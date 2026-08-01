package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	SupplierStatusDraft     = "draft"
	SupplierStatusPending   = "pending"
	SupplierStatusApproved  = "approved"
	SupplierStatusRejected  = "rejected"
	SupplierStatusSuspended = "suspended"
)

type Supplier struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	StoreName       string
	Slug            string
	Description     string
	Status          string
	OriginCityID    int
	PickupAddress   string
	OwnFleetEnabled bool
	FleetCoverageKM int
	FleetFlatRate   float64
	VerifiedAt      *time.Time
	CreatedAt       time.Time
}
