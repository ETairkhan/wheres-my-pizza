package brokermessage

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
	"wheres-my-pizza/internal/kitchen/app/core"
	"wheres-my-pizza/internal/kitchen/domain/dto"
	"wheres-my-pizza/internal/xpkg/config"
	"wheres-my-pizza/internal/xpkg/logger"

	amqp "github.com/rabbitmq/amqp091-go"
)

const exchange = "notifications"

type RabbitMQ struct {
	ctx          context.Context
	cfg          config.RabbitMQ
	conn         *amqp.Connection
	ch           *amqp.Channel
	mylog        logger.Logger
	reconnecting bool
	mu           *sync.Mutex

	prefetch int
}

// create RabbitMQ adapter
func New(
	ctx context.Context,
	rabbitmqCfg config.RabbitMQ,
	mylog logger.Logger,
	prefetch int,
) (core.IRabbitMQ, error) {
	r := &RabbitMQ{
		ctx:          ctx,
		cfg:          rabbitmqCfg,
		mylog:        mylog,
		mu:           &sync.Mutex{},
		reconnecting: false,
		prefetch:     prefetch,
	}
	err := r.connect()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// connect to rabbitmq
func (r *RabbitMQ) connect() error {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/%s",
		r.cfg.User,
		r.cfg.Password,
		r.cfg.Host,
		r.cfg.Port,
		r.cfg.VHost,
	))
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	if err := ch.Confirm(false); err != nil {
		return err
	}

	if err := ch.Qos(r.prefetch, 0, false); err != nil {
		return err
	}

	r.conn = conn
	r.ch = ch
	return nil
}

func (r *RabbitMQ) IsAlive() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if both connection and channel are initialized and not closed
	if r.conn == nil || r.conn.IsClosed() {
		return core.ErrMBConn
	}

	if r.ch == nil || r.ch.IsClosed() {
		return core.ErrMBCh
	}

	return nil
}

func (r *RabbitMQ) Close() error {
	if r.ch != nil && !r.ch.IsClosed() {
		if err := r.ch.Close(); err != nil {
			return fmt.Errorf("close rabbitmq channel: %v", err)
		}
	}

	if r.conn != nil && r.conn.IsClosed() {
		if err := r.conn.Close(); err != nil {
			return fmt.Errorf("close rabbitmq connection: %v", err)
		}
	}
	return nil
}

func (r *RabbitMQ) PushMessage(ctx context.Context, message dto.OrderMessage) error {
	log := r.mylog.With("action", "pushMessage")

	if r.conn.IsClosed() {
		log.Error("connection between rabbitmq is closed", fmt.Errorf("closed conn"))
		go r.reconnect(r.ctx)
		return fmt.Errorf("rabbitmq: connection lose")
	}

	routingKey := "order_update_messages"
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return r.ch.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (r *RabbitMQ) reconnect(ctx context.Context) {
	r.mu.Lock()
	if r.reconnecting {
		r.mu.Unlock()
		return
	}
	r.reconnecting = true
	r.mu.Unlock()

	t := time.NewTicker(time.Second * core.MBReconnInterval)
	log := r.mylog.With("action", "rabbitmq-reconnecting")

	for {
		select {
		case <-t.C:
			err := r.connect()
			if err == nil {
				log.Info("rabbitmq reconnected!")
				return
			}
			log.Info("rabbitmq failed to reconnect")

		case <-ctx.Done():
			t.Stop()
			return
		}
	}
}

func (r *RabbitMQ) ConsumeMessage(ctx context.Context, queue, workerName string) (<-chan amqp.Delivery, error) {
	return r.ch.ConsumeWithContext(ctx, queue, workerName, false, false, false, false, nil)
}
