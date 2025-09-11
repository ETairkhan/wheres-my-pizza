package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"wheres-my-pizza/internal/xpkg/logger"
	"wheres-my-pizza/internal/tracking/app/core"
	"wheres-my-pizza/internal/tracking/domain/models"
)

type OrderService struct {
	ctx       context.Context
	orderRepo core.IOrderRepo
	mylog     logger.Logger
}

func NewOrderService(
	ctx context.Context,
	orderRepo core.IOrderRepo,
	mylogger logger.Logger,
) *OrderService {
	return &OrderService{
		ctx:       ctx,
		orderRepo: orderRepo,
		mylog:     mylogger,
	}
}

func (os *OrderService) GetStatus(ctx context.Context, orderNumber string) (models.Order, error) {
	mylog := os.mylog.Action("GetStatus")
	order, err := os.orderRepo.GetStatus(ctx, orderNumber)
	if err != nil {
		if errors.Is(err, core.ErrDBConn) {
			mylog.Error("Failed to connect to db", err)
			return models.Order{}, err
		}
		if errors.Is(err, core.ErrOrderNotFound) {
			mylog.Error("Order not found", err)
			return models.Order{}, err
		}
		os.mylog.Action("GetStatus").Error("Failed to get status from db", err)
		return models.Order{}, fmt.Errorf("cannot get status: %w", err)
	}
	return order, nil
}

func (os *OrderService) GetHistory(ctx context.Context, orderNumber string) ([]models.OrderStatusLog, error) {
	mylog := os.mylog.Action("GetHistory")

	orders, err := os.orderRepo.GetHistory(ctx, orderNumber)
	if err != nil {
		if errors.Is(err, core.ErrDBConn) {
			mylog.Error("Failed to connect to db", err)
			return nil, err
		}
		if errors.Is(err, core.ErrOrderNotFound) {
			mylog.Error("Order not found", err)
			return nil, err
		}
		mylog.Error("Failed to get history", err)
		return nil, fmt.Errorf("cannot get history: %w", err)
	}
	return orders, nil
}

func (os *OrderService) GetEstimatedCompletion(createdAt time.Time, orderType string) time.Time {
	mylog := os.mylog.Action("GetEstimatedCompletion")

	sleepTime, ok := core.SleepTime[orderType]
	os.mylog.Debug("estimated completion time", "sleep_time", sleepTime, "order_type", orderType)
	if !ok {
		mylog.Warn("Failed to get estimated time completion")
		return createdAt
	}
	return createdAt.Add(time.Second * time.Duration(sleepTime))
}
