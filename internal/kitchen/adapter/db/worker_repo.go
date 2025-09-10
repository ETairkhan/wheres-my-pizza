package db

import (
	"context"
	"fmt"
	"wheres-my-pizza/internal/kitchen/app/core"
	"wheres-my-pizza/internal/kitchen/domain/models"
)

type WorkerRepo struct {
	ctx context.Context
	db  core.IDB
}

func NewWorkerRepo(ctx context.Context, db core.IDB) core.IWorkerRepo {
	return &WorkerRepo{
		ctx: ctx,
		db:  db,
	}
}

func (wr *WorkerRepo) Get(ctx context.Context, name string) (models.Worker, error) {
	worker := models.Worker{}
	q := `SELECT worker_id, created_at, name, type, status, last_active_at, total_orders_processed FROM workers WHERE name = $1`
	if err := wr.db.GetConn().QueryRow(ctx, q, name).Scan(
		&worker.WorkerId,
		&worker.CreatedAt,
		&worker.Name,
		&worker.Type,
		&worker.Status,
		&worker.LastActiveAt,
		&worker.TotalOrdersProcessed); err != nil{
			return models.Worker{}, err
		}
	return worker, nil
}

func (wr *WorkerRepo) Create(ctx context.Context, w models.Worker) (string ,error){
	if err := wr.db.IsAlive(); err != nil{
		return "", core.ErrDBConn
	}

	q  := `INSERT INTO workers (name, type) VALUES ($1, $2) RETURNING worker_id`
	id :=""
	if err := wr.db.GetConn().QueryRow(ctx, q, w.Name, w.Type).Scan(&id); err != nil{
		return "", err
	}

	return id, nil
}

func (wr *WorkerRepo) UpdateLastSeen(ctx context.Context, name string) error{
	q := `
	UPDATE
		workers
	SET
		last_active_at = NOW(),
		status = 'online'
	WHERE
		name =  $1`
	
	result, err := wr.db.GetConn().Exec(ctx, q, name)
	if err != nil {
		return fmt.Errorf("failed to execute update: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no worker found with name: %s", name)
	}

	return nil
}

func (wr *WorkerRepo) UpdateType(ctx context.Context, name, newType string) error {
	q := `
	UPDATE 
		workers
	SET
		type=$1 
	WHERE 
		name=$2`

	result, err := wr.db.GetConn().Exec(ctx, q, newType, name)
	if err != nil {
		return fmt.Errorf("failed to execute update: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no worker found with name: %s", name)
	}

	return nil
}

func (wr *WorkerRepo) SetOnline(ctx context.Context, name string) error {
	q := `
	UPDATE 
		workers
	SET
		status='online'
	WHERE 
		name=$1`

	result, err := wr.db.GetConn().Exec(ctx, q, name)
	if err != nil {
		return fmt.Errorf("failed to execute update: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no worker found with name: %s", name)
	}

	return nil
}

func (wr *WorkerRepo) SetOffline(ctx context.Context, name string) error {
	q := `
	UPDATE 
		workers
	SET
		status='offline'
	WHERE 
		name=$1`

	result, err := wr.db.GetConn().Exec(ctx, q, name)
	if err != nil {
		return fmt.Errorf("failed to execute update: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no worker found with name: %s", name)
	}

	return nil
}

