package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	paymentv1 "erdinhrmwn/bangunin/proto/payment/v1"
)

type paymentServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	xendit *xenditClient
	log    zerolog.Logger
}

func (s *paymentServer) CreateInvoice(ctx context.Context, req *paymentv1.CreateInvoiceRequest) (*paymentv1.CreateInvoiceResponse, error) {
	id, url, err := s.xendit.CreateInvoice(ctx, req.CheckoutGroupId, req.Amount, req.Description)
	if err != nil {
		s.log.Error().Err(err).Str("checkout_group_id", req.CheckoutGroupId).Msg("create invoice failed")
		return nil, fmt.Errorf("create invoice: %w", err)
	}
	return &paymentv1.CreateInvoiceResponse{InvoiceId: id, InvoiceUrl: url}, nil
}
