package models

import (
	"time"
)

type Order struct {
	ID              int64      `json:"id"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Number          string     `json:"number"`
	CustomerName    string     `json:"customer_name"`
	Type            string     `json:"type"`
	TableNumber     *int       `json:"table_number,omitempty"`
	DeliveryAddress *string    `json:"delivery_address,omitempty"`
	TotalAmount     float64    `json:"total_amount"`
	Priority        int        `json:"priority"`
	Status          string     `json:"status"`
	ProcessedBy     *string    `json:"processed_by,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

type OrderItem struct {
	ID        int64     `json:"id"`
	OrderID   int64     `json:"order_id"`
	Name      string    `json:"name"`
	Quantity  int       `json:"quantity"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

type OrderStatusLog struct {
	ID        int64     `json:"id"`
	OrderID   int64     `json:"order_id"`
	Status    string    `json:"status"`
	ChangedBy string    `json:"changed_by"`
	ChangedAt time.Time `json:"changed_at"`
	Notes     *string   `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Worker struct {
	ID              int64     `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	Name            string    `json:"name"`
	Type            string    `json:"type"`
	Status          string    `json:"status"`
	LastSeen        time.Time `json:"last_seen"`
	OrdersProcessed int       `json:"orders_processed"`
}

type CreateOrderRequest struct {
	CustomerName    string             `json:"customer_name"`
	OrderType       string             `json:"order_type"`
	TableNumber     *int               `json:"table_number,omitempty"`
	DeliveryAddress *string            `json:"delivery_address,omitempty"`
	Items           []OrderItemRequest `json:"items"`
}

type OrderItemRequest struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

type CreateOrderResponse struct {
	OrderNumber string  `json:"order_number"`
	Status      string  `json:"status"`
	TotalAmount float64 `json:"total_amount"`
}

type OrderMessage struct {
	OrderNumber     string             `json:"order_number"`
	CustomerName    string             `json:"customer_name"`
	OrderType       string             `json:"order_type"`
	TableNumber     *int               `json:"table_number,omitempty"`
	DeliveryAddress *string            `json:"delivery_address,omitempty"`
	Items           []OrderItemRequest `json:"items"`
	TotalAmount     float64            `json:"total_amount"`
	Priority        int                `json:"priority"`
}

type StatusUpdateMessage struct {
	OrderNumber         string    `json:"order_number"`
	OldStatus           string    `json:"old_status"`
	NewStatus           string    `json:"new_status"`
	ChangedBy           string    `json:"changed_by"`
	Timestamp           time.Time `json:"timestamp"`
	EstimatedCompletion time.Time `json:"estimated_completion"`
}
