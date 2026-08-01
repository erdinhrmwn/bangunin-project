// Package dto holds HTTP request/response bodies with validator tags.
// Business validation stays in the usecase layer (CLAUDE.md §7).
package dto

type RegisterRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,password"`
	Phone    string `json:"phone"`
	Role     string `json:"role" validate:"required,oneof=user supplier"`
}

type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

type ResendOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" validate:"required,email"`
	OTP         string `json:"otp" validate:"required,len=6"`
	NewPassword string `json:"new_password" validate:"required,password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
