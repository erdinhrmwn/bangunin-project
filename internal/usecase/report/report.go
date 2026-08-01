// Package report implements the supplier sales summary and CSV export
// (FR-7.3).
package report

import (
	"context"
	"time"

	"github.com/google/uuid"

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
	gmv, count, err := u.orders.SalesSummary(ctx, supplierID, from, to)
	if err != nil {
		return nil, err
	}
	top, err := u.orders.TopProducts(ctx, supplierID, from, to, topProductsLimit)
	if err != nil {
		return nil, err
	}
	perDay, err := u.orders.SalesPerDay(ctx, supplierID, from, to)
	if err != nil {
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
	gmv, byStatus, err := u.orders.PlatformSummary(ctx, from, to)
	if err != nil {
		return nil, err
	}
	commission, err := u.ledger.SumByType(ctx, entity.LedgerTypeDebitCommission, from, to)
	if err != nil {
		return nil, err
	}
	_, activeSuppliers, err := u.suppliers.List(ctx, "approved", "", 1, 1)
	if err != nil {
		return nil, err
	}
	newUsers, err := u.users.CountCreatedBetween(ctx, from, to)
	if err != nil {
		return nil, err
	}
	perDay, err := u.orders.PlatformSalesPerDay(ctx, from, to)
	if err != nil {
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
