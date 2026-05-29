package config

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewRabbitMQConnection(rabbitMQURL string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(rabbitMQURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	// Declare topic exchange
	err = ch.ExchangeDeclare(
		"wallet_events", // name
		"topic",         // type
		true,            // durable
		false,           // auto-deleted
		false,           // internal
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	queues := []string{
		"notification_queue",
		"audit_queue",
		"analytics_queue",
	}

	for _, qName := range queues {
		_, err := ch.QueueDeclare(
			qName, // name
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			ch.Close()
			conn.Close()
			return nil, nil, fmt.Errorf("failed to declare queue %s: %w", qName, err)
		}

		err = ch.QueueBind(
			qName,            // queue name
			"wallet.event.#", // routing key
			"wallet_events",  // exchange
			false,
			nil,
		)
		if err != nil {
			ch.Close()
			conn.Close()
			return nil, nil, fmt.Errorf("failed to bind queue %s: %w", qName, err)
		}
	}

	return conn, ch, nil
}
