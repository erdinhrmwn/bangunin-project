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

type BannerRepository struct {
	db *pgxpool.Pool
}

func NewBannerRepository(db *pgxpool.Pool) *BannerRepository {
	return &BannerRepository{db: db}
}

func (r *BannerRepository) Create(ctx context.Context, b *entity.Banner) error {
	const q = `
		INSERT INTO banners (id, image_url, link, starts_at, ends_at, sort_order, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`
	err := r.db.QueryRow(ctx, q, b.ID, b.ImageURL, b.Link, b.StartsAt, b.EndsAt, b.SortOrder, b.IsActive).
		Scan(&b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return fmt.Errorf("postgres: create banner: %w", err)
	}
	return nil
}

func (r *BannerRepository) Update(ctx context.Context, b *entity.Banner) error {
	const q = `
		UPDATE banners
		SET image_url = $2, link = $3, starts_at = $4, ends_at = $5, sort_order = $6, is_active = $7, updated_at = now()
		WHERE id = $1
		RETURNING updated_at
	`
	err := r.db.QueryRow(ctx, q, b.ID, b.ImageURL, b.Link, b.StartsAt, b.EndsAt, b.SortOrder, b.IsActive).Scan(&b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return errs.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("postgres: update banner: %w", err)
	}
	return nil
}

func (r *BannerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM banners WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres: delete banner: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (r *BannerRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Banner, error) {
	const q = selectBannerCols + `FROM banners WHERE id = $1`
	b, err := scanBanner(r.db.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find banner: %w", err)
	}
	return b, nil
}

func (r *BannerRepository) ListAll(ctx context.Context) ([]*entity.Banner, error) {
	const q = selectBannerCols + `FROM banners ORDER BY sort_order, created_at DESC`
	return r.list(ctx, q)
}

// ListActive returns banners flagged active and within their schedule
// (FR-7.2), ordered for display.
func (r *BannerRepository) ListActive(ctx context.Context) ([]*entity.Banner, error) {
	const q = selectBannerCols + `FROM banners WHERE is_active AND now() BETWEEN starts_at AND ends_at ORDER BY sort_order, created_at DESC`
	return r.list(ctx, q)
}

func (r *BannerRepository) list(ctx context.Context, query string) ([]*entity.Banner, error) {
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: list banners: %w", err)
	}
	defer rows.Close()

	var out []*entity.Banner
	for rows.Next() {
		b, err := scanBanner(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan banner: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

const selectBannerCols = `
	SELECT id, image_url, link, starts_at, ends_at, sort_order, is_active, created_at, updated_at
`

func scanBanner(row rowScanner) (*entity.Banner, error) {
	var b entity.Banner
	err := row.Scan(&b.ID, &b.ImageURL, &b.Link, &b.StartsAt, &b.EndsAt, &b.SortOrder, &b.IsActive, &b.CreatedAt, &b.UpdatedAt)
	return &b, err
}
