package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shopspring/decimal"
	"marcceljanara/wallet-ledger-service/internal/config"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/repository"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

var (
	ErrEmailConflict      = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, string, error)
	GetAllUsers(ctx context.Context, pagination dto.PaginationRequest) (*dto.AdminUserListResponse, error)
}
type authService struct {
	userRepo   repository.UserRepository
	walletRepo repository.WalletRepository
	pool       TxBeginner
	cfg        *config.Config
	rabbitCh   *utils.SafeChannel
}

func NewAuthService(
	userRepo repository.UserRepository,
	walletRepo repository.WalletRepository,
	pool TxBeginner,
	cfg *config.Config,
	rabbitCh *utils.SafeChannel,
) AuthService {
	return &authService{
		userRepo:   userRepo,
		walletRepo: walletRepo,
		pool:       pool,
		cfg:        cfg,
		rabbitCh:   rabbitCh,
	}
}

func (s *authService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResponse, error) {
	existingUser, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, ErrEmailConflict
	}

	pwdHash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	user, err := s.userRepo.Create(ctx, tx, req.Email, pwdHash, model.UserRoleUser)
	if err != nil {
		return nil, err
	}

	walletID, err := utils.GenerateWalletID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate wallet ID: %w", err)
	}
	wallet := &model.Wallet{
		ID:        walletID,
		UserID:    user.ID,
		Balance:   decimal.Zero,
		Currency:  "IDR",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	_, err = s.walletRepo.Create(ctx, tx, wallet)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		eventData := map[string]interface{}{
			"wallet_id": walletID,
		}
		if err := publishEvent(bgCtx, s.rabbitCh, "REGISTER", user.ID.String(), "", "", eventData); err != nil {
			slog.Error("failed to publish register event to rabbitmq", "error", err)
		}
	}()

	return &dto.RegisterResponse{
		UserID:    user.ID,
		Email:     user.Email,
		WalletID:  walletID,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *authService) Login(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, string, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", ErrInvalidCredentials
	}

	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		return nil, "", ErrInvalidCredentials
	}

	token, err := utils.GenerateToken(user.ID, user.Email, string(user.Role), s.cfg.JWTSecret, s.cfg.JWTExpiration)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := publishEvent(bgCtx, s.rabbitCh, "LOGIN", user.ID.String(), "", "", nil); err != nil {
			slog.Error("failed to publish login event to rabbitmq", "error", err)
		}
	}()

	return &dto.LoginResponse{
		UserID: user.ID,
		Email:  user.Email,
		Role:   string(user.Role),
	}, token, nil
}

func (s *authService) GetAllUsers(ctx context.Context, pagination dto.PaginationRequest) (*dto.AdminUserListResponse, error) {
	pagination.SetDefaults()

	users, total, err := s.userRepo.FindAll(ctx, pagination.Limit, pagination.Offset())
	if err != nil {
		return nil, err
	}

	adminUsers := make([]dto.AdminUserResponse, 0, len(users))
	for _, u := range users {
		adminUsers = append(adminUsers, dto.AdminUserResponse{
			UserID:    u.User.ID,
			Email:     u.User.Email,
			Role:      string(u.User.Role),
			WalletID:  u.WalletID,
			Balance:   u.WalletBalance,
			CreatedAt: u.User.CreatedAt,
		})
	}

	totalPages := (total + pagination.Limit - 1) / pagination.Limit

	return &dto.AdminUserListResponse{
		Users: adminUsers,
		Pagination: dto.PaginationResponse{
			CurrentPage: pagination.Page,
			PerPage:     pagination.Limit,
			TotalItems:  total,
			TotalPages:  totalPages,
		},
	}, nil
}

func publishEvent(ctx context.Context, ch *utils.SafeChannel, eventType string, userID string, ipAddress, endpoint string, data any) error {
	if ch == nil {
		return nil
	}
	evt := struct {
		EventType string    `json:"event_type"`
		UserID    string    `json:"user_id"`
		Data      any       `json:"data"`
		Timestamp time.Time `json:"timestamp"`
		IPAddress string    `json:"ip_address"`
		Endpoint  string    `json:"endpoint"`
	}{
		EventType: eventType,
		UserID:    userID,
		Data:      data,
		Timestamp: time.Now(),
		IPAddress: ipAddress,
		Endpoint:  endpoint,
	}

	body, err := json.Marshal(evt)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(ctx,
		"wallet_events",
		"wallet.event."+eventType,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
