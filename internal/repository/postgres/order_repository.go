package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
)

type OrderRepository struct {
	db *pgxpool.Pool
}

func NewOrderRepository(db *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, o *entity.Order) error {
	const q = `
		INSERT INTO orders (
			id, checkout_group_id, user_id, supplier_id, order_number, status,
			subtotal, shipping_cost, commission_amount, total, shipping_address
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Exec(ctx, q,
		o.ID, o.CheckoutGroupID, o.UserID, o.SupplierID, o.OrderNumber, o.Status,
		o.Subtotal, o.ShippingCost, o.CommissionAmount, o.Total, o.ShippingAddress,
	)
	if err != nil {
		return fmt.Errorf("postgres: create order: %w", err)
	}
	return nil
}

// UpdateStatus guards the update on the expected prior status; a mismatch
// (concurrent transition already applied) surfaces as errs.ErrConflict.
func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, fromStatus, toStatus string) error {
	tag, err := r.db.Exec(ctx, `UPDATE orders SET status = $2 WHERE id = $1 AND status = $3`, id, toStatus, fromStatus)
	if err != nil {
		return fmt.Errorf("postgres: update order status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errs.ErrConflict
	}
	return nil
}

func (r *OrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	o, err := scanOrder(r.db.QueryRow(ctx, selectOrderCols+`FROM orders WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find order: %w", err)
	}
	return o, nil
}

func (r *OrderRepository) ListByCheckoutGroupID(ctx context.Context, checkoutGroupID uuid.UUID) ([]*entity.Order, error) {
	rows, err := r.db.Query(ctx, selectOrderCols+`FROM orders WHERE checkout_group_id = $1 ORDER BY created_at`, checkoutGroupID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list orders by checkout group: %w", err)
	}
	defer rows.Close()
	return collectOrders(rows)
}

func (r *OrderRepository) ListByUserID(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Order, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM orders WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count orders by user: %w", err)
	}

	rows, err := r.db.Query(ctx,
		selectOrderCols+`FROM orders WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, perPage, (page-1)*perPage,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list orders by user: %w", err)
	}
	defer rows.Close()

	out, err := collectOrders(rows)
	return out, total, err
}

func (r *OrderRepository) ListAll(ctx context.Context, page, perPage int) ([]*entity.Order, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count orders: %w", err)
	}
	rows, err := r.db.Query(ctx,
		selectOrderCols+`FROM orders ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		perPage, (page-1)*perPage,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list orders: %w", err)
	}
	defer rows.Close()
	out, err := collectOrders(rows)
	return out, total, err
}

func (r *OrderRepository) ListBySupplierID(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.Order, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM orders WHERE supplier_id = $1`, supplierID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count orders by supplier: %w", err)
	}

	rows, err := r.db.Query(ctx,
		selectOrderCols+`FROM orders WHERE supplier_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		supplierID, perPage, (page-1)*perPage,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list orders by supplier: %w", err)
	}
	defer rows.Close()

	out, err := collectOrders(rows)
	return out, total, err
}

func (r *OrderRepository) ListDeliveredBefore(ctx context.Context, cutoff time.Time) ([]*entity.Order, error) {
	rows, err := r.db.Query(ctx,
		selectOrderCols+`FROM orders WHERE status = 'delivered' AND id IN (
			SELECT order_id FROM order_status_histories WHERE to_status = 'delivered' AND created_at < $1
		)`,
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres: list delivered before cutoff: %w", err)
	}
	defer rows.Close()
	return collectOrders(rows)
}

func (r *OrderRepository) CreateItems(ctx context.Context, items []*entity.OrderItem) error {
	for _, i := range items {
		const q = `
			INSERT INTO order_items (id, order_id, variant_id, product_name, variant_name, unit, price, qty, weight_gram)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`
		_, err := r.db.Exec(ctx, q, i.ID, i.OrderID, i.VariantID, i.ProductName, i.VariantName, i.Unit, i.Price, i.Qty, i.WeightGram)
		if err != nil {
			return fmt.Errorf("postgres: create order item: %w", err)
		}
	}
	return nil
}

func (r *OrderRepository) ListItemsByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entity.OrderItem, error) {
	rows, err := r.db.Query(ctx, selectOrderItemCols+`FROM order_items WHERE order_id = $1`, orderID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list order items: %w", err)
	}
	defer rows.Close()

	var out []*entity.OrderItem
	for rows.Next() {
		i, err := scanOrderItem(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan order item: %w", err)
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

const selectOrderCols = `
	SELECT id, checkout_group_id, user_id, supplier_id, order_number, status,
		subtotal, shipping_cost, commission_amount, total, shipping_address, created_at
`

func scanOrder(row rowScanner) (*entity.Order, error) {
	var o entity.Order
	err := row.Scan(
		&o.ID, &o.CheckoutGroupID, &o.UserID, &o.SupplierID, &o.OrderNumber, &o.Status,
		&o.Subtotal, &o.ShippingCost, &o.CommissionAmount, &o.Total, &o.ShippingAddress, &o.CreatedAt,
	)
	return &o, err
}

func collectOrders(rows pgx.Rows) ([]*entity.Order, error) {
	var out []*entity.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan order: %w", err)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *OrderRepository) FindItemByID(ctx context.Context, itemID uuid.UUID) (*entity.OrderItem, error) {
	i, err := scanOrderItem(r.db.QueryRow(ctx, selectOrderItemCols+`FROM order_items WHERE id = $1`, itemID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find order item: %w", err)
	}
	return i, nil
}

const selectOrderItemCols = `SELECT id, order_id, variant_id, product_name, variant_name, unit, price, qty, weight_gram `

func scanOrderItem(row rowScanner) (*entity.OrderItem, error) {
	var i entity.OrderItem
	err := row.Scan(&i.ID, &i.OrderID, &i.VariantID, &i.ProductName, &i.VariantName, &i.Unit, &i.Price, &i.Qty, &i.WeightGram)
	return &i, err
}

func (r *OrderRepository) SalesSummary(ctx context.Context, supplierID uuid.UUID, from, to time.Time) (float64, int, error) {
	var gmv float64
	var count int
	const q = `
		SELECT COALESCE(SUM(total), 0), COUNT(*) FROM orders
		WHERE supplier_id = $1 AND status = 'completed' AND created_at BETWEEN $2 AND $3
	`
	if err := r.db.QueryRow(ctx, q, supplierID, from, to).Scan(&gmv, &count); err != nil {
		return 0, 0, fmt.Errorf("postgres: sales summary: %w", err)
	}
	return gmv, count, nil
}

func (r *OrderRepository) TopProducts(ctx context.Context, supplierID uuid.UUID, from, to time.Time, limit int) ([]*entity.TopProduct, error) {
	const q = `
		SELECT oi.product_name, SUM(oi.qty), SUM(oi.price * oi.qty)
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE o.supplier_id = $1 AND o.status = 'completed' AND o.created_at BETWEEN $2 AND $3
		GROUP BY oi.product_name
		ORDER BY SUM(oi.price * oi.qty) DESC
		LIMIT $4
	`
	rows, err := r.db.Query(ctx, q, supplierID, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("postgres: top products: %w", err)
	}
	defer rows.Close()

	var out []*entity.TopProduct
	for rows.Next() {
		var p entity.TopProduct
		if err := rows.Scan(&p.ProductName, &p.Qty, &p.Revenue); err != nil {
			return nil, fmt.Errorf("postgres: scan top product: %w", err)
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (r *OrderRepository) SalesPerDay(ctx context.Context, supplierID uuid.UUID, from, to time.Time) ([]*entity.DailySales, error) {
	const q = `
		SELECT date_trunc('day', created_at), SUM(total), COUNT(*)
		FROM orders
		WHERE supplier_id = $1 AND status = 'completed' AND created_at BETWEEN $2 AND $3
		GROUP BY date_trunc('day', created_at)
		ORDER BY date_trunc('day', created_at)
	`
	rows, err := r.db.Query(ctx, q, supplierID, from, to)
	if err != nil {
		return nil, fmt.Errorf("postgres: sales per day: %w", err)
	}
	defer rows.Close()

	var out []*entity.DailySales
	for rows.Next() {
		var d entity.DailySales
		if err := rows.Scan(&d.Day, &d.GMV, &d.Orders); err != nil {
			return nil, fmt.Errorf("postgres: scan daily sales: %w", err)
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

func (r *OrderRepository) PlatformSummary(ctx context.Context, from, to time.Time) (float64, map[string]int, error) {
	const q = `
		SELECT status, COUNT(*), COALESCE(SUM(total), 0) FROM orders
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY status
	`
	rows, err := r.db.Query(ctx, q, from, to)
	if err != nil {
		return 0, nil, fmt.Errorf("postgres: platform summary: %w", err)
	}
	defer rows.Close()

	var gmv float64
	byStatus := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		var statusTotal float64
		if err := rows.Scan(&status, &count, &statusTotal); err != nil {
			return 0, nil, fmt.Errorf("postgres: scan platform summary: %w", err)
		}
		byStatus[status] = count
		if status == "completed" {
			gmv = statusTotal
		}
	}
	return gmv, byStatus, rows.Err()
}

func (r *OrderRepository) PlatformSalesPerDay(ctx context.Context, from, to time.Time) ([]*entity.DailySales, error) {
	const q = `
		SELECT date_trunc('day', created_at), SUM(total), COUNT(*)
		FROM orders
		WHERE status = 'completed' AND created_at BETWEEN $1 AND $2
		GROUP BY date_trunc('day', created_at)
		ORDER BY date_trunc('day', created_at)
	`
	rows, err := r.db.Query(ctx, q, from, to)
	if err != nil {
		return nil, fmt.Errorf("postgres: platform sales per day: %w", err)
	}
	defer rows.Close()

	var out []*entity.DailySales
	for rows.Next() {
		var d entity.DailySales
		if err := rows.Scan(&d.Day, &d.GMV, &d.Orders); err != nil {
			return nil, fmt.Errorf("postgres: scan platform daily sales: %w", err)
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}
