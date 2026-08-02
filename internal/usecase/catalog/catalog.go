// Package catalog implements the public storefront (FR-4.6–FR-4.9): product
// search, product detail, and supplier store pages. Product detail is
// cache-aside 60s, invalidated by the product usecase on mutation via
// cache.ProductDetailKey.
package catalog

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/pkg/cache"
)

const detailCacheTTL = 60 * time.Second

type Usecase struct {
	products  repository.ProductRepository
	suppliers repository.SupplierRepository
	rdb       *redis.Client
}

func New(products repository.ProductRepository, suppliers repository.SupplierRepository, rdb *redis.Client) *Usecase {
	return &Usecase{products: products, suppliers: suppliers, rdb: rdb}
}

// Search returns the public catalog listing (FR-4.6). Filtering to active
// products from approved suppliers happens in the repository query.
func (u *Usecase) Search(ctx context.Context, params repository.SearchParams) ([]*repository.ProductSummary, string, error) {
	return u.products.Search(ctx, params)
}

// Detail returns a published product by slug, cache-aside 60s (FR-4.7).
func (u *Usecase) Detail(ctx context.Context, slug string) (*entity.Product, error) {
	key := cache.ProductDetailKey(slug)
	if cached, ok, err := cache.GetJSON[*entity.Product](ctx, u.rdb, key); err == nil && ok {
		return cached, nil
	}
	p, err := u.products.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if p.Status != entity.ProductStatusActive {
		return nil, errs.ErrNotFound
	}
	_ = cache.SetJSON(ctx, u.rdb, key, p, detailCacheTTL)
	return p, nil
}

// SupplierStore returns an approved supplier's public profile plus their
// paginated active products (FR-4.9).
func (u *Usecase) SupplierStore(ctx context.Context, slug string, page, perPage int) (*entity.Supplier, []*entity.Product, int, error) {
	s, err := u.suppliers.FindBySlug(ctx, slug)
	if err != nil {
		return nil, nil, 0, err
	}
	if s.Status != entity.SupplierStatusApproved {
		return nil, nil, 0, errs.ErrNotFound
	}
	products, total, err := u.products.ListBySupplierPublic(ctx, s.ID, page, perPage)
	if err != nil {
		return nil, nil, 0, err
	}
	return s, products, total, nil
}
