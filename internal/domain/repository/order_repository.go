package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type OrderRepository interface {
	Create(ctx context.Context, o *entity.Order) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Order, error)
	ListByCheckoutGroupID(ctx context.Context, checkoutGroupID uuid.UUID) ([]*entity.Order, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Order, int, error)
	ListBySupplierID(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.Order, int, error)
	// ListDeliveredBefore returns orders in 'delivered' status whose
	// implicit delivered_at (from shipments) is before cutoff, for the
	// order:autocomplete job.
	ListDeliveredBefore(ctx context.Context, cutoff time.Time) ([]*entity.Order, error)

	CreateItems(ctx context.Context, items []*entity.OrderItem) error
	ListItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entity.OrderItem, error)
}
