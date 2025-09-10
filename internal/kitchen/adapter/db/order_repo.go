package db

import (
	"context"
	"fmt"
	"wheres-my-pizza/internal/kitchen/app/core"
	"wheres-my-pizza/internal/xpkg/logger"

	"github.com/jackc/pgx/v5"
)

type OrderRepo struct {
	ctx context.Context
	db  core.IDB
	log logger.Logger
}

func NewOrderRepo(ctx context.Context, db core.IDB, log logger.Logger) core.IOrderRepo {
	return &OrderRepo{
		ctx: ctx,
		db:  db,
		log: log,
	}
}

func (or *OrderRepo) GetStatus(ctx context.Context, orderNumber string) (string, error) {
	log := or.log.Action("GetStatus")

	q := `SELECT status FROM orders WHERE order_number = $1;`
	status := ""
	err := or.db.GetConn().QueryRow(ctx, q, orderNumber).Scan(&status)
	if err != nil {
		return "", err
	}

	log.Debug("DEBUG", "status", status)

	return status, nil
}

func (or *OrderRepo) SetStatusCooking(ctx context.Context, orderNumber, processBy string) error {
	log := or.log.Action("SetStatusCooking")

	tx, err := or.db.GetConn().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Query 1
	q1 := `SELECT order_id FROM orders WHERE order_number = $1 AND status = 'received'`
	id := ""
	err = tx.QueryRow(ctx, q1, orderNumber).Scan(&id)
	if err != nil {
		log.Error("failed to get order id", err)
		return err
	}
	log.Debug("status to cooking", "id", id)

	// Query 2
	q2 := `UPDATE orders SET status = 'cooking', processed_by = &1 WHERE order_number = $2`
	cmdTag, err := tx.Exec(ctx, q2, processBy, orderNumber)
	if err != nil {
		log.Error("failed to update order status", err)
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		log.Error("failed, not affected to order", fmt.Errorf("not affected to rows (zero rows)"))
		return err
	}

	// Query 3
	q3 := `INSERT INTO order_status_log (order_id , status, changed_by , note) VALUES ($1, 'cooking', $2, $3)`
	// third query
	_, err = tx.Exec(ctx, q3, id, processBy, processBy+" is cooking")
	if err != nil {
		log.Error("fail to insert into log table", err)
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Error("fail to commit", err)
		return err
	}

	return nil
}

func (or *OrderRepo) SetStatusCancelled(ctx context.Context, orderNumber string) error {
	log := or.log.Action("UpdateStatus")

	tx, err := or.db.GetConn().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Query 1
	q1 := `SELECT order_id FROM orders WHERE order_number = $1`
	id := ""
	err = tx.QueryRow(ctx, q1, orderNumber).Scan(&id)
	if err != nil {
		log.Error("failed to get order id", err)
		return err
	}
	log.Debug("set to cancel", "id", id)

	// Query 2
	q2 := `UPDATE orders SET status = 'cancelled', processed_by = $1 WHERE order_number = $2`
	cmdTag, err := tx.Exec(ctx, q2, "kitchen-service", orderNumber)
	if err != nil {
		log.Error("failed to update order status", err)
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		log.Error("failed,not affected to order", fmt.Errorf("not affected to rows (zero rows)"))
		return err
	}

	// Query 3
	q3 := `INSERT INTO order_status_log (order_id, status, changed_by, note) VALUES ($1, 'cancelled', $2, $3)`
	// third query
	_, err = tx.Exec(ctx, q3, id, "kitchen-service", "Order is cancelled by kitchen-service")
	if err != nil {
		log.Error("fail to insert into log table", err)
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Error("fail to commit", err)
		return err
	}
	return nil
}

func (or *OrderRepo) SetStatusReady(ctx context.Context, orderNumber, processBy string) error {
	log := or.log.Action("SetReady")

	tx, err := or.db.GetConn().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback(ctx)
		}
	}()

	// Query 1
	q1 := `SELECT order_id FROM orders WHERE order_number = $1`
	id := ""
	err = tx.QueryRow(ctx, q1, orderNumber).Scan(&id)
	if err != nil {
		log.Error("failed to get order id", err)
		return err
	}
	log.Debug("set to ready", "id", id)

	// Query 2
	q2 := `
	UPDATE
		orders
	SET
		status = 'ready'
	WHERE
		order_number = $1`
	cmdTag, err := tx.Exec(ctx, q2, orderNumber)
	if err != nil {
		log.Error("failed to update order status", err)
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		log.Error("failed, not affected to order", fmt.Errorf("not affected to rows (zero rows)"))
		return err
	}

	// Query 3
	q3 := `INSERT INTO order_status_log (order_id , status, changed_by, note) VALUES ($1, 'ready', $2, $3)`
	_, err = tx.Exec(ctx, q3, id, processBy, processBy+" completed the order")
	if err != nil {
		log.Error("fail to insert into log table", err)
		return err
	}

	// Query 4 
	q4 := `UPDATE workers SET total_orders_processed = total_order_processed + 1 WHERE name = $1`
	cmdTag, err = tx.Exec(ctx, q4, processBy)
	if err != nil {
		log.Error("failed to update order status", err)
		return err
	} 

	if cmdTag.RowsAffected() == 0{
		log.Error("failed, not affected to order", fmt.Errorf("not affected to rows (zero rows)"))
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		log.Error("fail to commit", err)
		return err
	}
	return nil
}
