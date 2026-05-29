package worker

import (
	"context"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationWorker struct {
	channel *amqp.Channel
}

func NewNotificationWorker(channel *amqp.Channel) *NotificationWorker {
	return &NotificationWorker{
		channel: channel,
	}
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

			slog.Info("Notification event received", "event", string(msg.Body))
			_ = msg.Ack(false)
		}
	}
}
