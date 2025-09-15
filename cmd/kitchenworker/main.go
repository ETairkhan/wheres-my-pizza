package kitchenworker

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"wheres-my-pizza/internal/kitchenworker/worker"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"
)

func Main() {
	workerName := flag.String("worker-name", "", "Unique name for the worker")
	orderTypes := flag.String("order-types", "", "Comma-separated list of order types")
	heartbeatInterval := flag.Int("heartbeat-interval", 30, "Interval between heartbeats in seconds")
	prefetch := flag.Int("prefetch", 1, "RabbitMQ prefetch count")
	flag.Parse()

	if *workerName == "" {
		log.Fatal("worker-name is required")
	}

	logger := logger.NewLogger("kitchen-worker")
	logger.Info("startup", "service_started", "Kitchen Worker starting")

	// Load configuration
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		logger.Error("startup", "config_load_failed", "Failed to load configuration", err)
		log.Fatal(err)
	}

	// Parse order types
	var orderTypesList []string
	if *orderTypes != "" {
		orderTypesList = strings.Split(*orderTypes, ",")
	}

	// Create worker
	kitchenWorker := worker.NewKitchenWorker(*workerName, orderTypesList, *heartbeatInterval, *prefetch, cfg, logger)

	// Start worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := kitchenWorker.Start(ctx); err != nil {
			logger.Error("startup", "worker_start_failed", "Failed to start worker", err)
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown", "graceful_shutdown", "Shutting down worker...")

	cancel()
	kitchenWorker.Stop()

	logger.Info("shutdown", "service_stopped", "Worker exiting")
}
