package services

import (
	"context"
	"errors"
	"fmt"

	"where-is-my-pizza/internal/mylogger"
	"where-is-my-pizza/internal/tracking/app/core"
	"where-is-my-pizza/internal/tracking/domain/models"
)

type WorkerService struct {
	ctx        context.Context
	workerRepo core.IWorkerRepo
	mylog      mylogger.Logger
}

func NewWorkerService(
	ctx context.Context,
	workerRepo core.IWorkerRepo,
	mylogger mylogger.Logger,
) *WorkerService {
	return &WorkerService{
		ctx:        ctx,
		workerRepo: workerRepo,
		mylog:      mylogger,
	}
}

func (ws *WorkerService) GetStatusOfAllWorkers(ctx context.Context) ([]models.Worker, error) {
	mylog := ws.mylog.Action("GetStatusOfAllWorkers")

	workers, err := ws.workerRepo.GetStatusOfAllWorkers(ctx)
	if err != nil {
		if errors.Is(err, core.ErrDBConn) {
			mylog.Error("Failed to connect to db", err)
			return nil, fmt.Errorf("cannot connect to db: %w", err)
		}
		if errors.Is(err, core.ErrWorkersNotFound) {
			mylog.Error("Worker not found", err)
			return nil, err
		}
		mylog.Error("Failed to get history", err)
		return nil, fmt.Errorf("cannot get history: %w", err)
	}
	return workers, nil
}
