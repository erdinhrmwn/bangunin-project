// Package order implements the order state machine (FR-5.8), listing
// (FR-5.10), shipment (FR-5.9), paid callback (FR-5.6), and the
// expire/autocomplete jobs (FR-5.7/FR-5.11).
package order

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/domain/service"
	"erdinhrmwn/bangunin/pkg/apperr"
)

// autocompleteAfter/movementNote: ponytail: fixed values, promote to config
// if ops needs to tune per-environment (CLAUDE.md §5.9 rung).
const autocompleteAfter = 3 * 24 * time.Hour

type Usecase struct {
	orders       repository.OrderRepository
	histories    repository.OrderStatusHistoryRepository
	shipments    repository.ShipmentRepository
	checkouts    repository.CheckoutGroupRepository
	payments     repository.PaymentRepository
	reservations repository.StockReservationRepository
	suppliers    repository.SupplierRepository
	users        repository.UserRepository
	notify       repository.NotificationRepository
	audit        repository.AuditLogRepository
	email        service.EmailEnqueuer
	ledger       repository.LedgerEntryRepository
	balances     repository.SupplierBalanceRepository
}

func New(
	orders repository.OrderRepository,
	histories repository.OrderStatusHistoryRepository,
	shipments repository.ShipmentRepository,
	checkouts repository.CheckoutGroupRepository,
	payments repository.PaymentRepository,
	reservations repository.StockReservationRepository,
	suppliers repository.SupplierRepository,
	users repository.UserRepository,
	notify repository.NotificationRepository,
	audit repository.AuditLogRepository,
	email service.EmailEnqueuer,
	ledger repository.LedgerEntryRepository,
	balances repository.SupplierBalanceRepository,
) *Usecase {
	return &Usecase{
		orders: orders, histories: histories, shipments: shipments, checkouts: checkouts,
		payments: payments, reservations: reservations, suppliers: suppliers,
		users: users, notify: notify, audit: audit, email: email,
		ledger: ledger, balances: balances,
	}
}

// transitions is the explicit FR-5.8 table: fromStatus -> toStatus -> allowed actor types.
var transitions = map[string]map[string][]string{
	entity.OrderStatusPendingPayment: {
		entity.OrderStatusPaid:      {entity.ActorTypeSystem},
		entity.OrderStatusExpired:   {entity.ActorTypeSystem},
		entity.OrderStatusCancelled: {entity.ActorTypeUser},
	},
	entity.OrderStatusPaid: {
		entity.OrderStatusProcessed: {entity.ActorTypeSupplier},
		entity.OrderStatusCancelled: {entity.ActorTypeUser, entity.ActorTypeSupplier},
	},
	entity.OrderStatusProcessed: {
		entity.OrderStatusShipped: {entity.ActorTypeSupplier},
	},
	entity.OrderStatusShipped: {
		entity.OrderStatusDelivered: {entity.ActorTypeSupplier},
	},
	entity.OrderStatusDelivered: {
		entity.OrderStatusCompleted: {entity.ActorTypeUser, entity.ActorTypeSystem},
	},
}

// transition validates + applies a status change, writing one history row (FR-5.8).
// actorID is nil for actorType=system.
func (u *Usecase) transition(ctx context.Context, o *entity.Order, toStatus, actorType string, actorID *uuid.UUID, note string) error {
	allowedActors, ok := transitions[o.Status][toStatus]
	if !ok {
		return apperr.New("INVALID_STATUS_TRANSITION", fmt.Sprintf("Cannot move order from %s to %s", o.Status, toStatus), 409)
	}
	allowed := false
	for _, a := range allowedActors {
		if a == actorType {
			allowed = true
			break
		}
	}
	if !allowed {
		return apperr.New("INVALID_STATUS_TRANSITION", fmt.Sprintf("Actor %s cannot move order to %s", actorType, toStatus), 409)
	}

	if err := u.orders.UpdateStatus(ctx, o.ID, toStatus); err != nil {
		return err
	}
	histID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("order: generate history id: %w", err)
	}
	if err := u.histories.Create(ctx, &entity.OrderStatusHistory{
		ID: histID, OrderID: o.ID, FromStatus: o.Status, ToStatus: toStatus,
		ActorType: actorType, ActorID: actorID, Note: note,
	}); err != nil {
		return err
	}
	o.Status = toStatus
	return nil
}

func (u *Usecase) Get(ctx context.Context, id uuid.UUID) (*entity.Order, error) {
	return u.orders.FindByID(ctx, id)
}

func (u *Usecase) History(ctx context.Context, orderID uuid.UUID) ([]*entity.OrderStatusHistory, error) {
	return u.histories.ListByOrderID(ctx, orderID)
}

func (u *Usecase) Shipment(ctx context.Context, orderID uuid.UUID) (*entity.Shipment, error) {
	return u.shipments.FindByOrderID(ctx, orderID)
}

func (u *Usecase) ListByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Order, int, error) {
	return u.orders.ListByUserID(ctx, userID, page, perPage)
}

func (u *Usecase) ListAll(ctx context.Context, page, perPage int) ([]*entity.Order, int, error) {
	return u.orders.ListAll(ctx, page, perPage)
}

func (u *Usecase) ListBySupplier(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.Order, int, error) {
	return u.orders.ListBySupplierID(ctx, supplierID, page, perPage)
}

// Process marks a paid order as processing (supplier acknowledges the order, FR-5.8).
func (u *Usecase) Process(ctx context.Context, orderID, supplierID uuid.UUID) error {
	o, err := u.orders.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if o.SupplierID != supplierID {
		return apperr.New("FORBIDDEN", "Order does not belong to your store", 403)
	}
	return u.transition(ctx, o, entity.OrderStatusProcessed, entity.ActorTypeSupplier, &supplierID, "Order processing started")
}

// Ship marks a processed order shipped (FR-5.9): courier requires a tracking
// number, fleet requires the supplier has own_fleet_enabled.
func (u *Usecase) Ship(ctx context.Context, orderID, supplierID uuid.UUID, method, courierCode, trackingNumber string) error {
	o, err := u.orders.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if o.SupplierID != supplierID {
		return apperr.New("FORBIDDEN", "Order does not belong to your store", 403)
	}

	switch method {
	case entity.ShipmentMethodCourier:
		if trackingNumber == "" {
			return apperr.New("VALIDATION_ERROR", "Tracking number is required for courier shipment", 422)
		}
	case entity.ShipmentMethodSupplierFleet:
		s, err := u.suppliers.FindByID(ctx, supplierID)
		if err != nil {
			return err
		}
		if !s.OwnFleetEnabled {
			return apperr.New("VALIDATION_ERROR", "Your store does not have fleet delivery enabled", 422)
		}
	default:
		return apperr.New("VALIDATION_ERROR", "Invalid shipment method", 422)
	}

	if err := u.transition(ctx, o, entity.OrderStatusShipped, entity.ActorTypeSupplier, &supplierID, "Order shipped"); err != nil {
		return err
	}

	shipID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("order: generate shipment id: %w", err)
	}
	now := time.Now()
	return u.shipments.Create(ctx, &entity.Shipment{
		ID: shipID, OrderID: orderID, Method: method, CourierCode: courierCode,
		TrackingNumber: trackingNumber, Status: entity.ShipmentStatusPickedUp, ShippedAt: &now,
	})
}

// Deliver marks a shipped order delivered (supplier-confirmed, FR-5.8).
func (u *Usecase) Deliver(ctx context.Context, orderID, supplierID uuid.UUID) error {
	o, err := u.orders.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if o.SupplierID != supplierID {
		return apperr.New("FORBIDDEN", "Order does not belong to your store", 403)
	}
	return u.transition(ctx, o, entity.OrderStatusDelivered, entity.ActorTypeSupplier, &supplierID, "Order delivered")
}

// Complete lets the buyer confirm receipt (FR-5.8); autocomplete job handles the system path.
func (u *Usecase) Complete(ctx context.Context, orderID, userID uuid.UUID) error {
	o, err := u.orders.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if o.UserID != userID {
		return apperr.New("FORBIDDEN", "Order does not belong to you", 403)
	}
	if err := u.transition(ctx, o, entity.OrderStatusCompleted, entity.ActorTypeUser, &userID, "Buyer confirmed receipt"); err != nil {
		return err
	}
	return u.onOrderCompleted(ctx, o)
}

// Cancel cancels a pending or paid order (FR-5.8/AC-5.d). Cancelling a paid
// order triggers a refund and restores stock; no real disbursement exists in
// this phase (escrow/settlement is phase 6) so the refund is recorded via the
// history note only — see plan decision on refund scope.
func (u *Usecase) Cancel(ctx context.Context, orderID, actorID uuid.UUID, actorType string) error {
	o, err := u.orders.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if actorType == entity.ActorTypeUser && o.UserID != actorID {
		return apperr.New("FORBIDDEN", "Order does not belong to you", 403)
	}
	if actorType == entity.ActorTypeSupplier && o.SupplierID != actorID {
		return apperr.New("FORBIDDEN", "Order does not belong to your store", 403)
	}

	wasPaid := o.Status == entity.OrderStatusPaid
	note := "Order cancelled"
	if wasPaid {
		note = "Order cancelled, refund processed (mock)"
	}
	if err := u.transition(ctx, o, entity.OrderStatusCancelled, actorType, &actorID, note); err != nil {
		return err
	}
	return u.restoreStock(ctx, o.ID, wasPaid)
}

// ForceStatus lets an admin override the status directly (FR-5.10), bypassing
// the state machine, with an audit log entry.
func (u *Usecase) ForceStatus(ctx context.Context, orderID, adminID uuid.UUID, toStatus, reason string) error {
	o, err := u.orders.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	before := o.Status
	if err := u.orders.UpdateStatus(ctx, orderID, toStatus); err != nil {
		return err
	}
	histID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("order: generate history id: %w", err)
	}
	if err := u.histories.Create(ctx, &entity.OrderStatusHistory{
		ID: histID, OrderID: orderID, FromStatus: before, ToStatus: toStatus,
		ActorType: entity.ActorTypeAdmin, ActorID: &adminID, Note: "Admin force status: " + reason,
	}); err != nil {
		return err
	}
	auditID, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("order: generate audit id: %w", err)
	}
	return u.audit.Create(ctx, &entity.AuditLog{
		ID: auditID, ActorID: adminID, Action: "order.force_status", EntityType: "order", EntityID: orderID,
		Metadata: map[string]any{"reason": reason, "before": before, "after": toStatus},
	})
}

// restoreStock reverses the reserve_convert decrement (if the order had been
// paid) or simply releases active reservations (if unpaid).
func (u *Usecase) restoreStock(ctx context.Context, orderID uuid.UUID, wasPaid bool) error {
	reservations, err := u.reservations.ListByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	var adjustments []repository.ReservationAdjustment
	for _, r := range reservations {
		if r.Status != entity.StockReservationStatusActive && r.Status != entity.StockReservationStatusConverted {
			continue
		}
		adj := repository.ReservationAdjustment{
			ReservationID: r.ID, VariantID: r.VariantID, Status: entity.StockReservationStatusReleased,
		}
		if wasPaid && r.Status == entity.StockReservationStatusConverted {
			adj.Delta = r.Qty
			adj.Movement = &entity.StockMovement{
				ID: mustNewV7(), VariantID: r.VariantID, Type: entity.MovementTypeAdjustment,
				Qty: r.Qty, RefType: entity.RefTypeOrder, RefID: &orderID, Note: "Order cancelled, stock restored",
			}
		}
		adjustments = append(adjustments, adj)
	}
	return u.reservations.ApplyAdjustments(ctx, adjustments)
}

// HandlePaidCallback applies FR-5.6, idempotent on the invoice already being
// paid (PaymentRepository.MarkPaid no-ops in that case).
func (u *Usecase) HandlePaidCallback(ctx context.Context, xenditInvoiceID string) error {
	p, err := u.payments.FindByXenditInvoiceID(ctx, xenditInvoiceID)
	if err != nil {
		return err
	}
	if p.Status == entity.PaymentStatusPaid {
		return nil
	}

	group, err := u.checkouts.FindByID(ctx, p.CheckoutGroupID)
	if err != nil {
		return err
	}
	if group.Status == entity.CheckoutGroupStatusPaid {
		return nil
	}

	if err := u.payments.MarkPaid(ctx, p.ID, time.Now()); err != nil {
		return err
	}
	if err := u.checkouts.UpdateStatus(ctx, group.ID, entity.CheckoutGroupStatusPaid); err != nil {
		return err
	}

	orders, err := u.orders.ListByCheckoutGroupID(ctx, group.ID)
	if err != nil {
		return err
	}
	for _, o := range orders {
		if err := u.transition(ctx, o, entity.OrderStatusPaid, entity.ActorTypeSystem, nil, "Payment received"); err != nil {
			return err
		}
		if err := u.convertReservations(ctx, o.ID); err != nil {
			return err
		}
		u.notifyOrderUpdate(ctx, o, "Order paid", "Your order "+o.OrderNumber+" has been paid.")
		u.notifySupplier(ctx, o, "New order received", "Order "+o.OrderNumber+" has been paid and is ready to process.")
	}
	return nil
}

func (u *Usecase) convertReservations(ctx context.Context, orderID uuid.UUID) error {
	reservations, err := u.reservations.ListByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	var adjustments []repository.ReservationAdjustment
	for _, r := range reservations {
		if r.Status != entity.StockReservationStatusActive {
			continue
		}
		adjustments = append(adjustments, repository.ReservationAdjustment{
			ReservationID: r.ID, VariantID: r.VariantID, Delta: -r.Qty,
			Status: entity.StockReservationStatusConverted,
			Movement: &entity.StockMovement{
				ID: mustNewV7(), VariantID: r.VariantID, Type: entity.MovementTypeReserveConvert,
				Qty: -r.Qty, RefType: entity.RefTypeOrder, RefID: &orderID, Note: "Reservation converted on payment",
			},
		})
	}
	return u.reservations.ApplyAdjustments(ctx, adjustments)
}

// HandleExpire runs the order:expire job (FR-5.7): releases reservations for
// checkout groups still pending past their payment's expiry.
func (u *Usecase) HandleExpire(ctx context.Context) error {
	reservations, err := u.reservations.ListExpired(ctx)
	if err != nil {
		return err
	}
	seen := make(map[uuid.UUID]bool)
	for _, r := range reservations {
		if seen[r.OrderID] {
			continue
		}
		seen[r.OrderID] = true
		if err := u.expireOrder(ctx, r.OrderID); err != nil {
			return err
		}
	}
	return nil
}

func (u *Usecase) expireOrder(ctx context.Context, orderID uuid.UUID) error {
	o, err := u.orders.FindByID(ctx, orderID)
	if err != nil {
		return err
	}
	if o.Status != entity.OrderStatusPendingPayment {
		return nil
	}
	if err := u.transition(ctx, o, entity.OrderStatusExpired, entity.ActorTypeSystem, nil, "Payment window expired"); err != nil {
		return err
	}
	if err := u.restoreStock(ctx, o.ID, false); err != nil {
		return err
	}
	u.notifyOrderUpdate(ctx, o, "Order expired", "Your order "+o.OrderNumber+" expired due to no payment.")
	return nil
}

// HandleAutocomplete runs the order:autocomplete job (FR-5.11): delivered
// orders older than autocompleteAfter move to completed.
func (u *Usecase) HandleAutocomplete(ctx context.Context) error {
	orders, err := u.orders.ListDeliveredBefore(ctx, time.Now().Add(-autocompleteAfter))
	if err != nil {
		return err
	}
	for _, o := range orders {
		if err := u.transition(ctx, o, entity.OrderStatusCompleted, entity.ActorTypeSystem, nil, "Auto-completed after delivery window"); err != nil {
			return err
		}
		if err := u.onOrderCompleted(ctx, o); err != nil {
			return err
		}
	}
	return nil
}

// onOrderCompleted is the single call site for post-completion effects
// (FR-6.1): credits the supplier balance for the order subtotal (+shipping
// if own-fleet, since courier shipping never enters escrow) and debits the
// pre-computed platform commission, recording each as an append-only ledger
// entry. Idempotent by construction: both call sites only invoke this right
// after a successful transition to completed, and "completed" is terminal
// in the transitions table, so a second attempt fails at transition() before
// reaching here.
func (u *Usecase) onOrderCompleted(ctx context.Context, o *entity.Order) error {
	shipment, err := u.shipments.FindByOrderID(ctx, o.ID)
	if err != nil {
		return fmt.Errorf("order: find shipment for settlement: %w", err)
	}

	credit := o.Subtotal
	if shipment.Method == entity.ShipmentMethodSupplierFleet {
		credit += o.ShippingCost
	}

	balance, err := u.balances.Increment(ctx, o.SupplierID, credit)
	if err != nil {
		return fmt.Errorf("order: credit supplier balance: %w", err)
	}
	if err := u.ledger.Create(ctx, &entity.LedgerEntry{
		ID: mustNewV7(), OrderID: &o.ID, SupplierID: o.SupplierID,
		Type: entity.LedgerTypeCreditOrder, Amount: credit, BalanceAfter: balance,
	}); err != nil {
		return fmt.Errorf("order: record credit ledger entry: %w", err)
	}

	balance, err = u.balances.Increment(ctx, o.SupplierID, -o.CommissionAmount)
	if err != nil {
		return fmt.Errorf("order: debit commission: %w", err)
	}
	if err := u.ledger.Create(ctx, &entity.LedgerEntry{
		ID: mustNewV7(), OrderID: &o.ID, SupplierID: o.SupplierID,
		Type: entity.LedgerTypeDebitCommission, Amount: o.CommissionAmount, BalanceAfter: balance,
	}); err != nil {
		return fmt.Errorf("order: record commission ledger entry: %w", err)
	}
	return nil
}

func (u *Usecase) notifyOrderUpdate(ctx context.Context, o *entity.Order, title, body string) {
	u.notifyUser(ctx, o.UserID, title, body)
}

func (u *Usecase) notifySupplier(ctx context.Context, o *entity.Order, title, body string) {
	s, err := u.suppliers.FindByID(ctx, o.SupplierID)
	if err != nil {
		return
	}
	u.notifyUser(ctx, s.UserID, title, body)
}

func (u *Usecase) notifyUser(ctx context.Context, userID uuid.UUID, title, body string) {
	notifID, err := uuid.NewV7()
	if err != nil {
		return
	}
	_ = u.notify.Create(ctx, &entity.Notification{
		ID: notifID, UserID: userID, Type: entity.NotificationTypeOrderUpdate, Title: title, Body: body,
	})
	if usr, err := u.users.FindByID(ctx, userID); err == nil {
		_ = u.email.EnqueueEmail(usr.Email, title, body)
	}
}

func mustNewV7() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(err)
	}
	return id
}
