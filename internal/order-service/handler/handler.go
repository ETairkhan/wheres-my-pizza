package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"wheres-my-pizza/internal/order-service/service"
	"wheres-my-pizza/internal/order-service/validation"
	"wheres-my-pizza/pkg/logger"
	"wheres-my-pizza/pkg/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rabbitmq/amqp091-go"
)

type OrderHandler struct {
	dbPool    *pgxpool.Pool
	rabbitMQ  *amqp091.Channel
	service   *service.OrderService
	validator *validation.OrderValidator
	logger    *logger.Logger
}

func NewOrderHandler(dbPool *pgxpool.Pool, rabbitMQ *amqp091.Channel, logger *logger.Logger) *OrderHandler {
	validator := validation.NewOrderValidator()
	orderService := service.NewOrderService(dbPool, rabbitMQ, logger)
	return &OrderHandler{
		dbPool:    dbPool,
		rabbitMQ:  rabbitMQ,
		service:   orderService,
		validator: validator,
		logger:    logger,
	}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = "req-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error(requestID, "validation_failed", "Invalid JSON payload", err)
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate the request
	if err := h.validator.Validate(&req); err != nil {
		h.logger.Error(requestID, "validation_failed", "Validation failed", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.logger.Debug(requestID, "order_received", "New order received")

	// Process the order
	response, err := h.service.CreateOrder(r.Context(), &req, requestID)
	if err != nil {
		h.logger.Error(requestID, "order_processing_failed", "Failed to create order", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
