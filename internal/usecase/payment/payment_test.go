package payment_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	servicemocks "erdinhrmwn/bangunin/internal/domain/service/mocks"
	paymentusecase "erdinhrmwn/bangunin/internal/usecase/payment"
)

func TestCreateInvoice_DelegatesToGateway(t *testing.T) {
	gw := servicemocks.NewMockPaymentGateway(t)
	groupID := uuid.Must(uuid.NewV7())
	gw.EXPECT().CreateInvoice(context.Background(), groupID, 100000.0, "desc").Return("inv-1", "https://pay.example/inv-1", nil)

	uc := paymentusecase.New(gw)
	invoiceID, invoiceURL, err := uc.CreateInvoice(context.Background(), groupID, 100000.0, "desc")

	require.NoError(t, err)
	require.Equal(t, "inv-1", invoiceID)
	require.Equal(t, "https://pay.example/inv-1", invoiceURL)
}
