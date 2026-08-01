package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type SupplierRepository interface {
	Create(ctx context.Context, s *entity.Supplier) error
	Update(ctx context.Context, s *entity.Supplier) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Supplier, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.Supplier, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Supplier, error)
	List(ctx context.Context, status, q string, page, perPage int) ([]*entity.Supplier, int, error)
}
