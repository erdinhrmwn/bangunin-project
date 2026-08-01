package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	CheckoutGroupStatusPendingPayment = "pending_payment"
	CheckoutGroupStatusPaid           = "paid"
	CheckoutGroupStatusPaymentFailed  = "payment_failed"
	CheckoutGroupStatusExpired        = "expired"
	CheckoutGroupStatusCancelled      = "cancelled"
)

type CheckoutGroup struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	GroupNumber string
	GrandTotal  float64
	Status      string
	CreatedAt   time.Time
}
