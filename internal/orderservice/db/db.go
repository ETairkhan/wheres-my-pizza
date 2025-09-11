package db

import (
	"context"
	"fmt"
	"time"

	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderDB struct {
	dbPool *pgxpool.Pool
	logger *logger.Logger
}

func NewOrderDB(dbPool *pgxpool.Pool, logger *logger.Logger) *OrderDB {
	return &OrderDB{
		dbPool: dbPool,
		logger: logger,
	}
}

func (d *OrderDB) GenerateOrderNumber(ctx context.Context) (string, error) {
	today := time.Now().UTC().Format("20060102")

	// Use a transaction to ensure atomic sequence generation
	tx, err := d.dbPool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var maxSeq int
	err = tx.QueryRow(ctx, `
        SELECT COALESCE(MAX(CAST(SUBSTRING(number FROM '\\d+$') AS INTEGER)), 0)
        FROM orders 
        WHERE created_at >= CURRENT_DATE AT TIME ZONE 'UTC'
        AND created_at < (CURRENT_DATE + INTERVAL '1 day') AT TIME ZONE 'UTC'
    `).Scan(&maxSeq)

	if err != nil {
		return "", err
	}

	orderNumber := fmt.Sprintf("ORD_%s_%03d", today, maxSeq+1)

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	return orderNumber, nil
}

func (d *OrderDB) CreateOrder(ctx context.Context, req *models.CreateOrderRequest, orderNumber string, totalAmount float64, priority int) (int64, error) {
	var orderID int64

	err := d.dbPool.QueryRow(ctx, `
        INSERT INTO orders (number, customer_name, type, table_number, delivery_address, total_amount, priority, status)
        VALUES ($1, $2, $3, $4, $5, $6, $7, 'received')
        RETURNING id
    `, orderNumber, req.CustomerName, req.OrderType, req.TableNumber, req.DeliveryAddress, totalAmount, priority).Scan(&orderID)

	if err != nil {
		return 0, err
	}

	return orderID, nil
}

func (d *OrderDB) CreateOrderItems(ctx context.Context, orderID int64, items []models.OrderItemRequest) error {
	batch := &pgx.Batch{}

	for _, item := range items {
		batch.Queue(`
            INSERT INTO order_items (order_id, name, quantity, price)
            VALUES ($1, $2, $3, $4)
        `, orderID, item.Name, item.Quantity, item.Price)
	}

	br := d.dbPool.SendBatch(ctx, batch)
	defer br.Close()

	for range items {
		_, err := br.Exec()
		if err != nil {
			return err
		}
	}

	return br.Close()
}

func (d *OrderDB) LogOrderStatus(ctx context.Context, orderID int64, status, changedBy, notes string) error {
	_, err := d.dbPool.Exec(ctx, `
        INSERT INTO order_status_log (order_id, status, changed_by, notes)
        VALUES ($1, $2, $3, $4)
    `, orderID, status, changedBy, notes)

	return err
}
