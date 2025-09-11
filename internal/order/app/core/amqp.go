package core

import (
	"context"

	"wheres-my-pizza/internal/order/domain/dto"
)

type IRabbitMQ interface {
	Close() error
	PushMessage(ctx context.Context, message dto.OrderRequest) error
}
