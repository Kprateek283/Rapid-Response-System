package mq

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// SetupQueues declares the exchange and all queues with bindings.
// Called once at application startup.
func SetupQueues(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel for setup: %w", err)
	}
	defer ch.Close()

	// Declare the topic exchange
	err = ch.ExchangeDeclare(
		"rapid_incidents",
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Queue definitions: name -> routing key
	queues := map[string]string{
		"ai_assessment_queue":      "incident.media_ready",
		"ground_truth_queue":       "incident.ground_truth",
		"report_compilation_queue": "incident.compile_report",
	}

	for queueName, routingKey := range queues {
		_, err := ch.QueueDeclare(
			queueName,
			true,  // durable
			false, // auto-delete
			false, // exclusive
			false, // no-wait
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queueName, err)
		}

		err = ch.QueueBind(
			queueName,
			routingKey,
			"rapid_incidents",
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to bind queue %s: %w", queueName, err)
		}

		log.Printf("[MQ] Queue '%s' bound to key '%s'", queueName, routingKey)
	}

	return nil
}
