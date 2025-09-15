package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
	"wheres-my-pizza/internal/trackingservice/handler"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/db"
	"wheres-my-pizza/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	port       int
	config     *config.Config
	logger     *logger.Logger
	httpServer *http.Server
	dbPool     *pgxpool.Pool
}

func NewServer(port int, cfg *config.Config, log *logger.Logger) *Server {
	return &Server{
		port:   port,
		config: cfg,
		logger: log,
	}
}

func (s *Server) Start() error {
	// Connect to database
	dbPool, err := db.ConnectDB(&s.config.Database, s.logger)
	if err != nil {
		return err
	}
	s.dbPool = dbPool

	// Create tracking handler
	trackingHandler := handler.NewTrackingHandler(s.dbPool, s.logger)

	// Setup HTTP server with routes
	mux := http.NewServeMux()

	// Order status endpoints
	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/orders" {
			http.Error(w, "Order number is required", http.StatusBadRequest)
			return
		}

		// Extract order number from path
		path := r.URL.Path[len("/orders/"):]

		// Check if it's a status or history request
		if strings.HasSuffix(path, "/status") {
			// Remove "/status" from the path to get just the order number
			orderNumber := path[:len(path)-7]
			if orderNumber == "" {
				http.Error(w, "Order number is required", http.StatusBadRequest)
				return
			}
			trackingHandler.GetOrderStatus(w, r)
		} else if strings.HasSuffix(path, "/history") {
			// Remove "/history" from the path to get just the order number
			orderNumber := path[:len(path)-8]
			if orderNumber == "" {
				http.Error(w, "Order number is required", http.StatusBadRequest)
				return
			}
			trackingHandler.GetOrderHistory(w, r)
		} else {
			// Default to status if no specific endpoint
			trackingHandler.GetOrderStatus(w, r)
		}
	})

	// Worker status endpoint
	mux.HandleFunc("/workers/status", trackingHandler.GetWorkersStatus)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "healthy", "service": "tracking-service"}`))
	})

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("startup", "server_started", fmt.Sprintf("Tracking Service started on port %d", s.port))
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.dbPool != nil {
		s.dbPool.Close()
	}
	return s.httpServer.Shutdown(ctx)
}
