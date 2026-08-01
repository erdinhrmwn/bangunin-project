package dto

import "time"

type BannerRequest struct {
	ImageURL  string    `json:"image_url" validate:"required"`
	Link      string    `json:"link"`
	StartsAt  time.Time `json:"starts_at" validate:"required"`
	EndsAt    time.Time `json:"ends_at" validate:"required,gtfield=StartsAt"`
	SortOrder int       `json:"sort_order"`
	IsActive  bool      `json:"is_active"`
}
