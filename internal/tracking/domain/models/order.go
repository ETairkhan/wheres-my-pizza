package models

import "time"

type Order struct {
	OrderID         string    `json:"order_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	OrderNumber     string    `json:"order_number"`
	CustomerName    string    `json:"customer_name"`
	Type            string    `json:"type"`
	TableNumber     int       `json:"table_number"`
	DeliveryAddress string    `json:"delivery_address"`
	TotalAmount     float64   `json:"total_amount"`
	Priority        int       `json:"priority"`
	Status          string    `json:"status"`
	ProcessedBy     string    `json:"processed_by"`
	CompletedAt     time.Time `json:"completed_at"`
}

type OrderItem struct {
	ItemID    int     `json:"item_id"`
	OrderID   int     `json:"order_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type OrderStatusLog struct {
	ID        int       `json:"log_id"`
	OrderID   int       `json:"order_id"`
	Status    string    `json:"status"`
	ChangedBy string    `json:"changed_by"`
	ChangedAt time.Time `json:"changed_at"`
	Note      string    `json:"note"`
}
