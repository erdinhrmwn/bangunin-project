package service

import (
	"context"

	"github.com/google/uuid"
)

// PaymentGateway creates a payment invoice / disbursement via payment-service
// (gRPC, CLAUDE.md §2). Implemented by infra/grpcclient.
type PaymentGateway interface {
	CreateInvoice(ctx context.Context, checkoutGroupID uuid.UUID, amount float64, description string) (invoiceID, invoiceURL string, err error)
	// Disburse pays out a withdraw request to a supplier's bank account.
	Disburse(ctx context.Context, withdrawID uuid.UUID, amount float64, bankCode, accountNumber, accountName string) (disbursementID string, err error)
}
