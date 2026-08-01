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

type ProductImageRepository struct {
	db *pgxpool.Pool
}

func NewProductImageRepository(db *pgxpool.Pool) *ProductImageRepository {
	return &ProductImageRepository{db: db}
}

func (r *ProductImageRepository) Create(ctx context.Context, img *entity.ProductImage) error {
	const q = `
		INSERT INTO product_images (id, product_id, url, is_primary, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at
	`
	err := r.db.QueryRow(ctx, q, img.ID, img.ProductID, img.URL, img.IsPrimary, img.SortOrder).Scan(&img.CreatedAt)
	if err != nil {
		return fmt.Errorf("postgres: create product image: %w", err)
	}
	return nil
}

func (r *ProductImageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM product_images WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres: delete product image: %w", err)
	}
	return nil
}

func (r *ProductImageRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.ProductImage, error) {
	const q = selectImageCols + `FROM product_images WHERE id = $1`
	img, err := scanImage(r.db.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find product image: %w", err)
	}
	return img, nil
}

func (r *ProductImageRepository) ListByProductID(ctx context.Context, productID uuid.UUID) ([]*entity.ProductImage, error) {
	const q = selectImageCols + `FROM product_images WHERE product_id = $1 ORDER BY sort_order, created_at`
	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list product images: %w", err)
	}
	defer rows.Close()

	var out []*entity.ProductImage
	for rows.Next() {
		img, err := scanImage(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan product image: %w", err)
		}
		out = append(out, img)
	}
	return out, rows.Err()
}

func (r *ProductImageRepository) CountByProductID(ctx context.Context, productID uuid.UUID) (int, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM product_images WHERE product_id = $1`, productID).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres: count product images: %w", err)
	}
	return n, nil
}

func (r *ProductImageRepository) UnsetPrimary(ctx context.Context, productID uuid.UUID) error {
	const q = `UPDATE product_images SET is_primary = false WHERE product_id = $1 AND is_primary = true`
	if _, err := r.db.Exec(ctx, q, productID); err != nil {
		return fmt.Errorf("postgres: unset primary product image: %w", err)
	}
	return nil
}

func (r *ProductImageRepository) SetPrimary(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE product_images SET is_primary = true WHERE id = $1`
	if _, err := r.db.Exec(ctx, q, id); err != nil {
		return fmt.Errorf("postgres: set primary product image: %w", err)
	}
	return nil
}

func (r *ProductImageRepository) FindPrimary(ctx context.Context, productID uuid.UUID) (*entity.ProductImage, error) {
	const q = selectImageCols + `FROM product_images WHERE product_id = $1 AND is_primary = true`
	img, err := scanImage(r.db.QueryRow(ctx, q, productID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find primary product image: %w", err)
	}
	return img, nil
}

const selectImageCols = `
	SELECT id, product_id, url, is_primary, sort_order, created_at
`

func scanImage(row rowScanner) (*entity.ProductImage, error) {
	var img entity.ProductImage
	err := row.Scan(&img.ID, &img.ProductID, &img.URL, &img.IsPrimary, &img.SortOrder, &img.CreatedAt)
	return &img, err
}
