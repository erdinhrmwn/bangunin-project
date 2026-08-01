package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type ShipmentRepository interface {
	Create(ctx context.Context, s *entity.Shipment) error
	Update(ctx context.Context, s *entity.Shipment) error
	FindByOrderID(ctx context.Context, orderID uuid.UUID) (*entity.Shipment, error)
}
