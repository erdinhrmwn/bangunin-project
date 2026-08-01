package entity

import (
	"time"

	"github.com/google/uuid"
)

type ProductImage struct {
	ID        uuid.UUID
	ProductID uuid.UUID
	URL       string
	IsPrimary bool
	SortOrder int
	CreatedAt time.Time
}
