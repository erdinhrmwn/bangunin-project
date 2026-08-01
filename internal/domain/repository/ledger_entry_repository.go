package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type LedgerEntryRepository interface {
	Create(ctx context.Context, e *entity.LedgerEntry) error
	ListBySupplier(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.LedgerEntry, int, error)
}
