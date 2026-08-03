package report_test

import (
	"bytes"
	"context"
	"encoding/csv"
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

func TestAdminGetSummary_Aggregates(t *testing.T) {
	orders := mocks.NewMockOrderRepository(t)
	ledger := mocks.NewMockLedgerEntryRepository(t)
	suppliers := mocks.NewMockSupplierRepository(t)
	users := mocks.NewMockUserRepository(t)
	reportSvc := servicemocks.NewMockReportEnqueuer(t)
	uc := reportusecase.NewAdmin(orders, ledger, suppliers, users, reportSvc)

	from, to := time.Now().AddDate(0, 0, -30), time.Now()
	byStatus := map[string]int{"completed": 4, "pending_payment": 1}

	orders.EXPECT().PlatformSummary(mock.Anything, from, to).Return(1000.0, byStatus, nil)
	ledger.EXPECT().SumByType(mock.Anything, entity.LedgerTypeDebitCommission, from, to).Return(40.0, nil)
	suppliers.EXPECT().List(mock.Anything, "approved", "", 1, 1).Return(nil, 12, nil)
	users.EXPECT().CountCreatedBetween(mock.Anything, from, to).Return(7, nil)
	orders.EXPECT().PlatformSalesPerDay(mock.Anything, from, to).Return([]*entity.DailySales{
		{Day: to, GMV: 1000, Orders: 4},
	}, nil)

	s, err := uc.GetSummary(context.Background(), from, to)

	require.NoError(t, err)
	require.Equal(t, 1000.0, s.GMV)
	require.Equal(t, 40.0, s.Commission)
	require.Equal(t, byStatus, s.OrdersByStatus)
	require.Equal(t, 12, s.ActiveSuppliers)
	require.Equal(t, 7, s.NewUsers)
	require.Len(t, s.SalesPerDay, 1)
}

func TestAdminExport_EnqueuesAdminReportGenerate(t *testing.T) {
	orders := mocks.NewMockOrderRepository(t)
	ledger := mocks.NewMockLedgerEntryRepository(t)
	suppliers := mocks.NewMockSupplierRepository(t)
	users := mocks.NewMockUserRepository(t)
	reportSvc := servicemocks.NewMockReportEnqueuer(t)
	uc := reportusecase.NewAdmin(orders, ledger, suppliers, users, reportSvc)

	adminID := uuid.Must(uuid.NewV7())
	from, to := time.Now().AddDate(0, 0, -30), time.Now()

	reportSvc.EXPECT().EnqueueAdminReportGenerate(adminID, from, to).Return(nil)

	err := uc.Export(context.Background(), adminID, from, to)

	require.NoError(t, err)
}

func TestSummary_BuildCSV(t *testing.T) {
	s := &reportusecase.Summary{
		GMV: 1000.5, OrdersCount: 4, AOV: 250.125,
		TopProducts: []*entity.TopProduct{{ProductName: "Semen", Qty: 10, Revenue: 500}},
		SalesPerDay: []*entity.DailySales{{Day: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), GMV: 1000, Orders: 4}},
	}

	data, err := s.BuildCSV()
	require.NoError(t, err)

	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	require.NoError(t, err)
	require.Equal(t, [][]string{
		{"GMV", "Orders", "AOV"},
		{"1000.50", "4", "250.12"},
		{"Product", "Qty", "Revenue"},
		{"Semen", "10", "500.00"},
		{"Day", "GMV", "Orders"},
		{"2026-01-15", "1000.00", "4"},
	}, rows)
}

func TestAdminSummary_BuildCSV(t *testing.T) {
	s := &reportusecase.AdminSummary{
		GMV: 5000, Commission: 200, OrdersByStatus: map[string]int{"completed": 3},
		ActiveSuppliers: 12, NewUsers: 7,
		SalesPerDay: []*entity.DailySales{{Day: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), GMV: 5000, Orders: 3}},
	}

	data, err := s.BuildCSV()
	require.NoError(t, err)

	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	rows, err := r.ReadAll()
	require.NoError(t, err)
	require.Equal(t, [][]string{
		{"GMV", "Commission", "ActiveSuppliers", "NewUsers"},
		{"5000.00", "200.00", "12", "7"},
		{"Status", "Count"},
		{"completed", "3"},
		{"Day", "GMV", "Orders"},
		{"2026-01-15", "5000.00", "3"},
	}, rows)
}
