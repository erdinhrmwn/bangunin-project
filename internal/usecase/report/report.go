// Package report implements the supplier sales summary and CSV export
// (FR-7.3).
package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/domain/service"
)

// topProductsLimit: ponytail: fixed at 5 per FR-7.3 spec, promote to a
// query param if suppliers ever need more.
const topProductsLimit = 5

type Summary struct {
	GMV         float64
	OrdersCount int
	AOV         float64
	TopProducts []*entity.TopProduct
	SalesPerDay []*entity.DailySales
}

type Usecase struct {
	orders repository.OrderRepository
	report service.ReportEnqueuer
}

func New(orders repository.OrderRepository, report service.ReportEnqueuer) *Usecase {
	return &Usecase{orders: orders, report: report}
}

// GetSummary aggregates GMV/orders/AOV/top-products/sales-per-day for a
// supplier's completed orders within [from, to] (FR-7.3).
func (u *Usecase) GetSummary(ctx context.Context, supplierID uuid.UUID, from, to time.Time) (*Summary, error) {
	var gmv float64
	var count int
	var top []*entity.TopProduct
	var perDay []*entity.DailySales

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		gmv, count, err = u.orders.SalesSummary(gctx, supplierID, from, to)
		return err
	})
	g.Go(func() (err error) {
		top, err = u.orders.TopProducts(gctx, supplierID, from, to, topProductsLimit)
		return err
	})
	g.Go(func() (err error) {
		perDay, err = u.orders.SalesPerDay(gctx, supplierID, from, to)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	var aov float64
	if count > 0 {
		aov = gmv / float64(count)
	}
	return &Summary{GMV: gmv, OrdersCount: count, AOV: aov, TopProducts: top, SalesPerDay: perDay}, nil
}

// Export enqueues a report:generate job that builds a CSV, uploads it, and
// notifies the supplier with a presigned download link (FR-7.3).
func (u *Usecase) Export(ctx context.Context, supplierID uuid.UUID, from, to time.Time) error {
	return u.report.EnqueueReportGenerate(supplierID, from, to)
}

// BuildCSV renders a supplier Summary as CSV, for the report:generate worker task.
func (s *Summary) BuildCSV() ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"GMV", "Orders", "AOV"})
	_ = w.Write([]string{fmt.Sprintf("%.2f", s.GMV), fmt.Sprintf("%d", s.OrdersCount), fmt.Sprintf("%.2f", s.AOV)})
	_ = w.Write([]string{})
	_ = w.Write([]string{"Product", "Qty", "Revenue"})
	for _, tp := range s.TopProducts {
		_ = w.Write([]string{tp.ProductName, fmt.Sprintf("%d", tp.Qty), fmt.Sprintf("%.2f", tp.Revenue)})
	}
	_ = w.Write([]string{})
	_ = w.Write([]string{"Day", "GMV", "Orders"})
	for _, d := range s.SalesPerDay {
		_ = w.Write([]string{d.Day.Format("2006-01-02"), fmt.Sprintf("%.2f", d.GMV), fmt.Sprintf("%d", d.Orders)})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("report: write csv: %w", err)
	}
	return buf.Bytes(), nil
}

// AdminSummary is the platform-wide summary for FR-7.4.
type AdminSummary struct {
	GMV             float64
	Commission      float64
	OrdersByStatus  map[string]int
	ActiveSuppliers int
	NewUsers        int
	SalesPerDay     []*entity.DailySales
}

// AdminUsecase aggregates platform-wide metrics for admins (FR-7.4).
type AdminUsecase struct {
	orders    repository.OrderRepository
	ledger    repository.LedgerEntryRepository
	suppliers repository.SupplierRepository
	users     repository.UserRepository
	report    service.ReportEnqueuer
}

func NewAdmin(orders repository.OrderRepository, ledger repository.LedgerEntryRepository, suppliers repository.SupplierRepository, users repository.UserRepository, report service.ReportEnqueuer) *AdminUsecase {
	return &AdminUsecase{orders: orders, ledger: ledger, suppliers: suppliers, users: users, report: report}
}

// GetSummary aggregates GMV/commission/orders-by-status/active-suppliers/
// new-users/sales-per-day platform-wide within [from, to] (FR-7.4).
func (u *AdminUsecase) GetSummary(ctx context.Context, from, to time.Time) (*AdminSummary, error) {
	var gmv, commission float64
	var byStatus map[string]int
	var activeSuppliers, newUsers int
	var perDay []*entity.DailySales

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		gmv, byStatus, err = u.orders.PlatformSummary(gctx, from, to)
		return err
	})
	g.Go(func() (err error) {
		commission, err = u.ledger.SumByType(gctx, entity.LedgerTypeDebitCommission, from, to)
		return err
	})
	g.Go(func() (err error) {
		_, activeSuppliers, err = u.suppliers.List(gctx, "approved", "", 1, 1)
		return err
	})
	g.Go(func() (err error) {
		newUsers, err = u.users.CountCreatedBetween(gctx, from, to)
		return err
	})
	g.Go(func() (err error) {
		perDay, err = u.orders.PlatformSalesPerDay(gctx, from, to)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return &AdminSummary{
		GMV: gmv, Commission: commission, OrdersByStatus: byStatus,
		ActiveSuppliers: activeSuppliers, NewUsers: newUsers, SalesPerDay: perDay,
	}, nil
}

// Export enqueues a report:generate-admin job that builds a platform CSV
// and notifies the requesting admin with a presigned download link (FR-7.4).
func (u *AdminUsecase) Export(ctx context.Context, adminID uuid.UUID, from, to time.Time) error {
	return u.report.EnqueueAdminReportGenerate(adminID, from, to)
}

// BuildCSV renders an AdminSummary as CSV, for the report:generate-admin worker task.
func (s *AdminSummary) BuildCSV() ([]byte, error) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"GMV", "Commission", "ActiveSuppliers", "NewUsers"})
	_ = w.Write([]string{
		fmt.Sprintf("%.2f", s.GMV), fmt.Sprintf("%.2f", s.Commission),
		fmt.Sprintf("%d", s.ActiveSuppliers), fmt.Sprintf("%d", s.NewUsers),
	})
	_ = w.Write([]string{})
	_ = w.Write([]string{"Status", "Count"})
	for status, count := range s.OrdersByStatus {
		_ = w.Write([]string{status, fmt.Sprintf("%d", count)})
	}
	_ = w.Write([]string{})
	_ = w.Write([]string{"Day", "GMV", "Orders"})
	for _, d := range s.SalesPerDay {
		_ = w.Write([]string{d.Day.Format("2006-01-02"), fmt.Sprintf("%.2f", d.GMV), fmt.Sprintf("%d", d.Orders)})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("report: write admin csv: %w", err)
	}
	return buf.Bytes(), nil
}
