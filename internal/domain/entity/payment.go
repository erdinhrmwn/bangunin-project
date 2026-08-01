package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	PaymentMethodVA      = "va"
	PaymentMethodEWallet = "ewallet"
	PaymentMethodQRIS    = "qris"

	PaymentStatusPending = "pending"
	PaymentStatusPaid    = "paid"
	PaymentStatusExpired = "expired"
	PaymentStatusFailed  = "failed"
)

type Payment struct {
	ID              uuid.UUID
	CheckoutGroupID uuid.UUID
	XenditInvoiceID string
	Method          string
	Amount          float64
	Status          string
	InvoiceURL      string
	PaidAt          *time.Time
	ExpiresAt       *time.Time
	CreatedAt       time.Time
}
