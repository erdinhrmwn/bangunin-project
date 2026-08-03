package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type OrderRepository interface {
	Create(ctx context.Context, o *entity.Order) error
	// UpdateStatus is guarded by fromStatus (WHERE status = fromStatus) to
	// prevent a race between two concurrent status transitions (e.g. a
	// webhook and a cron job) from both succeeding against a stale
	// in-memory status. Returns errs.ErrConflict if no row matched.
	UpdateStatus(ctx context.Context, id uuid.UUID, fromStatus, toStatus string) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Order, error)
	ListByCheckoutGroupID(ctx context.Context, checkoutGroupID uuid.UUID) ([]*entity.Order, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Order, int, error)
	ListAll(ctx context.Context, page, perPage int) ([]*entity.Order, int, error)
	ListBySupplierID(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.Order, int, error)
	// ListDeliveredBefore returns orders in 'delivered' status whose
	// implicit delivered_at (from shipments) is before cutoff, for the
	// order:autocomplete job.
	ListDeliveredBefore(ctx context.Context, cutoff time.Time) ([]*entity.Order, error)

	CreateItems(ctx context.Context, items []*entity.OrderItem) error
	ListItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entity.OrderItem, error)
	FindItemByID(ctx context.Context, itemID uuid.UUID) (*entity.OrderItem, error)

	// SalesSummary returns GMV (sum of Total) and order count for a
	// supplier's completed orders created within [from, to] (FR-7.3).
	SalesSummary(ctx context.Context, supplierID uuid.UUID, from, to time.Time) (gmv float64, ordersCount int, err error)
	// TopProducts returns the top-selling products by revenue for a
	// supplier's completed orders within [from, to] (FR-7.3).
	TopProducts(ctx context.Context, supplierID uuid.UUID, from, to time.Time, limit int) ([]*entity.TopProduct, error)
	// SalesPerDay returns a daily GMV/order-count series for a supplier's
	// completed orders within [from, to] (FR-7.3).
	SalesPerDay(ctx context.Context, supplierID uuid.UUID, from, to time.Time) ([]*entity.DailySales, error)

	// PlatformSummary returns GMV (sum of Total for completed orders) and a
	// per-status order count, platform-wide, within [from, to] (FR-7.4).
	PlatformSummary(ctx context.Context, from, to time.Time) (gmv float64, byStatus map[string]int, err error)
	// PlatformSalesPerDay returns a daily GMV/order-count series across all
	// suppliers' completed orders within [from, to] (FR-7.4).
	PlatformSalesPerDay(ctx context.Context, from, to time.Time) ([]*entity.DailySales, error)
}
