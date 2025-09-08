package db

import (
	"context"
	"wheres-my-pizza/internal/xpkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

type KitchenDB struct {
	DbPool *pgxpool.Pool
	logger *logger.Logger
}

func NewKitchenDB(dbPool *pgxpool.Pool, logger *logger.Logger) *KitchenDB {
	return &KitchenDB{
		DbPool: dbPool,
		logger: logger,
	}
}

func (d *KitchenDB) GetOrderIDByNumber(ctx context.Context, orderNumber string) (int64, error) {
	var orderID int64
	err := d.DbPool.QueryRow(ctx, `SELECT id FROM orders WHERE number = $1`, orderNumber).Scan(&orderID)
	return orderID, err
}

func (d *KitchenDB) UpdateOrderStatus(ctx context.Context, orderID int64, status, processedBy string) error {
	_, err := d.DbPool.Exec(ctx, `
        UPDATE orders 
        SET status = $1, processed_by = $2, updated_at = NOW() 
        WHERE id = $3
    `, status, processedBy, orderID)
	return err
}

func (d *KitchenDB) UpdateOrderStatusToReady(ctx context.Context, orderID int64, processedBy string) error {
	_, err := d.DbPool.Exec(ctx, `
        UPDATE orders 
        SET status = 'ready', processed_by = $1, completed_at = NOW(), updated_at = NOW() 
        WHERE id = $2
    `, processedBy, orderID)
	return err
}

func (d *KitchenDB) LogOrderStatus(ctx context.Context, orderID int64, status, changedBy, notes string) error {
	_, err := d.DbPool.Exec(ctx, `
        INSERT INTO order_status_log (order_id, status, changed_by, notes)
        VALUES ($1, $2, $3, $4)
    `, orderID, status, changedBy, notes)
	return err
}

func (d *KitchenDB) IncrementOrdersProcessed(ctx context.Context, workerName string) error {
	_, err := d.DbPool.Exec(ctx, `
        UPDATE workers 
        SET orders_processed = orders_processed + 1, last_seen = NOW() 
        WHERE name = $1
    `, workerName)
	return err
}
