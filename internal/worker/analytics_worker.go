package worker

import (
	"context"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AnalyticsWorker struct {
	channel *amqp.Channel
}

func NewAnalyticsWorker(channel *amqp.Channel) *AnalyticsWorker {
	return &AnalyticsWorker{
		channel: channel,
	}
}

func (w *AnalyticsWorker) Start(ctx context.Context) {
	msgs, err := w.channel.Consume(
		"analytics_queue",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		slog.Error("failed to consume analytics queue", "error", err)
		return
	}

	slog.Info("Analytics worker started successfully")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Analytics worker shutting down")
			return
		case msg, ok := <-msgs:
			if !ok {
				slog.Info("Analytics queue channel closed")
				return
			}

			slog.Info("Analytics event received", "event", string(msg.Body))
			_ = msg.Ack(false)
		}
	}
}
