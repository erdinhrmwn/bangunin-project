package entity

import (
	"time"

	"github.com/google/uuid"
)

type CartItem struct {
	ID        uuid.UUID
	CartID    uuid.UUID
	VariantID uuid.UUID
	Qty       int
	CreatedAt time.Time
}
