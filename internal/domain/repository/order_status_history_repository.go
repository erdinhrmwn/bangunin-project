package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type OrderStatusHistoryRepository interface {
	Create(ctx context.Context, h *entity.OrderStatusHistory) error
	ListByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entity.OrderStatusHistory, error)
}
