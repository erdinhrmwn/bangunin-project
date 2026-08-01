package service

import (
	"context"
	"errors"
)

// ErrShippingUnavailable is returned by ShippingGateway when the courier API
// is unreachable or errors after retry (FR-5.2).
var ErrShippingUnavailable = errors.New("shipping gateway unavailable")

// ShippingOption is one courier/service quote returned by ShippingGateway.
type ShippingOption struct {
	CourierCode    string
	CourierService string
	Cost           float64
	ETD            string // e.g. "2-3 days", as returned by RajaOngkir
}

// TrackInfo is waybill tracking status. TrackWaybill is a stub for phase 5
// (FR-5.9) — real tracking sync is out of scope beyond the method existing.
type TrackInfo struct {
	Status  string
	History []string
}

// ShippingGateway resolves courier shipping cost & tracking via RajaOngkir.
// Implemented by infra/rajaongkir.
type ShippingGateway interface {
	GetCost(ctx context.Context, originCityID, destCityID int, weightGram int) ([]ShippingOption, error)
	TrackWaybill(ctx context.Context, waybill string) (*TrackInfo, error)
}
