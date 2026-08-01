package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type CartRepository interface {
	// FindOrCreateByUserID returns the user's cart, creating an empty one if
	// none exists yet (every user gets exactly one cart, lazily).
	FindOrCreateByUserID(ctx context.Context, userID uuid.UUID) (*entity.Cart, error)
	Touch(ctx context.Context, cartID uuid.UUID) error

	// UpsertItem inserts a cart item or increments qty if (cart_id,
	// variant_id) already exists (decision #8: ON CONFLICT DO UPDATE).
	UpsertItem(ctx context.Context, item *entity.CartItem) error
	UpdateItemQty(ctx context.Context, itemID uuid.UUID, qty int) error
	RemoveItem(ctx context.Context, itemID uuid.UUID) error
	ListItemsByCartID(ctx context.Context, cartID uuid.UUID) ([]*entity.CartItem, error)
	FindItemByID(ctx context.Context, itemID uuid.UUID) (*entity.CartItem, error)
}
