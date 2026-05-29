package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"marcceljanara/wallet-ledger-service/internal/config"
	"marcceljanara/wallet-ledger-service/internal/handler"
	"marcceljanara/wallet-ledger-service/internal/middleware"
	"marcceljanara/wallet-ledger-service/internal/repository"
	"marcceljanara/wallet-ledger-service/internal/service"
	"marcceljanara/wallet-ledger-service/internal/utils"
	"marcceljanara/wallet-ledger-service/internal/worker"
)

func main() {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// 2. Initialize infrastructure
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2a. PostgreSQL pool
	pool, err := config.NewDatabasePool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to initialize database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// 2b. Run migrations
	err = config.RunMigrations(cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// 2c. Redis client
	redisClient, err := config.NewRedisClient(ctx, cfg.RedisURL)
	if err != nil {
		slog.Error("Failed to initialize Redis client", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// 2d. RabbitMQ
	rabbitConn, rabbitCh, err := config.NewRabbitMQConnection(cfg.RabbitMQURL)
	if err != nil {
		slog.Error("Failed to initialize RabbitMQ connection", "error", err)
		os.Exit(1)
	}
	defer rabbitCh.Close()
	defer rabbitConn.Close()

	safeRabbitCh := utils.NewSafeChannel(rabbitCh)

	// 3. Initialize repositories
	userRepo := repository.NewUserRepository(pool)
	walletRepo := repository.NewWalletRepository(pool)
	transactionRepo := repository.NewTransactionRepository(pool)
	ledgerRepo := repository.NewLedgerRepository(pool)
	auditRepo := repository.NewAuditRepository(pool)

	// 4. Initialize services
	auditService := service.NewAuditService(auditRepo)
	authService := service.NewAuthService(userRepo, walletRepo, pool, cfg, safeRabbitCh)
	walletService := service.NewWalletService(walletRepo)
	transactionService := service.NewTransactionService(walletRepo, transactionRepo, ledgerRepo, pool, safeRabbitCh)
	ledgerService := service.NewLedgerService(walletRepo, ledgerRepo)

	// 5. Initialize handlers
	authHandler := handler.NewAuthHandler(authService, int(cfg.JWTExpiration.Seconds()), cfg.CookieSecure)
	walletHandler := handler.NewWalletHandler(walletService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	ledgerHandler := handler.NewLedgerHandler(ledgerService)
	auditHandler := handler.NewAuditHandler(auditService)

	// 6. Start RabbitMQ workers
	auditWorker := worker.NewAuditWorker(rabbitCh, auditService)
	notifWorker := worker.NewNotificationWorker(rabbitCh)
	analyticsWorker := worker.NewAnalyticsWorker(rabbitCh)
	go auditWorker.Start(ctx)
	go notifWorker.Start(ctx)
	go analyticsWorker.Start(ctx)

	// 7. Configure Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(gin.Logger())
	r.Use(middleware.RateLimit(redisClient, 100, time.Minute))

	v1 := r.Group("/api/v1")
	{
		// Public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(cfg.JWTSecret))
		protected.Use(middleware.AuditLog(safeRabbitCh))
		{
			protected.POST("/auth/logout", authHandler.Logout)

			protected.GET("/wallets/me", walletHandler.GetMyWallet)

			// Endpoints with idempotency
			idempotent := protected.Group("")
			idempotent.Use(middleware.Idempotency(redisClient))
			{
				idempotent.POST("/wallets/topup", transactionHandler.TopUp)
				idempotent.POST("/transfers", transactionHandler.Transfer)
			}

			protected.GET("/transactions", transactionHandler.GetTransactions)
			protected.GET("/transactions/:transactionId", transactionHandler.GetTransactionDetail)

			protected.GET("/ledger/entries", ledgerHandler.GetLedgerEntries)

			protected.GET("/audit/logs", auditHandler.GetAuditLogs)

			// Admin-only routes
			admin := protected.Group("/admin")
			admin.Use(middleware.RoleGuard("ADMIN"))
			{
				admin.GET("/users", authHandler.GetUsersAdmin)
				admin.GET("/transactions", transactionHandler.GetTransactionsAdmin)
				admin.GET("/transactions/:transactionId", transactionHandler.GetTransactionDetailAdmin)
				admin.GET("/audit-logs", auditHandler.GetAuditLogsAdmin)
			}
		}
	}

	// 8. Start HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	go func() {
		slog.Info("Server starting", "port", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 9. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
	cancel() // stops workers
	slog.Info("Server exited")
}
