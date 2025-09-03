package interfaces

import (
	// "where-is-my-pizza/internal/order/domain/dto"
	// "where-is-my-pizza/internal/order/domain/models"

	"github.com/jackc/pgx/v5"
)

type IDB interface {
	Stop() error
	IsAlive() (bool, error)
	GetConn() *pgx.Conn
}

// type IOrderRepo interface {
// 	Create(order dto.OrderRequest) (models.Order, error)
// }
