package dto

import "erdinhrmwn/bangunin/internal/domain/entity"

// ProductDetailResponse wraps entity.Product with the viewer's wishlist
// state (FR-7.1) — never baked into the entity itself since product detail
// is Redis-cached and shared across all users (see catalog.Usecase.Detail).
type ProductDetailResponse struct {
	*entity.Product
	IsWishlisted bool `json:"is_wishlisted"`
}
