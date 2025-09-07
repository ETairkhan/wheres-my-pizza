package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	kdb "wheres-my-pizza/internal/kitchen-worker/db"
	"wheres-my-pizza/internal/kitchen-worker/processor"
	"wheres-my-pizza/internal/kitchen-worker/status_update"
	"wheres-my-pizza/internal/kitchen-worker/worker_registration"

	"wheres-my-pizza/pkg/config"
	pkgdb "wheres-my-pizza/pkg/db"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/models"
	"wheres-my-pizza/pkg/rabbitmq"

	"github.com/jackc/pgx/v5/pgxpool"
	amqp "github.com/rabbitmq/amqp091-go"
)

type KitchenWorker struct {
	workerName        string
	orderTypes        []string
	heartbeatInterval int
	prefetch          int
	config            *config.Config
	logger            *logger.Logger
	dbPool            *pgxpool.Pool
	rabbitMQ          *rabbitmq.RabbitMQ
	dbService         *kdb.KitchenDB
	processor         *processor.OrderProcessor
	statusService     *status_update.StatusUpdateService
	registration      *worker_registration.WorkerRegistration
	stopChan          chan struct{}
}

func NewKitchenWorker(workerName string, orderTypes []string, heartbeatInterval, prefetch int, cfg *config.Config, log *logger.Logger) *KitchenWorker {
	return &KitchenWorker{
		workerName:        workerName,
		orderTypes:        orderTypes,
		heartbeatInterval: heartbeatInterval,
		prefetch:          prefetch,
		config:            cfg,
		logger:            log,
		stopChan:          make(chan struct{}),
	}
}

func (w *KitchenWorker) Start(ctx context.Context) error {
	// Connect to Postgres using pkg/db.ConnectDB
	dbPool, err := pkgdb.ConnectDB(&w.config.Database, w.logger)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	w.dbPool = dbPool
	w.dbService = kdb.NewKitchenDB(dbPool, w.logger)

	// Connect to RabbitMQ
	rmq, err := rabbitmq.ConnectRabbitMQ(&w.config.RabbitMQ, w.logger)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	w.rabbitMQ = rmq

	// Register worker
	w.registration = worker_registration.NewWorkerRegistration(w.dbService, w.logger)
	if err := w.registration.RegisterWorker(w.workerName, "chef", w.orderTypes); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// Initialize services
	w.processor = processor.NewOrderProcessor(w.logger)
	w.statusService = status_update.NewStatusUpdateService(rmq, w.logger)

	// Set prefetch
	if err := w.rabbitMQ.Channel.Qos(w.prefetch, 0, false); err != nil {
		return fmt.Errorf("failed to set prefetch: %w", err)
	}

	// Start heartbeat
	go w.startHeartbeat(ctx)

	// Start consuming messages
	return w.consumeMessages(ctx)
}

func (w *KitchenWorker) consumeMessages(ctx context.Context) error {
	// Consume messages from the queue
	messages, err := w.rabbitMQ.Channel.Consume(
		"kitchen_queue", // queue
		w.workerName,    // consumer
		false,           // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	w.logger.Info("startup", "consuming_started", "Started consuming messages from kitchen_queue")

	// Message processing loop
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-messages:
			if !ok {
				return fmt.Errorf("message channel closed")
			}
			go w.processMessage(ctx, msg)
		}
	}
}

func (w *KitchenWorker) processMessage(ctx context.Context, msg amqp.Delivery) {
	requestID := fmt.Sprintf("msg-%d", time.Now().UnixNano())

	// Parse the message
	var orderMsg models.OrderMessage
	if err := json.Unmarshal(msg.Body, &orderMsg); err != nil {
		w.logger.Error(requestID, "message_parsing_failed", "Failed to parse message", err)
		if err2 := msg.Nack(false, true); err2 != nil {
			w.logger.Error(requestID, "nack_failed", "Failed to Nack message after parse error", err2)
		}
		return
	}

	// Check if the worker can handle this order type
	if len(w.orderTypes) > 0 && !w.canHandleOrderType(orderMsg.OrderType) {
		w.logger.Debug(requestID, "order_type_mismatch",
			fmt.Sprintf("Worker %s cannot handle order type %s", w.workerName, orderMsg.OrderType))
		if err := msg.Nack(false, true); err != nil {
			w.logger.Error(requestID, "nack_failed", "Failed to Nack message (type mismatch)", err)
		}
		return
	}

	w.logger.Debug(requestID, "order_processing_started",
		fmt.Sprintf("Processing order %s", orderMsg.OrderNumber))

	// Process the order
	if err := w.processOrder(ctx, &orderMsg); err != nil {
		w.logger.Error(requestID, "order_processing_failed",
			fmt.Sprintf("Failed to process order %s", orderMsg.OrderNumber), err)
		if err2 := msg.Nack(false, true); err2 != nil {
			w.logger.Error(requestID, "nack_failed", "Failed to Nack message after processing error", err2)
		}
		return
	}

	// Acknowledge the message
	if err := msg.Ack(false); err != nil {
		w.logger.Error(requestID, "ack_failed", "Failed to Ack processed message", err)
		return
	}
	w.logger.Debug(requestID, "order_completed",
		fmt.Sprintf("Completed processing order %s", orderMsg.OrderNumber))
}

func (w *KitchenWorker) canHandleOrderType(orderType string) bool {
	for _, t := range w.orderTypes {
		if t == orderType {
			return true
		}
	}
	return false
}

func (w *KitchenWorker) processOrder(ctx context.Context, orderMsg *models.OrderMessage) error {
	// Get order ID from the database
	orderID, err := w.dbService.GetOrderIDByNumber(ctx, orderMsg.OrderNumber)
	if err != nil {
		return err
	}

	// Update status to 'cooking'
	if err := w.dbService.UpdateOrderStatus(ctx, orderID, "cooking", w.workerName); err != nil {
		return err
	}

	// Log status change
	if err := w.dbService.LogOrderStatus(ctx, orderID, "cooking", w.workerName, "Order started cooking"); err != nil {
		return err
	}

	// Publish status update to RabbitMQ
	estimatedCompletion := time.Now().Add(w.getCookingTime(orderMsg.OrderType))
	statusUpdate := models.StatusUpdateMessage{
		OrderNumber:         orderMsg.OrderNumber,
		OldStatus:           "received",
		NewStatus:           "cooking",
		ChangedBy:           w.workerName,
		Timestamp:           time.Now(),
		EstimatedCompletion: estimatedCompletion,
	}
	if err := w.statusService.PublishStatusUpdate(&statusUpdate); err != nil {
		return err
	}

	// Simulate cooking process
	w.processor.SimulateCooking(orderMsg.OrderType)

	// Update status to 'ready'
	if err := w.dbService.UpdateOrderStatusToReady(ctx, orderID, w.workerName); err != nil {
		return err
	}

	// Log status change again
	if err := w.dbService.LogOrderStatus(ctx, orderID, "ready", w.workerName, "Order is ready"); err != nil {
		return err
	}

	// Publish final status update
	statusUpdate = models.StatusUpdateMessage{
		OrderNumber:         orderMsg.OrderNumber,
		OldStatus:           "cooking",
		NewStatus:           "ready",
		ChangedBy:           w.workerName,
		Timestamp:           time.Now(),
		EstimatedCompletion: time.Now(),
	}
	return w.statusService.PublishStatusUpdate(&statusUpdate)
}

func (w *KitchenWorker) getCookingTime(orderType string) time.Duration {
	switch orderType {
	case "dine_in":
		return 8 * time.Second
	case "takeout":
		return 10 * time.Second
	case "delivery":
		return 12 * time.Second
	default:
		return 10 * time.Second
	}
}

func (w *KitchenWorker) startHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(w.heartbeatInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.registration.SendHeartbeat(w.workerName); err != nil {
				w.logger.Error("heartbeat", "heartbeat_failed", "Failed to send heartbeat", err)
			} else {
				w.logger.Debug("heartbeat", "heartbeat_sent", "Heartbeat sent successfully")
			}
		}
	}
}

func (w *KitchenWorker) Stop() {
	if w.rabbitMQ != nil {
		w.rabbitMQ.Close()
	}
	if w.dbPool != nil {
		w.dbPool.Close()
	}
	if w.registration != nil {
		if err := w.registration.MarkWorkerOffline(w.workerName); err != nil {
			w.logger.Error("shutdown", "mark_offline_failed", "Failed to mark worker offline", err)
		}
	}
	close(w.stopChan)
}
