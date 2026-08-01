package report_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	servicemocks "erdinhrmwn/bangunin/internal/domain/service/mocks"
	reportusecase "erdinhrmwn/bangunin/internal/usecase/report"
)

func TestGetSummary_ComputesAOV(t *testing.T) {
	orders := mocks.NewMockOrderRepository(t)
	reportSvc := servicemocks.NewMockReportEnqueuer(t)
	uc := reportusecase.New(orders, reportSvc)

	supplierID := uuid.Must(uuid.NewV7())
	from, to := time.Now().AddDate(0, 0, -30), time.Now()

	orders.EXPECT().SalesSummary(mock.Anything, supplierID, from, to).Return(1000.0, 4, nil)
	orders.EXPECT().TopProducts(mock.Anything, supplierID, from, to, 5).Return([]*entity.TopProduct{
		{ProductName: "Semen", Qty: 10, Revenue: 500},
	}, nil)
	orders.EXPECT().SalesPerDay(mock.Anything, supplierID, from, to).Return([]*entity.DailySales{
		{Day: to, GMV: 1000, Orders: 4},
	}, nil)

	s, err := uc.GetSummary(context.Background(), supplierID, from, to)

	require.NoError(t, err)
	require.Equal(t, 1000.0, s.GMV)
	require.Equal(t, 4, s.OrdersCount)
	require.Equal(t, 250.0, s.AOV)
	require.Len(t, s.TopProducts, 1)
	require.Len(t, s.SalesPerDay, 1)
}

func TestGetSummary_ZeroOrders_AOVIsZero(t *testing.T) {
	orders := mocks.NewMockOrderRepository(t)
	reportSvc := servicemocks.NewMockReportEnqueuer(t)
	uc := reportusecase.New(orders, reportSvc)

	supplierID := uuid.Must(uuid.NewV7())
	from, to := time.Now().AddDate(0, 0, -30), time.Now()

	orders.EXPECT().SalesSummary(mock.Anything, supplierID, from, to).Return(0.0, 0, nil)
	orders.EXPECT().TopProducts(mock.Anything, supplierID, from, to, 5).Return(nil, nil)
	orders.EXPECT().SalesPerDay(mock.Anything, supplierID, from, to).Return(nil, nil)

	s, err := uc.GetSummary(context.Background(), supplierID, from, to)

	require.NoError(t, err)
	require.Equal(t, 0.0, s.AOV)
}

func TestExport_EnqueuesReportGenerate(t *testing.T) {
	orders := mocks.NewMockOrderRepository(t)
	reportSvc := servicemocks.NewMockReportEnqueuer(t)
	uc := reportusecase.New(orders, reportSvc)

	supplierID := uuid.Must(uuid.NewV7())
	from, to := time.Now().AddDate(0, 0, -30), time.Now()

	reportSvc.EXPECT().EnqueueReportGenerate(supplierID, from, to).Return(nil)

	err := uc.Export(context.Background(), supplierID, from, to)

	require.NoError(t, err)
}
