package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type LedgerEntryRepository interface {
	Create(ctx context.Context, e *entity.LedgerEntry) error
	ListBySupplier(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.LedgerEntry, int, error)
	// SumByType sums entries of the given type, platform-wide, created
	// within [from, to] (FR-7.4, used for total commission).
	SumByType(ctx context.Context, entryType string, from, to time.Time) (float64, error)
}
