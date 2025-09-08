package core

import (
	"context"
	"wheres-my-pizza/internal/kitchen/domain/dto"

	amqp "github.com/rabbitmq/amqp091-go"
)

type IRabbitMQ interface {
	IsAlive() error
	Close() error
	PushMessage(ctx context.Context, message dto.OrderMessage) error
	ConsumeMessage(ctx context.Context, queue string, workerName string) (<-chan amqp.Delivery, error)
}
