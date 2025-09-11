package services

import (
	"context"

	"where-is-my-pizza/internal/mylogger"
)

type OrderService struct {
	ctx   context.Context
	mylog mylogger.Logger
}

func NewOrderService(
	ctx context.Context,
	mylogger mylogger.Logger,
) *OrderService {
	return &OrderService{
		ctx:   ctx,
		mylog: mylogger,
	}
}
