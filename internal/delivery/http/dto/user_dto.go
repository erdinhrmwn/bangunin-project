package dto

type UpdateProfileRequest struct {
	Name  string `json:"name" validate:"required"`
	Phone string `json:"phone"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,password"`
}

type UserResponse struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Email           string  `json:"email"`
	Phone           string  `json:"phone"`
	Role            string  `json:"role"`
	Status          string  `json:"status"`
	EmailVerifiedAt *string `json:"email_verified_at"`
}
