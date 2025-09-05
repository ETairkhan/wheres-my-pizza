package processor

import (
	"time"

	"wheres-my-pizza/pkg/logger"
)

type OrderProcessor struct {
	logger *logger.Logger
}

func NewOrderProcessor(logger *logger.Logger) *OrderProcessor {
	return &OrderProcessor{
		logger: logger,
	}
}

func (p *OrderProcessor) SimulateCooking(orderType string) {
	var cookingTime time.Duration

	switch orderType {
	case "dine_in":
		cookingTime = 8 * time.Second
	case "takeout":
		cookingTime = 10 * time.Second
	case "delivery":
		cookingTime = 12 * time.Second
	default:
		cookingTime = 10 * time.Second
	}

	time.Sleep(cookingTime)
}
