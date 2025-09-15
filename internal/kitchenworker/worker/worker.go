package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"wheres-my-pizza/internal/kitchenworker/processor"
	"wheres-my-pizza/internal/kitchenworker/statusupdate"
	"wheres-my-pizza/internal/kitchenworker/workerregistration"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/models"
	"wheres-my-pizza/pkg/rabbitmq"

	kdb "wheres-my-pizza/internal/kitchenworker/db"

	pkgdb "wheres-my-pizza/pkg/db"

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
	statusService     *statusupdate.StatusUpdateService
	registration      *workerregistration.WorkerRegistration
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
	w.registration = workerregistration.NewWorkerRegistration(w.dbService, w.logger)
	if err := w.registration.RegisterWorker(w.workerName, "chef", w.orderTypes); err != nil {
		return fmt.Errorf("failed to register worker: %w", err)
	}

	// Initialize services
	w.processor = processor.NewOrderProcessor(w.logger)
	w.statusService = statusupdate.NewStatusUpdateService(rmq, w.logger)

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
	// Setup specialized queues
	if err := w.setupQueues(); err != nil {
		return fmt.Errorf("failed to setup queues: %w", err)
	}

	// Determine which queue to consume from
	queueName := "kitchen_queue"
	if len(w.orderTypes) == 1 {
		queueName = fmt.Sprintf("kitchen_%s_queue", w.orderTypes[0])
	}

	messages, err := w.rabbitMQ.Channel.Consume(
		queueName,    // queue
		w.workerName, // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	w.logger.Info("startup", "consuming_started",
		fmt.Sprintf("Started consuming messages from %s", queueName))

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
		w.logger.Debug(requestID, "order_type_mismatch", fmt.Sprintf("Worker %s cannot handle order type %s", w.workerName, orderMsg.OrderType))
		if err := msg.Nack(false, true); err != nil {
			w.logger.Error(requestID, "nack_failed", "Failed to Nack message (type mismatch)", err)
		}
		return
	}

	w.logger.Debug(requestID, "order_processing_started", fmt.Sprintf("Processing order %s", orderMsg.OrderNumber))

	// Process the order
	if err := w.processOrder(ctx, &orderMsg); err != nil {
		w.logger.Error(requestID, "order_processing_failed", fmt.Sprintf("Failed to process order %s", orderMsg.OrderNumber), err)
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

	w.logger.Debug(requestID, "order_completed", fmt.Sprintf("Completed processing order %s", orderMsg.OrderNumber))
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

// Add this method to the KitchenWorker struct
func (w *KitchenWorker) setupQueues() error {
	// Declare specialized queues based on order types
	if len(w.orderTypes) > 0 {
		for _, orderType := range w.orderTypes {
			queueName := fmt.Sprintf("kitchen_%s_queue", orderType)

			_, err := w.rabbitMQ.Channel.QueueDeclare(
				queueName, // name
				true,      // durable
				false,     // delete when unused
				false,     // exclusive
				false,     // no-wait
				amqp.Table{
					"x-dead-letter-exchange": "orders_topic_dlx",
				}, // arguments
			)
			if err != nil {
				return err
			}

			// Bind to specific order type and all priorities
			routingKey := fmt.Sprintf("kitchen.%s.*", orderType)
			err = w.rabbitMQ.Channel.QueueBind(
				queueName,      // queue name
				routingKey,     // routing key
				"orders_topic", // exchange
				false,          // no-wait
				nil,            // arguments
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
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
