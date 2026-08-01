package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
)

type CartRepository struct {
	db *pgxpool.Pool
}

func NewCartRepository(db *pgxpool.Pool) *CartRepository {
	return &CartRepository{db: db}
}

func (r *CartRepository) FindOrCreateByUserID(ctx context.Context, userID uuid.UUID) (*entity.Cart, error) {
	const q = `
		INSERT INTO carts (id, user_id) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET user_id = carts.user_id
		RETURNING id, user_id, updated_at
	`
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("postgres: generate cart id: %w", err)
	}

	var c entity.Cart
	if err := r.db.QueryRow(ctx, q, id, userID).Scan(&c.ID, &c.UserID, &c.UpdatedAt); err != nil {
		return nil, fmt.Errorf("postgres: find or create cart: %w", err)
	}
	return &c, nil
}

func (r *CartRepository) Touch(ctx context.Context, cartID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `UPDATE carts SET updated_at = now() WHERE id = $1`, cartID); err != nil {
		return fmt.Errorf("postgres: touch cart: %w", err)
	}
	return nil
}

func (r *CartRepository) UpsertItem(ctx context.Context, item *entity.CartItem) error {
	const q = `
		INSERT INTO cart_items (id, cart_id, variant_id, qty)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (cart_id, variant_id) DO UPDATE SET qty = cart_items.qty + excluded.qty
	`
	if _, err := r.db.Exec(ctx, q, item.ID, item.CartID, item.VariantID, item.Qty); err != nil {
		return fmt.Errorf("postgres: upsert cart item: %w", err)
	}
	return nil
}

func (r *CartRepository) UpdateItemQty(ctx context.Context, itemID uuid.UUID, qty int) error {
	if _, err := r.db.Exec(ctx, `UPDATE cart_items SET qty = $2 WHERE id = $1`, itemID, qty); err != nil {
		return fmt.Errorf("postgres: update cart item qty: %w", err)
	}
	return nil
}

func (r *CartRepository) RemoveItem(ctx context.Context, itemID uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM cart_items WHERE id = $1`, itemID); err != nil {
		return fmt.Errorf("postgres: remove cart item: %w", err)
	}
	return nil
}

func (r *CartRepository) ListItemsByCartID(ctx context.Context, cartID uuid.UUID) ([]*entity.CartItem, error) {
	rows, err := r.db.Query(ctx, selectCartItemCols+`FROM cart_items WHERE cart_id = $1 ORDER BY created_at`, cartID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list cart items: %w", err)
	}
	defer rows.Close()

	var out []*entity.CartItem
	for rows.Next() {
		i, err := scanCartItem(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan cart item: %w", err)
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *CartRepository) FindItemByID(ctx context.Context, itemID uuid.UUID) (*entity.CartItem, error) {
	i, err := scanCartItem(r.db.QueryRow(ctx, selectCartItemCols+`FROM cart_items WHERE id = $1`, itemID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find cart item: %w", err)
	}
	return i, nil
}

const selectCartItemCols = `SELECT id, cart_id, variant_id, qty, created_at `

func scanCartItem(row rowScanner) (*entity.CartItem, error) {
	var i entity.CartItem
	err := row.Scan(&i.ID, &i.CartID, &i.VariantID, &i.Qty, &i.CreatedAt)
	return &i, err
}
