package services

import (
	"context"

	"wheres-my-pizza/internal/xpkg/logger"
)

type OrderService struct {
	ctx   context.Context
	mylog logger.Logger
}

func NewOrderService(
	ctx context.Context,
	mylogger logger.Logger,
) *OrderService {
	return &OrderService{
		ctx:   ctx,
		mylog: mylogger,
	}
}
