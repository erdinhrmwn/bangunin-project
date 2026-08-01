package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type PaymentRepository interface {
	Create(ctx context.Context, p *entity.Payment) error
	FindByCheckoutGroupID(ctx context.Context, checkoutGroupID uuid.UUID) (*entity.Payment, error)
	FindByXenditInvoiceID(ctx context.Context, xenditInvoiceID string) (*entity.Payment, error)
	// MarkPaid is a no-op (returns nil, no rows touched signalled via
	// unchanged PaidAt on re-read) if the payment is already paid, making
	// duplicate webhook delivery idempotent (AC-5.c).
	MarkPaid(ctx context.Context, id uuid.UUID, paidAt time.Time) error
}
