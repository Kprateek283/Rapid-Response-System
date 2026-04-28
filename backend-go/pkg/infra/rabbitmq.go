package infra

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func NewRabbitMQConnection(dsn string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	log.Println("Successfully connected to RabbitMQ")
	return conn, nil
}
