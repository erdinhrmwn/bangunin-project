package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	ShipmentMethodCourier       = "courier"
	ShipmentMethodSupplierFleet = "supplier_fleet"

	ShipmentStatusPending   = "pending"
	ShipmentStatusPickedUp  = "picked_up"
	ShipmentStatusInTransit = "in_transit"
	ShipmentStatusDelivered = "delivered"
)

type Shipment struct {
	ID             uuid.UUID
	OrderID        uuid.UUID
	Method         string
	CourierCode    string
	CourierService string
	TrackingNumber string
	Cost           float64
	Status         string
	ShippedAt      *time.Time
	DeliveredAt    *time.Time
	CreatedAt      time.Time
}
