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
	"marcceljanara/wallet-ledger-service/internal/utils"
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

func TestAuthService_Login_Success(t *testing.T) {
	mockUserRepo := mocks.NewUserRepository(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	cfg := &config.Config{
		JWTSecret:     "secret",
		JWTExpiration: 10 * time.Minute,
	}

	userID := uuid.New()
	pwdHash, _ := utils.HashPassword("password123")

	mockUserRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(&model.User{
		ID:           userID,
		Email:        "test@example.com",
		PasswordHash: pwdHash,
		Role:         model.UserRoleUser,
	}, nil)

	svc := service.NewAuthService(mockUserRepo, mockWalletRepo, nil, cfg, nil)

	res, token, err := svc.Login(context.Background(), dto.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.NotEmpty(t, token)
	assert.Equal(t, "test@example.com", res.Email)
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	mockUserRepo := mocks.NewUserRepository(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	pwdHash, _ := utils.HashPassword("password123")

	mockUserRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(&model.User{
		ID:           uuid.New(),
		Email:        "test@example.com",
		PasswordHash: pwdHash,
		Role:         model.UserRoleUser,
	}, nil)

	svc := service.NewAuthService(mockUserRepo, mockWalletRepo, nil, nil, nil)

	_, _, err := svc.Login(context.Background(), dto.LoginRequest{
		Email:    "test@example.com",
		Password: "wrongpassword",
	})

	assert.ErrorIs(t, err, service.ErrInvalidCredentials)
}

func TestAuthService_GetAllUsers_Success(t *testing.T) {
	mockUserRepo := mocks.NewUserRepository(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	userID := uuid.New()
	mockUsers := []model.UserWithWallet{
		{
			User: model.User{
				ID:        userID,
				Email:     "user1@example.com",
				Role:      model.UserRoleUser,
				CreatedAt: time.Now(),
			},
			WalletID:      "WLT-USER1",
			WalletBalance: decimal.NewFromInt(100),
		},
	}

	mockUserRepo.On("FindAll", mock.Anything, 10, 0).Return(mockUsers, 1, nil)

	svc := service.NewAuthService(mockUserRepo, mockWalletRepo, nil, nil, nil)

	res, err := svc.GetAllUsers(context.Background(), dto.PaginationRequest{
		Page:  1,
		Limit: 10,
	})

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Len(t, res.Users, 1)
	assert.Equal(t, "user1@example.com", res.Users[0].Email)
	assert.Equal(t, "WLT-USER1", res.Users[0].WalletID)
	assert.Equal(t, decimal.NewFromInt(100), res.Users[0].Balance)
	assert.Equal(t, 1, res.Pagination.TotalItems)
}
