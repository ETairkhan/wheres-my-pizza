package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"wheres-my-pizza/internal/tracking-service/service"
	"wheres-my-pizza/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TrackingHandler struct {
	service *service.TrackingService
	logger  *logger.Logger
}

func NewTrackingHandler(dbPool *pgxpool.Pool, logger *logger.Logger) *TrackingHandler {
	trackingService := service.NewTrackingService(dbPool, logger)
	return &TrackingHandler{
		service: trackingService,
		logger:  logger,
	}
}

func (h *TrackingHandler) GetOrderStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order number from URL path
	path := r.URL.Path[len("/orders/"):]
	orderNumber := strings.TrimSuffix(path, "/status")
	orderNumber = strings.TrimSuffix(orderNumber, "/history")

	if orderNumber == "" {
		http.Error(w, "Order number is required", http.StatusBadRequest)
		return
	}

	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = "req-" + time.Now().Format("20060102150405")
	}

	h.logger.Debug(requestID, "request_received", "Get order status request for order: "+orderNumber)

	status, err := h.service.GetOrderStatus(r.Context(), orderNumber)
	if err != nil {
		h.logger.Error(requestID, "db_query_failed", "Failed to get order status", err)
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (h *TrackingHandler) GetOrderHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract order number from URL path
	path := r.URL.Path[len("/orders/"):]
	orderNumber := strings.TrimSuffix(path, "/history")

	if orderNumber == "" {
		http.Error(w, "Order number is required", http.StatusBadRequest)
		return
	}

	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = "req-" + time.Now().Format("20060102150405")
	}

	h.logger.Debug(requestID, "request_received", "Get order history request for order: "+orderNumber)

	history, err := h.service.GetOrderHistory(r.Context(), orderNumber)
	if err != nil {
		h.logger.Error(requestID, "db_query_failed", "Failed to get order history", err)
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func (h *TrackingHandler) GetWorkersStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = "req-" + time.Now().Format("20060102150405")
	}

	h.logger.Debug(requestID, "request_received", "Get workers status request")

	workers, err := h.service.GetWorkersStatus(r.Context())
	if err != nil {
		h.logger.Error(requestID, "db_query_failed", "Failed to get workers status", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workers)
}
