// Package inventory implements supplier stock adjustment and movement
// history (FR-4.5). Adjustments and their movement record are written
// atomically so stock_after is always consistent.
package inventory

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
)

type Usecase struct {
	variants  repository.ProductVariantRepository
	movements repository.StockMovementRepository
}

func New(variants repository.ProductVariantRepository, movements repository.StockMovementRepository) *Usecase {
	return &Usecase{variants: variants, movements: movements}
}

// AdjustStock changes a variant's stock by qty (positive or negative),
// ownership-checked via supplierID, and records a stock_movements row of
// type "adjustment" in the same transaction. Returns errs.ErrConflict if
// the result would go negative.
func (u *Usecase) AdjustStock(ctx context.Context, supplierID, variantID uuid.UUID, qty int, note string) (*entity.ProductVariant, error) {
	v, err := u.variants.FindByIDAndSupplier(ctx, variantID, supplierID)
	if err != nil {
		return nil, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("inventory: generate movement id: %w", err)
	}
	m := &entity.StockMovement{
		ID:        id,
		VariantID: variantID,
		Type:      entity.MovementTypeAdjustment,
		Qty:       qty,
		RefType:   entity.RefTypeManual,
		Note:      note,
	}
	newStock, err := u.variants.AdjustStockWithMovement(ctx, variantID, qty, m)
	if err != nil {
		return nil, err
	}
	v.Stock = newStock
	return v, nil
}

func (u *Usecase) ListMovements(ctx context.Context, supplierID, variantID uuid.UUID, page, perPage int) ([]*entity.StockMovement, int, error) {
	if _, err := u.variants.FindByIDAndSupplier(ctx, variantID, supplierID); err != nil {
		return nil, 0, err
	}
	return u.movements.ListByVariantID(ctx, variantID, page, perPage)
}
