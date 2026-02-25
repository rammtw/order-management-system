package model

import "time"

type Product struct {
	ID               string
	SKU              string
	Name             string
	Description      string
	PriceCents       int64
	Quantity         int32
	ReservedQuantity int32
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (p *Product) Available() int32 {
	return p.Quantity - p.ReservedQuantity
}

type ReservationItem struct {
	ProductID string
	Quantity  int32
}
