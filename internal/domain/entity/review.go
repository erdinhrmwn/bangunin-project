package entity

import (
	"time"

	"github.com/google/uuid"
)

type Review struct {
	ID          uuid.UUID
	OrderItemID uuid.UUID
	ProductID   uuid.UUID
	UserID      uuid.UUID
	Rating      int
	Comment     string
	CreatedAt   time.Time
}
