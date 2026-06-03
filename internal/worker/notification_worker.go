package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/shopspring/decimal"
	"marcceljanara/wallet-ledger-service/internal/repository"
	"marcceljanara/wallet-ledger-service/internal/service"
)

type NotificationWorker struct {
	channel    *amqp.Channel
	notifServ  service.NotificationService
	walletRepo repository.WalletRepository
}

func NewNotificationWorker(
	channel *amqp.Channel,
	notifServ service.NotificationService,
	walletRepo repository.WalletRepository,
) *NotificationWorker {
	return &NotificationWorker{
		channel:    channel,
		notifServ:  notifServ,
		walletRepo: walletRepo,
	}
}

type RegisterEventData struct {
	WalletID string `json:"wallet_id"`
}

type TopUpEventData struct {
	TransactionID string          `json:"transaction_id"`
	ReferenceNo   string          `json:"reference_no"`
	WalletID      string          `json:"wallet_id"`
	Amount        decimal.Decimal `json:"amount"`
	BalanceAfter  decimal.Decimal `json:"balance_after"`
}

type TransferEventData struct {
	TransactionID  string          `json:"transaction_id"`
	ReferenceNo    string          `json:"reference_no"`
	SourceWalletID string          `json:"source_wallet_id"`
	TargetWalletID string          `json:"target_wallet_id"`
	Amount         decimal.Decimal `json:"amount"`
	BalanceAfter   decimal.Decimal `json:"balance_after"`
}

func (w *NotificationWorker) Start(ctx context.Context) {
	msgs, err := w.channel.Consume(
		"notification_queue",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		slog.Error("failed to consume notification queue", "error", err)
		return
	}

	slog.Info("Notification worker started successfully")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Notification worker shutting down")
			return
		case msg, ok := <-msgs:
			if !ok {
				slog.Info("Notification queue channel closed")
				return
			}

			if err := w.processMessage(ctx, msg); err != nil {
				slog.Error("failed to process notification message", "error", err)
				if nackErr := msg.Nack(false, true); nackErr != nil {
					slog.Error("failed to nack notification message", "error", nackErr)
				}
			} else {
				if ackErr := msg.Ack(false); ackErr != nil {
					slog.Error("failed to ack notification message", "error", ackErr)
				}
			}
		}
	}
}

func (w *NotificationWorker) processMessage(ctx context.Context, msg amqp.Delivery) error {
	var evt WalletEventMsg
	if err := json.Unmarshal(msg.Body, &evt); err != nil {
		return fmt.Errorf("failed to unmarshal notification event: %w", err)
	}

	userID, err := uuid.Parse(evt.UserID)
	if err != nil && evt.EventType != "AUDIT" {
		return fmt.Errorf("invalid user id: %w", err)
	}

	switch evt.EventType {
	case "REGISTER":
		var data RegisterEventData
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			return fmt.Errorf("failed to unmarshal register data: %w", err)
		}
		title := "Pendaftaran Berhasil"
		message := fmt.Sprintf("Selamat datang! Dompet Anda dengan ID %s telah berhasil dibuat.", data.WalletID)
		_, err = w.notifServ.CreateAndPushNotification(ctx, userID, title, message)
		if err != nil {
			return fmt.Errorf("failed to create register notification: %w", err)
		}

	case "LOGIN":
		title := "Login Berhasil"
		message := "Anda baru saja masuk ke akun Anda."
		_, err = w.notifServ.CreateAndPushNotification(ctx, userID, title, message)
		if err != nil {
			return fmt.Errorf("failed to create login notification: %w", err)
		}

	case "TOPUP":
		var data TopUpEventData
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			return fmt.Errorf("failed to unmarshal topup data: %w", err)
		}
		title := "Top-up Berhasil"
		message := fmt.Sprintf("Top-up sebesar Rp %s berhasil. Ref: %s.", data.Amount.String(), data.ReferenceNo)
		_, err = w.notifServ.CreateAndPushNotification(ctx, userID, title, message)
		if err != nil {
			return fmt.Errorf("failed to create topup notification: %w", err)
		}

	case "TRANSFER":
		var data TransferEventData
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			return fmt.Errorf("failed to unmarshal transfer data: %w", err)
		}

		// 1. Send notification to the Sender
		titleSender := "Transfer Berhasil"
		msgSender := fmt.Sprintf("Transfer sebesar Rp %s ke dompet %s berhasil. Ref: %s.", data.Amount.String(), data.TargetWalletID, data.ReferenceNo)
		_, err = w.notifServ.CreateAndPushNotification(ctx, userID, titleSender, msgSender)
		if err != nil {
			slog.Error("failed to create transfer sender notification", "error", err)
		}

		// 2. Fetch Receiver Wallet to get Receiver's User ID
		targetWallet, err := w.walletRepo.FindByID(ctx, data.TargetWalletID)
		if err != nil {
			return fmt.Errorf("failed to find target wallet: %w", err)
		}
		if targetWallet == nil {
			slog.Warn("target wallet not found for transfer notification", "wallet_id", data.TargetWalletID)
			break
		}

		// 3. Send notification to the Receiver
		titleReceiver := "Transfer Diterima"
		msgReceiver := fmt.Sprintf("Anda menerima transfer sebesar Rp %s dari dompet %s. Ref: %s.", data.Amount.String(), data.SourceWalletID, data.ReferenceNo)
		_, err = w.notifServ.CreateAndPushNotification(ctx, targetWallet.UserID, titleReceiver, msgReceiver)
		if err != nil {
			slog.Error("failed to create transfer receiver notification", "error", err)
		}

	case "AUDIT":
		// Audit events do not trigger user visible push notifications, skip.
	}

	return nil
}
