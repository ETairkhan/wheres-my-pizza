package notificationsubscriber

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"wheres-my-pizza/internal/notificationsubscriber/subscriber"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"
)

func Main() {
	flag.Parse() // No specific flags for notification subscriber

	logger := logger.NewLogger("notification-subscriber")
	logger.Info("startup", "service_started", "Notification Subscriber starting")

	// Load configuration
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		logger.Error("startup", "config_load_failed", "Failed to load configuration", err)
		log.Fatal(err)
	}

	// Create subscriber
	notifSubscriber := subscriber.NewNotificationSubscriber(cfg, logger)

	// Start subscriber
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := notifSubscriber.Start(ctx); err != nil {
			logger.Error("startup", "subscribe_start_failed", "Failed to start subscriber", err)
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown", "graceful_shutdown", "Shutting down subscriber...")
	cancel()
	notifSubscriber.Stop()

	logger.Info("shutdown", "service_stopped", "Subscriber exiting")
}
