package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"where-is-my-pizza/internal/mylogger"
	"where-is-my-pizza/internal/notsub/app/core"
	"where-is-my-pizza/internal/notsub/config"
	"where-is-my-pizza/internal/notsub/domain/dto"

	brokermessage "where-is-my-pizza/internal/notsub/adapter/broker_message"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	online    = "online"
	offline   = "offline"
	queueName = "order_update"
)

type Notification struct {
	cfg    *config.Config
	mylog  mylogger.Logger
	mb     core.IRabbitMQ
	ctx    context.Context
	appCtx context.Context

	mu sync.Mutex
	wg sync.WaitGroup
}

func NewNotification(
	ctx context.Context,
	appCtx context.Context,
	cfg *config.Config,
	mylog mylogger.Logger,
) *Notification {
	return &Notification{
		ctx:    ctx,
		appCtx: appCtx,
		cfg:    cfg,
		mylog:  mylog,
	}
}

// Run initializes worker that starts n.rking. It returns n.en the n.rker stops.
func (n *Notification) Run() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	mylog := n.mylog.Action("Run-notifications")

	// Initialize RabbitMQ connection
	if err := n.initializeRabbitMQ(); err != nil {
		mylog.Action("mb_connection_failed").Error("Failed to connect to message broker", err)
		return err
	}
	mylog.Action("mb_connected").Info("Successful message broker connection")

	messageBus, err := n.mb.ConsumeMessage(n.appCtx, queueName, "")
	if err != nil {
		return fmt.Errorf("failed to consume message from rabbitmq: %v", err)
	}

	n.work(messageBus)

	return nil
}

func (n *Notification) Stop(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.mylog.Action("graceful_shutdown_started").Info("Shutting down")

	// wait for all workers to finish
	n.wg.Wait()

	if n.mb != nil {
		if err := n.mb.Close(); err != nil {
			n.mylog.Action("mb_close_failed").Error("Failed to close message broker", err)
			return fmt.Errorf("mb close: %w", err)
		}
		n.mylog.Action("mb_closed").Info("Message broker closed")
	}

	n.mylog.Action("graceful_shutdown_completed").Info("Successfully shutted down")
	return nil
}

func (n *Notification) work(notifCh <-chan amqp.Delivery) {
	for {
		select {
		case <-n.ctx.Done():
			n.mylog.Action("work_shutdown").Info("Stopping message consumption due to context cancel")
			return

		case msg, ok := <-notifCh:
			if !ok {
				return
			}
			n.wg.Add(1) // Track a new goroutine
			go func(msg amqp.Delivery) {
				defer n.wg.Done() // Decrease counter when done

				if err, dlq := n.processMsg(msg); err != nil {
					n.mylog.Action("processMsg").Error("Failed to process order", err)
					err = msg.Nack(false, dlq)
					if dlq {
						n.mylog.Action("Nack").Debug("send to dead letter queue")
					}
					if err != nil {
						n.mylog.Action("Nack").Error("Failed to nack", err)
					}
				}
			}(msg)
		}
	}
}

// should not to send dead letter queue or not, this n.at is mean bool argument that return
func (n *Notification) processMsg(msg amqp.Delivery) (error, bool) {
	order := dto.OrderMessage{}
	err := json.Unmarshal(msg.Body, &order)
	if err != nil {
		return fmt.Errorf("unmarshal message: %v", err), false
	}

	log := n.mylog.WithGroup("details").With("order_number", order.OrderNumber, "new_status", order.NewStatus)

	log.Action("notification_received").Info("Received status update for order")

	fmt.Printf("Notification for order %s: Status changed from '%s' to '%s' by %s.\n", order.OrderNumber, order.OldStatus, order.NewStatus, order.ChangedBy)

	// Acknowledged message
	if err := msg.Ack(false); err != nil {
		return fmt.Errorf("acknowledge message: %v", err), true
	}
	n.mylog.Debug("Ack message", "order", order)
	return nil, true
}

func (n *Notification) initializeRabbitMQ() error {
	mb, err := brokermessage.New(n.appCtx, n.cfg.RMQ, n.mylog)
	if err != nil {
		return fmt.Errorf("failed to connect to rabbitmq: %v", err)
	}
	n.mb = mb
	return nil
}
