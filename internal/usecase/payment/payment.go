// Package payment wraps domain/service.PaymentGateway for callers outside
// checkout (which calls the gateway directly at confirm time, FR-5.4 step 4).
package payment

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/service"
)

type Usecase struct {
	gateway service.PaymentGateway
}

func New(gateway service.PaymentGateway) *Usecase {
	return &Usecase{gateway: gateway}
}

func (u *Usecase) CreateInvoice(ctx context.Context, checkoutGroupID uuid.UUID, amount float64, description string) (invoiceID, invoiceURL string, err error) {
	return u.gateway.CreateInvoice(ctx, checkoutGroupID, amount, description)
}
