package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"time"

	"github.com/shopspring/decimal"
	"marcceljanara/wallet-ledger-service/internal/config"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/repository"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

func main() {
	emailFlag := flag.String("email", "admin@example.com", "Admin email address")
	passwordFlag := flag.String("password", "admin123", "Admin password")
	flag.Parse()

	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// 2. Connect to database
	pool, err := config.NewDatabasePool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	walletRepo := repository.NewWalletRepository(pool)

	// 3. Check if user already exists
	existing, err := userRepo.FindByEmail(ctx, *emailFlag)
	if err != nil {
		slog.Error("Failed to query existing user", "error", err)
		os.Exit(1)
	}
	if existing != nil {
		slog.Info("Admin user already exists", "email", *emailFlag)
		return
	}

	// 4. Hash password
	pwdHash, err := utils.HashPassword(*passwordFlag)
	if err != nil {
		slog.Error("Failed to hash password", "error", err)
		os.Exit(1)
	}

	// 5. Create user and wallet in transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		slog.Error("Failed to begin transaction", "error", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx)

	user, err := userRepo.Create(ctx, tx, *emailFlag, pwdHash, model.UserRoleAdmin)
	if err != nil {
		slog.Error("Failed to create admin user", "error", err)
		os.Exit(1)
	}

	walletID, err := utils.GenerateWalletID()
	if err != nil {
		slog.Error("Failed to generate wallet ID", "error", err)
		os.Exit(1)
	}

	wallet := &model.Wallet{
		ID:        walletID,
		UserID:    user.ID,
		Balance:   decimal.Zero,
		Currency:  "IDR",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = walletRepo.Create(ctx, tx, wallet)
	if err != nil {
		slog.Error("Failed to create admin wallet", "error", err)
		os.Exit(1)
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("Failed to commit transaction", "error", err)
		os.Exit(1)
	}

	slog.Info("Admin user and wallet seeded successfully", "email", user.Email, "wallet_id", walletID)
}
