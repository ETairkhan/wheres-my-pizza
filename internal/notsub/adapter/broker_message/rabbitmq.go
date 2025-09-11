package brokermessage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"wheres-my-pizza/internal/xpkg/logger"
	"wheres-my-pizza/internal/notsub/app/core"
	"wheres-my-pizza/internal/xpkg/config"

	amqp "github.com/rabbitmq/amqp091-go"
)

const exchange = "notifications"

type RabbitMQ struct {
	ctx          context.Context
	cfg          *config.RabbitMQ
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
	rabbitmqCfg *config.RabbitMQ,
	mylog logger.Logger,
) (core.IRabbitMQ, error) {
	r := &RabbitMQ{
		ctx:          ctx,
		cfg:          rabbitmqCfg,
		mylog:        mylog,
		mu:           &sync.Mutex{},
		reconnecting: false,
	}
	err := r.connect()
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *RabbitMQ) IsAlive() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if both connection and channel are initialized and not closed
	if r.conn == nil || r.conn.IsClosed() {
		return false
	}
	if r.ch == nil || r.ch.IsClosed() {
		return false
	}

	return true
}

func (r *RabbitMQ) Close() error {
	if r.ch != nil && !r.ch.IsClosed() {
		if err := r.ch.Close(); err != nil {
			return fmt.Errorf("close rabbitmq channel: %v", err)
		}
	}

	if r.conn != nil && !r.conn.IsClosed() {
		if err := r.conn.Close(); err != nil {
			return fmt.Errorf("close rabbitmq connection: %v", err)
		}
	}
	return nil
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

	// try channel
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

func (r *RabbitMQ) reconnect(ctx context.Context) {
	r.mu.Lock()
	if r.reconnecting {
		r.mu.Unlock()
		return
	}
	r.reconnecting = true
	r.mu.Unlock()

	t := time.NewTicker(time.Second * 5)
	log := r.mylog.With("action", "rabbitmq-reconnecting")

	for {
		select {
		case <-t.C:
			// ..
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
