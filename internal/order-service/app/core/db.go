package core

import (
	"context"

	"where-is-my-pizza/internal/order/domain/dto"
	"where-is-my-pizza/internal/order/domain/models"

	"github.com/jackc/pgx/v5"
)

type IDB interface {
	Close() error
	IsAlive() error
	GetConn() *pgx.Conn
}

type IOrderRepo interface {
	Create(ctx context.Context, order dto.OrderRequest) (models.Order, error)
	Delete(ctx context.Context, order dto.OrderRequest) error
}
