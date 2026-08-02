// Package wishlist implements user product wishlists (FR-7.1): add/remove/
// list, and an IsWishlisted lookup used to flag product detail responses.
package wishlist

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
)

type Usecase struct {
	wishlists repository.WishlistRepository
	products  repository.ProductRepository
}

func New(wishlists repository.WishlistRepository, products repository.ProductRepository) *Usecase {
	return &Usecase{wishlists: wishlists, products: products}
}

func (u *Usecase) Add(ctx context.Context, userID, productID uuid.UUID) error {
	if _, err := u.products.FindByID(ctx, productID); err != nil {
		return err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("wishlist: generate id: %w", err)
	}
	return u.wishlists.Create(ctx, &entity.Wishlist{ID: id, UserID: userID, ProductID: productID})
}

func (u *Usecase) Remove(ctx context.Context, userID, productID uuid.UUID) error {
	return u.wishlists.Delete(ctx, userID, productID)
}

func (u *Usecase) IsWishlisted(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	return u.wishlists.ExistsByUserAndProduct(ctx, userID, productID)
}

func (u *Usecase) ListMine(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Wishlist, int, error) {
	return u.wishlists.ListByUser(ctx, userID, page, perPage)
}
