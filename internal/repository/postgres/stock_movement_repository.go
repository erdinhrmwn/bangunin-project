package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type StockMovementRepository struct {
	db *pgxpool.Pool
}

func NewStockMovementRepository(db *pgxpool.Pool) *StockMovementRepository {
	return &StockMovementRepository{db: db}
}

func (r *StockMovementRepository) Create(ctx context.Context, m *entity.StockMovement) error {
	const q = `
		INSERT INTO stock_movements (id, variant_id, type, qty, ref_type, ref_id, stock_after, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`
	err := r.db.QueryRow(ctx, q, m.ID, m.VariantID, m.Type, m.Qty, m.RefType, m.RefID, m.StockAfter, m.Note).
		Scan(&m.CreatedAt)
	if err != nil {
		return fmt.Errorf("postgres: create stock movement: %w", err)
	}
	return nil
}

func (r *StockMovementRepository) ListByVariantID(ctx context.Context, variantID uuid.UUID, page, perPage int) ([]*entity.StockMovement, int, error) {
	var total int
	const countQ = `SELECT count(*) FROM stock_movements WHERE variant_id = $1`
	if err := r.db.QueryRow(ctx, countQ, variantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count stock movements: %w", err)
	}

	const q = `
		SELECT id, variant_id, type, qty, ref_type, ref_id, stock_after, note, created_at
		FROM stock_movements WHERE variant_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, variantID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list stock movements: %w", err)
	}
	defer rows.Close()

	var out []*entity.StockMovement
	for rows.Next() {
		var m entity.StockMovement
		if err := rows.Scan(&m.ID, &m.VariantID, &m.Type, &m.Qty, &m.RefType, &m.RefID, &m.StockAfter, &m.Note, &m.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("postgres: scan stock movement: %w", err)
		}
		out = append(out, &m)
	}
	return out, total, rows.Err()
}
