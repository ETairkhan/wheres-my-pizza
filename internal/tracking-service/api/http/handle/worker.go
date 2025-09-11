package handle

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"where-is-my-pizza/internal/mylogger"
	"where-is-my-pizza/internal/tracking/app/core"
	"where-is-my-pizza/internal/tracking/app/services"
)

type WorkerHandler struct {
	workerService *services.WorkerService
	mylog         mylogger.Logger
}

func NewWorkerHandler(workerService *services.WorkerService, mylog mylogger.Logger) *WorkerHandler {
	return &WorkerHandler{
		workerService: workerService,
		mylog:         mylog,
	}
}

func (wh *WorkerHandler) GetStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), core.WaitTime*time.Second)
		defer cancel()

		workers, err := wh.workerService.GetStatusOfAllWorkers(ctx)
		if err != nil {
			if errors.Is(err, core.ErrWorkersNotFound) {
				jsonError(w, http.StatusNotFound, core.ErrWorkersNotFound)
				return
			}
			jsonError(w, http.StatusInternalServerError, fmt.Errorf("failed to get status of all registered kitchen workers: %v", err))
			return
		}

		type resp struct {
			WorkerName      string `json:"worker_name"`
			Status          string `json:"status"`
			OrdersProcessed int    `json:"orders_processed"`
			LastSeen        string `json:"last_seen"`
		}

		var allRegisteredWorkers []resp
		for _, worker := range workers {
			allRegisteredWorkers = append(allRegisteredWorkers,
				resp{
					WorkerName:      worker.Name,
					Status:          worker.Status,
					OrdersProcessed: worker.TotalOrdersProcessed,
					LastSeen:        worker.LastActiveAt.UTC().Format(time.RFC3339),
				},
			)
		}

		jsonResponse(w, http.StatusOK, allRegisteredWorkers)
	}
}
