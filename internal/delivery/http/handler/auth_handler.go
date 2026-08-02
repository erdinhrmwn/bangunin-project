package handler

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/pkg/apperr"
	"erdinhrmwn/bangunin/pkg/jwt"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"

	authusecase "erdinhrmwn/bangunin/internal/usecase/auth"
)

type AuthHandler struct {
	auth   *authusecase.Usecase
	jwtSvc *jwt.Service
}

func NewAuthHandler(auth *authusecase.Usecase, jwtSvc *jwt.Service) *AuthHandler {
	return &AuthHandler{auth: auth, jwtSvc: jwtSvc}
}

func (h *AuthHandler) Register(c fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}

	usr, err := h.auth.Register(c.Context(), authusecase.RegisterInput{
		Name: req.Name, Email: req.Email, Password: req.Password, Phone: req.Phone, Role: req.Role,
	})
	if err != nil {
		return errJSON(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(response.Success("Registered, please verify your email", fiber.Map{"id": usr.ID}))
}

func (h *AuthHandler) VerifyEmail(c fiber.Ctx) error {
	var req dto.VerifyEmailRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}

	if err := h.auth.VerifyEmail(c.Context(), req.Email, req.OTP); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Email verified", nil))
}

func (h *AuthHandler) ResendOTP(c fiber.Ctx) error {
	var req dto.ResendOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}

	if err := h.auth.ResendOTP(c.Context(), req.Email); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OTP sent", nil))
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}

	pair, err := h.auth.Login(c.Context(), req.Email, req.Password, c.IP())
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Login successful", dto.TokenResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken}))
}

func (h *AuthHandler) Refresh(c fiber.Ctx) error {
	var req dto.RefreshRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}

	pair, err := h.auth.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Token refreshed", dto.TokenResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken}))
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	token, _ := strings.CutPrefix(c.Get("Authorization"), "Bearer ")
	claims, err := h.jwtSvc.Parse(token)
	if err != nil {
		return errJSON(c, apperr.New("UNAUTHORIZED", "Authentication required", fiber.StatusUnauthorized))
	}

	var req dto.RefreshRequest
	_ = c.Bind().Body(&req) // refresh_token optional on logout

	remaining := time.Until(claims.ExpiresAt.Time)
	if err := h.auth.Logout(c.Context(), req.RefreshToken, claims.JTI, remaining); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Logged out", nil))
}

func (h *AuthHandler) ForgotPassword(c fiber.Ctx) error {
	var req dto.ForgotPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}

	_ = h.auth.ForgotPassword(c.Context(), req.Email)
	return c.JSON(response.Success("If the email exists, an OTP has been sent", nil))
}

func (h *AuthHandler) ResetPassword(c fiber.Ctx) error {
	var req dto.ResetPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}

	if err := h.auth.ResetPassword(c.Context(), req.Email, req.OTP, req.NewPassword); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Password reset", nil))
}

func badRequest(c fiber.Ctx) error {
	return c.Status(fiber.StatusBadRequest).JSON(response.Error("Invalid request body", nil))
}

// parseUUIDParam parses a route param as a UUID (FR-7.5 centralized
// validation), returning a ready-to-return 400 response error on failure.
func parseUUIDParam(c fiber.Ctx, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params(param))
	if err != nil {
		return uuid.Nil, badRequest(c)
	}
	return id, nil
}

func errJSON(c fiber.Ctx, err error) error {
	ae := apperr.From(err)
	return c.Status(ae.HTTPStatus).JSON(response.Error(ae.Message, fiber.Map{"code": ae.Code}))
}
