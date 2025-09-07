package status_update

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
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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
		return err
	}

	s.logger.Debug("", "status_update_published",
		fmt.Sprintf("Status update published for order %s", statusUpdate.OrderNumber))
	return nil
}
