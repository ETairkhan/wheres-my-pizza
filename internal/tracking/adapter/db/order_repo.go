package db

import (
	"context"

	"where-is-my-pizza/internal/tracking/app/core"
	"where-is-my-pizza/internal/tracking/domain/models"

	"github.com/jackc/pgx/v5"
)

type OrderRepo struct {
	ctx context.Context
	db  core.IDB
}

func NewOrderRepo(ctx context.Context, db core.IDB) *OrderRepo {
	return &OrderRepo{
		ctx: ctx,
		db:  db,
	}
}

func (or *OrderRepo) GetStatus(ctx context.Context, orderNumber string) (models.Order, error) {
	if err := or.db.IsAlive(); err != nil {
		return models.Order{}, core.ErrDBConn
	}

	q := `
	SELECT
		order_number,
		type,
		status,
		created_at,
		updated_at,
		processed_by
	FROM 
		orders
	WHERE
		order_number = $1
	`
	var order models.Order
	if err := or.db.GetConn().QueryRow(ctx, q, orderNumber).Scan(
		&order.OrderNumber,
		&order.Type,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
		&order.ProcessedBy,
	); err != nil {
		if err == pgx.ErrNoRows {
			return models.Order{}, core.ErrOrderNotFound
		}
		return models.Order{}, err
	}

	return order, nil
}

func (or *OrderRepo) GetHistory(ctx context.Context, orderNumber string) ([]models.OrderStatusLog, error) {
	if err := or.db.IsAlive(); err != nil {
		return nil, core.ErrDBConn
	}

	q := `
	SELECT
		osl.status,
		osl.changed_at,
		osl.changed_by
	FROM 
		order_status_log osl
	JOIN orders o ON osl.order_id = o.order_id  
	WHERE
		o.order_number = $1
	`
	var orders []models.OrderStatusLog

	rows, err := or.db.GetConn().Query(ctx, q, orderNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var order models.OrderStatusLog
		err := rows.Scan(
			&order.Status,
			&order.ChangedAt,
			&order.ChangedBy,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	if len(orders) == 0 {
		return nil, core.ErrOrderNotFound
	}
	return orders, nil
}
