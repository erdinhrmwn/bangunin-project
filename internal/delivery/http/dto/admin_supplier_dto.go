package dto

type RejectRequest struct {
	Reason string `json:"reason" validate:"required"`
}

type SuspendRequest struct {
	Reason string `json:"reason" validate:"required"`
}
