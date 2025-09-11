package core

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type IRabbitMQ interface {
	Close() error
	ConsumeMessage(ctx context.Context, queue string, workerName string) (<-chan amqp.Delivery, error)
}
