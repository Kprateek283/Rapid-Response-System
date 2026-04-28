package mq

import (
	"context"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher wraps an AMQP channel for publishing messages.
type Publisher struct {
	ch *amqp.Channel
}

// NewPublisher creates a new RabbitMQ publisher and declares the exchange.
func NewPublisher(conn *amqp.Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare the topic exchange
	err = ch.ExchangeDeclare(
		"rapid_incidents", // exchange name
		"topic",           // type
		true,              // durable
		false,             // auto-deleted
		false,             // internal
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	log.Println("[MQ] Publisher initialized with exchange 'rapid_incidents'")
	return &Publisher{ch: ch}, nil
}

// Publish sends a message to the exchange with the given routing key.
func (p *Publisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	return p.ch.PublishWithContext(ctx,
		"rapid_incidents", // exchange
		routingKey,        // routing key
		false,             // mandatory
		false,             // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

// Close closes the publisher channel.
func (p *Publisher) Close() error {
	return p.ch.Close()
}
