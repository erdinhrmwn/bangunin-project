package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type UserAddressRepository interface {
	Create(ctx context.Context, a *entity.UserAddress) error
	Update(ctx context.Context, a *entity.UserAddress) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.UserAddress, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.UserAddress, error)
	// ClearDefault unsets IsDefault on every address for userID except keepID.
	ClearDefault(ctx context.Context, userID uuid.UUID, keepID uuid.UUID) error
}
