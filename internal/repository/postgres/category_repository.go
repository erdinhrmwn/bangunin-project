package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
)

type CategoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(ctx context.Context, c *entity.Category) error {
	const q = `
		INSERT INTO categories (parent_id, name, slug, is_active, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, q, c.ParentID, c.Name, c.Slug, c.IsActive, c.SortOrder).
		Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return fmt.Errorf("postgres: create category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) Update(ctx context.Context, c *entity.Category) error {
	const q = `
		UPDATE categories
		SET parent_id = $2, name = $3, slug = $4, is_active = $5, sort_order = $6
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q, c.ID, c.ParentID, c.Name, c.Slug, c.IsActive, c.SortOrder)
	if err != nil {
		return fmt.Errorf("postgres: update category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Exec(ctx, `DELETE FROM categories WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres: delete category: %w", err)
	}
	return nil
}

func (r *CategoryRepository) FindByID(ctx context.Context, id int) (*entity.Category, error) {
	const q = selectCategoryCols + `FROM categories WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *CategoryRepository) FindBySlug(ctx context.Context, slug string) (*entity.Category, error) {
	const q = selectCategoryCols + `FROM categories WHERE slug = $1`
	return r.scanOne(ctx, q, slug)
}

func (r *CategoryRepository) ListTree(ctx context.Context) ([]*entity.Category, error) {
	const q = selectCategoryCols + `FROM categories ORDER BY parent_id NULLS FIRST, sort_order, name`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("postgres: list categories: %w", err)
	}
	defer rows.Close()

	var out []*entity.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan category: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *CategoryRepository) CountChildren(ctx context.Context, id int) (int, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM categories WHERE parent_id = $1`, id).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres: count category children: %w", err)
	}
	return n, nil
}

func (r *CategoryRepository) CountProducts(ctx context.Context, id int) (int, error) {
	var n int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM products WHERE category_id = $1`, id).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres: count category products: %w", err)
	}
	return n, nil
}

const selectCategoryCols = `
	SELECT id, parent_id, name, slug, is_active, sort_order, created_at
`

func scanCategory(row rowScanner) (*entity.Category, error) {
	var c entity.Category
	err := row.Scan(&c.ID, &c.ParentID, &c.Name, &c.Slug, &c.IsActive, &c.SortOrder, &c.CreatedAt)
	return &c, err
}

func (r *CategoryRepository) scanOne(ctx context.Context, query string, arg any) (*entity.Category, error) {
	c, err := scanCategory(r.db.QueryRow(ctx, query, arg))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find category: %w", err)
	}
	return c, nil
}
