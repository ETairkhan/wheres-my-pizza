package workerregistration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"wheres-my-pizza/internal/kitchenworker/db"
	"wheres-my-pizza/pkg/logger"
)

type WorkerRegistration struct {
	dbService *db.KitchenDB
	logger    *logger.Logger
}

func NewWorkerRegistration(dbService *db.KitchenDB, logger *logger.Logger) *WorkerRegistration {
	return &WorkerRegistration{
		dbService: dbService,
		logger:    logger,
	}
}

func (r *WorkerRegistration) RegisterWorker(workerName, workerType string, orderTypes []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if worker already exists and is online
	var existingStatus string
	err := r.dbService.DbPool.QueryRow(ctx,
		"SELECT status FROM workers WHERE name = $1",
		workerName).Scan(&existingStatus)

	if err == nil && existingStatus == "online" {
		return errors.New("worker with this name is already online")
	}

	// Convert order types to a comma-separated string for storage
	orderTypesStr := ""
	if len(orderTypes) > 0 {
		orderTypesStr = strings.Join(orderTypes, ",")
	}

	// Register or update worker
	if err != nil {
		// Worker doesn't exist, create new
		_, err = r.dbService.DbPool.Exec(ctx, `
			INSERT INTO workers (name, type, order_types, status, last_seen, orders_processed)
			VALUES ($1, $2, $3, 'online', NOW(), 0)
		`, workerName, workerType, orderTypesStr)
	} else {
		// Worker exists, update status and order types
		_, err = r.dbService.DbPool.Exec(ctx, `
			UPDATE workers 
			SET status = 'online', last_seen = NOW(), type = $2, order_types = $3
			WHERE name = $1
		`, workerName, workerType, orderTypesStr)
	}

	if err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	r.logger.Info("startup", "worker_registered",
		fmt.Sprintf("Worker %s registered successfully with order types: %v", workerName, orderTypes))
	return nil
}

func (r *WorkerRegistration) SendHeartbeat(workerName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.dbService.DbPool.Exec(ctx, `
		UPDATE workers 
		SET last_seen = NOW() 
		WHERE name = $1
	`, workerName)

	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}

	return nil
}

func (r *WorkerRegistration) MarkWorkerOffline(workerName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.dbService.DbPool.Exec(ctx, `
		UPDATE workers 
		SET status = 'offline', last_seen = NOW() 
		WHERE name = $1
	`, workerName)

	if err != nil {
		return fmt.Errorf("failed to mark worker offline: %w", err)
	}

	r.logger.Info("shutdown", "worker_offline",
		fmt.Sprintf("Worker %s marked as offline", workerName))
	return nil
}

func (r *WorkerRegistration) GetWorkerOrderTypes(workerName string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var orderTypesStr string
	err := r.dbService.DbPool.QueryRow(ctx, `
		SELECT order_types FROM workers WHERE name = $1
	`, workerName).Scan(&orderTypesStr)

	if err != nil {
		return nil, fmt.Errorf("failed to get worker order types: %w", err)
	}

	if orderTypesStr == "" {
		return []string{}, nil
	}

	return strings.Split(orderTypesStr, ","), nil
}

func (r *WorkerRegistration) IncrementOrdersProcessed(workerName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.dbService.DbPool.Exec(ctx, `
		UPDATE workers 
		SET orders_processed = orders_processed + 1, last_seen = NOW() 
		WHERE name = $1
	`, workerName)

	if err != nil {
		return fmt.Errorf("failed to increment orders processed: %w", err)
	}

	return nil
}
