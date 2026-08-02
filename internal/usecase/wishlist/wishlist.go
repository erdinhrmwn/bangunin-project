// Package wishlist implements user product wishlists (FR-7.1): add/remove/
// list, and an IsWishlisted lookup used to flag product detail responses.
package wishlist

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/pkg/cache"
)

const isWishlistedCacheTTL = 60 * time.Second

func isWishlistedCacheKey(userID, productID uuid.UUID) string {
	return "wishlist:flag:" + userID.String() + ":" + productID.String()
}

type Usecase struct {
	wishlists repository.WishlistRepository
	products  repository.ProductRepository
	rdb       *redis.Client
}

func New(wishlists repository.WishlistRepository, products repository.ProductRepository, rdb *redis.Client) *Usecase {
	return &Usecase{wishlists: wishlists, products: products, rdb: rdb}
}

func (u *Usecase) Add(ctx context.Context, userID, productID uuid.UUID) error {
	if _, err := u.products.FindByID(ctx, productID); err != nil {
		return err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("wishlist: generate id: %w", err)
	}
	if err := u.wishlists.Create(ctx, &entity.Wishlist{ID: id, UserID: userID, ProductID: productID}); err != nil {
		return err
	}
	_ = cache.Delete(ctx, u.rdb, isWishlistedCacheKey(userID, productID))
	return nil
}

func (u *Usecase) Remove(ctx context.Context, userID, productID uuid.UUID) error {
	if err := u.wishlists.Delete(ctx, userID, productID); err != nil {
		return err
	}
	_ = cache.Delete(ctx, u.rdb, isWishlistedCacheKey(userID, productID))
	return nil
}

// IsWishlisted is on the product-detail hot path, cached separately from the
// shared product cache entry (a per-user flag can't be baked into a cache
// key shared by every viewer without leaking one user's wishlist state to
// another).
func (u *Usecase) IsWishlisted(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	key := isWishlistedCacheKey(userID, productID)
	if v, ok, err := cache.GetJSON[bool](ctx, u.rdb, key); err == nil && ok {
		return v, nil
	}
	v, err := u.wishlists.ExistsByUserAndProduct(ctx, userID, productID)
	if err != nil {
		return false, err
	}
	_ = cache.SetJSON(ctx, u.rdb, key, v, isWishlistedCacheTTL)
	return v, nil
}

func (u *Usecase) ListMine(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Wishlist, int, error) {
	return u.wishlists.ListByUser(ctx, userID, page, perPage)
}
