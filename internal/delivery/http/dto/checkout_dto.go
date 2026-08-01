package dto

type CheckoutItemRequest struct {
	VariantID string `json:"variant_id" validate:"required,uuid"`
	Qty       int    `json:"qty" validate:"required,gt=0"`
}

type CheckoutPreviewRequest struct {
	AddressID string                `json:"address_id" validate:"required,uuid"`
	Items     []CheckoutItemRequest `json:"items" validate:"required,min=1,dive"`
}

type CheckoutGroupChoiceRequest struct {
	SupplierID   string                `json:"supplier_id" validate:"required,uuid"`
	Items        []CheckoutItemRequest `json:"items" validate:"required,min=1,dive"`
	ShippingCost float64               `json:"shipping_cost" validate:"gte=0"`
}

type CheckoutConfirmRequest struct {
	AddressID string                       `json:"address_id" validate:"required,uuid"`
	Groups    []CheckoutGroupChoiceRequest `json:"groups" validate:"required,min=1,dive"`
}
