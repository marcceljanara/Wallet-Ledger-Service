package worker

import (
	"context"
	"log/slog"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

type AnalyticsWorker struct {
	rabbitCh *utils.SafeChannel
}

func NewAnalyticsWorker(rabbitCh *utils.SafeChannel) *AnalyticsWorker {
	return &AnalyticsWorker{
		rabbitCh: rabbitCh,
	}
}

func (w *AnalyticsWorker) Start(ctx context.Context) {
	slog.Info("Analytics worker starting")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Analytics worker shutting down")
			return
		default:
		}

		ch, err := w.rabbitCh.NewChannel()
		if err != nil {
			slog.Error("Analytics worker failed to create channel, retrying...", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		slog.Info("Analytics worker acquired channel, entering consume loop")
		w.consumeLoop(ctx, ch)
	}
}

func (w *AnalyticsWorker) consumeLoop(ctx context.Context, ch *amqp.Channel) {
	defer ch.Close()

	msgs, err := ch.Consume(
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

	for {
		select {
		case <-ctx.Done():
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
