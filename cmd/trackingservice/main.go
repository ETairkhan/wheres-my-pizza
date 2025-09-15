package trackingservice

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wheres-my-pizza/cmd/trackingservice/server"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"
)

func Main() {
	port := flag.Int("port", 3002, "HTTP port for the API")
	flag.Parse()

	logger := logger.NewLogger("tracking-service")
	logger.Info("startup", "service_started", "Tracking Service starting")

	// Load configuration
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		logger.Error("startup", "config_load_failed", "Failed to load configuration", err)
		log.Fatal(err)
	}

	// Create server
	srv := server.NewServer(*port, cfg, logger)

	// Start server in a goroutine
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Error("startup", "server_start_failed", "Failed to start server", err)
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutdown", "graceful_shutdown", "Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown", "shutdown_failed", "Server forced to shutdown", err)
		log.Fatal(err)
	}

	logger.Info("shutdown", "service_stopped", "Server exiting")
}
