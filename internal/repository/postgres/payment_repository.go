package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
)

type PaymentRepository struct {
	db *pgxpool.Pool
}

func NewPaymentRepository(db *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{db: db}
}

func (r *PaymentRepository) Create(ctx context.Context, p *entity.Payment) error {
	const q = `
		INSERT INTO payments (id, checkout_group_id, xendit_invoice_id, method, amount, status, invoice_url, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, q,
		p.ID, p.CheckoutGroupID, p.XenditInvoiceID, p.Method, p.Amount, p.Status, p.InvoiceURL, p.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("postgres: create payment: %w", err)
	}
	return nil
}

func (r *PaymentRepository) FindByCheckoutGroupID(ctx context.Context, checkoutGroupID uuid.UUID) (*entity.Payment, error) {
	p, err := scanPayment(r.db.QueryRow(ctx, selectPaymentCols+`FROM payments WHERE checkout_group_id = $1`, checkoutGroupID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find payment by checkout group: %w", err)
	}
	return p, nil
}

func (r *PaymentRepository) FindByXenditInvoiceID(ctx context.Context, xenditInvoiceID string) (*entity.Payment, error) {
	p, err := scanPayment(r.db.QueryRow(ctx, selectPaymentCols+`FROM payments WHERE xendit_invoice_id = $1`, xenditInvoiceID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find payment by invoice id: %w", err)
	}
	return p, nil
}

func (r *PaymentRepository) MarkPaid(ctx context.Context, id uuid.UUID, paidAt time.Time) error {
	const q = `UPDATE payments SET status = 'paid', paid_at = $2 WHERE id = $1`
	if _, err := r.db.Exec(ctx, q, id, paidAt); err != nil {
		return fmt.Errorf("postgres: mark payment paid: %w", err)
	}
	return nil
}

const selectPaymentCols = `
	SELECT id, checkout_group_id, xendit_invoice_id, method, amount, status, invoice_url, paid_at, expires_at, created_at
`

func scanPayment(row rowScanner) (*entity.Payment, error) {
	var p entity.Payment
	err := row.Scan(
		&p.ID, &p.CheckoutGroupID, &p.XenditInvoiceID, &p.Method, &p.Amount, &p.Status,
		&p.InvoiceURL, &p.PaidAt, &p.ExpiresAt, &p.CreatedAt,
	)
	return &p, err
}
