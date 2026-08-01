package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type CheckoutGroupRepository interface {
	Create(ctx context.Context, g *entity.CheckoutGroup) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.CheckoutGroup, error)
	// Confirm atomically inserts the checkout group and every order bundle
	// (order, items, reservations, initial status history) in one TX,
	// re-validating stock >= requested per variant before committing
	// (FR-5.4). Returns errs.ErrConflict if any variant's stock is
	// insufficient at commit time.
	Confirm(ctx context.Context, g *entity.CheckoutGroup, orders []ConfirmOrder) error
}

// ConfirmOrder bundles one supplier's order with its line items, stock
// reservations, and initial status history entry for CheckoutGroupRepository.Confirm.
type ConfirmOrder struct {
	Order        *entity.Order
	Items        []*entity.OrderItem
	Reservations []*entity.StockReservation
	History      *entity.OrderStatusHistory
}
