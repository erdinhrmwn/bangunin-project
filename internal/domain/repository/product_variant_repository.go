package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type ProductVariantRepository interface {
	Create(ctx context.Context, v *entity.ProductVariant) error
	Update(ctx context.Context, v *entity.ProductVariant) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.ProductVariant, error)
	FindByIDAndSupplier(ctx context.Context, id, supplierID uuid.UUID) (*entity.ProductVariant, error)
	ListByProductID(ctx context.Context, productID uuid.UUID) ([]*entity.ProductVariant, error)
	CountActiveByProductID(ctx context.Context, productID uuid.UUID) (int, error)
	// AdjustStock applies delta atomically and returns the new stock value.
	// Implementations must guard stock + delta >= 0 in the UPDATE's WHERE
	// clause and return errs.ErrConflict when no row is updated.
	AdjustStock(ctx context.Context, variantID uuid.UUID, delta int) (int, error)
}
