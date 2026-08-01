package dto

type AddCartItemRequest struct {
	VariantID string `json:"variant_id" validate:"required,uuid"`
	Qty       int    `json:"qty" validate:"required,gt=0"`
}

type UpdateCartItemRequest struct {
	Qty int `json:"qty" validate:"required,gt=0"`
}
