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

func (s *paymentServer) Disburse(ctx context.Context, req *paymentv1.DisburseRequest) (*paymentv1.DisburseResponse, error) {
	id, err := s.xendit.Disburse(ctx, req.WithdrawId, req.Amount, req.BankCode, req.AccountNumber, req.AccountName)
	if err != nil {
		s.log.Error().Err(err).Str("withdraw_id", req.WithdrawId).Msg("disburse failed")
		return nil, fmt.Errorf("disburse: %w", err)
	}
	return &paymentv1.DisburseResponse{DisbursementId: id}, nil
}
