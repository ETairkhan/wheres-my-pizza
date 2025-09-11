package db

import (
	"context"
	"fmt"
	"time"

	"wheres-my-pizza/internal/order/app/core"
	"wheres-my-pizza/internal/order/domain/dto"
	"wheres-my-pizza/internal/order/domain/models"
)

type OrderRepo struct {
	ctx           context.Context
	db            core.IDB
	maxConcurrent int
}

func NewOrderRepo(ctx context.Context, db core.IDB, maxConcurrent int) *OrderRepo {
	return &OrderRepo{
		ctx:           ctx,
		db:            db,
		maxConcurrent: maxConcurrent,
	}
}

func (or *OrderRepo) Create(ctx context.Context, order dto.OrderRequest) (models.Order, error) {
	if err := or.db.IsAlive(); err != nil {
		return models.Order{}, core.ErrDBConn
	}

	// Get the current date in UTC format (YYYYMMDD)
	currentDate := time.Now().UTC().Format("20060102")

	// Start a new transaction
	tx, err := or.db.GetConn().Begin(ctx)
	if err != nil {
		return models.Order{}, fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback(ctx) // Ensure the transaction is rolled back if an error occurs

	var CurrentConcurent int
	q0 := `SELECT COUNT(1) FROM orders WHERE created_at::DATE = $1 AND status = 'received'`
	err = tx.QueryRow(ctx, q0, currentDate).Scan(&CurrentConcurent)
	if err != nil {
		return models.Order{}, err
	}

	if CurrentConcurent >= or.maxConcurrent {
		return models.Order{}, core.ErrMaxConcurentExceeded
	}
	// Step 1: Count the number of orders already placed for today
	var orderCount int
	err = tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM orders
			WHERE created_at::DATE = $1
		`, currentDate).Scan(&orderCount)
	if err != nil {
		return models.Order{}, fmt.Errorf("failed to count today's orders: %v", err)
	}

	// Step 2: Generate the order number as ORD_YYYYMMDD_NNN
	// The sequence starts from 1 and we increment it by 1 for each new order of the day
	sequenceNumber := orderCount + 1
	orderNumber := fmt.Sprintf("ORD_%s_%03d", currentDate, sequenceNumber)

	// Step 3: Insert the order data into the orders table
	newOrder := models.Order{
		OrderNumber: orderNumber,
		Type:        order.Type,
		Status:      core.DefaultOrderStatus,
		TotalAmount: order.TotalAmount,
	}
	err = tx.QueryRow(ctx, `
			INSERT INTO orders (
				order_number,
				customer_name,
				type,
				table_number,
				delivery_address,
				total_amount,
				priority,
				status,
				processed_by
			)
			VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9
			)
			RETURNING order_id
		`,
		orderNumber,
		order.CustomerName,
		order.Type,
		order.TableNumber,
		order.DeliveryAddress,
		order.TotalAmount,
		order.Priority,
		core.DefaultOrderStatus,
		core.DefaultProcessedBy,
	).Scan(&newOrder.OrderID)
	if err != nil {
		return models.Order{}, fmt.Errorf("failed to insert order: %v", err)
	}

	// Insert each item associated with the order
	for _, item := range order.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (
				order_id,
				name,
				quantity,
				unit_price
			)
			VALUES ($1, $2, $3, $4)
		`, newOrder.OrderID, item.Name, item.Quantity, item.Price)
		if err != nil {
			return models.Order{}, fmt.Errorf("failed to insert item: %v", err)
		}
	}

	// Insert the status log into order_status_log
	changedAt := time.Now()
	_, err = tx.Exec(ctx, `
		INSERT INTO order_status_log (
			order_id, 
			status, 
			changed_by, 
			changed_at, 
			note
		)
		VALUES ($1, $2, $3, $4, $5)
	`, newOrder.OrderID, "received", core.DefaultProcessedBy, changedAt, "")
	if err != nil {
		return models.Order{}, fmt.Errorf("failed to insert order status log: %v", err)
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return models.Order{}, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return newOrder, nil
}

func (or *OrderRepo) Delete(ctx context.Context, order dto.OrderRequest) error {
	q := `DELETE FROM orders WHERE order_number = $1`

	_, err := or.db.GetConn().Exec(ctx, q, order.OrderNumber)
	return err
}
