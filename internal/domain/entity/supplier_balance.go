package entity

import (
	"time"

	"github.com/google/uuid"
)

type SupplierBalance struct {
	SupplierID uuid.UUID
	Balance    float64
	UpdatedAt  time.Time
}
