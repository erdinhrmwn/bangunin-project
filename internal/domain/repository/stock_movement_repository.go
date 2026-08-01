package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type StockMovementRepository interface {
	Create(ctx context.Context, m *entity.StockMovement) error
	ListByVariantID(ctx context.Context, variantID uuid.UUID, page, perPage int) ([]*entity.StockMovement, int, error)
}
