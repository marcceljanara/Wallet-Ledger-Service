package worker

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ProcessMessageForTest exposes the unexported processMessage method for use in
// external test packages (package worker_test). This file is compiled only during
// testing and does not affect the production binary.
func (w *NotificationWorker) ProcessMessageForTest(ctx context.Context, msg amqp.Delivery) error {
	return w.processMessage(ctx, msg)
}
