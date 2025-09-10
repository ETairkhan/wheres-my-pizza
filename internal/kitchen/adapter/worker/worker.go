package worker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	brokermessage "wheres-my-pizza/internal/kitchen/adapter/broker_message"
	"wheres-my-pizza/internal/kitchen/adapter/db"
	"wheres-my-pizza/internal/kitchen/app/core"
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

func (w *Worker) work(jobsCh <-chan amqp.Delivery){
	log := w.mylog.Action("work")
	ctx, cancel := context.WithTimeout(context.Background(), core.WaitTime * time.Second)
	defer cancel()

	if err := w.workerRepo.UpdateLastSeen(ctx, w.workerParams.WorkerName); err != nil{
		w.mylog.Action("updateLastSeen").Error("Failed to update last seen and status field", err)
	}

	for {
		select{
			case msg, ok := <-jobsCh:
			if !ok{
				log.Debug("main work is done")
				return 
			} 

			// Process message 
			if err, dlq := w.processMsg(msg); err != nil{
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
				if !dlq{
					w.mylog.Action("Nack").Debug("send to dead letter queue")
				}
				if err != nil {
					w.mylog.Action("Nack").Error("Failed to nack", err)
					if err := w.mb.IsAlive(); err != nil{
						w.mylog.Action("Nack").Info("no point to live, app is shutting down")
						w.notifyCancel()
					}
				}
			}

		case <-w.ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), core.WaitTime * time.Second)
			defer cancel()

			if err := w.workerRepo.UpdateLastSeen(ctx, w.workerParams.WorkerName); err != nil{
				w.mylog.Action("UpdateLastSeen").Error("Failed to update last seen field"), err
			}
			w.mylog.Action("UpdateLastSeen").Info("Successfully updated last seen")
		}
	}
}
