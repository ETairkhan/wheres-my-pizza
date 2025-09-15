package db

import (
	"context"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TrackingDB struct {
	dbPool *pgxpool.Pool
	logger *logger.Logger
}

func NewTrackingDB(dbPool *pgxpool.Pool, logger *logger.Logger) *TrackingDB {
	return &TrackingDB{
		dbPool: dbPool,
		logger: logger,
	}
}

func (d *TrackingDB) GetOrderByNumber(ctx context.Context, orderNumber string) (*models.Order, error) {
	var order models.Order
	err := d.dbPool.QueryRow(ctx, `
        SELECT id, created_at, updated_at, number, customer_name, type, 
               table_number, delivery_address, total_amount, priority, 
               status, processed_by, completed_at
        FROM orders 
        WHERE number = $1
    `, orderNumber).Scan(
		&order.ID, &order.CreatedAt, &order.UpdatedAt, &order.Number,
		&order.CustomerName, &order.Type, &order.TableNumber, &order.DeliveryAddress,
		&order.TotalAmount, &order.Priority, &order.Status, &order.ProcessedBy, &order.CompletedAt,
	)
	return &order, err
}

func (d *TrackingDB) GetOrderStatusHistory(ctx context.Context, orderID int64) ([]models.OrderStatusLog, error) {
	rows, err := d.dbPool.Query(ctx, `
        SELECT id, created_at, order_id, status, changed_by, changed_at, notes
        FROM order_status_log 
        WHERE order_id = $1 
        ORDER BY created_at ASC
    `, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []models.OrderStatusLog
	for rows.Next() {
		var entry models.OrderStatusLog
		err := rows.Scan(
			&entry.ID, &entry.CreatedAt, &entry.OrderID, &entry.Status,
			&entry.ChangedBy, &entry.ChangedAt, &entry.Notes,
		)
		if err != nil {
			return nil, err
		}
		history = append(history, entry)
	}

	return history, nil
}

func (d *TrackingDB) GetAllWorkers(ctx context.Context) ([]models.Worker, error) {
	rows, err := d.dbPool.Query(ctx, `
        SELECT id, created_at, name, type, status, last_seen, orders_processed
        FROM workers 
        ORDER BY name ASC
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workers []models.Worker
	for rows.Next() {
		var worker models.Worker
		err := rows.Scan(
			&worker.ID, &worker.CreatedAt, &worker.Name, &worker.Type,
			&worker.Status, &worker.LastSeen, &worker.OrdersProcessed,
		)
		if err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}

	return workers, nil
}
