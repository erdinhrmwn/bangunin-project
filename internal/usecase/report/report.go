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
