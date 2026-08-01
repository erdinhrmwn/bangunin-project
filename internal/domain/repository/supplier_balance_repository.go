package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type SupplierBalanceRepository interface {
	FindBySupplierID(ctx context.Context, supplierID uuid.UUID) (*entity.SupplierBalance, error)
	// Increment adds delta (negative to debit) to the supplier's balance,
	// creating the row with balance=delta if it doesn't exist yet, and
	// returns the resulting balance. Must be called within the caller's TX.
	Increment(ctx context.Context, supplierID uuid.UUID, delta float64) (newBalance float64, err error)
}
