package models

import "time"

type Worker struct {
	WorkerId             string // uuid
	CreatedAt            time.Time
	Name                 string
	Type                 string
	Status               string
	LastActiveAt         time.Time
	TotalOrdersProcessed int
}
