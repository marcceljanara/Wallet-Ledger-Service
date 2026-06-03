package utils

import (
	"context"
	"fmt"
	"sync"

	"marcceljanara/wallet-ledger-service/internal/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

// SafeChannel wraps amqp.Connection and amqp.Channel to manage connections and concurrent publishing.
type SafeChannel struct {
	mu        sync.Mutex
	url       string
	conn      *amqp.Connection
	ch        *amqp.Channel
	connClose chan *amqp.Error
	chClose   chan *amqp.Error
}

// NewSafeChannel creates a new SafeChannel connection/channel manager.
func NewSafeChannel(url string) *SafeChannel {
	return &SafeChannel{url: url}
}

// EnsureConnected checks if the connection and channel are open, and if not, reconnects and redeclares everything.
func (s *SafeChannel) EnsureConnected() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.ensureConnectedLocked()
}

func (s *SafeChannel) ensureConnectedLocked() error {
	// Check if conn has closed asynchronously
	if s.connClose != nil {
		select {
		case <-s.connClose:
			s.conn = nil
		default:
		}
	}
	// Check if ch has closed asynchronously
	if s.chClose != nil {
		select {
		case <-s.chClose:
			s.ch = nil
		default:
		}
	}

	if s.conn == nil || s.conn.IsClosed() {
		if s.ch != nil {
			_ = s.ch.Close()
		}
		if s.conn != nil {
			_ = s.conn.Close()
		}
		s.ch = nil
		s.conn = nil

		conn, err := amqp.Dial(s.url)
		if err != nil {
			return fmt.Errorf("failed to connect to rabbitmq: %w", err)
		}
		s.conn = conn
		s.connClose = make(chan *amqp.Error, 1)
		s.conn.NotifyClose(s.connClose)
	}

	if s.ch == nil {
		ch, err := s.conn.Channel()
		if err != nil {
			return fmt.Errorf("failed to open channel: %w", err)
		}

		err = config.DeclareExchangeAndQueues(ch)
		if err != nil {
			_ = ch.Close()
			return fmt.Errorf("failed to declare exchange/queues: %w", err)
		}

		s.ch = ch
		s.chClose = make(chan *amqp.Error, 1)
		s.ch.NotifyClose(s.chClose)
	}

	return nil
}

// PublishWithContext sends a message to the exchange concurrently in a thread-safe manner.
// If the publish fails, it will attempt to reconnect once and retry publishing.
func (s *SafeChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Try to ensure we are connected
	if err := s.ensureConnectedLocked(); err != nil {
		return fmt.Errorf("publish failed, unable to connect: %w", err)
	}

	// First attempt
	err := s.ch.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
	if err == nil {
		return nil
	}

	// If publish failed, we mark the channel as nil to force recreation
	s.ch = nil
	if err := s.ensureConnectedLocked(); err != nil {
		return fmt.Errorf("retry publish failed, unable to reconnect: %w", err)
	}

	// Second attempt after reconnecting
	err = s.ch.PublishWithContext(ctx, exchange, key, mandatory, immediate, msg)
	if err != nil {
		return fmt.Errorf("failed to publish after reconnect: %w", err)
	}

	return nil
}

// NewChannel creates a new, independent channel using the current connection.
// It will ensure the connection is open first.
func (s *SafeChannel) NewChannel() (*amqp.Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureConnectedLocked(); err != nil {
		return nil, err
	}

	ch, err := s.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open new channel: %w", err)
	}

	return ch, nil
}

// Channel returns the underlying raw channel thread-safely.
func (s *SafeChannel) Channel() *amqp.Channel {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ch
}

// Close closes the underlying connection and channel.
func (s *SafeChannel) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ch != nil {
		_ = s.ch.Close()
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
	s.ch = nil
	s.conn = nil
}
