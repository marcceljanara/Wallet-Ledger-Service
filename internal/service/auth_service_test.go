package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"marcceljanara/wallet-ledger-service/internal/config"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/mocks"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/service"
)

type mockTx struct {
	pgx.Tx
}

func (m *mockTx) Commit(ctx context.Context) error   { return nil }
func (m *mockTx) Rollback(ctx context.Context) error { return nil }

type mockTxBeginner struct {
	beginFunc func(ctx context.Context) (pgx.Tx, error)
}

func (m *mockTxBeginner) Begin(ctx context.Context) (pgx.Tx, error) {
	return m.beginFunc(ctx)
}

func TestAuthService_Register_Success(t *testing.T) {
	mockUserRepo := mocks.NewUserRepository(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	cfg := &config.Config{
		JWTSecret:     "secret",
		JWTExpiration: 10 * time.Minute,
	}

	mockUserRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(nil, nil)

	mockTxInstance := &mockTx{}
	mockPool := &mockTxBeginner{
		beginFunc: func(ctx context.Context) (pgx.Tx, error) {
			return mockTxInstance, nil
		},
	}

	userID := uuid.New()
	mockUserRepo.On("Create", mock.Anything, mockTxInstance, "test@example.com", mock.Anything, model.UserRoleUser).
		Return(&model.User{
			ID:        userID,
			Email:     "test@example.com",
			Role:      model.UserRoleUser,
			CreatedAt: time.Now(),
		}, nil)

	mockWalletRepo.On("Create", mock.Anything, mockTxInstance, mock.Anything).
		Return(&model.Wallet{
			ID:        "WLT-TEST",
			UserID:    userID,
			Balance:   decimal.Zero,
			Currency:  "IDR",
			CreatedAt: time.Now(),
		}, nil)

	svc := service.NewAuthService(mockUserRepo, mockWalletRepo, mockPool, cfg, nil)

	res, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:           "test@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "test@example.com", res.Email)
	assert.Equal(t, userID, res.UserID)
}

func TestAuthService_Register_EmailConflict(t *testing.T) {
	mockUserRepo := mocks.NewUserRepository(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	mockUserRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(&model.User{Email: "test@example.com"}, nil)

	svc := service.NewAuthService(mockUserRepo, mockWalletRepo, nil, nil, nil)

	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:           "test@example.com",
		Password:        "password123",
		ConfirmPassword: "password123",
	})

	assert.ErrorIs(t, err, service.ErrEmailConflict)
}
