package service

import (
	"context"
	"time"

	"wheres-my-pizza/internal/trackingservice/db"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TrackingService struct {
	dbService *db.TrackingDB
	logger    *logger.Logger
}

func NewTrackingService(dbPool *pgxpool.Pool, logger *logger.Logger) *TrackingService {
	dbService := db.NewTrackingDB(dbPool, logger)
	return &TrackingService{
		dbService: dbService,
		logger:    logger,
	}
}

func (s *TrackingService) GetOrderStatus(ctx context.Context, orderNumber string) (*models.OrderStatusResponse, error) {
	order, err := s.dbService.GetOrderByNumber(ctx, orderNumber)
	if err != nil {
		return nil, err
	}

	// Calculate estimated completion if cooking
	var estimatedCompletion *time.Time
	if order.Status == "cooking" {
		// Simple estimation: 10 minutes from updated_at
		est := order.UpdatedAt.Add(10 * time.Minute)
		estimatedCompletion = &est
	}

	return &models.OrderStatusResponse{
		OrderNumber:         order.Number,
		CurrentStatus:       order.Status,
		UpdatedAt:           order.UpdatedAt,
		EstimatedCompletion: estimatedCompletion,
		ProcessedBy:         order.ProcessedBy,
	}, nil
}

func (s *TrackingService) GetOrderHistory(ctx context.Context, orderNumber string) ([]models.OrderHistoryEntry, error) {
	order, err := s.dbService.GetOrderByNumber(ctx, orderNumber)
	if err != nil {
		return nil, err
	}

	history, err := s.dbService.GetOrderStatusHistory(ctx, order.ID)
	if err != nil {
		return nil, err
	}

	var historyEntries []models.OrderHistoryEntry
	for _, entry := range history {
		historyEntries = append(historyEntries, models.OrderHistoryEntry{
			Status:    entry.Status,
			Timestamp: entry.ChangedAt,
			ChangedBy: entry.ChangedBy,
		})
	}

	return historyEntries, nil
}

func (s *TrackingService) GetWorkersStatus(ctx context.Context) ([]models.WorkerStatus, error) {
	workers, err := s.dbService.GetAllWorkers(ctx)
	if err != nil {
		return nil, err
	}

	var workerStatuses []models.WorkerStatus
	for _, worker := range workers {
		// Check if worker is offline (last seen > 2 minutes ago)
		status := worker.Status
		if worker.Status == "online" && time.Since(worker.LastSeen) > 2*time.Minute {
			status = "offline"
		}

		workerStatuses = append(workerStatuses, models.WorkerStatus{
			WorkerName:      worker.Name,
			Status:          status,
			OrdersProcessed: worker.OrdersProcessed,
			LastSeen:        worker.LastSeen,
		})
	}

	return workerStatuses, nil
}
