package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"wheres-my-pizza/internal/xpkg/logger"
	"wheres-my-pizza/internal/order/api/http/handle"
	"wheres-my-pizza/internal/order/app/core"
	"wheres-my-pizza/internal/order/app/services"
	"wheres-my-pizza/internal/xpkg/config"

	brokermessage "wheres-my-pizza/internal/order/adapter/broker_message"
	database "wheres-my-pizza/internal/order/adapter/db"
)

var ErrServerClosed = errors.New("Server closed")

type Server struct {
	mux         *http.ServeMux
	cfg         *config.Config
	srv         *http.Server
	orderParams *core.OrderParams
	mylog       logger.Logger
	db          core.IDB
	mb          core.IRabbitMQ
	ctx         context.Context
	appCtx      context.Context
	mu          sync.Mutex
	wg          sync.WaitGroup
}

func NewServer(ctx, appCtx context.Context, cfg *config.Config, orderParams *core.OrderParams, mylog logger.Logger) *Server {
	return &Server{
		ctx:         ctx,
		appCtx:      appCtx,
		cfg:         cfg,
		orderParams: orderParams,
		mylog:       mylog,
		mux:         http.NewServeMux(),
	}
}

// Run initializes routes and starts listening. It returns when the server stops.
func (s *Server) Run() error {
	mylog := s.mylog.Action("server_started")
	// Initialize database connection
	if err := s.initializeDatabase(); err != nil {
		mylog.Action("db_connection_failed").Error("Failed to connect to database", err)
		return err
	}
	mylog.Action("db_connected").Info("Successful database connection")

	// Initialize RabbitMQ connection
	if err := s.initializeRabbitMQ(); err != nil {
		mylog.Action("mb_connection_failed").Error("Failed to connect to message broker", err)
		return err
	}
	mylog.Action("mb_connected").Info("Successful message broker connection")

	// Configure routes and handlers
	s.Configure()

	s.mu.Lock()
	s.srv = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.orderParams.Port),
		Handler: s.mux,
	}
	s.mu.Unlock()

	mylog = mylog.WithGroup("details").With("port", s.orderParams.Port, "max-concurrent", s.orderParams.MaxConcurrent)

	mylog.Info("server is running")
	// Start the HTTP server and handle graceful shutdown
	return s.startHTTPServer()
}

// Stop provides a programmatic shutdown. Accepts a context for timeout control.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mylog.Action("graceful_shutdown_started").Info("Shutting down HTTP server...")

	s.wg.Wait()

	if s.srv != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, core.WaitTime*time.Second)
		defer cancel()

		if err := s.srv.Shutdown(shutdownCtx); err != nil {
			s.mylog.Action("graceful_shutdown_failed").Error("Failed to shut down HTTP server gracefully", err)
			return fmt.Errorf("http server shutdown: %w", err)
		}
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.mylog.Action("db_close_failed").Error("Failed to close database", err)
			return fmt.Errorf("db close: %w", err)
		}
		s.mylog.Action("db_closed").Info("Database closed")
	}

	if s.mb != nil {
		if err := s.mb.Close(); err != nil {
			s.mylog.Action("mb_close_failed").Error("Failed to close message broker", err)
			return fmt.Errorf("mb close: %w", err)
		}
		s.mylog.Action("mb_closed").Info("Message broker closed")
	}

	s.mylog.Action("graceful_shutdown_completed").Info("HTTP server shut down gracefully")
	return nil
}

func (s *Server) startHTTPServer() error {
	errCh := make(chan error, 1)

	go func() {
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		} else {
			errCh <- nil
		}
	}()

	select {
	case <-s.ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) initializeDatabase() error {
	db, err := database.Start(s.appCtx, s.cfg.DB, s.mylog)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	s.db = db
	return nil
}

func (s *Server) initializeRabbitMQ() error {
	mb, err := brokermessage.New(s.appCtx, *s.cfg.RMQ, s.mylog)
	if err != nil {
		return fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}
	s.mb = mb
	return nil
}

// Configure sets up the HTTP handlers for various APIs including Market Data, Data Mode control, and Health checks.
func (s *Server) Configure() {
	// Repositories and services
	orderRepo := database.NewOrderRepo(s.ctx, s.db, s.orderParams.MaxConcurrent)

	orderService := services.NewOrderService(s.ctx, orderRepo, s.mb, s.mylog)

	orderHandler := handle.NewOrderHandler(orderService, s.mylog)
	// Register routes
	s.mux.Handle("POST /orders", orderHandler.Create())
}
