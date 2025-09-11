package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"wheres-my-pizza/internal/orderservice/handler"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/db"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/rabbitmq"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	port          int
	maxConcurrent int
	config        *config.Config
	logger        *logger.Logger
	httpServer    *http.Server
	dbPool        *pgxpool.Pool
	rabbitMQ      *rabbitmq.RabbitMQ
}

func NewServer(port, maxConcurrent int, cfg *config.Config, log *logger.Logger) *Server {
	return &Server{
		port:          port,
		maxConcurrent: maxConcurrent,
		config:        cfg,
		logger:        log,
	}
}

func (s *Server) Start() error {
	// Connect to database
	pool, err := db.ConnectDB(&s.config.Database, s.logger)
	if err != nil {
		return err
	}
	s.dbPool = pool

	// Connect to RabbitMQ
	rm, err := rabbitmq.ConnectRabbitMQ(&s.config.RabbitMQ, s.logger)
	if err != nil {
		return err
	}
	s.rabbitMQ = rm

	orderHandler := handler.NewOrderHandler(s.dbPool, rm.Channel, s.logger)

	// Setup HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/orders", orderHandler.CreateOrder)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("startup", "server_started", fmt.Sprintf("Order Service started on port %d", s.port))
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	// Close MQ
	if s.rabbitMQ != nil {
		s.rabbitMQ.Close()
	}
	// Close DB pool
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
