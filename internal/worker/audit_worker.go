package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"marcceljanara/wallet-ledger-service/internal/model"
	"marcceljanara/wallet-ledger-service/internal/service"
)

type AuditWorker struct {
	channel      *amqp.Channel
	auditService service.AuditService
}

func NewAuditWorker(channel *amqp.Channel, auditService service.AuditService) *AuditWorker {
	return &AuditWorker{
		channel:      channel,
		auditService: auditService,
	}
}

type WalletEventMsg struct {
	EventType string          `json:"event_type"`
	UserID    string          `json:"user_id"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
	IPAddress string          `json:"ip_address"`
	Endpoint  string          `json:"endpoint"`
}

func (w *AuditWorker) Start(ctx context.Context) {
	msgs, err := w.channel.Consume(
		"audit_queue",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		slog.Error("failed to consume audit queue", "error", err)
		return
	}

	slog.Info("Audit worker started successfully")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Audit worker shutting down")
			return
		case msg, ok := <-msgs:
			if !ok {
				slog.Info("Audit queue channel closed")
				return
			}

			if err := w.processMessage(ctx, msg); err != nil {
				slog.Error("failed to process audit message", "error", err)
				_ = msg.Nack(false, true)
			} else {
				_ = msg.Ack(false)
			}
		}
	}
}

func (w *AuditWorker) processMessage(ctx context.Context, msg amqp.Delivery) error {
	var evt WalletEventMsg
	if err := json.Unmarshal(msg.Body, &evt); err != nil {
		return fmt.Errorf("failed to unmarshal wallet event message: %w", err)
	}

	uid, err := uuid.Parse(evt.UserID)
	if err != nil {
		return fmt.Errorf("invalid user id in event: %w", err)
	}

	action := evt.EventType
	ipAddress := evt.IPAddress
	endpoint := evt.Endpoint

	if evt.EventType == "AUDIT" {
		var auditData struct {
			UserID    string `json:"user_id"`
			Action    string `json:"action"`
			IPAddress string `json:"ip_address"`
			Endpoint  string `json:"endpoint"`
		}
		if err := json.Unmarshal(evt.Data, &auditData); err == nil {
			if auditData.Action != "" {
				action = auditData.Action
			}
			if auditData.IPAddress != "" {
				ipAddress = auditData.IPAddress
			}
			if auditData.Endpoint != "" {
				endpoint = auditData.Endpoint
			}
		}
	}

	log := &model.AuditLog{
		UserID:    uid,
		Action:    action,
		IPAddress: ipAddress,
		Endpoint:  endpoint,
		CreatedAt: evt.Timestamp,
	}

	return w.auditService.CreateLog(ctx, log)
}
