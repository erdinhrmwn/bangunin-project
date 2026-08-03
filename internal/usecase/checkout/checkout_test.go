package checkout_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	"erdinhrmwn/bangunin/internal/domain/service"
	"erdinhrmwn/bangunin/pkg/apperr"

	servicemocks "erdinhrmwn/bangunin/internal/domain/service/mocks"
	infraredis "erdinhrmwn/bangunin/internal/repository/redis"
	checkoutusecase "erdinhrmwn/bangunin/internal/usecase/checkout"
)

type deps struct {
	checkoutGroups *mocks.MockCheckoutGroupRepository
	payments       *mocks.MockPaymentRepository
	reservations   *mocks.MockStockReservationRepository
	addresses      *mocks.MockUserAddressRepository
	carts          *mocks.MockCartRepository
	variants       *mocks.MockProductVariantRepository
	products       *mocks.MockProductRepository
	suppliers      *mocks.MockSupplierRepository
	shippingGW     *servicemocks.MockShippingGateway
	paymentGW      *servicemocks.MockPaymentGateway
	quotes         *infraredis.KVStore
}

// newUsecase wires a checkout.Usecase against mockery-mocked repos and a
// real StockLock backed by a local Redis (skips if unreachable, matching
// usecase/product's precedent for redis-dependent unit tests).
func newUsecase(t *testing.T) (*checkoutusecase.Usecase, *deps) {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not reachable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })

	d := &deps{
		checkoutGroups: mocks.NewMockCheckoutGroupRepository(t),
		payments:       mocks.NewMockPaymentRepository(t),
		reservations:   mocks.NewMockStockReservationRepository(t),
		addresses:      mocks.NewMockUserAddressRepository(t),
		carts:          mocks.NewMockCartRepository(t),
		variants:       mocks.NewMockProductVariantRepository(t),
		products:       mocks.NewMockProductRepository(t),
		suppliers:      mocks.NewMockSupplierRepository(t),
		shippingGW:     servicemocks.NewMockShippingGateway(t),
		paymentGW:      servicemocks.NewMockPaymentGateway(t),
		quotes:         infraredis.NewKVStore(rdb),
	}
	uc := checkoutusecase.New(
		d.checkoutGroups, d.payments, d.reservations, d.addresses, d.carts,
		d.variants, d.products, d.suppliers, d.shippingGW, d.paymentGW,
		infraredis.NewStockLock(rdb), d.quotes,
	)
	return uc, d
}

// seedQuote caches a valid shipping-cost quote for supplierID+variantID as
// Preview would, so Confirm-exercising tests pass the new server-side cost
// check. The cache key must match checkout.quoteKey's basket fingerprint
// (userID + addrID + sha256 of sorted "variantID:qty" pairs) or Confirm
// rejects it as QUOTE_EXPIRED before ever reaching the mocked repos.
func seedQuote(t *testing.T, d *deps, userID, addrID, supplierID, variantID uuid.UUID, qty int, cost float64) {
	t.Helper()
	data, err := json.Marshal(map[string][]float64{supplierID.String(): {cost}})
	require.NoError(t, err)
	sum := sha256.Sum256([]byte(variantID.String() + ":" + strconv.Itoa(qty)))
	key := "checkout_quote:" + userID.String() + ":" + addrID.String() + ":" + hex.EncodeToString(sum[:])
	require.NoError(t, d.quotes.Set(context.Background(), key, string(data), time.Minute))
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

func TestPreview_ForbiddenWhenAddressNotOwned(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: uuid.Must(uuid.NewV7())}, nil)

	_, err := uc.Preview(context.Background(), userID, addrID, nil)

	require.Equal(t, "FORBIDDEN", appErrCode(t, err))
}

func TestPreview_VariantInactive_Rejected(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID, variantID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: userID}, nil)
	d.variants.EXPECT().FindByIDs(mock.Anything, []uuid.UUID{variantID}).Return([]*entity.ProductVariant{{ID: variantID, IsActive: false}}, nil)
	d.products.EXPECT().FindByIDs(mock.Anything, mock.Anything).Return(nil, nil)

	_, err := uc.Preview(context.Background(), userID, addrID, []checkoutusecase.ItemRequest{{VariantID: variantID, Qty: 1}})

	require.Equal(t, "VARIANT_INACTIVE", appErrCode(t, err))
}

func TestPreview_OutOfStock_Rejected(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID, variantID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: userID}, nil)
	d.variants.EXPECT().FindByIDs(mock.Anything, []uuid.UUID{variantID}).Return([]*entity.ProductVariant{{ID: variantID, IsActive: true, MinOrderQty: 1, Stock: 1}}, nil)
	d.products.EXPECT().FindByIDs(mock.Anything, mock.Anything).Return(nil, nil)

	_, err := uc.Preview(context.Background(), userID, addrID, []checkoutusecase.ItemRequest{{VariantID: variantID, Qty: 2}})

	require.Equal(t, "OUT_OF_STOCK", appErrCode(t, err))
}

func TestPreview_SupplierNotApproved_Rejected(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID, variantID, productID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: userID}, nil)
	d.variants.EXPECT().FindByIDs(mock.Anything, []uuid.UUID{variantID}).Return([]*entity.ProductVariant{{
		ID: variantID, ProductID: productID, SupplierID: supplierID, IsActive: true, MinOrderQty: 1, Stock: 10,
	}}, nil)
	d.products.EXPECT().FindByIDs(mock.Anything, mock.Anything).Return([]*entity.Product{{ID: productID, Status: entity.ProductStatusActive}}, nil)
	d.suppliers.EXPECT().FindByID(mock.Anything, supplierID).Return(&entity.Supplier{ID: supplierID, Status: entity.SupplierStatusPending}, nil)

	_, err := uc.Preview(context.Background(), userID, addrID, []checkoutusecase.ItemRequest{{VariantID: variantID, Qty: 1}})

	require.Equal(t, "SUPPLIER_NOT_APPROVED", appErrCode(t, err))
}

func TestPreview_Success_GroupsPerSupplierWithShippingAndFleet(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	variantID, productID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: userID, CityID: 200}, nil)
	d.variants.EXPECT().FindByIDs(mock.Anything, []uuid.UUID{variantID}).Return([]*entity.ProductVariant{{
		ID: variantID, ProductID: productID, SupplierID: supplierID, IsActive: true,
		MinOrderQty: 1, Stock: 10, Price: 50000, WeightGram: 1000,
	}}, nil)
	d.products.EXPECT().FindByIDs(mock.Anything, mock.Anything).Return([]*entity.Product{{ID: productID, Name: "Semen", Status: entity.ProductStatusActive}}, nil)
	d.suppliers.EXPECT().FindByID(mock.Anything, supplierID).Return(&entity.Supplier{
		ID: supplierID, StoreName: "Toko A", Status: entity.SupplierStatusApproved,
		OriginCityID: 100, OwnFleetEnabled: true, FleetFlatRate: 25000,
	}, nil)
	d.shippingGW.EXPECT().GetCost(mock.Anything, 100, 200, 1000).Return(
		[]service.ShippingOption{{CourierCode: "jne", Cost: 15000}}, nil,
	)

	got, err := uc.Preview(context.Background(), userID, addrID, []checkoutusecase.ItemRequest{{VariantID: variantID, Qty: 1}})

	require.NoError(t, err)
	require.Len(t, got.Groups, 1)
	pg := got.Groups[0]
	require.Equal(t, supplierID, pg.SupplierID)
	require.Equal(t, 50000.0, pg.Subtotal)
	require.Len(t, pg.ShippingOptions, 1)
	require.NotNil(t, pg.FleetOption)
	require.Equal(t, 25000.0, pg.FleetOption.Cost)
	require.Equal(t, 50000.0+15000.0, got.GrandTotalEstimate)
}

func confirmChoice(supplierID, variantID uuid.UUID, qty int, shippingCost float64) checkoutusecase.ConfirmGroupChoice {
	return checkoutusecase.ConfirmGroupChoice{
		SupplierID:   supplierID,
		Items:        []checkoutusecase.ItemRequest{{VariantID: variantID, Qty: qty}},
		ShippingCost: shippingCost,
	}
}

func stubConfirmItemLookups(d *deps, variantID, productID, supplierID uuid.UUID) {
	d.variants.EXPECT().FindByIDs(mock.Anything, mock.Anything).Return([]*entity.ProductVariant{{
		ID: variantID, ProductID: productID, SupplierID: supplierID, Price: 50000, WeightGram: 1000,
	}}, nil)
	d.products.EXPECT().FindByIDs(mock.Anything, mock.Anything).Return([]*entity.Product{{ID: productID, Name: "Semen"}}, nil)
}

func TestConfirm_ForbiddenWhenAddressNotOwned(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: uuid.Must(uuid.NewV7())}, nil)

	_, err := uc.Confirm(context.Background(), userID, addrID, nil)

	require.Equal(t, "FORBIDDEN", appErrCode(t, err))
}

func TestConfirm_Success_ClearsCartAndCreatesInvoice(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	variantID, productID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	cartID, itemID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: userID}, nil)
	seedQuote(t, d, userID, addrID, supplierID, variantID, 1, 10000)
	stubConfirmItemLookups(d, variantID, productID, supplierID)
	d.checkoutGroups.EXPECT().Confirm(mock.Anything, mock.AnythingOfType("*entity.CheckoutGroup"), mock.AnythingOfType("[]repository.ConfirmOrder")).
		Run(func(_ context.Context, g *entity.CheckoutGroup, orders []repository.ConfirmOrder) {
			require.Len(t, orders, 1)
			require.Equal(t, 50000.0+10000.0, g.GrandTotal)
		}).Return(nil)
	d.paymentGW.EXPECT().CreateInvoice(mock.Anything, mock.Anything, 50000.0+10000.0, mock.Anything).Return("inv-1", "https://pay.example/inv-1", nil)
	d.payments.EXPECT().Create(mock.Anything, mock.MatchedBy(func(p *entity.Payment) bool {
		return p.XenditInvoiceID == "inv-1" && p.Status == entity.PaymentStatusPending
	})).Return(nil)
	d.carts.EXPECT().FindOrCreateByUserID(mock.Anything, userID).Return(&entity.Cart{ID: cartID}, nil)
	d.carts.EXPECT().ListItemsByCartID(mock.Anything, cartID).Return([]*entity.CartItem{{ID: itemID, CartID: cartID, VariantID: variantID}}, nil)
	d.carts.EXPECT().RemoveItem(mock.Anything, itemID).Return(nil)
	d.carts.EXPECT().Touch(mock.Anything, cartID).Return(nil)

	got, err := uc.Confirm(context.Background(), userID, addrID, []checkoutusecase.ConfirmGroupChoice{
		confirmChoice(supplierID, variantID, 1, 10000),
	})

	require.NoError(t, err)
	require.Equal(t, "https://pay.example/inv-1", got.InvoiceURL)
}

func TestConfirm_InvoiceFailure_ReleasesReservationsAndMarksFailed(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	variantID, productID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: userID}, nil)
	seedQuote(t, d, userID, addrID, supplierID, variantID, 1, 10000)
	stubConfirmItemLookups(d, variantID, productID, supplierID)
	d.checkoutGroups.EXPECT().Confirm(mock.Anything, mock.Anything, mock.Anything).Return(nil)
	d.paymentGW.EXPECT().CreateInvoice(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", "", service.ErrShippingUnavailable)
	d.checkoutGroups.EXPECT().UpdateStatus(mock.Anything, mock.Anything, entity.CheckoutGroupStatusPaymentFailed).Return(nil)
	d.reservations.EXPECT().ApplyAdjustments(mock.Anything, mock.MatchedBy(func(adjs []repository.ReservationAdjustment) bool {
		return len(adjs) == 1 && adjs[0].VariantID == variantID && adjs[0].Status == entity.StockReservationStatusReleased
	})).Return(nil)

	_, err := uc.Confirm(context.Background(), userID, addrID, []checkoutusecase.ConfirmGroupChoice{
		confirmChoice(supplierID, variantID, 1, 10000),
	})

	ae := apperr.From(err)
	require.Equal(t, "PAYMENT_UNAVAILABLE", ae.Code)
	require.Equal(t, 502, ae.HTTPStatus)
}

func TestConfirm_StockConflictFromRepo_Propagates409(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	variantID, productID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: userID}, nil)
	seedQuote(t, d, userID, addrID, supplierID, variantID, 1, 10000)
	stubConfirmItemLookups(d, variantID, productID, supplierID)
	d.checkoutGroups.EXPECT().Confirm(mock.Anything, mock.Anything, mock.Anything).Return(errs.ErrConflict)

	_, err := uc.Confirm(context.Background(), userID, addrID, []checkoutusecase.ConfirmGroupChoice{
		confirmChoice(supplierID, variantID, 1, 10000),
	})

	require.ErrorIs(t, err, errs.ErrConflict)
}

// TestConfirm_ConcurrentSameVariant_OnlyOneAcquiresLock covers AC-5.b: two
// concurrent Confirm calls for the same variant must not both proceed past
// the stock lock. Uses a real StockLock (Redis SETNX), not a mock, since
// checkout.Usecase depends on the concrete *redis.StockLock (no interface).
// The winner is pinned mid-flight (blocked inside checkoutGroups.Confirm) via
// a channel barrier so the second call deterministically hits the still-held
// lock, instead of relying on goroutine-scheduling luck against near-instant
// mocked calls.
func TestConfirm_ConcurrentSameVariant_OnlyOneAcquiresLock(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	variantID, productID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: userID}, nil).Twice()
	seedQuote(t, d, userID, addrID, supplierID, variantID, 1, 10000)
	stubConfirmItemLookups(d, variantID, productID, supplierID)

	holding := make(chan struct{})
	release := make(chan struct{})
	d.checkoutGroups.EXPECT().Confirm(mock.Anything, mock.Anything, mock.Anything).
		Run(func(context.Context, *entity.CheckoutGroup, []repository.ConfirmOrder) {
			close(holding)
			<-release
		}).Return(nil)
	d.paymentGW.EXPECT().CreateInvoice(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("inv-1", "https://pay.example/inv-1", nil)
	d.payments.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	d.carts.EXPECT().FindOrCreateByUserID(mock.Anything, userID).Return(&entity.Cart{ID: uuid.Must(uuid.NewV7())}, nil)
	d.carts.EXPECT().ListItemsByCartID(mock.Anything, mock.Anything).Return(nil, nil)
	d.carts.EXPECT().Touch(mock.Anything, mock.Anything).Return(nil)

	var wg sync.WaitGroup
	var errA error
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, errA = uc.Confirm(context.Background(), userID, addrID, []checkoutusecase.ConfirmGroupChoice{
			confirmChoice(supplierID, variantID, 1, 10000),
		})
	}()

	<-holding // wait until goroutine A is holding the lock, blocked mid-Confirm

	_, errB := uc.Confirm(context.Background(), userID, addrID, []checkoutusecase.ConfirmGroupChoice{
		confirmChoice(supplierID, variantID, 1, 10000),
	})

	close(release) // let A finish
	wg.Wait()

	require.NoError(t, errA, "the lock holder should complete successfully")
	require.Equal(t, "CONFLICT", appErrCode(t, errB), "the second confirm should get 409 while the lock is held")
}

func TestConfirm_NoPriorQuote_RejectsAsExpired(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	variantID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: userID}, nil)

	_, err := uc.Confirm(context.Background(), userID, addrID, []checkoutusecase.ConfirmGroupChoice{
		confirmChoice(supplierID, variantID, 1, 10000),
	})

	require.Equal(t, "QUOTE_EXPIRED", appErrCode(t, err))
}

func TestConfirm_ShippingCostMismatch_Rejected(t *testing.T) {
	uc, d := newUsecase(t)
	userID, addrID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	variantID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	d.addresses.EXPECT().FindByID(mock.Anything, addrID).Return(&entity.UserAddress{ID: addrID, UserID: userID}, nil)
	seedQuote(t, d, userID, addrID, supplierID, variantID, 1, 10000)

	_, err := uc.Confirm(context.Background(), userID, addrID, []checkoutusecase.ConfirmGroupChoice{
		confirmChoice(supplierID, variantID, 1, 0),
	})

	require.Equal(t, "SHIPPING_COST_MISMATCH", appErrCode(t, err))
}
