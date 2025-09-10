package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	brokermessage "wheres-my-pizza/internal/kitchen/adapter/broker_message"
	"wheres-my-pizza/internal/kitchen/adapter/db"
	"wheres-my-pizza/internal/kitchen/app/core"
	"wheres-my-pizza/internal/kitchen/domain/dto"
	"wheres-my-pizza/internal/kitchen/domain/models"
	"wheres-my-pizza/internal/xpkg/config"
	"wheres-my-pizza/internal/xpkg/logger"

	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	online  = "online"
	offline = "offline"
)

type Worker struct {
	cfg   *config.Config
	mylog logger.Logger

	workerParams *core.WorkerParams
	orderRepo    core.IOrderRepo
	workerRepo   core.IWorkerRepo
	db           core.IDB
	mb           core.IRabbitMQ

	ctx          context.Context
	notifyCancel context.CancelFunc
	appCtx       context.Context

	mu sync.Mutex
	wg sync.WaitGroup

	ticker *time.Ticker
}

func NewWorker(
	notifyCtx context.Context,
	notifyCancel context.CancelFunc,
	appCtx context.Context,
	cfg *config.Config,
	workerParams *core.WorkerParams,
	mylog logger.Logger,
) *Worker {
	return &Worker{
		ctx:          notifyCtx,
		notifyCancel: notifyCancel,
		cfg:          cfg,
		appCtx:       appCtx,
		workerParams: workerParams,
		mylog:        mylog,
	}
}

// Run initializes worker that starts working. It return when the worker stops.
func (w *Worker) Run() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	mylog := w.mylog.Action("run-worker")

	// Initialize database connection
	if err := w.initializeDatabase(); err != nil {
		mylog.Action("db_connection_failed").Error("Failed to connect to database", err)
		return err
	}
	mylog.Action("db_connected").Info("Successful database connection")

	// Initialize RabbitMQ connection
	if err := w.initializeRabbitMQ(); err != nil {
		mylog.Action("mb_connection_failed").Error("Failed to connect to message broker", err)
		return err
	}
	mylog.Action("mb_connected").Info("Successful message broker connection")

	// Worker heartbeat
	w.ticker = time.NewTicker(time.Duration(w.workerParams.HeartbeatInterval) * time.Second)

	// Validate order
	if err := w.validateWorker(); err != nil {
		return fmt.Errorf("validate worker: %v", err)
	}

	// Consume messages from mb to give workers job
	jobsCh := w.startConsumers()
	w.work(jobsCh)

	return nil
}

func (w *Worker) validateWorker() error {
	mylog := w.mylog.Action("validateWorkerk")

	worker, err := w.workerRepo.Get(w.ctx, w.workerParams.WorkerName)
	typesToStr := strings.Join(w.workerParams.OrderTypes, ",")

	ctx, cancel := context.WithTimeout(w.ctx, time.Second*time.Duration(core.WaitTime))
	defer cancel()

	if err != nil {
		// if worker doesn't exist
		if err == pgx.ErrNoRows {
			mylog.Info("Starting to create worker")
			worker = models.Worker{
				Name: w.workerParams.WorkerName,
				Type: typesToStr,
			}
			id, err := w.workerRepo.Create(ctx, worker)
			if err != nil {
				mylog.Error("Failed to add worker to db", err)
				return err
			}
			worker.WorkerId = id
			mylog.Debug("Successfully created", "worker_id", id)
		} else {
			mylog.Error("Failed to get worker from db", err)
			return err
		}
	} else {
		if worker.Status == online {
			return fmt.Errorf("worker is online right now")
		}
		if worker.Type != typesToStr {
			mylog.Warn("worker have different type, updating type")
			return w.workerRepo.UpdateType(ctx, worker.Name, typesToStr)
		}
	}
	return nil
}

func (w *Worker) initializeDatabase() error {
	d, err := db.Start(context.Background(), w.cfg.DB, w.mylog)
	if err != nil {
		return err
	}

	orderRepo := db.NewOrderRepo(context.Background(), d, w.mylog)
	workerRepo := db.NewWorkerRepo(context.Background(), d)

	w.orderRepo = orderRepo
	w.workerRepo = workerRepo
	w.db = d
	return nil
}

func (w *Worker) initializeRabbitMQ() error {
	mb, err := brokermessage.New(w.ctx, *w.cfg.RMQ, w.mylog, w.workerParams.Prefetch)
	if err != nil {
		return fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}
	w.mb = mb
	return nil
}

func (w *Worker) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.mylog.Action("graceful_shutdown_started").Info("Shutting down")

	// Stop the ticker and cancel context
	w.ticker.Stop()

	// Wait for all workers to finish
	w.wg.Wait()

	ctx, cancel := context.WithTimeout(w.appCtx, core.WaitTime*time.Second)
	defer cancel()

	if err := w.workerRepo.SetOffline(ctx, w.workerParams.WorkerName); err != nil {
		w.mylog.Action("set_worker_status_offline_failed").Error("Failed to set worker status to 'offline'", err)
		return err
	}
	w.mylog.Action("set_worker_status_offline_completed").Info("Successfully set worker status to 'offline'")

	if w.db != nil {
		if err := w.db.Close(); err != nil {
			w.mylog.Action("db_close_failed").Error("failed to close database", err)
			return fmt.Errorf("db close %w", err)
		}
		w.mylog.Action("db_closed").Info("Database closed")
	}

	if w.mb != nil {
		if err := w.mb.Close(); err != nil {
			w.mylog.Action("mb_close_failed").Error("Failed to close message broker", err)
			return fmt.Errorf("mb close: %w", err)
		}
		w.mylog.Action("mb_closed").Info("Message broker closed")
	}

	w.mylog.Action("graceful_shutdown_completed").Info("Successfully shutted down")
	return nil
}

func (w *Worker) work(jobsCh <-chan amqp.Delivery) {
	log := w.mylog.Action("work")
	ctx, cancel := context.WithTimeout(context.Background(), core.WaitTime*time.Second)
	defer cancel()

	if err := w.workerRepo.UpdateLastSeen(ctx, w.workerParams.WorkerName); err != nil {
		w.mylog.Action("updateLastSeen").Error("Failed to update last seen and status field", err)
	}

	for {
		select {
		case msg, ok := <-jobsCh:
			if !ok {
				log.Debug("main work is done")
				return
			}

			// Process message
			if err, dlq := w.processMsg(msg); err != nil {
				w.mylog.Action("processMsg").Error("Failed to process order", err)
				if errors.Is(err, core.ErrDBConn) {
					w.mylog.Warn("db conn lose, trying reconnecting")
					// reconnecting (10 attempts)
					err := w.db.Reconnect()
					if err != nil {
						w.mylog.Warn("reconnecting failed, app is shutting down")
						w.notifyCancel()
					}
				}
				err = msg.Nack(false, dlq)
				if !dlq {
					w.mylog.Action("Nack").Debug("send to dead letter queue")
				}
				if err != nil {
					w.mylog.Action("Nack").Error("Failed to nack", err)
					if err := w.mb.IsAlive(); err != nil {
						w.mylog.Action("Nack").Info("no point to live, app is shutting down")
						w.notifyCancel()
					}
				}
			}

		case <-w.ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), core.WaitTime*time.Second)
			defer cancel()

			if err := w.workerRepo.UpdateLastSeen(ctx, w.workerParams.WorkerName); err != nil {
				w.mylog.Action("UpdateLastSeen").Error("Failed to update last seen field", err)
			}
			w.mylog.Action("UpdateLastSeen").Info("Successfully updated last seen")
		}
	}
}

// should not to send dead letter queue or not, this what is mean bool argument that return
func (w *Worker) processMsg(msg amqp.Delivery) (error, bool) {
	order := dto.OrderRequest{}
	err := json.Unmarshal(msg.Body, &order)
	if err != nil {
		return fmt.Errorf("unmarshal message: %v", err), false
	}
	w.mylog.Debug("new order received", "order-number", order.OrderNumber)
	// Change status of order
	ctx, cancel := context.WithTimeout(w.appCtx, time.Second*5)
	defer cancel()

	if err = w.orderRepo.SetStatusCooking(ctx, order.OrderNumber, w.workerParams.WorkerName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.mylog.Warn("possible error: contain order that already cooked")
			// order that already ready
			return fmt.Errorf("possible duplicate"), false
		}
		// reconnecting to db
		return core.ErrDBConn, true
	}

	workingDelay := core.SleepTime[order.Type]
	orderMsg := dto.OrderMessage{
		OrderNumber:         order.OrderNumber,
		ChangedBy:           w.workerParams.WorkerName,
		OldStatus:           "received",
		NewStatus:           "cooking",
		Timestamp:           time.Now().UTC().Format(time.RFC3339),
		EstimatedCompletion: time.Now().Add(time.Duration(workingDelay) * time.Second).UTC().Format(time.RFC3339),
	}

	if err := w.mb.PushMessage(w.appCtx, orderMsg); err != nil {
		w.mylog.Action("Publish").Error("Failed to publish message", err)
	}

	// Working delay
	time.Sleep(time.Duration(workingDelay) * time.Second)

	// Complete order
	ctx, cancel = context.WithTimeout(w.appCtx, time.Second*5)
	defer cancel()
	err = w.orderRepo.SetStatusReady(ctx, order.OrderNumber, w.workerParams.WorkerName)
	if err != nil {
		return core.ErrDBConn, true
	}

	orderMsg.OldStatus = "cooking"
	orderMsg.NewStatus = "ready"
	orderMsg.Timestamp = time.Now().UTC().Format(time.RFC3339)
	orderMsg.EstimatedCompletion = ""
	if err := w.mb.PushMessage(w.appCtx, orderMsg); err != nil {
		w.mylog.Action("mb_publish").Error("Failed to publish message", err)
	}

	// Acknowledged message
	if err := msg.Ack(false); err != nil{
		w.mylog.Action("Ack").Error("Failed to ack message", err)
		return err, true
	}
	w.mylog.Debug("Ack message", "order", order)
	return nil, true
}

func (w *Worker) startConsumers() chan amqp.Delivery{
	channels := make([]<- chan amqp.Delivery, 0 , len(core.AllowedOrderTypes))
	for _, orderType := range w.workerParams.OrderTypes{
		messageBus, err := w.mb.ConsumeMessage(w.appCtx, orderType, "")
		if err != nil {
			w.mylog.Action("consume_restart_failed").Error("Failed to re-consume", err)
			continue
		}
		channels = append(channels, messageBus)
	}
	return w.fanIn(channels...)
}

// fanIn merges multiple channels into one channel.

func (w *Worker) fanIn(channels ...<-chan amqp.Delivery) chan amqp.Delivery{
	out := make(chan amqp.Delivery, len(channels) * w.workerParams.Prefetch)

	w.wg.Add(len(channels))

	// Start a goroutine for each channel to handle the message consumption.
	for _, ch := range channels{
		go func(ch <-chan amqp.Delivery){
			defer w.wg.Done()
			for {
				select {
				case <-w.ctx.Done():
					w.mylog.Debug("ctx done fanIn is done")
					return
				case msg := <-ch:
					out <- msg
				}
			}
		}(ch)
	}

	// Close the output channel once all consumers have finished 
	go func(){
		w.wg.Wait()
		close(out)
	}()
	return out
}

