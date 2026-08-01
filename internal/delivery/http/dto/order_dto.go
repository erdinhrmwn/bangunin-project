package dto

type ShipOrderRequest struct {
	Method         string `json:"method" validate:"required,oneof=courier supplier_fleet"`
	CourierCode    string `json:"courier_code"`
	TrackingNumber string `json:"tracking_number"`
}

type ForceStatusRequest struct {
	Status string `json:"status" validate:"required"`
	Reason string `json:"reason" validate:"required"`
}

type CancelOrderRequest struct {
	Reason string `json:"reason"`
}
