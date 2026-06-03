package worker_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"marcceljanara/wallet-ledger-service/internal/mocks"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/worker"
)

// buildDelivery creates a fake amqp.Delivery with a JSON-marshalled body.
func buildDelivery(t *testing.T, eventType, userID string, data any) amqp.Delivery {
	t.Helper()
	rawData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("buildDelivery: failed to marshal data: %v", err)
	}

	type envelope struct {
		EventType string          `json:"event_type"`
		UserID    string          `json:"user_id"`
		Data      json.RawMessage `json:"data"`
	}

	body, err := json.Marshal(envelope{
		EventType: eventType,
		UserID:    userID,
		Data:      json.RawMessage(rawData),
	})
	if err != nil {
		t.Fatalf("buildDelivery: failed to marshal envelope: %v", err)
	}
	return amqp.Delivery{Body: body}
}

func TestNotificationWorker_ProcessMessage_Register(t *testing.T) {
	mockNotifSvc := mocks.NewNotificationService(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	userID := uuid.New()
	walletID := "WLT-ABC123"

	mockNotifSvc.On(
		"CreateAndPushNotification",
		mock.Anything,
		userID,
		"Pendaftaran Berhasil",
		mock.MatchedBy(func(msg string) bool {
			// message should contain the wallet ID
			return len(msg) > 0 && containsString(msg, walletID)
		}),
	).Return(&model.Notification{}, nil)

	w := worker.NewNotificationWorker(nil, mockNotifSvc, mockWalletRepo)
	msg := buildDelivery(t, "REGISTER", userID.String(), map[string]string{"wallet_id": walletID})

	err := w.ProcessMessageForTest(context.Background(), msg)
	assert.NoError(t, err)
}

func TestNotificationWorker_ProcessMessage_Login(t *testing.T) {
	mockNotifSvc := mocks.NewNotificationService(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	userID := uuid.New()

	mockNotifSvc.On(
		"CreateAndPushNotification",
		mock.Anything,
		userID,
		"Login Berhasil",
		"Anda baru saja masuk ke akun Anda.",
	).Return(&model.Notification{}, nil)

	w := worker.NewNotificationWorker(nil, mockNotifSvc, mockWalletRepo)
	msg := buildDelivery(t, "LOGIN", userID.String(), map[string]string{})

	err := w.ProcessMessageForTest(context.Background(), msg)
	assert.NoError(t, err)
}

func TestNotificationWorker_ProcessMessage_TopUp(t *testing.T) {
	mockNotifSvc := mocks.NewNotificationService(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	userID := uuid.New()
	amount := decimal.NewFromInt(500000)
	refNo := "TXN-20260603-ABC123"

	mockNotifSvc.On(
		"CreateAndPushNotification",
		mock.Anything,
		userID,
		"Top-up Berhasil",
		mock.MatchedBy(func(msg string) bool {
			return containsString(msg, amount.String()) && containsString(msg, refNo)
		}),
	).Return(&model.Notification{}, nil)

	w := worker.NewNotificationWorker(nil, mockNotifSvc, mockWalletRepo)
	msg := buildDelivery(t, "TOPUP", userID.String(), map[string]any{
		"transaction_id": uuid.New().String(),
		"reference_no":   refNo,
		"wallet_id":      "WLT-SENDER",
		"amount":         amount,
		"balance_after":  decimal.NewFromInt(1500000),
	})

	err := w.ProcessMessageForTest(context.Background(), msg)
	assert.NoError(t, err)
}

func TestNotificationWorker_ProcessMessage_Transfer_BothParties(t *testing.T) {
	mockNotifSvc := mocks.NewNotificationService(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	senderUserID := uuid.New()
	receiverUserID := uuid.New()
	sourceWalletID := "WLT-SRC"
	targetWalletID := "WLT-TGT"
	amount := decimal.NewFromInt(250000)
	refNo := "TXN-20260603-XYZ"

	// Sender notification
	mockNotifSvc.On(
		"CreateAndPushNotification",
		mock.Anything,
		senderUserID,
		"Transfer Berhasil",
		mock.MatchedBy(func(msg string) bool {
			return containsString(msg, amount.String()) && containsString(msg, targetWalletID)
		}),
	).Return(&model.Notification{}, nil).Once()

	// Receiver wallet lookup
	mockWalletRepo.On("FindByID", mock.Anything, targetWalletID).
		Return(&model.Wallet{ID: targetWalletID, UserID: receiverUserID}, nil)

	// Receiver notification
	mockNotifSvc.On(
		"CreateAndPushNotification",
		mock.Anything,
		receiverUserID,
		"Transfer Diterima",
		mock.MatchedBy(func(msg string) bool {
			return containsString(msg, amount.String()) && containsString(msg, sourceWalletID)
		}),
	).Return(&model.Notification{}, nil).Once()

	w := worker.NewNotificationWorker(nil, mockNotifSvc, mockWalletRepo)
	msg := buildDelivery(t, "TRANSFER", senderUserID.String(), map[string]any{
		"transaction_id":   uuid.New().String(),
		"reference_no":     refNo,
		"source_wallet_id": sourceWalletID,
		"target_wallet_id": targetWalletID,
		"amount":           amount,
		"balance_after":    decimal.NewFromInt(750000),
	})

	err := w.ProcessMessageForTest(context.Background(), msg)
	assert.NoError(t, err)
}

func TestNotificationWorker_ProcessMessage_Transfer_TargetWalletNotFound(t *testing.T) {
	mockNotifSvc := mocks.NewNotificationService(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	senderUserID := uuid.New()
	targetWalletID := "WLT-NONEXISTENT"
	amount := decimal.NewFromInt(100000)

	// Sender notification still succeeds
	mockNotifSvc.On(
		"CreateAndPushNotification",
		mock.Anything,
		senderUserID,
		"Transfer Berhasil",
		mock.Anything,
	).Return(&model.Notification{}, nil).Once()

	// Receiver wallet is not found (nil)
	mockWalletRepo.On("FindByID", mock.Anything, targetWalletID).Return(nil, nil)

	w := worker.NewNotificationWorker(nil, mockNotifSvc, mockWalletRepo)
	msg := buildDelivery(t, "TRANSFER", senderUserID.String(), map[string]any{
		"transaction_id":   uuid.New().String(),
		"reference_no":     "REF-001",
		"source_wallet_id": "WLT-SRC",
		"target_wallet_id": targetWalletID,
		"amount":           amount,
		"balance_after":    decimal.NewFromInt(900000),
	})

	// Should not return an error even if target wallet is not found
	err := w.ProcessMessageForTest(context.Background(), msg)
	assert.NoError(t, err)
}

func TestNotificationWorker_ProcessMessage_Transfer_WalletRepoError(t *testing.T) {
	mockNotifSvc := mocks.NewNotificationService(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	senderUserID := uuid.New()
	targetWalletID := "WLT-TGT"
	amount := decimal.NewFromInt(100000)

	// Sender notification succeeds
	mockNotifSvc.On(
		"CreateAndPushNotification",
		mock.Anything,
		senderUserID,
		"Transfer Berhasil",
		mock.Anything,
	).Return(&model.Notification{}, nil).Once()

	// Wallet repo returns error
	repoErr := errors.New("db connection timeout")
	mockWalletRepo.On("FindByID", mock.Anything, targetWalletID).Return(nil, repoErr)

	w := worker.NewNotificationWorker(nil, mockNotifSvc, mockWalletRepo)
	msg := buildDelivery(t, "TRANSFER", senderUserID.String(), map[string]any{
		"transaction_id":   uuid.New().String(),
		"reference_no":     "REF-002",
		"source_wallet_id": "WLT-SRC",
		"target_wallet_id": targetWalletID,
		"amount":           amount,
		"balance_after":    decimal.NewFromInt(900000),
	})

	err := w.ProcessMessageForTest(context.Background(), msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find target wallet")
}

func TestNotificationWorker_ProcessMessage_Audit_Skipped(t *testing.T) {
	mockNotifSvc := mocks.NewNotificationService(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	// No notification service methods should be called for AUDIT events.
	w := worker.NewNotificationWorker(nil, mockNotifSvc, mockWalletRepo)
	msg := buildDelivery(t, "AUDIT", uuid.New().String(), map[string]string{})

	err := w.ProcessMessageForTest(context.Background(), msg)
	assert.NoError(t, err)
}

func TestNotificationWorker_ProcessMessage_InvalidJSON(t *testing.T) {
	mockNotifSvc := mocks.NewNotificationService(t)
	mockWalletRepo := mocks.NewWalletRepository(t)

	w := worker.NewNotificationWorker(nil, mockNotifSvc, mockWalletRepo)
	msg := amqp.Delivery{Body: []byte("not-valid-json")}

	err := w.ProcessMessageForTest(context.Background(), msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal notification event")
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
