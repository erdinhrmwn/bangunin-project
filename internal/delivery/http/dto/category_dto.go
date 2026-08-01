package dto

type CategoryRequest struct {
	ParentID  *int   `json:"parent_id"`
	Name      string `json:"name" validate:"required"`
	IsActive  bool   `json:"is_active"`
	SortOrder int    `json:"sort_order"`
}
