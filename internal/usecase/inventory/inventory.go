// Package inventory implements supplier stock adjustment and movement
// history (FR-4.5). Adjustments and their movement record are written
// atomically so stock_after is always consistent.
package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/pkg/notify"
)

// lowStockNotifyTTL caps low-stock notifications to once per variant per
// day (FR-4.10), tracked via a Redis SETNX-style key.
const lowStockNotifyTTL = 24 * time.Hour

type Usecase struct {
	variants  repository.ProductVariantRepository
	movements repository.StockMovementRepository
	suppliers repository.SupplierRepository
	notify    repository.NotificationRepository
	rdb       *redis.Client
	threshold int
}

func New(variants repository.ProductVariantRepository, movements repository.StockMovementRepository, suppliers repository.SupplierRepository, notify repository.NotificationRepository, rdb *redis.Client, threshold int) *Usecase {
	return &Usecase{variants: variants, movements: movements, suppliers: suppliers, notify: notify, rdb: rdb, threshold: threshold}
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

// CheckLowStock scans active variants under the configured threshold and
// notifies their supplier, at most once per variant per 24h (FR-4.10).
func (u *Usecase) CheckLowStock(ctx context.Context) error {
	variants, err := u.variants.ListBelowThreshold(ctx, u.threshold)
	if err != nil {
		return fmt.Errorf("inventory: list low-stock variants: %w", err)
	}

	for _, v := range variants {
		key := "lowstock:notified:" + v.ID.String()
		ok, err := u.rdb.SetNX(ctx, key, 1, lowStockNotifyTTL).Result()
		if err != nil {
			return fmt.Errorf("inventory: dedupe low-stock notify: %w", err)
		}
		if !ok {
			continue // already notified within the last 24h
		}

		s, err := u.suppliers.FindByID(ctx, v.SupplierID)
		if err != nil {
			return fmt.Errorf("inventory: find supplier for low-stock variant: %w", err)
		}
		title := "Low stock: " + v.Name
		body := fmt.Sprintf("%s (SKU %s) has %d units left, below the %d threshold.", v.Name, v.SKU, v.Stock, u.threshold)
		if err := notify.Send(ctx, u.notify, s.UserID, entity.NotificationTypeLowStock, title, body); err != nil {
			return fmt.Errorf("inventory: create low-stock notification: %w", err)
		}
	}
	return nil
}
