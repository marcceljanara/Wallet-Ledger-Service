package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"marcceljanara/wallet-ledger-service/internal/dto"
	"marcceljanara/wallet-ledger-service/internal/mocks"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/service"
)

func TestTransactionService_TopUp_Success(t *testing.T) {
	mockWalletRepo := mocks.NewWalletRepository(t)
	mockTxnRepo := mocks.NewTransactionRepository(t)
	mockLedgerRepo := mocks.NewLedgerRepository(t)

	userID := uuid.New()
	walletID := "WLT-USER"

	mockWalletRepo.On("FindByUserID", mock.Anything, userID).Return(&model.Wallet{
		ID:      walletID,
		UserID:  userID,
		Balance: decimal.NewFromInt(1000),
	}, nil)

	mockTxInstance := &mockTx{}
	mockPool := &mockTxBeginner{
		beginFunc: func(ctx context.Context) (pgx.Tx, error) {
			return mockTxInstance, nil
		},
	}

	mockWalletRepo.On("FindByIDForUpdate", mock.Anything, mockTxInstance, walletID).Return(&model.Wallet{
		ID:      walletID,
		UserID:  userID,
		Balance: decimal.NewFromInt(1000),
	}, nil)

	txnID := uuid.New()
	mockTxnRepo.On("Create", mock.Anything, mockTxInstance, mock.Anything).Return(&model.Transaction{
		ID:          txnID,
		ReferenceNo: "TX-REF",
		Type:        model.TransactionTypeTopUp,
		Status:      model.TransactionStatusPending,
		Amount:      decimal.NewFromInt(500),
	}, nil)

	mockWalletRepo.On("UpdateBalance", mock.Anything, mockTxInstance, walletID, decimal.NewFromInt(1500)).Return(nil)

	mockLedgerRepo.On("Create", mock.Anything, mockTxInstance, mock.Anything).Return(&model.LedgerEntry{}, nil)

	mockTxnRepo.On("UpdateStatus", mock.Anything, mockTxInstance, txnID, model.TransactionStatusCompleted).Return(nil)

	svc := service.NewTransactionService(mockWalletRepo, mockTxnRepo, mockLedgerRepo, mockPool, nil)

	res, err := svc.TopUp(context.Background(), userID, dto.TopUpRequest{
		Amount: decimal.NewFromInt(500),
	})

	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, decimal.NewFromInt(1500), res.BalanceAfter)
	assert.Equal(t, walletID, res.WalletID)
}
