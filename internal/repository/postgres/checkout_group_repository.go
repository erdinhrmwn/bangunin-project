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

type CheckoutGroupRepository struct {
	db *pgxpool.Pool
}

func NewCheckoutGroupRepository(db *pgxpool.Pool) *CheckoutGroupRepository {
	return &CheckoutGroupRepository{db: db}
}

func (r *CheckoutGroupRepository) Create(ctx context.Context, g *entity.CheckoutGroup) error {
	const q = `
		INSERT INTO checkout_groups (id, user_id, group_number, grand_total, status)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := r.db.Exec(ctx, q, g.ID, g.UserID, g.GroupNumber, g.GrandTotal, g.Status); err != nil {
		return fmt.Errorf("postgres: create checkout group: %w", err)
	}
	return nil
}

func (r *CheckoutGroupRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if _, err := r.db.Exec(ctx, `UPDATE checkout_groups SET status = $2 WHERE id = $1`, id, status); err != nil {
		return fmt.Errorf("postgres: update checkout group status: %w", err)
	}
	return nil
}

func (r *CheckoutGroupRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.CheckoutGroup, error) {
	g, err := scanCheckoutGroup(r.db.QueryRow(ctx, selectCheckoutGroupCols+`FROM checkout_groups WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find checkout group: %w", err)
	}
	return g, nil
}

const selectCheckoutGroupCols = `SELECT id, user_id, group_number, grand_total, status, created_at `

func scanCheckoutGroup(row rowScanner) (*entity.CheckoutGroup, error) {
	var g entity.CheckoutGroup
	err := row.Scan(&g.ID, &g.UserID, &g.GroupNumber, &g.GrandTotal, &g.Status, &g.CreatedAt)
	return &g, err
}
