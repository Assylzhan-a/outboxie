package model

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Amount     float64   `json:"amount"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type OrderCreatedEvent struct {
	OrderID    uuid.UUID `json:"order_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Amount     float64   `json:"amount"`
	CreatedAt  time.Time `json:"created_at"`
}
