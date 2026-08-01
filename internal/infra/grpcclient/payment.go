package grpcclient

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	paymentv1 "erdinhrmwn/bangunin/proto/payment/v1"
)

// PaymentClient implements domain/service.PaymentGateway against the
// standalone services/payment gRPC server (FR-5.5).
type PaymentClient struct {
	client paymentv1.PaymentServiceClient
}

func NewPaymentClient(addr string) (*PaymentClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpcclient: dial payment-service: %w", err)
	}
	return &PaymentClient{client: paymentv1.NewPaymentServiceClient(conn)}, nil
}

func (c *PaymentClient) CreateInvoice(ctx context.Context, checkoutGroupID uuid.UUID, amount float64, description string) (invoiceID, invoiceURL string, err error) {
	resp, err := c.client.CreateInvoice(ctx, &paymentv1.CreateInvoiceRequest{
		CheckoutGroupId: checkoutGroupID.String(),
		Amount:          amount,
		Description:     description,
	})
	if err != nil {
		return "", "", fmt.Errorf("grpcclient: create invoice: %w", err)
	}
	return resp.InvoiceId, resp.InvoiceUrl, nil
}

func (c *PaymentClient) Disburse(ctx context.Context, withdrawID uuid.UUID, amount float64, bankCode, accountNumber, accountName string) (disbursementID string, err error) {
	resp, err := c.client.Disburse(ctx, &paymentv1.DisburseRequest{
		WithdrawId:    withdrawID.String(),
		Amount:        amount,
		BankCode:      bankCode,
		AccountNumber: accountNumber,
		AccountName:   accountName,
	})
	if err != nil {
		return "", fmt.Errorf("grpcclient: disburse: %w", err)
	}
	return resp.DisbursementId, nil
}
