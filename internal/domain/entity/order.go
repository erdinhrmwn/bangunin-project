package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	OrderStatusPendingPayment = "pending_payment"
	OrderStatusPaid           = "paid"
	OrderStatusProcessed      = "processed"
	OrderStatusShipped        = "shipped"
	OrderStatusDelivered      = "delivered"
	OrderStatusCompleted      = "completed"
	OrderStatusExpired        = "expired"
	OrderStatusCancelled      = "cancelled"
)

// ShippingAddressSnapshot is stored as JSONB on Order.ShippingAddress —
// a copy of UserAddress at checkout time (CLAUDE.md §5.8: never join to
// master data for historical order data).
type ShippingAddressSnapshot struct {
	RecipientName  string `json:"recipient_name"`
	RecipientPhone string `json:"recipient_phone"`
	ProvinceID     int    `json:"province_id"`
	CityID         int    `json:"city_id"`
	Subdistrict    string `json:"subdistrict"`
	PostalCode     string `json:"postal_code"`
	AddressDetail  string `json:"address_detail"`
}

type Order struct {
	ID               uuid.UUID
	CheckoutGroupID  uuid.UUID
	UserID           uuid.UUID
	SupplierID       uuid.UUID
	OrderNumber      string
	Status           string
	Subtotal         float64
	ShippingCost     float64
	CommissionAmount float64
	Total            float64
	ShippingAddress  ShippingAddressSnapshot
	CreatedAt        time.Time
}
