package service

import (
	"context"
	"fmt"
	"wheres-my-pizza/internal/order-service/db"
	"wheres-my-pizza/internal/order-service/message"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabbitmq/amqp091-go"
)

type OrderService struct {
	dbPool     *pgxpool.Pool
	rabbitMQ   *amqp091.Channel
	dbService  *db.OrderDB
	msgService *message.MessageService
	logger     *logger.Logger
}

func NewOrderService(dbPool *pgxpool.Pool, rabbitMQ *amqp091.Channel, logger *logger.Logger) *OrderService {
	dbService := db.NewOrderDB(dbPool, logger)
	msgService := message.NewMessageService(rabbitMQ, logger)
	return &OrderService{
		dbPool:     dbPool,
		rabbitMQ:   rabbitMQ,
		dbService:  dbService,
		msgService: msgService,
		logger:     logger,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, req *models.CreateOrderRequest, requestID string) (*models.CreateOrderResponse, error) {
	// Calculate total amount
	totalAmount := 0.0
	for _, item := range req.Items {
		totalAmount += item.Price * float64(item.Quantity)
	}

	// Determine priority
	priority := 1
	if totalAmount > 100 {
		priority = 10
	} else if totalAmount >= 50 {
		priority = 5
	}

	// Generate order number
	orderNumber, err := s.dbService.GenerateOrderNumber(ctx)
	if err != nil {
		return nil, err
	}

	// Create order in database
	orderID, err := s.dbService.CreateOrder(ctx, req, orderNumber, totalAmount, priority)
	if err != nil {
		return nil, err
	}

	// Create order items
	err = s.dbService.CreateOrderItems(ctx, orderID, req.Items)
	if err != nil {
		return nil, err
	}

	// Log initial status
	err = s.dbService.LogOrderStatus(ctx, orderID, "received", "order-service", "Order received")
	if err != nil {
		return nil, err
	}

	// Publish message to RabbitMQ
	orderMessage := models.OrderMessage{
		OrderNumber:     orderNumber,
		CustomerName:    req.CustomerName,
		OrderType:       req.OrderType,
		TableNumber:     req.TableNumber,
		DeliveryAddress: req.DeliveryAddress,
		Items:           req.Items,
		TotalAmount:     totalAmount,
		Priority:        priority,
	}

	routingKey := fmt.Sprintf("kitchen.%s.%d", req.OrderType, priority)
	err = s.msgService.PublishOrderMessage(&orderMessage, routingKey, uint8(priority))
	if err != nil {
		return nil, err
	}

	s.logger.Debug(requestID, "order_published", "Order published to RabbitMQ")

	return &models.CreateOrderResponse{
		OrderNumber: orderNumber,
		Status:      "received",
		TotalAmount: totalAmount,
	}, nil
}
