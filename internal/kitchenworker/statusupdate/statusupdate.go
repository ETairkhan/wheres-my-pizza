package statusupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/models"
	"wheres-my-pizza/pkg/rabbitmq"

	"github.com/rabbitmq/amqp091-go"
)

type StatusUpdateService struct {
	rabbitMQ *rabbitmq.RabbitMQ
	logger   *logger.Logger
}

func NewStatusUpdateService(rabbitMQ *rabbitmq.RabbitMQ, logger *logger.Logger) *StatusUpdateService {
	return &StatusUpdateService{
		rabbitMQ: rabbitMQ,
		logger:   logger,
	}
}

func (s *StatusUpdateService) PublishStatusUpdate(statusUpdate *models.StatusUpdateMessage) error {
	messageBytes, err := json.Marshal(statusUpdate)
	if err != nil {
		return fmt.Errorf("failed to marshal status update: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ensure the notifications exchange exists
	err = s.rabbitMQ.Channel.ExchangeDeclare(
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

	err = s.rabbitMQ.Channel.PublishWithContext(ctx,
		"notifications_fanout", // exchange
		"",                     // routing key
		false,                  // mandatory
		false,                  // immediate
		amqp091.Publishing{
			DeliveryMode: amqp091.Persistent,
			ContentType:  "application/json",
			Body:         messageBytes,
			Timestamp:    time.Now(),
		})
	if err != nil {
		return fmt.Errorf("failed to publish status update: %w", err)
	}

	s.logger.Debug("", "status_update_published",
		fmt.Sprintf("Status update published for order %s: %s -> %s",
			statusUpdate.OrderNumber, statusUpdate.OldStatus, statusUpdate.NewStatus))
	return nil
}

func (s *StatusUpdateService) EnsureExchange() error {
	return s.rabbitMQ.Channel.ExchangeDeclare(
		"notifications_fanout", // name
		"fanout",               // type
		true,                   // durable
		false,                  // auto-deleted
		false,                  // internal
		false,                  // no-wait
		nil,                    // arguments
	)
}
