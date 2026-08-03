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

type ProductVariantRepository struct {
	db *pgxpool.Pool
}

func NewProductVariantRepository(db *pgxpool.Pool) *ProductVariantRepository {
	return &ProductVariantRepository{db: db}
}

func (r *ProductVariantRepository) Create(ctx context.Context, v *entity.ProductVariant) error {
	const q = `
		INSERT INTO product_variants (
			id, product_id, supplier_id, sku, name, unit, price, weight_gram,
			length_cm, width_cm, height_cm, min_order_qty, stock, is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`
	_, err := r.db.Exec(ctx, q,
		v.ID, v.ProductID, v.SupplierID, v.SKU, v.Name, v.Unit, v.Price, v.WeightGram,
		v.LengthCM, v.WidthCM, v.HeightCM, v.MinOrderQty, v.Stock, v.IsActive,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return errs.ErrConflict
		}
		return fmt.Errorf("postgres: create product variant: %w", err)
	}
	return nil
}

func (r *ProductVariantRepository) Update(ctx context.Context, v *entity.ProductVariant) error {
	const q = `
		UPDATE product_variants
		SET sku = $2, name = $3, unit = $4, price = $5, weight_gram = $6,
			length_cm = $7, width_cm = $8, height_cm = $9, min_order_qty = $10, is_active = $11
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q,
		v.ID, v.SKU, v.Name, v.Unit, v.Price, v.WeightGram,
		v.LengthCM, v.WidthCM, v.HeightCM, v.MinOrderQty, v.IsActive,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return errs.ErrConflict
		}
		return fmt.Errorf("postgres: update product variant: %w", err)
	}
	return nil
}

func (r *ProductVariantRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.ProductVariant, error) {
	const q = selectVariantCols + `FROM product_variants WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *ProductVariantRepository) FindByIDAndSupplier(ctx context.Context, id, supplierID uuid.UUID) (*entity.ProductVariant, error) {
	const q = selectVariantCols + `FROM product_variants WHERE id = $1 AND supplier_id = $2`
	v, err := scanVariant(r.db.QueryRow(ctx, q, id, supplierID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find product variant: %w", err)
	}
	return v, nil
}

func (r *ProductVariantRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.ProductVariant, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	const q = selectVariantCols + `FROM product_variants WHERE id = ANY($1)`
	rows, err := r.db.Query(ctx, q, ids)
	if err != nil {
		return nil, fmt.Errorf("postgres: find product variants: %w", err)
	}
	defer rows.Close()

	var out []*entity.ProductVariant
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan product variant: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *ProductVariantRepository) ListByProductID(ctx context.Context, productID uuid.UUID) ([]*entity.ProductVariant, error) {
	const q = selectVariantCols + `FROM product_variants WHERE product_id = $1 ORDER BY created_at`
	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list product variants: %w", err)
	}
	defer rows.Close()

	var out []*entity.ProductVariant
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan product variant: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *ProductVariantRepository) CountActiveByProductID(ctx context.Context, productID uuid.UUID) (int, error) {
	var n int
	const q = `SELECT count(*) FROM product_variants WHERE product_id = $1 AND is_active = true`
	if err := r.db.QueryRow(ctx, q, productID).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres: count active product variants: %w", err)
	}
	return n, nil
}

// ListBelowThreshold returns active variants with stock < threshold.
func (r *ProductVariantRepository) ListBelowThreshold(ctx context.Context, threshold int) ([]*entity.ProductVariant, error) {
	const q = selectVariantCols + `FROM product_variants WHERE is_active = true AND stock < $1 ORDER BY id`
	rows, err := r.db.Query(ctx, q, threshold)
	if err != nil {
		return nil, fmt.Errorf("postgres: list variants below threshold: %w", err)
	}
	defer rows.Close()

	var out []*entity.ProductVariant
	for rows.Next() {
		v, err := scanVariant(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan product variant: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// AdjustStock guards stock + delta >= 0 in the WHERE clause; a missing row
// (guard failed or variant not found) surfaces as errs.ErrConflict.
func (r *ProductVariantRepository) AdjustStock(ctx context.Context, variantID uuid.UUID, delta int) (int, error) {
	const q = `
		UPDATE product_variants
		SET stock = stock + $2
		WHERE id = $1 AND stock + $2 >= 0
		RETURNING stock
	`
	var newStock int
	err := r.db.QueryRow(ctx, q, variantID, delta).Scan(&newStock)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errs.ErrConflict
	}
	if err != nil {
		return 0, fmt.Errorf("postgres: adjust stock: %w", err)
	}
	return newStock, nil
}

// AdjustStockWithMovement applies delta and inserts m atomically in one
// transaction. Same stock + delta >= 0 guard as AdjustStock.
func (r *ProductVariantRepository) AdjustStockWithMovement(ctx context.Context, variantID uuid.UUID, delta int, m *entity.StockMovement) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("postgres: begin adjust stock tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const updateQ = `
		UPDATE product_variants
		SET stock = stock + $2
		WHERE id = $1 AND stock + $2 >= 0
		RETURNING stock
	`
	var newStock int
	if err := tx.QueryRow(ctx, updateQ, variantID, delta).Scan(&newStock); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errs.ErrConflict
		}
		return 0, fmt.Errorf("postgres: adjust stock: %w", err)
	}

	m.StockAfter = newStock
	const insertQ = `
		INSERT INTO stock_movements (id, variant_id, type, qty, ref_type, ref_id, stock_after, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`
	if err := tx.QueryRow(ctx, insertQ, m.ID, m.VariantID, m.Type, m.Qty, m.RefType, m.RefID, m.StockAfter, m.Note).
		Scan(&m.CreatedAt); err != nil {
		return 0, fmt.Errorf("postgres: create stock movement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("postgres: commit adjust stock tx: %w", err)
	}
	return newStock, nil
}

const selectVariantCols = `
	SELECT id, product_id, supplier_id, sku, name, unit, price, weight_gram,
		length_cm, width_cm, height_cm, min_order_qty, stock, is_active, created_at
`

func scanVariant(row rowScanner) (*entity.ProductVariant, error) {
	var v entity.ProductVariant
	err := row.Scan(
		&v.ID, &v.ProductID, &v.SupplierID, &v.SKU, &v.Name, &v.Unit, &v.Price, &v.WeightGram,
		&v.LengthCM, &v.WidthCM, &v.HeightCM, &v.MinOrderQty, &v.Stock, &v.IsActive, &v.CreatedAt,
	)
	return &v, err
}

func (r *ProductVariantRepository) scanOne(ctx context.Context, query string, arg any) (*entity.ProductVariant, error) {
	v, err := scanVariant(r.db.QueryRow(ctx, query, arg))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find product variant: %w", err)
	}
	return v, nil
}
