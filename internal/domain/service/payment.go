package service

import (
	"context"

	"github.com/google/uuid"
)

// PaymentGateway creates a payment invoice via payment-service (gRPC,
// CLAUDE.md §2). Implemented by infra/grpcclient.
type PaymentGateway interface {
	CreateInvoice(ctx context.Context, checkoutGroupID uuid.UUID, amount float64, description string) (invoiceID, invoiceURL string, err error)
}
