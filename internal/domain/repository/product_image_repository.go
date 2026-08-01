package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type ProductImageRepository interface {
	Create(ctx context.Context, img *entity.ProductImage) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.ProductImage, error)
	ListByProductID(ctx context.Context, productID uuid.UUID) ([]*entity.ProductImage, error)
	CountByProductID(ctx context.Context, productID uuid.UUID) (int, error)
	UnsetPrimary(ctx context.Context, productID uuid.UUID) error
	FindPrimary(ctx context.Context, productID uuid.UUID) (*entity.ProductImage, error)
}
