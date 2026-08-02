package entity

import (
	"time"

	"github.com/google/uuid"
)

type Wishlist struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ProductID uuid.UUID
	CreatedAt time.Time
}
