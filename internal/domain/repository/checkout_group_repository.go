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
}
