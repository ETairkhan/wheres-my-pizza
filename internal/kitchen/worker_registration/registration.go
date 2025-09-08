package worker_registration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"wheres-my-pizza/internal/kitchen-worker/db"
	"wheres-my-pizza/internal/xpkg/logger"
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if worker already exists and is online
	var existingStatus string
	err := r.dbService.DbPool.QueryRow(ctx,
		"SELECT status FROM workers WHERE name = $1", workerName).Scan(&existingStatus)

	if err == nil && existingStatus == "online" {
		return errors.New("worker with this name is already online")
	}

	// Register or update worker
	if err != nil {
		// Worker doesn't exist, create new
		_, err = r.dbService.DbPool.Exec(ctx, `
            INSERT INTO workers (name, type, status, last_seen, orders_processed)
            VALUES ($1, $2, 'online', NOW(), 0)
        `, workerName, workerType)
	} else {
		// Worker exists, update status
		_, err = r.dbService.DbPool.Exec(ctx, `
            UPDATE workers 
            SET status = 'online', last_seen = NOW(), type = $2
            WHERE name = $1
        `, workerName, workerType)
	}

	if err != nil {
		return err
	}

	r.logger.Info("startup", "worker_registered",
		fmt.Sprintf("Worker %s registered successfully", workerName))
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
	return err
}

func (r *WorkerRegistration) MarkWorkerOffline(workerName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.dbService.DbPool.Exec(ctx, `
        UPDATE workers 
        SET status = 'offline', last_seen = NOW() 
        WHERE name = $1
    `, workerName)
	return err
}
