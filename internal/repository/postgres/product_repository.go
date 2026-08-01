package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository"
)

type ProductRepository struct {
	db *pgxpool.Pool
}

func NewProductRepository(db *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, p *entity.Product) error {
	const q = `
		INSERT INTO products (id, supplier_id, category_id, name, slug, description, specs, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`
	specs := p.Specs
	if specs == nil {
		specs = map[string]any{}
	}
	err := r.db.QueryRow(ctx, q, p.ID, p.SupplierID, p.CategoryID, p.Name, p.Slug, p.Description, specs, p.Status).
		Scan(&p.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return errs.ErrConflict
		}
		return fmt.Errorf("postgres: create product: %w", err)
	}
	return nil
}

func (r *ProductRepository) Update(ctx context.Context, p *entity.Product) error {
	const q = `
		UPDATE products
		SET category_id = $2, name = $3, slug = $4, description = $5, specs = $6, status = $7
		WHERE id = $1
	`
	specs := p.Specs
	if specs == nil {
		specs = map[string]any{}
	}
	_, err := r.db.Exec(ctx, q, p.ID, p.CategoryID, p.Name, p.Slug, p.Description, specs, p.Status)
	if err != nil {
		if isUniqueViolation(err) {
			return errs.ErrConflict
		}
		return fmt.Errorf("postgres: update product: %w", err)
	}
	return nil
}

func (r *ProductRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Product, error) {
	const q = selectProductCols + `FROM products WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *ProductRepository) FindBySlug(ctx context.Context, slug string) (*entity.Product, error) {
	const q = selectProductCols + `FROM products WHERE slug = $1`
	return r.scanOne(ctx, q, slug)
}

func (r *ProductRepository) FindBySupplierAndID(ctx context.Context, supplierID, id uuid.UUID) (*entity.Product, error) {
	const q = selectProductCols + `FROM products WHERE id = $1 AND supplier_id = $2`
	p, err := scanProduct(r.db.QueryRow(ctx, q, id, supplierID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find product: %w", err)
	}
	return p, nil
}

func (r *ProductRepository) ListBySupplier(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.Product, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM products WHERE supplier_id = $1`, supplierID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count products: %w", err)
	}

	const q = selectProductCols + `FROM products WHERE supplier_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, supplierID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list products: %w", err)
	}
	defer rows.Close()

	var out []*entity.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan product: %w", err)
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

func (r *ProductRepository) ListBySupplierPublic(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.Product, int, error) {
	var total int
	const countQ = `SELECT count(*) FROM products WHERE supplier_id = $1 AND status = 'active'`
	if err := r.db.QueryRow(ctx, countQ, supplierID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count supplier products: %w", err)
	}

	const q = selectProductCols + `FROM products WHERE supplier_id = $1 AND status = 'active' ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.db.Query(ctx, q, supplierID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list supplier products: %w", err)
	}
	defer rows.Close()

	var out []*entity.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan product: %w", err)
		}
		out = append(out, p)
	}
	return out, total, rows.Err()
}

const selectProductCols = `
	SELECT id, supplier_id, category_id, name, slug, description, specs, status, rating_avg, rating_count, created_at
`

func scanProduct(row rowScanner) (*entity.Product, error) {
	var p entity.Product
	err := row.Scan(
		&p.ID, &p.SupplierID, &p.CategoryID, &p.Name, &p.Slug, &p.Description, &p.Specs,
		&p.Status, &p.RatingAvg, &p.RatingCount, &p.CreatedAt,
	)
	return &p, err
}

func (r *ProductRepository) scanOne(ctx context.Context, query string, arg any) (*entity.Product, error) {
	p, err := scanProduct(r.db.QueryRow(ctx, query, arg))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find product: %w", err)
	}
	return p, nil
}

// --- Search (FR-4.6) ---
//
// searchCursor is the decoded form of SearchParams.Cursor: the sort column's
// value at the last row of the previous page, plus its product id as a
// tiebreaker for a stable keyset. V's meaning depends on Sort:
// newest -> RFC3339 timestamp, price_asc/price_desc -> price_min decimal
// string, rating -> rating_avg decimal string, relevance -> ts_rank decimal
// string.
type searchCursor struct {
	V  string    `json:"v"`
	ID uuid.UUID `json:"id"`
}

func encodeCursor(v string, id uuid.UUID) string {
	raw, _ := json.Marshal(searchCursor{V: v, ID: id})
	return base64.URLEncoding.EncodeToString(raw)
}

func decodeCursor(s string) (*searchCursor, error) {
	raw, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	var c searchCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// searchOrder holds the per-sort-mode SQL fragments: the aggregated column
// to sort/paginate on, its keyset comparison operator, and the cast used to
// decode the opaque cursor value back into that column's type.
type searchOrder struct {
	column     string
	direction  string
	cursorCast string
}

var searchOrders = map[string]searchOrder{
	repository.SortNewest:    {"created_at", "DESC", "::timestamptz"},
	repository.SortPriceAsc:  {"price_min", "ASC", "::numeric"},
	repository.SortPriceDesc: {"price_min", "DESC", "::numeric"},
	repository.SortRating:    {"rating_avg", "DESC", "::numeric"},
	"relevance":              {"rank", "DESC", "::real"},
}

func (r *ProductRepository) Search(ctx context.Context, params repository.SearchParams) ([]*repository.ProductSummary, string, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	sortMode := params.Sort
	if sortMode == "" {
		if params.Query != "" {
			sortMode = "relevance"
		} else {
			sortMode = repository.SortNewest
		}
	}
	order, ok := searchOrders[sortMode]
	if !ok {
		order = searchOrders[repository.SortNewest]
	}

	args := []any{params.Query}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	where := "p.status = 'active' AND s.status = 'approved'"
	where += " AND ($1 = '' OR p.search_vector @@ websearch_to_tsquery('indonesian', $1))"
	if params.CategoryID > 0 {
		where += fmt.Sprintf(" AND p.category_id = %s", arg(params.CategoryID))
	}
	if params.CityID > 0 {
		where += fmt.Sprintf(" AND s.origin_city_id = %s", arg(params.CityID))
	}

	having := ""
	if params.MinPrice > 0 {
		having += fmt.Sprintf(" AND COALESCE(MIN(v.price) FILTER (WHERE v.is_active), 0) >= %s", arg(params.MinPrice))
	}
	if params.MaxPrice > 0 {
		having += fmt.Sprintf(" AND COALESCE(MAX(v.price) FILTER (WHERE v.is_active), 0) <= %s", arg(params.MaxPrice))
	}

	keyset := ""
	if params.Cursor != "" {
		c, err := decodeCursor(params.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("postgres: decode cursor: %w", err)
		}
		op := ">"
		if order.direction == "DESC" {
			op = "<"
		}
		vArg := arg(c.V)
		idArg := arg(c.ID)
		keyset = fmt.Sprintf(" AND (%s, id) %s (%s%s, %s)", order.column, op, vArg, order.cursorCast, idArg)
	}

	limitArg := arg(limit + 1)

	q := fmt.Sprintf(`
		SELECT id, slug, name, primary_image, price_min, price_max, unit,
			supplier_name, supplier_city, rating_avg, rating_count, %s::text AS cursor_val
		FROM (
			SELECT
				p.id, p.slug, p.name, p.created_at, p.rating_avg, p.rating_count,
				s.store_name AS supplier_name, s.origin_city_id AS supplier_city,
				COALESCE(MIN(v.price) FILTER (WHERE v.is_active), 0) AS price_min,
				COALESCE(MAX(v.price) FILTER (WHERE v.is_active), 0) AS price_max,
				(SELECT v2.unit FROM product_variants v2 WHERE v2.product_id = p.id AND v2.is_active ORDER BY v2.price LIMIT 1) AS unit,
				(SELECT pi.url FROM product_images pi WHERE pi.product_id = p.id AND pi.is_primary LIMIT 1) AS primary_image,
				ts_rank(p.search_vector, websearch_to_tsquery('indonesian', $1)) AS rank
			FROM products p
			JOIN suppliers s ON s.id = p.supplier_id
			LEFT JOIN product_variants v ON v.product_id = p.id
			WHERE %s
			GROUP BY p.id, s.store_name, s.origin_city_id
			HAVING true%s
		) agg
		WHERE true%s
		ORDER BY %s %s, id %s
		LIMIT %s
	`, order.column, where, having, keyset, order.column, order.direction, order.direction, limitArg)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("postgres: search products: %w", err)
	}
	defer rows.Close()

	var out []*repository.ProductSummary
	var cursorVals []string
	for rows.Next() {
		var s repository.ProductSummary
		var primaryImage *string
		var unit *string
		var cursorVal string
		if err := rows.Scan(
			&s.ID, &s.Slug, &s.Name, &primaryImage, &s.PriceMin, &s.PriceMax, &unit,
			&s.SupplierName, &s.SupplierCity, &s.RatingAvg, &s.RatingCount, &cursorVal,
		); err != nil {
			return nil, "", fmt.Errorf("postgres: scan product summary: %w", err)
		}
		if primaryImage != nil {
			s.PrimaryImage = *primaryImage
		}
		if unit != nil {
			s.Unit = *unit
		}
		out = append(out, &s)
		cursorVals = append(cursorVals, cursorVal)
	}
	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("postgres: search products: %w", err)
	}

	var next string
	if len(out) > limit {
		out = out[:limit]
		cursorVals = cursorVals[:limit]
		next = encodeCursor(cursorVals[limit-1], out[limit-1].ID)
	}
	return out, next, nil
}
