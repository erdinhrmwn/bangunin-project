package entity

import (
	"time"

	"github.com/google/uuid"
)

type SupplierBankAccount struct {
	ID            uuid.UUID
	SupplierID    uuid.UUID
	BankCode      string
	AccountNumber string
	AccountName   string
	IsDefault     bool
	CreatedAt     time.Time
}
