package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type BannerRepository interface {
	Create(ctx context.Context, b *entity.Banner) error
	Update(ctx context.Context, b *entity.Banner) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Banner, error)
	ListAll(ctx context.Context) ([]*entity.Banner, error)
	ListActive(ctx context.Context) ([]*entity.Banner, error)
}
