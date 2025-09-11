package db

import (
	"context"

	"wheres-my-pizza/internal/tracking/app/core"
	"wheres-my-pizza/internal/tracking/domain/models"
)

type WorkerRepo struct {
	ctx context.Context
	db  core.IDB
}

func NewWorkerRepo(ctx context.Context, db core.IDB) *WorkerRepo {
	return &WorkerRepo{
		ctx: ctx,
		db:  db,
	}
}

func (wr *WorkerRepo) GetStatusOfAllWorkers(ctx context.Context) ([]models.Worker, error) {
	if err := wr.db.IsAlive(); err != nil {
		return nil, core.ErrDBConn
	}

	q := `
	SELECT
		name,
		status,
		total_orders_processed,
		last_active_at
	FROM 
		workers
	`
	var workers []models.Worker

	rows, err := wr.db.GetConn().Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var worker models.Worker
		err := rows.Scan(
			&worker.Name,
			&worker.Status,
			&worker.TotalOrdersProcessed,
			&worker.LastActiveAt,
		)
		if err != nil {
			return nil, err
		}
		workers = append(workers, worker)
	}

	if len(workers) == 0 {
		return nil, core.ErrWorkersNotFound
	}
	return workers, nil
}
