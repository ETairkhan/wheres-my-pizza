package subscriber

import (
	"context"
	"fmt"
	"wheres-my-pizza/internal/notificationsubscriber/message"
	"wheres-my-pizza/internal/notificationsubscriber/notifier"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/rabbitmq"
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
		config:    cfg,
		logger:    logger,
		stopChan:  make(chan struct{}),
		notifier:  notifier.NewNotifier(logger),
		msgParser: message.NewMessageParser(logger),
	}
}

func (s *NotificationSubscriber) Start(ctx context.Context) error {
	// Connect to RabbitMQ
	rmq, err := rabbitmq.ConnectRabbitMQ(&s.config.RabbitMQ, s.logger)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	s.rabbitMQ = rmq

	// Declare exchange
	err = rmq.Channel.ExchangeDeclare(
		"notifications_fanout", // name
		"fanout",               // type
		true,                   // durable
		false,                  // auto-deleted
		false,                  // internal
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	q, err := rmq.Channel.QueueDeclare(
		"",    // name (let server generate)
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = rmq.Channel.QueueBind(
		q.Name,                 // queue name
		"",                     // routing key
		"notifications_fanout", // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	// Consume messages
	messages, err := rmq.Channel.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming messages: %w", err)
	}

	s.logger.Info("startup", "subscriber_started", "Notification subscriber started successfully")

	// Process messages
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-messages:
			if !ok {
				return fmt.Errorf("message channel closed")
			}
			if err := s.processMessage(msg.Body); err != nil {
				s.logger.Error("message_processing", "process_failed", "Failed to process message", err)
			}
		}
	}
}

func (s *NotificationSubscriber) processMessage(messageBytes []byte) error {
	// Parse the message
	statusUpdate, err := s.msgParser.ParseStatusUpdate(messageBytes)
	if err != nil {
		return err
	}

	// Display notification
	s.notifier.DisplayNotification(statusUpdate)

	s.logger.Debug("message_processing", "notification_displayed",
		fmt.Sprintf("Displayed notification for order %s", statusUpdate.OrderNumber))
	return nil
}

func (s *NotificationSubscriber) Stop() {
	if s.rabbitMQ != nil {
		s.rabbitMQ.Close()
	}
	close(s.stopChan)
}
