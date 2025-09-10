package worker

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"wheres-my-pizza/internal/kitchen/adapter/brokermessage"
	"wheres-my-pizza/internal/kitchen/adapter/db"
	"wheres-my-pizza/internal/kitchen/app/core"
	"wheres-my-pizza/internal/kitchen/domain/models"
	"wheres-my-pizza/internal/xpkg/config"
	"wheres-my-pizza/internal/xpkg/logger"

	"github.com/jackc/pgx/v5"
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
			return fmt.Erorrf("worker is online right now")
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
