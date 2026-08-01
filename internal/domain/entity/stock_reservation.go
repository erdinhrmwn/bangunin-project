package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	StockReservationStatusActive    = "active"
	StockReservationStatusConverted = "converted"
	StockReservationStatusReleased  = "released"
)

type StockReservation struct {
	ID        uuid.UUID
	VariantID uuid.UUID
	OrderID   uuid.UUID
	Qty       int
	Status    string
	ExpiresAt time.Time
	CreatedAt time.Time
}
