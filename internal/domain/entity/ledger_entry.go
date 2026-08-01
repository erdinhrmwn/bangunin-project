package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	LedgerTypeCreditOrder     = "credit_order"
	LedgerTypeDebitCommission = "debit_commission"
	LedgerTypeDebitWithdraw   = "debit_withdraw"
)

// LedgerEntry is append-only (enforced by a DB trigger, CLAUDE.md §5.4) —
// never update or delete rows, only insert. Exactly one of OrderID /
// WithdrawRequestID is set, matching Type (DB CHECK enforces this).
type LedgerEntry struct {
	ID                uuid.UUID
	OrderID           *uuid.UUID
	WithdrawRequestID *uuid.UUID
	SupplierID        uuid.UUID
	Type              string
	Amount            float64
	BalanceAfter      float64
	CreatedAt         time.Time
}
