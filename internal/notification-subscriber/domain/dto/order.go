package dto

import "time"

type OrderRequest struct {
	OrderNumber     string    `json:"order_number"`
	CreatedAt       time.Time `json:"created_at"`
	CustomerName    string    `json:"customer_name"`
	Type            string    `json:"order_type"`
	Items           []Item    `json:"items"`
	TableNumber     int       `json:"table_number,omitempty"`
	DeliveryAddress string    `json:"delivery_address,omitempty"`
	TotalAmount     float64   `json:"total_amount"`
	Priority        int       `json:"priority"`
}

type OrderMessage struct {
	OrderNumber         string `json:"order_number"`
	OldStatus           string `json:"old_status"`
	NewStatus           string `json:"new_status"`
	ChangedBy           string `json:"changed_by"`
	Timestamp           string `json:"timestamp"`
	EstimatedCompletion string `json:"estimated_completion"`
}

type Item struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

type OrderResponse struct {
	OrderNumber string  `json:"order_number"`
	Status      string  `json:"status"`
	TotalAmount float64 `json:"total_amount"`
}

// type OrderStatusLog struct {
// 	Status    string    `json:"status"`
// 	ChangedBy string    `json:"changed_by"`
// 	ChangedAt time.Time `json:"changed_at"`
// 	Note      string    `json:"note"`
// }
