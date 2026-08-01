package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type WishlistRepository interface {
	Create(ctx context.Context, w *entity.Wishlist) error
	Delete(ctx context.Context, userID, productID uuid.UUID) error
	ExistsByUserAndProduct(ctx context.Context, userID, productID uuid.UUID) (bool, error)
	ListByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Wishlist, int, error)
}
