package model

import "time"

type OrderStatus int32

const (
	OrderStatusUnspecified OrderStatus = iota
	OrderStatusPending
	OrderStatusConfirmed
	OrderStatusProcessing
	OrderStatusShipped
	OrderStatusDelivered
	OrderStatusCancelled
)

type OrderItem struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int32  `json:"quantity"`
	PriceCents  int64  `json:"price_cents"`
}

type Order struct {
	ID         string
	CustomerID string
	Items      []OrderItem
	Status     OrderStatus
	TotalCents int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
