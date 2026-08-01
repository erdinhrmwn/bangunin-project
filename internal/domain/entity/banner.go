package entity

import (
	"time"

	"github.com/google/uuid"
)

type Banner struct {
	ID        uuid.UUID
	ImageURL  string
	Link      string
	StartsAt  time.Time
	EndsAt    time.Time
	SortOrder int
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
