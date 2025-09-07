package subscriber

import (
	"context"
	"fmt"
	"time"

	"wheres-my-pizza/internal/notification-subscriber/message"
	"wheres-my-pizza/internal/notification-subscriber/notifier"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/rabbitmq"

	amqp "github.com/rabbitmq/amqp091-go"
)

type NotificationSubscriber struct {
	config    *config.Config
	logger    *logger.Logger
	rabbitMQ  *rabbitmq.RabbitMQ
	notifier  *notifier.Notifier
	msgParser *message.MessageParser
	stopChan  chan struct{}
}

func NewNotificationSubscriber(cfg *config.Config, logger *logger.Logger) *NotificationSubscriber {
	return &NotificationSubscriber{
		config:   cfg,
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

func (s *NotificationSubscriber) Start(ctx context.Context) error {
	// Connect to RabbitMQ
	rabbitMQ, err := rabbitmq.ConnectRabbitMQ(&s.config.RabbitMQ, s.logger)
	if err != nil {
		return err
	}
	s.rabbitMQ = rabbitMQ

	// Initialize services
	s.notifier = notifier.NewNotifier(s.logger)
	s.msgParser = message.NewMessageParser(s.logger)

	// Start consuming messages
	return s.consumeMessages(ctx)
}

func (s *NotificationSubscriber) consumeMessages(ctx context.Context) error {
	msgs, err := s.rabbitMQ.Channel.Consume(
		"notifications_queue", // queue
		"notification-sub",    // consumer
		false,                 // auto-ack
		false,                 // exclusive
		false,                 // no-local
		false,                 // no-wait
		nil,                   // args
	)
	if err != nil {
		return err
	}

	s.logger.Info("startup", "consuming_started", "Started consuming messages from notifications_queue")

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("message channel closed")
			}
			go s.processMessage(ctx, msg)
		}
	}
}

func (s *NotificationSubscriber) processMessage(ctx context.Context, msg amqp.Delivery) {
	requestID := fmt.Sprintf("notif-%d", time.Now().UnixNano())

	// Parse message
	statusUpdate, err := s.msgParser.ParseStatusUpdate(msg.Body)
	if err != nil {
		s.logger.Error(requestID, "message_parsing_failed", "Failed to parse status update", err)
		msg.Nack(false, false) // don't requeue
		return
	}

	// Display notification
	s.notifier.DisplayNotification(statusUpdate)

	// Log the notification
	s.logger.Debug(requestID, "notification_received",
		fmt.Sprintf("Status update for order %s: %s -> %s",
			statusUpdate.OrderNumber, statusUpdate.OldStatus, statusUpdate.NewStatus))

	// Acknowledge message
	msg.Ack(false)
}

func (s *NotificationSubscriber) Stop() {
	if s.rabbitMQ != nil {
		s.rabbitMQ.Close()
	}
	close(s.stopChan)
}
