package utils

import (
	"context"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
)

// SafeChannel wraps amqp.Channel to make concurrent publishing safe.
type SafeChannel struct {
	mu sync.Mutex
	ch *amqp.Channel
}

// NewSafeChannel creates a new SafeChannel.
func NewSafeChannel(ch *amqp.Channel) *SafeChannel {
	return &SafeChannel{ch: ch}
}

// PublishWithContext sends a message to the exchange concurrently in a thread-safe manner.
func (s *SafeChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	if s == nil || s.ch == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ch.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
}

// Channel returns the underlying raw channel.
func (s *SafeChannel) Channel() *amqp.Channel {
	if s == nil {
		return nil
	}
	return s.ch
}
