package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	DocTypeNIB  = "nib"
	DocTypeKTP  = "ktp"
	DocTypeNPWP = "npwp"

	DocStatusPending  = "pending"
	DocStatusApproved = "approved"
	DocStatusRejected = "rejected"
)

type SupplierDocument struct {
	ID            uuid.UUID
	SupplierID    uuid.UUID
	DocType       string
	FileURL       string
	Status        string
	RejectionNote string
	CreatedAt     time.Time
}
