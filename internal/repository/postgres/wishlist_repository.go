package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
)

type WishlistRepository struct {
	db *pgxpool.Pool
}

func NewWishlistRepository(db *pgxpool.Pool) *WishlistRepository {
	return &WishlistRepository{db: db}
}

func (r *WishlistRepository) Create(ctx context.Context, w *entity.Wishlist) error {
	const q = `
		INSERT INTO wishlists (id, user_id, product_id)
		VALUES ($1, $2, $3)
		RETURNING created_at
	`
	err := r.db.QueryRow(ctx, q, w.ID, w.UserID, w.ProductID).Scan(&w.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return errs.ErrConflict
		}
		return fmt.Errorf("postgres: create wishlist: %w", err)
	}
	return nil
}

func (r *WishlistRepository) Delete(ctx context.Context, userID, productID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM wishlists WHERE user_id = $1 AND product_id = $2`, userID, productID)
	if err != nil {
		return fmt.Errorf("postgres: delete wishlist: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *WishlistRepository) ExistsByUserAndProduct(ctx context.Context, userID, productID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM wishlists WHERE user_id = $1 AND product_id = $2)`, userID, productID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("postgres: check wishlist exists: %w", err)
	}
	return exists, nil
}

func (r *WishlistRepository) ListByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Wishlist, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM wishlists WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count wishlists: %w", err)
	}

	const q = `SELECT id, user_id, product_id, created_at FROM wishlists WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, userID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list wishlists: %w", err)
	}
	defer rows.Close()

	var out []*entity.Wishlist
	for rows.Next() {
		var w entity.Wishlist
		if err := rows.Scan(&w.ID, &w.UserID, &w.ProductID, &w.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("postgres: scan wishlist: %w", err)
		}
		out = append(out, &w)
	}
	return out, total, rows.Err()
}
