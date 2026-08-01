package repository

import (
	"context"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type CategoryRepository interface {
	Create(ctx context.Context, c *entity.Category) error
	Update(ctx context.Context, c *entity.Category) error
	Delete(ctx context.Context, id int) error
	FindByID(ctx context.Context, id int) (*entity.Category, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Category, error)
	ListTree(ctx context.Context) ([]*entity.Category, error)
	CountChildren(ctx context.Context, id int) (int, error)
	CountProducts(ctx context.Context, id int) (int, error)
}
