package dto

import "github.com/google/uuid"

type ToggleWishlistRequest struct {
	ProductID uuid.UUID `json:"product_id" validate:"required"`
}
