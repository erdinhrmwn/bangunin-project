package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type StockReservationRepository interface {
	Create(ctx context.Context, r *entity.StockReservation) error
	ListByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entity.StockReservation, error)
	// ListExpired returns 'active' reservations whose expires_at is in the
	// past, for the order:expire job.
	ListExpired(ctx context.Context) ([]*entity.StockReservation, error)
	// UpdateStatus transitions a reservation (active -> converted|released).
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	// ApplyAdjustments atomically applies every adjustment (variant stock
	// delta + movement + reservation status) in one TX, so a partial failure
	// across multiple reservations for one order never leaves stock and
	// reservation status inconsistent with each other.
	ApplyAdjustments(ctx context.Context, adjustments []ReservationAdjustment) error
}

// ReservationAdjustment bundles one reservation's stock delta (with its
// movement record, skipped if Movement is nil) and target status for
// StockReservationRepository.ApplyAdjustments.
type ReservationAdjustment struct {
	ReservationID uuid.UUID
	VariantID     uuid.UUID
	Delta         int
	Movement      *entity.StockMovement
	Status        string
}
