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
	// ListBelowThreshold returns all active variants with stock < threshold,
	// for the low-stock notification job (FR-4.10).
	ListBelowThreshold(ctx context.Context, threshold int) ([]*entity.ProductVariant, error)
	// AdjustStock applies delta atomically and returns the new stock value.
	// Implementations must guard stock + delta >= 0 in the UPDATE's WHERE
	// clause and return errs.ErrConflict when no row is updated.
	AdjustStock(ctx context.Context, variantID uuid.UUID, delta int) (int, error)
	// AdjustStockWithMovement applies delta and inserts m in the same DB
	// transaction, setting m.StockAfter and m.CreatedAt on success. Same
	// stock + delta >= 0 guard and errs.ErrConflict as AdjustStock.
	AdjustStockWithMovement(ctx context.Context, variantID uuid.UUID, delta int, m *entity.StockMovement) (int, error)
}
