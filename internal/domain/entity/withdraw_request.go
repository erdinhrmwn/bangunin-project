package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	WithdrawStatusPending   = "pending"
	WithdrawStatusApproved  = "approved"
	WithdrawStatusRejected  = "rejected"
	WithdrawStatusDisbursed = "disbursed"
	WithdrawStatusFailed    = "failed"
)

type WithdrawRequest struct {
	ID             uuid.UUID
	SupplierID     uuid.UUID
	BankAccountID  uuid.UUID
	Amount         float64
	Status         string
	DisbursementID string
	RequestedAt    time.Time
	ProcessedAt    *time.Time
}
