package order_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	servicemocks "erdinhrmwn/bangunin/internal/domain/service/mocks"
	orderusecase "erdinhrmwn/bangunin/internal/usecase/order"
	"erdinhrmwn/bangunin/pkg/apperr"
)

type deps struct {
	orders       *mocks.MockOrderRepository
	histories    *mocks.MockOrderStatusHistoryRepository
	shipments    *mocks.MockShipmentRepository
	checkouts    *mocks.MockCheckoutGroupRepository
	payments     *mocks.MockPaymentRepository
	reservations *mocks.MockStockReservationRepository
	variants     *mocks.MockProductVariantRepository
	suppliers    *mocks.MockSupplierRepository
	users        *mocks.MockUserRepository
	notify       *mocks.MockNotificationRepository
	audit        *mocks.MockAuditLogRepository
	email        *servicemocks.MockEmailEnqueuer
}

func newUsecase(t *testing.T) (*orderusecase.Usecase, *deps) {
	t.Helper()
	d := &deps{
		orders:       mocks.NewMockOrderRepository(t),
		histories:    mocks.NewMockOrderStatusHistoryRepository(t),
		shipments:    mocks.NewMockShipmentRepository(t),
		checkouts:    mocks.NewMockCheckoutGroupRepository(t),
		payments:     mocks.NewMockPaymentRepository(t),
		reservations: mocks.NewMockStockReservationRepository(t),
		variants:     mocks.NewMockProductVariantRepository(t),
		suppliers:    mocks.NewMockSupplierRepository(t),
		users:        mocks.NewMockUserRepository(t),
		notify:       mocks.NewMockNotificationRepository(t),
		audit:        mocks.NewMockAuditLogRepository(t),
		email:        servicemocks.NewMockEmailEnqueuer(t),
	}
	uc := orderusecase.New(
		d.orders, d.histories, d.shipments, d.checkouts, d.payments, d.reservations,
		d.variants, d.suppliers, d.users, d.notify, d.audit, d.email,
	)
	return uc, d
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

// stubNotify allows notifyUser's best-effort notification+email calls for any user.
func stubNotify(d *deps) {
	d.notify.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Maybe()
	d.users.EXPECT().FindByID(mock.Anything, mock.Anything).Return(&entity.User{Email: "buyer@example.com"}, nil).Maybe()
	d.email.EXPECT().EnqueueEmail(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	d.suppliers.EXPECT().FindByID(mock.Anything, mock.Anything).Return(&entity.Supplier{UserID: uuid.Must(uuid.NewV7())}, nil).Maybe()
}

func TestProcess_Success(t *testing.T) {
	uc, d := newUsecase(t)
	orderID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, SupplierID: supplierID, Status: entity.OrderStatusPaid}, nil)
	d.orders.EXPECT().UpdateStatus(mock.Anything, orderID, entity.OrderStatusProcessed).Return(nil)
	d.histories.EXPECT().Create(mock.Anything, mock.MatchedBy(func(h *entity.OrderStatusHistory) bool {
		return h.ToStatus == entity.OrderStatusProcessed && h.ActorType == entity.ActorTypeSupplier
	})).Return(nil)

	err := uc.Process(context.Background(), orderID, supplierID)

	require.NoError(t, err)
}

func TestProcess_WrongSupplier_Returns403(t *testing.T) {
	uc, d := newUsecase(t)
	orderID, supplierID, otherSupplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, SupplierID: supplierID, Status: entity.OrderStatusPaid}, nil)

	err := uc.Process(context.Background(), orderID, otherSupplierID)

	ae := apperr.From(err)
	require.Equal(t, "FORBIDDEN", ae.Code)
	require.Equal(t, 403, ae.HTTPStatus)
}

func TestShip_Courier_RequiresTrackingNumber(t *testing.T) {
	uc, d := newUsecase(t)
	orderID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, SupplierID: supplierID, Status: entity.OrderStatusProcessed}, nil)

	err := uc.Ship(context.Background(), orderID, supplierID, entity.ShipmentMethodCourier, "jne", "")

	require.Equal(t, "VALIDATION_ERROR", appErrCode(t, err))
}

func TestShip_Fleet_RequiresOwnFleetEnabled(t *testing.T) {
	uc, d := newUsecase(t)
	orderID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, SupplierID: supplierID, Status: entity.OrderStatusProcessed}, nil)
	d.suppliers.EXPECT().FindByID(mock.Anything, supplierID).Return(&entity.Supplier{ID: supplierID, OwnFleetEnabled: false}, nil)

	err := uc.Ship(context.Background(), orderID, supplierID, entity.ShipmentMethodSupplierFleet, "", "")

	require.Equal(t, "VALIDATION_ERROR", appErrCode(t, err))
}

func TestShip_Courier_Success(t *testing.T) {
	uc, d := newUsecase(t)
	orderID, supplierID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, SupplierID: supplierID, Status: entity.OrderStatusProcessed}, nil)
	d.orders.EXPECT().UpdateStatus(mock.Anything, orderID, entity.OrderStatusShipped).Return(nil)
	d.histories.EXPECT().Create(mock.Anything, mock.MatchedBy(func(h *entity.OrderStatusHistory) bool {
		return h.ToStatus == entity.OrderStatusShipped && h.ActorType == entity.ActorTypeSupplier
	})).Return(nil)
	d.shipments.EXPECT().Create(mock.Anything, mock.MatchedBy(func(s *entity.Shipment) bool {
		return s.TrackingNumber == "TRACK123" && s.Status == entity.ShipmentStatusPickedUp
	})).Return(nil)

	err := uc.Ship(context.Background(), orderID, supplierID, entity.ShipmentMethodCourier, "jne", "TRACK123")

	require.NoError(t, err)
}

func TestForceStatus_BypassesStateMachine(t *testing.T) {
	uc, d := newUsecase(t)
	orderID, adminID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, Status: entity.OrderStatusPendingPayment}, nil)
	d.orders.EXPECT().UpdateStatus(mock.Anything, orderID, entity.OrderStatusShipped).Return(nil)
	d.histories.EXPECT().Create(mock.Anything, mock.MatchedBy(func(h *entity.OrderStatusHistory) bool {
		return h.FromStatus == entity.OrderStatusPendingPayment && h.ToStatus == entity.OrderStatusShipped && h.ActorType == entity.ActorTypeAdmin
	})).Return(nil)
	d.audit.EXPECT().Create(mock.Anything, mock.MatchedBy(func(a *entity.AuditLog) bool {
		return a.Action == "order.force_status" && a.EntityID == orderID
	})).Return(nil)

	err := uc.ForceStatus(context.Background(), orderID, adminID, entity.OrderStatusShipped, "manual override")

	require.NoError(t, err)
}

func TestCancel_PaidOrder_RestoresStockAndRecordsMockRefund(t *testing.T) {
	uc, d := newUsecase(t)
	orderID, userID, variantID, resID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, UserID: userID, Status: entity.OrderStatusPaid}, nil)
	d.orders.EXPECT().UpdateStatus(mock.Anything, orderID, entity.OrderStatusCancelled).Return(nil)
	d.histories.EXPECT().Create(mock.Anything, mock.MatchedBy(func(h *entity.OrderStatusHistory) bool {
		return h.Note == "Order cancelled, refund processed (mock)"
	})).Return(nil)
	d.reservations.EXPECT().ListByOrderID(mock.Anything, orderID).Return([]*entity.StockReservation{
		{ID: resID, OrderID: orderID, VariantID: variantID, Qty: 3, Status: entity.StockReservationStatusConverted},
	}, nil)
	d.variants.EXPECT().AdjustStockWithMovement(mock.Anything, variantID, 3, mock.MatchedBy(func(m *entity.StockMovement) bool {
		return m.Type == entity.MovementTypeAdjustment && m.Qty == 3
	})).Return(10, nil)
	d.reservations.EXPECT().UpdateStatus(mock.Anything, resID, entity.StockReservationStatusReleased).Return(nil)

	err := uc.Cancel(context.Background(), orderID, userID, entity.ActorTypeUser)

	require.NoError(t, err)
}

func TestCancel_IllegalTransition_Returns409(t *testing.T) {
	uc, d := newUsecase(t)
	orderID, userID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, UserID: userID, Status: entity.OrderStatusShipped}, nil)

	err := uc.Cancel(context.Background(), orderID, userID, entity.ActorTypeUser)

	ae := apperr.From(err)
	require.Equal(t, "INVALID_STATUS_TRANSITION", ae.Code)
	require.Equal(t, 409, ae.HTTPStatus)
}

func TestHandlePaidCallback_AlreadyPaid_NoOps(t *testing.T) {
	uc, d := newUsecase(t)
	d.payments.EXPECT().FindByXenditInvoiceID(mock.Anything, "inv-1").Return(&entity.Payment{Status: entity.PaymentStatusPaid}, nil)

	err := uc.HandlePaidCallback(context.Background(), "inv-1")

	require.NoError(t, err)
}

func TestHandlePaidCallback_MarksPaidConvertsReservationsAndNotifiesBoth(t *testing.T) {
	uc, d := newUsecase(t)
	stubNotify(d)
	groupID, paymentID, orderID, variantID, resID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	d.payments.EXPECT().FindByXenditInvoiceID(mock.Anything, "inv-1").Return(&entity.Payment{
		ID: paymentID, CheckoutGroupID: groupID, Status: entity.PaymentStatusPending,
	}, nil)
	d.checkouts.EXPECT().FindByID(mock.Anything, groupID).Return(&entity.CheckoutGroup{ID: groupID, Status: entity.CheckoutGroupStatusPendingPayment}, nil)
	d.payments.EXPECT().MarkPaid(mock.Anything, paymentID, mock.AnythingOfType("time.Time")).Return(nil)
	d.checkouts.EXPECT().UpdateStatus(mock.Anything, groupID, entity.CheckoutGroupStatusPaid).Return(nil)
	d.orders.EXPECT().ListByCheckoutGroupID(mock.Anything, groupID).Return([]*entity.Order{
		{ID: orderID, Status: entity.OrderStatusPendingPayment, OrderNumber: "ORD-1"},
	}, nil)
	d.orders.EXPECT().UpdateStatus(mock.Anything, orderID, entity.OrderStatusPaid).Return(nil)
	d.histories.EXPECT().Create(mock.Anything, mock.MatchedBy(func(h *entity.OrderStatusHistory) bool {
		return h.ToStatus == entity.OrderStatusPaid && h.ActorType == entity.ActorTypeSystem
	})).Return(nil)
	d.reservations.EXPECT().ListByOrderID(mock.Anything, orderID).Return([]*entity.StockReservation{
		{ID: resID, OrderID: orderID, VariantID: variantID, Qty: 2, Status: entity.StockReservationStatusActive},
	}, nil)
	d.variants.EXPECT().AdjustStockWithMovement(mock.Anything, variantID, -2, mock.MatchedBy(func(m *entity.StockMovement) bool {
		return m.Type == entity.MovementTypeReserveConvert
	})).Return(8, nil)
	d.reservations.EXPECT().UpdateStatus(mock.Anything, resID, entity.StockReservationStatusConverted).Return(nil)

	err := uc.HandlePaidCallback(context.Background(), "inv-1")

	require.NoError(t, err)
}

func TestHandleExpire_ReleasesReservationAndExpiresOrderOnce(t *testing.T) {
	uc, d := newUsecase(t)
	stubNotify(d)
	orderID, variantID, res1, res2 := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())

	d.reservations.EXPECT().ListExpired(mock.Anything).Return([]*entity.StockReservation{
		{ID: res1, OrderID: orderID, VariantID: variantID, Status: entity.StockReservationStatusActive},
		{ID: res2, OrderID: orderID, VariantID: variantID, Status: entity.StockReservationStatusActive},
	}, nil)
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, Status: entity.OrderStatusPendingPayment, OrderNumber: "ORD-1"}, nil)
	d.orders.EXPECT().UpdateStatus(mock.Anything, orderID, entity.OrderStatusExpired).Return(nil)
	d.histories.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)
	d.reservations.EXPECT().ListByOrderID(mock.Anything, orderID).Return([]*entity.StockReservation{
		{ID: res1, OrderID: orderID, VariantID: variantID, Status: entity.StockReservationStatusActive},
		{ID: res2, OrderID: orderID, VariantID: variantID, Status: entity.StockReservationStatusActive},
	}, nil)
	d.reservations.EXPECT().UpdateStatus(mock.Anything, res1, entity.StockReservationStatusReleased).Return(nil)
	d.reservations.EXPECT().UpdateStatus(mock.Anything, res2, entity.StockReservationStatusReleased).Return(nil)

	err := uc.HandleExpire(context.Background())

	require.NoError(t, err)
}

func TestHandleAutocomplete_CompletesDeliveredOrders(t *testing.T) {
	uc, d := newUsecase(t)
	orderID := uuid.Must(uuid.NewV7())

	d.orders.EXPECT().ListDeliveredBefore(mock.Anything, mock.AnythingOfType("time.Time")).Return([]*entity.Order{
		{ID: orderID, Status: entity.OrderStatusDelivered},
	}, nil)
	d.orders.EXPECT().UpdateStatus(mock.Anything, orderID, entity.OrderStatusCompleted).Return(nil)
	d.histories.EXPECT().Create(mock.Anything, mock.MatchedBy(func(h *entity.OrderStatusHistory) bool {
		return h.ToStatus == entity.OrderStatusCompleted && h.ActorType == entity.ActorTypeSystem
	})).Return(nil)

	err := uc.HandleAutocomplete(context.Background())

	require.NoError(t, err)
}
