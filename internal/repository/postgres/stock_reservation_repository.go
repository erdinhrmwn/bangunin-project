package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
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
