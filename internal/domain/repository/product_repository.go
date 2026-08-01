package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

// SearchParams drives the public catalog search (FR-4.6). Cursor is an
// opaque, base64-encoded keyset token from a previous Search call's
// nextCursor — empty for the first page.
type SearchParams struct {
	Query      string
	CategoryID int
	MinPrice   float64
	MaxPrice   float64
	Cursor     string
	Limit      int
}

type ProductRepository interface {
	Create(ctx context.Context, p *entity.Product) error
	Update(ctx context.Context, p *entity.Product) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Product, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Product, error)
	FindBySupplierAndID(ctx context.Context, supplierID, id uuid.UUID) (*entity.Product, error)
	ListBySupplier(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.Product, int, error)
	Search(ctx context.Context, params SearchParams) ([]*entity.Product, string, error)
	ListBySupplierPublic(ctx context.Context, supplierSlug string, page, perPage int) ([]*entity.Product, int, error)
}
