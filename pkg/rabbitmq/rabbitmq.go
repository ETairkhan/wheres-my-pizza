package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
	Logger  *logger.Logger
}

func ConnectRabbitMQ(cfg *config.RabbitMQConfig, log *logger.Logger) (*RabbitMQ, error) {
	connStr := fmt.Sprintf("amqp://%s:%s@%s:%d/",
		cfg.User, cfg.Password, cfg.Host, cfg.Port)

	conn, err := amqp.Dial(connStr)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Declare exchanges
	err = channel.ExchangeDeclare(
		"orders_topic", // name
		"topic",        // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return nil, err
	}

	err = channel.ExchangeDeclare(
		"notifications_fanout", // name
		"fanout",               // type
		true,                   // durable
		false,                  // auto-deleted
		false,                  // internal
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		return nil, err
	}

	// Declare queues
	_, err = channel.QueueDeclare(
		"kitchen_queue", // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		amqp.Table{
			"x-dead-letter-exchange": "dlx",
		}, // arguments
	)
	if err != nil {
		return nil, err
	}

	_, err = channel.QueueDeclare(
		"notifications_queue", // name
		true,                  // durable
		false,                 // delete when unused
		false,                 // exclusive
		false,                 // no-wait
		nil,                   // arguments
	)
	if err != nil {
		return nil, err
	}

	// Bind queues to exchanges
	err = channel.QueueBind(
		"kitchen_queue", // queue name
		"kitchen.*",     // routing key
		"orders_topic",  // exchange
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		return nil, err
	}

	err = channel.QueueBind(
		"notifications_queue",  // queue name
		"",                     // routing key
		"notifications_fanout", // exchange
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		return nil, err
	}

	log.Info("startup", "rabbitmq_connected", "Connected to RabbitMQ")
	return &RabbitMQ{
		Conn:    conn,
		Channel: channel,
		Logger:  log,
	}, nil
}

func (r *RabbitMQ) Close() {
	if r.Channel != nil {
		r.Channel.Close()
	}
	if r.Conn != nil {
		r.Conn.Close()
	}
}

func (r *RabbitMQ) PublishMessage(exchange, routingKey string, message []byte, priority uint8) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return r.Channel.PublishWithContext(ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			Priority:     priority,
			ContentType:  "application/json",
			Body:         message,
		})
}
