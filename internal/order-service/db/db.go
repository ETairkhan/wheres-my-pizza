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
	var orderNumber string
	today := time.Now().UTC().Format("20060102")

	err := d.dbPool.QueryRow(ctx, `
        SELECT COALESCE(MAX(SUBSTRING(number FROM '\\d+$')::integer), 0) + 1 
        FROM orders 
        WHERE created_at::date = CURRENT_DATE
    `).Scan(&orderNumber)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("ORD_%s_%03d", today, orderNumber), nil
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
