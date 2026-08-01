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

type SupplierBalanceRepository struct {
	db *pgxpool.Pool
}

func NewSupplierBalanceRepository(db *pgxpool.Pool) *SupplierBalanceRepository {
	return &SupplierBalanceRepository{db: db}
}

func (r *SupplierBalanceRepository) FindBySupplierID(ctx context.Context, supplierID uuid.UUID) (*entity.SupplierBalance, error) {
	const q = `SELECT supplier_id, balance, updated_at FROM supplier_balances WHERE supplier_id = $1`
	var b entity.SupplierBalance
	err := r.db.QueryRow(ctx, q, supplierID).Scan(&b.SupplierID, &b.Balance, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find supplier balance: %w", err)
	}
	return &b, nil
}

func (r *SupplierBalanceRepository) Increment(ctx context.Context, supplierID uuid.UUID, delta float64) (float64, error) {
	const q = `
		INSERT INTO supplier_balances (supplier_id, balance, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (supplier_id) DO UPDATE SET balance = supplier_balances.balance + $2, updated_at = now()
		RETURNING balance
	`
	var newBalance float64
	if err := r.db.QueryRow(ctx, q, supplierID, delta).Scan(&newBalance); err != nil {
		return 0, fmt.Errorf("postgres: increment supplier balance: %w", err)
	}
	return newBalance, nil
}
