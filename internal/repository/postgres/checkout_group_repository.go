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

// Confirm inserts the checkout group and every order/items/reservations/
// history bundle in one TX. For each variant referenced by a reservation,
// re-checks stock - sum(active reservation qty) >= requested qty (available
// stock accounting for other in-flight reservations, since stock itself is
// only decremented at paid-conversion, never at reservation time). Returns
// errs.ErrConflict if any variant is short.
func (r *CheckoutGroupRepository) Confirm(ctx context.Context, g *entity.CheckoutGroup, orders []repository.ConfirmOrder) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin confirm checkout tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO checkout_groups (id, user_id, group_number, grand_total, status)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := tx.Exec(ctx, q, g.ID, g.UserID, g.GroupNumber, g.GrandTotal, g.Status); err != nil {
		return fmt.Errorf("postgres: create checkout group: %w", err)
	}

	const orderQ = `
		INSERT INTO orders (
			id, checkout_group_id, user_id, supplier_id, order_number, status,
			subtotal, shipping_cost, commission_amount, total, shipping_address
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	const itemQ = `
		INSERT INTO order_items (id, order_id, variant_id, product_name, variant_name, unit, price, qty, weight_gram)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	const availQ = `
		SELECT stock - COALESCE((SELECT sum(qty) FROM stock_reservations WHERE variant_id = $1 AND status = 'active'), 0)
		FROM product_variants WHERE id = $1 FOR UPDATE
	`
	const reservationQ = `
		INSERT INTO stock_reservations (id, variant_id, order_id, qty, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	const historyQ = `
		INSERT INTO order_status_histories (id, order_id, from_status, to_status, actor_type, actor_id, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	for _, o := range orders {
		if _, err := tx.Exec(ctx, orderQ,
			o.Order.ID, o.Order.CheckoutGroupID, o.Order.UserID, o.Order.SupplierID, o.Order.OrderNumber, o.Order.Status,
			o.Order.Subtotal, o.Order.ShippingCost, o.Order.CommissionAmount, o.Order.Total, o.Order.ShippingAddress,
		); err != nil {
			return fmt.Errorf("postgres: create order: %w", err)
		}

		for _, it := range o.Items {
			if _, err := tx.Exec(ctx, itemQ, it.ID, it.OrderID, it.VariantID, it.ProductName, it.VariantName, it.Unit, it.Price, it.Qty, it.WeightGram); err != nil {
				return fmt.Errorf("postgres: create order item: %w", err)
			}
		}

		for _, res := range o.Reservations {
			var available int
			if err := tx.QueryRow(ctx, availQ, res.VariantID).Scan(&available); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return errs.ErrNotFound
				}
				return fmt.Errorf("postgres: check variant availability: %w", err)
			}
			if available < res.Qty {
				return errs.ErrConflict
			}
			if _, err := tx.Exec(ctx, reservationQ, res.ID, res.VariantID, res.OrderID, res.Qty, res.Status, res.ExpiresAt); err != nil {
				return fmt.Errorf("postgres: create stock reservation: %w", err)
			}
		}

		if _, err := tx.Exec(ctx, historyQ, o.History.ID, o.History.OrderID, o.History.FromStatus, o.History.ToStatus, o.History.ActorType, o.History.ActorID, o.History.Note); err != nil {
			return fmt.Errorf("postgres: create order status history: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("postgres: commit confirm checkout tx: %w", err)
	}
	return nil
}

const selectCheckoutGroupCols = `SELECT id, user_id, group_number, grand_total, status, created_at `

func scanCheckoutGroup(row rowScanner) (*entity.CheckoutGroup, error) {
	var g entity.CheckoutGroup
	err := row.Scan(&g.ID, &g.UserID, &g.GroupNumber, &g.GrandTotal, &g.Status, &g.CreatedAt)
	return &g, err
}
