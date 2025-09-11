package models

import "time"

// only for db
type Order struct {
	OrderID         string
	OrderNumber     string
	CustomerName    string
	Type            string
	TableNumber     int
	DeliveryAddress string
	TotalAmount     float64
	Priority        int
	Status          string
	ProcessedBy     string
	CompletedAt     time.Time
}

type OrderItem struct {
	ItemID    int
	OrderID   int
	Name      string
	Quantity  int
	UnitPrice float64
}

type OrderStatusLog struct {
	ID        int
	OrderID   int
	Status    string
	ChangedBy string
	ChangedAt time.Time
	Note      string
}
