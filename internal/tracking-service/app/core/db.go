package core

import (
	"context"

	"where-is-my-pizza/internal/tracking/domain/models"

	"github.com/jackc/pgx/v5"
)

type IDB interface {
	Close() error
	IsAlive() error
	GetConn() *pgx.Conn
}

type IOrderRepo interface {
	GetStatus(ctx context.Context, orderNumber string) (models.Order, error)
	GetHistory(ctx context.Context, orderNumber string) ([]models.OrderStatusLog, error)
}

type IWorkerRepo interface {
	GetStatusOfAllWorkers(ctx context.Context) ([]models.Worker, error)
}
