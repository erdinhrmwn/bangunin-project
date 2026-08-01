// Package cart implements the user's shopping cart (FR-5.1): add/update/
// remove items and list them with read-time issue flags.
package cart

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/pkg/apperr"
)

type Usecase struct {
	carts    repository.CartRepository
	variants repository.ProductVariantRepository
	products repository.ProductRepository
}

func New(carts repository.CartRepository, variants repository.ProductVariantRepository, products repository.ProductRepository) *Usecase {
	return &Usecase{carts: carts, variants: variants, products: products}
}

// Item is a cart line enriched with current variant/product state. Flags
// are computed here at read-time (decision #8), not stored — cart_items has
// no price/name snapshot column, so unlike order_items there's nothing to
// diff a "price changed" flag against; only stock/min-order/active are
// checkable against current state.
type Item struct {
	ID              uuid.UUID
	VariantID       uuid.UUID
	ProductID       uuid.UUID
	ProductName     string
	VariantName     string
	Unit            string
	Price           float64
	Qty             int
	Stock           int
	MinOrderQty     int
	OutOfStock      bool
	BelowMinOrder   bool
	ProductInactive bool
}

func (u *Usecase) Get(ctx context.Context, userID uuid.UUID) ([]*Item, error) {
	c, err := u.carts.FindOrCreateByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	rows, err := u.carts.ListItemsByCartID(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	out := make([]*Item, 0, len(rows))
	for _, ci := range rows {
		v, err := u.variants.FindByID(ctx, ci.VariantID)
		if err != nil {
			return nil, err
		}
		p, err := u.products.FindByID(ctx, v.ProductID)
		if err != nil {
			return nil, err
		}
		out = append(out, &Item{
			ID:              ci.ID,
			VariantID:       v.ID,
			ProductID:       p.ID,
			ProductName:     p.Name,
			VariantName:     v.Name,
			Unit:            v.Unit,
			Price:           v.Price,
			Qty:             ci.Qty,
			Stock:           v.Stock,
			MinOrderQty:     v.MinOrderQty,
			OutOfStock:      v.Stock < ci.Qty,
			BelowMinOrder:   ci.Qty < v.MinOrderQty,
			ProductInactive: !v.IsActive || p.Status != entity.ProductStatusActive,
		})
	}
	return out, nil
}

func (u *Usecase) Add(ctx context.Context, userID, variantID uuid.UUID, qty int) error {
	if _, err := u.variants.FindByID(ctx, variantID); err != nil {
		return err
	}
	c, err := u.carts.FindOrCreateByUserID(ctx, userID)
	if err != nil {
		return err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("cart: generate item id: %w", err)
	}
	if err := u.carts.UpsertItem(ctx, &entity.CartItem{ID: id, CartID: c.ID, VariantID: variantID, Qty: qty}); err != nil {
		return err
	}
	return u.carts.Touch(ctx, c.ID)
}

func (u *Usecase) UpdateQty(ctx context.Context, userID, itemID uuid.UUID, qty int) error {
	c, err := u.ownedItem(ctx, userID, itemID)
	if err != nil {
		return err
	}
	if err := u.carts.UpdateItemQty(ctx, itemID, qty); err != nil {
		return err
	}
	return u.carts.Touch(ctx, c.ID)
}

func (u *Usecase) Remove(ctx context.Context, userID, itemID uuid.UUID) error {
	c, err := u.ownedItem(ctx, userID, itemID)
	if err != nil {
		return err
	}
	if err := u.carts.RemoveItem(ctx, itemID); err != nil {
		return err
	}
	return u.carts.Touch(ctx, c.ID)
}

// ownedItem verifies itemID belongs to userID's cart, returning
// errs.ErrForbidden otherwise so a user can't mutate another user's items
// by guessing an item UUID.
func (u *Usecase) ownedItem(ctx context.Context, userID, itemID uuid.UUID) (*entity.Cart, error) {
	item, err := u.carts.FindItemByID(ctx, itemID)
	if err != nil {
		return nil, err
	}
	c, err := u.carts.FindOrCreateByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if item.CartID != c.ID {
		return nil, apperr.New("FORBIDDEN", "Cart item does not belong to you", 403)
	}
	return c, nil
}
