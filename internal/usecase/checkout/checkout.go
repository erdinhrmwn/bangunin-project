// Package checkout implements checkout preview (shipping quotes per
// supplier group) and confirm (stock reservation + order split + payment
// invoice), per FR-5.3/FR-5.4.
package checkout

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/domain/service"
	"erdinhrmwn/bangunin/pkg/apperr"

	infraredis "erdinhrmwn/bangunin/internal/repository/redis"
)

const (
	reservationTTL = 2 * time.Hour
	lockTTL        = 10 * time.Second
	commissionRate = 0.04
	// quoteTTL: ponytail: fixed window a client has to Confirm after
	// Preview before the quoted shipping cost expires; promote to config
	// if ops needs to tune it per-environment.
	quoteTTL = 15 * time.Minute
)

type Usecase struct {
	checkoutGroups repository.CheckoutGroupRepository
	payments       repository.PaymentRepository
	reservations   repository.StockReservationRepository
	addresses      repository.UserAddressRepository
	carts          repository.CartRepository
	variants       repository.ProductVariantRepository
	products       repository.ProductRepository
	suppliers      repository.SupplierRepository
	shippingGW     service.ShippingGateway
	paymentGW      service.PaymentGateway
	lock           *infraredis.StockLock
	quotes         *infraredis.KVStore
}

func New(
	checkoutGroups repository.CheckoutGroupRepository,
	payments repository.PaymentRepository,
	reservations repository.StockReservationRepository,
	addresses repository.UserAddressRepository,
	carts repository.CartRepository,
	variants repository.ProductVariantRepository,
	products repository.ProductRepository,
	suppliers repository.SupplierRepository,
	shippingGW service.ShippingGateway,
	paymentGW service.PaymentGateway,
	lock *infraredis.StockLock,
	quotes *infraredis.KVStore,
) *Usecase {
	return &Usecase{
		checkoutGroups: checkoutGroups, payments: payments, reservations: reservations,
		addresses: addresses, carts: carts, variants: variants, products: products,
		suppliers: suppliers, shippingGW: shippingGW, paymentGW: paymentGW, lock: lock,
		quotes: quotes,
	}
}

// quoteKey scopes cached shipping quotes to the user + delivery address, so a
// quote from one address can't validate a Confirm against another.
func quoteKey(userID, addressID uuid.UUID) string {
	return "checkout_quote:" + userID.String() + ":" + addressID.String()
}

// validateShippingCosts rejects any ConfirmGroupChoice whose client-supplied
// ShippingCost doesn't match one of the costs Preview actually quoted for
// that supplier (courier options or the fleet flat rate), closing the price-
// manipulation gap where a client could set an arbitrary ShippingCost.
func (u *Usecase) validateShippingCosts(ctx context.Context, userID, addressID uuid.UUID, choices []ConfirmGroupChoice) error {
	raw, err := u.quotes.Get(ctx, quoteKey(userID, addressID))
	if err != nil {
		return apperr.New("QUOTE_EXPIRED", "Shipping quote expired, please preview checkout again", 422)
	}
	var quoted map[string][]float64
	if err := json.Unmarshal([]byte(raw), &quoted); err != nil {
		return fmt.Errorf("checkout: unmarshal cached shipping quote: %w", err)
	}
	for _, c := range choices {
		costs, ok := quoted[c.SupplierID.String()]
		if !ok {
			return apperr.New("QUOTE_EXPIRED", "Shipping quote expired, please preview checkout again", 422)
		}
		match := false
		for _, cost := range costs {
			if cost == c.ShippingCost {
				match = true
				break
			}
		}
		if !match {
			return apperr.New("SHIPPING_COST_MISMATCH", "Shipping cost does not match the previewed quote", 422)
		}
	}
	return nil
}

// ItemRequest is one checkout line: a variant + desired qty.
type ItemRequest struct {
	VariantID uuid.UUID
	Qty       int
}

// PreviewGroup is one supplier's slice of the cart, with shipping quotes.
type PreviewGroup struct {
	SupplierID      uuid.UUID
	SupplierName    string
	Items           []PreviewItem
	Subtotal        float64
	ShippingOptions []service.ShippingOption
	FleetOption     *FleetOption
}

type FleetOption struct {
	Cost float64
}

type PreviewItem struct {
	VariantID   uuid.UUID
	ProductName string
	VariantName string
	Unit        string
	Price       float64
	Qty         int
}

type PreviewResult struct {
	Groups             []PreviewGroup
	GrandTotalEstimate float64
}

// Preview groups requested items per supplier, validates each (stock,
// min-order, active, supplier approved), and resolves shipping quotes in
// parallel per group (FR-5.3).
func (u *Usecase) Preview(ctx context.Context, userID, addressID uuid.UUID, items []ItemRequest) (*PreviewResult, error) {
	addr, err := u.addresses.FindByID(ctx, addressID)
	if err != nil {
		return nil, err
	}
	if addr.UserID != userID {
		return nil, apperr.New("FORBIDDEN", "Address does not belong to you", 403)
	}

	type supplierBucket struct {
		supplier *entity.Supplier
		items    []PreviewItem
		weighted []shippingItem
		subtotal float64
	}
	buckets := make(map[uuid.UUID]*supplierBucket)
	var order []uuid.UUID

	for _, it := range items {
		v, err := u.variants.FindByID(ctx, it.VariantID)
		if err != nil {
			return nil, err
		}
		if !v.IsActive {
			return nil, apperr.New("VARIANT_INACTIVE", fmt.Sprintf("Variant %s is inactive", v.Name), 422)
		}
		if it.Qty < v.MinOrderQty {
			return nil, apperr.New("BELOW_MIN_ORDER", fmt.Sprintf("Variant %s requires min qty %d", v.Name, v.MinOrderQty), 422)
		}
		if v.Stock < it.Qty {
			return nil, apperr.New("OUT_OF_STOCK", fmt.Sprintf("Variant %s is out of stock", v.Name), 422)
		}
		p, err := u.products.FindByID(ctx, v.ProductID)
		if err != nil {
			return nil, err
		}
		if p.Status != entity.ProductStatusActive {
			return nil, apperr.New("PRODUCT_INACTIVE", fmt.Sprintf("Product %s is not active", p.Name), 422)
		}

		b, ok := buckets[v.SupplierID]
		if !ok {
			s, err := u.suppliers.FindByID(ctx, v.SupplierID)
			if err != nil {
				return nil, err
			}
			if s.Status != entity.SupplierStatusApproved {
				return nil, apperr.New("SUPPLIER_NOT_APPROVED", fmt.Sprintf("Supplier %s is not approved", s.StoreName), 422)
			}
			b = &supplierBucket{supplier: s}
			buckets[v.SupplierID] = b
			order = append(order, v.SupplierID)
		}
		b.items = append(b.items, PreviewItem{
			VariantID: v.ID, ProductName: p.Name, VariantName: v.Name,
			Unit: v.Unit, Price: v.Price, Qty: it.Qty,
		})
		b.weighted = append(b.weighted, shippingItem{variant: v, qty: it.Qty})
		b.subtotal += v.Price * float64(it.Qty)
	}

	groups := make([]PreviewGroup, len(order))
	g, gctx := errgroup.WithContext(ctx)
	for i, supplierID := range order {
		i, b := i, buckets[supplierID]
		g.Go(func() error {
			weight := chargeableWeight(b.weighted)
			opts, err := u.shippingGW.GetCost(gctx, b.supplier.OriginCityID, addr.CityID, weight)
			if err != nil {
				return err
			}
			pg := PreviewGroup{
				SupplierID: b.supplier.ID, SupplierName: b.supplier.StoreName,
				Items: b.items, Subtotal: b.subtotal, ShippingOptions: opts,
			}
			// ponytail: no city/address distance data in schema (only
			// FleetCoverageKM exists, nothing to compare it against) — always
			// offer the fleet option when enabled. Add real distance check
			// once city coordinates exist.
			if b.supplier.OwnFleetEnabled {
				pg.FleetOption = &FleetOption{Cost: b.supplier.FleetFlatRate}
			}
			groups[i] = pg
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	var grandTotal float64
	quoted := make(map[string][]float64, len(groups))
	for _, pg := range groups {
		grandTotal += pg.Subtotal
		if len(pg.ShippingOptions) > 0 {
			grandTotal += pg.ShippingOptions[0].Cost
		}
		costs := make([]float64, 0, len(pg.ShippingOptions)+1)
		for _, opt := range pg.ShippingOptions {
			costs = append(costs, opt.Cost)
		}
		if pg.FleetOption != nil {
			costs = append(costs, pg.FleetOption.Cost)
		}
		quoted[pg.SupplierID.String()] = costs
	}
	if data, err := json.Marshal(quoted); err == nil {
		_ = u.quotes.Set(ctx, quoteKey(userID, addressID), string(data), quoteTTL)
	}

	return &PreviewResult{Groups: groups, GrandTotalEstimate: grandTotal}, nil
}

type shippingItem struct {
	variant *entity.ProductVariant
	qty     int
}

func chargeableWeight(items []shippingItem) int {
	total := 0
	for _, it := range items {
		volumetric := it.variant.LengthCM * it.variant.WidthCM * it.variant.HeightCM / 6000
		w := it.variant.WeightGram
		if volumetric > w {
			w = volumetric
		}
		total += w * it.qty
	}
	return total
}

// ConfirmGroupChoice is one supplier group's chosen shipping method + cost,
// echoed back from Preview by the client.
type ConfirmGroupChoice struct {
	SupplierID   uuid.UUID
	Items        []ItemRequest
	ShippingCost float64
}

type ConfirmResult struct {
	CheckoutGroupID uuid.UUID
	InvoiceURL      string
}

// Confirm re-validates stock under a per-variant Redis lock, atomically
// creates the checkout group + orders + reservations, then requests a
// payment invoice. Invoice failure marks the group payment_failed and
// releases reservations (compensation), returning a 502 (FR-5.4).
func (u *Usecase) Confirm(ctx context.Context, userID, addressID uuid.UUID, choices []ConfirmGroupChoice) (*ConfirmResult, error) {
	addr, err := u.addresses.FindByID(ctx, addressID)
	if err != nil {
		return nil, err
	}
	if addr.UserID != userID {
		return nil, apperr.New("FORBIDDEN", "Address does not belong to you", 403)
	}

	if err := u.validateShippingCosts(ctx, userID, addressID, choices); err != nil {
		return nil, err
	}

	variantIDs := make([]uuid.UUID, 0)
	for _, c := range choices {
		for _, it := range c.Items {
			variantIDs = append(variantIDs, it.VariantID)
		}
	}
	sort.Slice(variantIDs, func(i, j int) bool { return variantIDs[i].String() < variantIDs[j].String() })

	locked := make([]uuid.UUID, 0, len(variantIDs))
	defer func() {
		for _, id := range locked {
			_ = u.lock.Release(ctx, id)
		}
	}()
	for _, id := range variantIDs {
		ok, err := u.lock.Acquire(ctx, id, lockTTL)
		if err != nil {
			return nil, fmt.Errorf("checkout: acquire stock lock: %w", err)
		}
		if !ok {
			return nil, apperr.New("CONFLICT", "Stock is being processed by another checkout, try again", 409)
		}
		locked = append(locked, id)
	}

	groupID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("checkout: generate group id: %w", err)
	}
	groupNumber, err := generateNumber("GRP")
	if err != nil {
		return nil, err
	}

	shippingAddr := entity.ShippingAddressSnapshot{
		RecipientName: addr.RecipientName, RecipientPhone: addr.RecipientPhone,
		ProvinceID: addr.ProvinceID, CityID: addr.CityID, Subdistrict: addr.Subdistrict,
		PostalCode: addr.PostalCode, AddressDetail: addr.AddressDetail,
	}

	var grandTotal float64
	confirmOrders := make([]repository.ConfirmOrder, 0, len(choices))
	for _, c := range choices {
		orderID, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("checkout: generate order id: %w", err)
		}
		orderNumber, err := generateOrderNumber()
		if err != nil {
			return nil, err
		}

		var subtotal float64
		items := make([]*entity.OrderItem, 0, len(c.Items))
		reservations := make([]*entity.StockReservation, 0, len(c.Items))
		for _, it := range c.Items {
			v, err := u.variants.FindByID(ctx, it.VariantID)
			if err != nil {
				return nil, err
			}
			p, err := u.products.FindByID(ctx, v.ProductID)
			if err != nil {
				return nil, err
			}
			subtotal += v.Price * float64(it.Qty)

			itemID, err := uuid.NewV7()
			if err != nil {
				return nil, fmt.Errorf("checkout: generate item id: %w", err)
			}
			items = append(items, &entity.OrderItem{
				ID: itemID, OrderID: orderID, VariantID: v.ID,
				ProductName: p.Name, VariantName: v.Name, Unit: v.Unit,
				Price: v.Price, Qty: it.Qty, WeightGram: v.WeightGram,
			})

			resID, err := uuid.NewV7()
			if err != nil {
				return nil, fmt.Errorf("checkout: generate reservation id: %w", err)
			}
			reservations = append(reservations, &entity.StockReservation{
				ID: resID, VariantID: v.ID, OrderID: orderID, Qty: it.Qty,
				Status: entity.StockReservationStatusActive, ExpiresAt: time.Now().Add(reservationTTL),
			})
		}

		commission := subtotal * commissionRate
		total := subtotal + c.ShippingCost
		grandTotal += total

		histID, err := uuid.NewV7()
		if err != nil {
			return nil, fmt.Errorf("checkout: generate history id: %w", err)
		}

		confirmOrders = append(confirmOrders, repository.ConfirmOrder{
			Order: &entity.Order{
				ID: orderID, CheckoutGroupID: groupID, UserID: userID, SupplierID: c.SupplierID,
				OrderNumber: orderNumber, Status: entity.OrderStatusPendingPayment,
				Subtotal: subtotal, ShippingCost: c.ShippingCost, CommissionAmount: commission,
				Total: total, ShippingAddress: shippingAddr,
			},
			Items:        items,
			Reservations: reservations,
			History: &entity.OrderStatusHistory{
				ID: histID, OrderID: orderID, FromStatus: "", ToStatus: entity.OrderStatusPendingPayment,
				ActorType: entity.ActorTypeUser, ActorID: &userID, Note: "Checkout confirmed",
			},
		})
	}

	group := &entity.CheckoutGroup{
		ID: groupID, UserID: userID, GroupNumber: groupNumber,
		GrandTotal: grandTotal, Status: entity.CheckoutGroupStatusPendingPayment,
	}

	if err := u.checkoutGroups.Confirm(ctx, group, confirmOrders); err != nil {
		return nil, err
	}

	invoiceID, invoiceURL, err := u.paymentGW.CreateInvoice(ctx, groupID, grandTotal, "Order "+groupNumber)
	if err != nil {
		_ = u.checkoutGroups.UpdateStatus(ctx, groupID, entity.CheckoutGroupStatusPaymentFailed)
		for _, co := range confirmOrders {
			for _, r := range co.Reservations {
				_ = u.reservations.UpdateStatus(ctx, r.ID, entity.StockReservationStatusReleased)
			}
		}
		return nil, apperr.New("PAYMENT_UNAVAILABLE", "Failed to create payment invoice, please try again", 502)
	}

	paymentID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("checkout: generate payment id: %w", err)
	}
	if err := u.payments.Create(ctx, &entity.Payment{
		ID: paymentID, CheckoutGroupID: groupID, XenditInvoiceID: invoiceID,
		Amount: grandTotal, Status: entity.PaymentStatusPending, InvoiceURL: invoiceURL,
	}); err != nil {
		return nil, err
	}

	if err := u.clearCartItems(ctx, userID, variantIDs); err != nil {
		return nil, err
	}

	return &ConfirmResult{CheckoutGroupID: groupID, InvoiceURL: invoiceURL}, nil
}

func (u *Usecase) clearCartItems(ctx context.Context, userID uuid.UUID, variantIDs []uuid.UUID) error {
	c, err := u.carts.FindOrCreateByUserID(ctx, userID)
	if err != nil {
		return err
	}
	rows, err := u.carts.ListItemsByCartID(ctx, c.ID)
	if err != nil {
		return err
	}
	wanted := make(map[uuid.UUID]bool, len(variantIDs))
	for _, id := range variantIDs {
		wanted[id] = true
	}
	for _, ci := range rows {
		if wanted[ci.VariantID] {
			if err := u.carts.RemoveItem(ctx, ci.ID); err != nil {
				return err
			}
		}
	}
	return u.carts.Touch(ctx, c.ID)
}

func generateOrderNumber() (string, error) {
	suffix, err := randomDigits(5)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ORD-%s-%s", time.Now().Format("20060102"), suffix), nil
}

func generateNumber(prefix string) (string, error) {
	suffix, err := randomDigits(8)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s", prefix, time.Now().Format("20060102"), suffix), nil
}

func randomDigits(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	digits := make([]byte, n)
	for i, c := range b {
		digits[i] = '0' + c%10
	}
	return string(digits), nil
}
