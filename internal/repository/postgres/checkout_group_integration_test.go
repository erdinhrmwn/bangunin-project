//go:build integration

// Package postgres_test — money-critical repo coverage (checkout confirm,
// payment lifecycle, stock reservation adjustments) against real Postgres.
// Run with: go test -tags=integration ./internal/repository/postgres/...
package postgres_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository"
	postgresrepo "erdinhrmwn/bangunin/internal/repository/postgres"
)

// seedID builds a unique order/group/invoice number so parallel subtests
// never collide on a UNIQUE constraint. Truncated to fit group_number's and
// order_number's VARCHAR(30) columns (migrations 000015/000017).
func seedID(prefix string) string {
	return prefix + "-" + strings.ReplaceAll(uuid.Must(uuid.NewV7()).String(), "-", "")[:20]
}

// seedUser inserts a minimal buyer row (role_id=3 'user', per seeds/roles.go)
// to satisfy checkout_groups/orders' user_id FK.
func seedUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `INSERT INTO roles (id, name) VALUES (3, 'user') ON CONFLICT (id) DO NOTHING`)
	require.NoError(t, err)
	userID := uuid.Must(uuid.NewV7())
	_, err = pool.Exec(ctx,
		`INSERT INTO users (id, role_id, name, email, password_hash, phone) VALUES ($1, 3, 'Buyer', $2, 'x', '0800')`,
		userID, userID.String()+"@example.com")
	require.NoError(t, err)
	return userID
}

// variantIDFor looks up the single variant seedProduct created for productID.
func variantIDFor(t *testing.T, pool *pgxpool.Pool, productID uuid.UUID) uuid.UUID {
	t.Helper()
	var variantID uuid.UUID
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT id FROM product_variants WHERE product_id = $1`, productID).Scan(&variantID))
	return variantID
}

// seedOrderOnly inserts a bare order row (no items/reservations) to satisfy
// stock_movements/stock_reservations' order_id FK in tests that call
// StockReservationRepository directly rather than through Confirm.
func seedOrderOnly(t *testing.T, pool *pgxpool.Pool, groupID, userID, supplierID uuid.UUID) uuid.UUID {
	t.Helper()
	orderID := uuid.Must(uuid.NewV7())
	_, err := pool.Exec(context.Background(),
		`INSERT INTO orders (id, checkout_group_id, user_id, supplier_id, order_number, status, subtotal, shipping_cost, commission_amount, total, shipping_address)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, '{}')`,
		orderID, groupID, userID, supplierID, seedID("ORD"), entity.OrderStatusPendingPayment,
		50000, 15000, 0, 65000)
	require.NoError(t, err)
	return orderID
}

// seedConfirmedGroup runs a full Confirm to produce a checkout_group with an
// order attached, for tests exercising PaymentRepository against a
// realistic FK graph.
func seedConfirmedGroup(t *testing.T, pool *pgxpool.Pool, supplierID, userID, variantID uuid.UUID) uuid.UUID {
	t.Helper()
	repo := postgresrepo.NewCheckoutGroupRepository(pool)
	groupID := uuid.Must(uuid.NewV7())
	group := &entity.CheckoutGroup{
		ID: groupID, UserID: userID, GroupNumber: seedID("GRP"),
		GrandTotal: 65000, Status: entity.CheckoutGroupStatusPendingPayment,
	}
	order := newConfirmOrder(groupID, userID, supplierID, variantID, 1, 50000)
	require.NoError(t, repo.Confirm(context.Background(), group, []repository.ConfirmOrder{order}))
	return groupID
}

func newConfirmOrder(groupID, userID, supplierID, variantID uuid.UUID, qty int, price float64) repository.ConfirmOrder {
	orderID := uuid.Must(uuid.NewV7())
	return repository.ConfirmOrder{
		Order: &entity.Order{
			ID: orderID, CheckoutGroupID: groupID, UserID: userID, SupplierID: supplierID,
			OrderNumber: seedID("ORD"), Status: entity.OrderStatusPendingPayment,
			Subtotal: price * float64(qty), ShippingCost: 15000, CommissionAmount: 0,
			Total: price*float64(qty) + 15000,
			ShippingAddress: entity.ShippingAddressSnapshot{
				RecipientName: "Budi", RecipientPhone: "0812", ProvinceID: 1, CityID: 501,
				Subdistrict: "Sub", PostalCode: "12345", AddressDetail: "Jl. Test No. 1",
			},
		},
		Items: []*entity.OrderItem{{
			ID: uuid.Must(uuid.NewV7()), OrderID: orderID, VariantID: variantID,
			ProductName: "Semen", VariantName: "Default", Unit: "sak",
			Price: price, Qty: qty, WeightGram: 40000,
		}},
		Reservations: []*entity.StockReservation{{
			ID: uuid.Must(uuid.NewV7()), VariantID: variantID, OrderID: orderID, Qty: qty,
			Status: entity.StockReservationStatusActive, ExpiresAt: time.Now().Add(2 * time.Hour),
		}},
		History: &entity.OrderStatusHistory{
			ID: uuid.Must(uuid.NewV7()), OrderID: orderID, FromStatus: "", ToStatus: entity.OrderStatusPendingPayment,
			ActorType: entity.ActorTypeSystem,
		},
	}
}

// TestCheckoutGroupRepository_Confirm exercises the JSONB shipping-address
// snapshot round-trip and multi-table commit against real Postgres — the
// direct-struct-as-param pattern is unverified against a real driver
// anywhere else in the test suite.
func TestCheckoutGroupRepository_Confirm(t *testing.T) {
	pool := setupDB(t)
	f := seedBase(t, pool)
	productID := f.seedProduct(t, f.supplierID, "Semen Confirm", "semen-confirm", 50000, entity.ProductStatusActive)
	variantID := variantIDFor(t, pool, productID)
	userID := seedUser(t, pool)

	repo := postgresrepo.NewCheckoutGroupRepository(pool)
	groupID := uuid.Must(uuid.NewV7())
	group := &entity.CheckoutGroup{
		ID: groupID, UserID: userID, GroupNumber: seedID("GRP"),
		GrandTotal: 65000, Status: entity.CheckoutGroupStatusPendingPayment,
	}
	order := newConfirmOrder(groupID, userID, f.supplierID, variantID, 1, 50000)

	err := repo.Confirm(context.Background(), group, []repository.ConfirmOrder{order})
	require.NoError(t, err, "shipping_address JSONB struct param must round-trip via pgx")

	got, err := repo.FindByID(context.Background(), groupID)
	require.NoError(t, err)
	require.Equal(t, 65000.0, got.GrandTotal)

	var raw []byte
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT shipping_address FROM orders WHERE id = $1`, order.Order.ID).Scan(&raw))
	var addr entity.ShippingAddressSnapshot
	require.NoError(t, json.Unmarshal(raw, &addr))
	require.Equal(t, "Budi", addr.RecipientName)
	require.Equal(t, "Jl. Test No. 1", addr.AddressDetail)

	var resStatus string
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT status FROM stock_reservations WHERE id = $1`, order.Reservations[0].ID).Scan(&resStatus))
	require.Equal(t, entity.StockReservationStatusActive, resStatus)
}

// TestCheckoutGroupRepository_Confirm_ConcurrentSameVariant_OneWins is the
// DB-level analog of usecase/checkout's mocked-repo concurrency test — proves
// the real `SELECT ... FOR UPDATE` availability check serializes two
// concurrent Confirm calls against the same variant so only one succeeds
// when stock is exactly exhausted by the first (stock=10, 6 each = only one fits).
func TestCheckoutGroupRepository_Confirm_ConcurrentSameVariant_OneWins(t *testing.T) {
	pool := setupDB(t)
	f := seedBase(t, pool)
	productID := f.seedProduct(t, f.supplierID, "Semen Race", "semen-race", 50000, entity.ProductStatusActive)
	variantID := variantIDFor(t, pool, productID)
	userID := seedUser(t, pool)

	repo := postgresrepo.NewCheckoutGroupRepository(pool)
	run := func() error {
		groupID := uuid.Must(uuid.NewV7())
		group := &entity.CheckoutGroup{
			ID: groupID, UserID: userID, GroupNumber: seedID("GRP"),
			GrandTotal: 300000, Status: entity.CheckoutGroupStatusPendingPayment,
		}
		order := newConfirmOrder(groupID, userID, f.supplierID, variantID, 6, 50000)
		return repo.Confirm(context.Background(), group, []repository.ConfirmOrder{order})
	}

	var wg sync.WaitGroup
	results := make([]error, 2)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i] = run()
		}(i)
	}
	wg.Wait()

	var successes, conflicts int32
	for _, e := range results {
		switch {
		case e == nil:
			atomic.AddInt32(&successes, 1)
		case errors.Is(e, errs.ErrConflict):
			atomic.AddInt32(&conflicts, 1)
		}
	}
	require.EqualValues(t, 1, successes, "exactly one confirm should win the row lock and fit within stock")
	require.EqualValues(t, 1, conflicts, "the loser must get errs.ErrConflict, not a generic error")
}

// TestPaymentRepository_CreateMarkPaidFindByInvoice covers the full
// create -> mark-paid -> lookup round trip against real Postgres.
func TestPaymentRepository_CreateMarkPaidFindByInvoice(t *testing.T) {
	pool := setupDB(t)
	f := seedBase(t, pool)
	productID := f.seedProduct(t, f.supplierID, "Semen Pay", "semen-pay", 50000, entity.ProductStatusActive)
	variantID := variantIDFor(t, pool, productID)
	userID := seedUser(t, pool)
	groupID := seedConfirmedGroup(t, pool, f.supplierID, userID, variantID)

	repo := postgresrepo.NewPaymentRepository(pool)
	paymentID := uuid.Must(uuid.NewV7())
	invoiceID := seedID("inv")
	p := &entity.Payment{
		ID: paymentID, CheckoutGroupID: groupID, XenditInvoiceID: invoiceID,
		Method: entity.PaymentMethodVA, Amount: 65000, Status: entity.PaymentStatusPending,
		InvoiceURL: "https://pay.example/" + invoiceID,
	}
	require.NoError(t, repo.Create(context.Background(), p))

	found, err := repo.FindByXenditInvoiceID(context.Background(), invoiceID)
	require.NoError(t, err)
	require.Equal(t, entity.PaymentStatusPending, found.Status)
	require.Nil(t, found.PaidAt)

	paidAt := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, repo.MarkPaid(context.Background(), paymentID, paidAt))

	found2, err := repo.FindByCheckoutGroupID(context.Background(), groupID)
	require.NoError(t, err)
	require.Equal(t, entity.PaymentStatusPaid, found2.Status)
	require.NotNil(t, found2.PaidAt)
	require.WithinDuration(t, paidAt, *found2.PaidAt, time.Second)
}

// TestStockReservationRepository_ApplyAdjustments covers both the guarded
// negative-stock rejection and a valid delta committing movement+status.
func TestStockReservationRepository_ApplyAdjustments(t *testing.T) {
	pool := setupDB(t)
	f := seedBase(t, pool)
	productID := f.seedProduct(t, f.supplierID, "Semen Adjust", "semen-adjust", 50000, entity.ProductStatusActive)
	variantID := variantIDFor(t, pool, productID)
	userID := seedUser(t, pool)
	groupID := uuid.Must(uuid.NewV7())
	_, err := pool.Exec(context.Background(),
		`INSERT INTO checkout_groups (id, user_id, group_number, grand_total, status) VALUES ($1, $2, $3, $4, $5)`,
		groupID, userID, seedID("GRP"), 65000, entity.CheckoutGroupStatusPendingPayment)
	require.NoError(t, err)
	orderID := seedOrderOnly(t, pool, groupID, userID, f.supplierID)

	repo := postgresrepo.NewStockReservationRepository(pool)

	t.Run("negative delta beyond stock is rejected, stock untouched", func(t *testing.T) {
		err := repo.ApplyAdjustments(context.Background(), []repository.ReservationAdjustment{{
			ReservationID: uuid.Must(uuid.NewV7()), VariantID: variantID, Delta: -999,
			Movement: &entity.StockMovement{
				ID: uuid.Must(uuid.NewV7()), VariantID: variantID, Type: entity.MovementTypeReserveConvert,
				Qty: 999, RefType: entity.RefTypeOrder, RefID: &orderID,
			},
			Status: entity.StockReservationStatusConverted,
		}})
		require.ErrorIs(t, err, errs.ErrConflict)

		var stock int
		require.NoError(t, pool.QueryRow(context.Background(),
			`SELECT stock FROM product_variants WHERE id = $1`, variantID).Scan(&stock))
		require.Equal(t, 10, stock, "stock must be untouched after a rejected adjustment")
	})

	t.Run("valid delta commits movement and reservation status", func(t *testing.T) {
		resID := uuid.Must(uuid.NewV7())
		_, err := pool.Exec(context.Background(),
			`INSERT INTO stock_reservations (id, variant_id, order_id, qty, status, expires_at) VALUES ($1, $2, $3, $4, $5, $6)`,
			resID, variantID, orderID, 3, entity.StockReservationStatusActive, time.Now().Add(2*time.Hour))
		require.NoError(t, err)

		movementID := uuid.Must(uuid.NewV7())
		err = repo.ApplyAdjustments(context.Background(), []repository.ReservationAdjustment{{
			ReservationID: resID, VariantID: variantID, Delta: -3,
			Movement: &entity.StockMovement{
				ID: movementID, VariantID: variantID, Type: entity.MovementTypeReserveConvert,
				Qty: -3, RefType: entity.RefTypeOrder, RefID: &orderID,
			},
			Status: entity.StockReservationStatusConverted,
		}})
		require.NoError(t, err)

		var stock int
		require.NoError(t, pool.QueryRow(context.Background(),
			`SELECT stock FROM product_variants WHERE id = $1`, variantID).Scan(&stock))
		require.Equal(t, 7, stock)

		var status string
		require.NoError(t, pool.QueryRow(context.Background(),
			`SELECT status FROM stock_reservations WHERE id = $1`, resID).Scan(&status))
		require.Equal(t, entity.StockReservationStatusConverted, status)

		var movementStockAfter int
		require.NoError(t, pool.QueryRow(context.Background(),
			`SELECT stock_after FROM stock_movements WHERE id = $1`, movementID).Scan(&movementStockAfter))
		require.Equal(t, 7, movementStockAfter)
	})
}
