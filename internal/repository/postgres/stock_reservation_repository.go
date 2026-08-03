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
	"erdinhrmwn/bangunin/internal/domain/repository"
)

type StockReservationRepository struct {
	db *pgxpool.Pool
}

func NewStockReservationRepository(db *pgxpool.Pool) *StockReservationRepository {
	return &StockReservationRepository{db: db}
}

func (r *StockReservationRepository) Create(ctx context.Context, res *entity.StockReservation) error {
	const q = `
		INSERT INTO stock_reservations (id, variant_id, order_id, qty, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, q, res.ID, res.VariantID, res.OrderID, res.Qty, res.Status, res.ExpiresAt)
	if err != nil {
		return fmt.Errorf("postgres: create stock reservation: %w", err)
	}
	return nil
}

func (r *StockReservationRepository) ListByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entity.StockReservation, error) {
	rows, err := r.db.Query(ctx, selectStockReservationCols+`FROM stock_reservations WHERE order_id = $1`, orderID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list stock reservations by order: %w", err)
	}
	defer rows.Close()
	return collectStockReservations(rows)
}

func (r *StockReservationRepository) ListExpired(ctx context.Context) ([]*entity.StockReservation, error) {
	rows, err := r.db.Query(ctx, selectStockReservationCols+`FROM stock_reservations WHERE status = 'active' AND expires_at < now()`)
	if err != nil {
		return nil, fmt.Errorf("postgres: list expired stock reservations: %w", err)
	}
	defer rows.Close()
	return collectStockReservations(rows)
}

func (r *StockReservationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if _, err := r.db.Exec(ctx, `UPDATE stock_reservations SET status = $2 WHERE id = $1`, id, status); err != nil {
		return fmt.Errorf("postgres: update stock reservation status: %w", err)
	}
	return nil
}

// ApplyAdjustments applies every adjustment's variant stock delta (guarded
// stock + delta >= 0, same as ProductVariantRepository.AdjustStock), movement
// insert (skipped if Movement is nil), and reservation status update in one
// TX, mirroring CheckoutGroupRepository.Confirm's per-item-loop-in-one-TX
// pattern.
func (r *StockReservationRepository) ApplyAdjustments(ctx context.Context, adjustments []repository.ReservationAdjustment) error {
	if len(adjustments) == 0 {
		return nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin apply reservation adjustments tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const stockQ = `
		UPDATE product_variants
		SET stock = stock + $2
		WHERE id = $1 AND stock + $2 >= 0
		RETURNING stock
	`
	const movementQ = `
		INSERT INTO stock_movements (id, variant_id, type, qty, ref_type, ref_id, stock_after, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`
	const statusQ = `UPDATE stock_reservations SET status = $2 WHERE id = $1`

	for _, adj := range adjustments {
		if adj.Movement != nil {
			var newStock int
			if err := tx.QueryRow(ctx, stockQ, adj.VariantID, adj.Delta).Scan(&newStock); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return errs.ErrConflict
				}
				return fmt.Errorf("postgres: adjust stock: %w", err)
			}
			adj.Movement.StockAfter = newStock
			if err := tx.QueryRow(ctx, movementQ,
				adj.Movement.ID, adj.Movement.VariantID, adj.Movement.Type, adj.Movement.Qty,
				adj.Movement.RefType, adj.Movement.RefID, adj.Movement.StockAfter, adj.Movement.Note,
			).Scan(&adj.Movement.CreatedAt); err != nil {
				return fmt.Errorf("postgres: create stock movement: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, statusQ, adj.ReservationID, adj.Status); err != nil {
			return fmt.Errorf("postgres: update stock reservation status: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit apply reservation adjustments tx: %w", err)
	}
	return nil
}

const selectStockReservationCols = `SELECT id, variant_id, order_id, qty, status, expires_at, created_at `

func scanStockReservation(row rowScanner) (*entity.StockReservation, error) {
	var res entity.StockReservation
	err := row.Scan(&res.ID, &res.VariantID, &res.OrderID, &res.Qty, &res.Status, &res.ExpiresAt, &res.CreatedAt)
	return &res, err
}

func collectStockReservations(rows pgx.Rows) ([]*entity.StockReservation, error) {
	var out []*entity.StockReservation
	for rows.Next() {
		res, err := scanStockReservation(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan stock reservation: %w", err)
		}
		out = append(out, res)
	}
	return out, rows.Err()
}
