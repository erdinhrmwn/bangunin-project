package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	ProductStatusDraft    = "draft"
	ProductStatusActive   = "active"
	ProductStatusInactive = "inactive"
	ProductStatusBanned   = "banned"
)

type Product struct {
	ID          uuid.UUID
	SupplierID  uuid.UUID
	CategoryID  int
	Name        string
	Slug        string
	Description string
	Specs       map[string]any
	Status      string
	RatingAvg   float64
	RatingCount int
	CreatedAt   time.Time
}
