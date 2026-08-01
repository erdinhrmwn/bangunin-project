package dto

type CreateReviewRequest struct {
	Rating  int    `json:"rating" validate:"required,min=1,max=5"`
	Comment string `json:"comment"`
}
