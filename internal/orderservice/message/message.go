package message

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/models"

	"github.com/rabbitmq/amqp091-go"
)

type MessageService struct {
	rabbitMQ *amqp091.Channel
	logger   *logger.Logger
}

func NewMessageService(rabbitMQ *amqp091.Channel, logger *logger.Logger) *MessageService {
	return &MessageService{
		rabbitMQ: rabbitMQ,
		logger:   logger,
	}
}

func (m *MessageService) PublishOrderMessage(orderMsg *models.OrderMessage, routingKey string, priority uint8) error {
	messageBytes, err := json.Marshal(orderMsg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = m.rabbitMQ.PublishWithContext(ctx,
		"orders_topic", // exchange
		routingKey,     // routing key
		false,          // mandatory
		false,          // immediate
		amqp091.Publishing{
			DeliveryMode: amqp091.Persistent,
			Priority:     priority,
			ContentType:  "application/json",
			Body:         messageBytes,
			Timestamp:    time.Now(),
		})

	if err != nil {
		return err
	}

	m.logger.Debug("", "message_published", fmt.Sprintf("Order message published with routing key: %s", routingKey))
	return nil
}
