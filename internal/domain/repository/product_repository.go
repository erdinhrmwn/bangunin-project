package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

const (
	SortNewest    = "newest"
	SortPriceAsc  = "price_asc"
	SortPriceDesc = "price_desc"
	SortRating    = "rating"
)

// SearchParams drives the public catalog search (FR-4.6). Cursor is an
// opaque, base64-encoded keyset token from a previous Search call's
// nextCursor — empty for the first page. Zero value fields (CategoryID,
// CityID, MinPrice, MaxPrice) mean "no filter".
type SearchParams struct {
	Query      string
	CategoryID int
	CityID     int
	MinPrice   float64
	MaxPrice   float64
	Sort       string
	Cursor     string
	Limit      int
}

// ProductSummary is the compact public catalog projection (FR-4.6): id,
// slug, name, primary image, price range, unit, supplier name/city, rating.
type ProductSummary struct {
	ID           uuid.UUID
	Slug         string
	Name         string
	PrimaryImage string
	PriceMin     float64
	PriceMax     float64
	Unit         string
	SupplierName string
	SupplierCity int
	RatingAvg    float64
	RatingCount  int
}

type ProductRepository interface {
	Create(ctx context.Context, p *entity.Product) error
	Update(ctx context.Context, p *entity.Product) error
	// UpdateRating recomputes the denormalized rating fields (FR-6.5) —
	// separate from Update so review creation doesn't touch unrelated columns.
	UpdateRating(ctx context.Context, productID uuid.UUID, avg float64, count int) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.Product, error)
	// FindByIDs batches lookup for N products in one query; result order is
	// not guaranteed to match ids, and missing ids are silently omitted.
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.Product, error)
	FindBySlug(ctx context.Context, slug string) (*entity.Product, error)
	FindBySupplierAndID(ctx context.Context, supplierID, id uuid.UUID) (*entity.Product, error)
	ListBySupplier(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.Product, int, error)
	Search(ctx context.Context, params SearchParams) ([]*ProductSummary, string, error)
	ListBySupplierPublic(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.Product, int, error)
}
