package core

import (
	"context"
	"wheres-my-pizza/internal/xpkg/models"

	"github.com/jackc/pgx/v5"
)

type IDB interface {
	Close() error
	GetConn() *pgx.Conn
	IsAlive() error
	Reconnect() error
}

type IOrderRepo interface {
	GetStatus(ctx context.Context, roderNumber string) (string, error)
	// transactional requests for orders -> orders_log
	SetStatusCooking(ctx context.Context, orderNumber, processBy string) error
	SetStatusReady(ctx context.Context, orderNumber, processBy string) error
	SetStatusCancelled(ctx context.Context, orderNumber string) error
}


type IWorkerRepo interface{
	Get(ctx context.Context, name string) (models.Worker, error)
	Create(ctx context.Context, w models.Worker) (string, error)
	SetOnline(ctx context.Context, name string) error
	SetOffline(ctx context.Context, name string) error
	UpdateLastSeen(ctx context.Context, name string) error
	UpdateType(ctx context.Context, name, newType string) error 
}