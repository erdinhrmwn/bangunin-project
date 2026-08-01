package handler

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	userusecase "erdinhrmwn/bangunin/internal/usecase/user"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"
)

// UserHandler serves GET/PATCH /me — mounted under /user, /supplier, /admin
// route groups (FR-2.8, FR-2.9), each restricted by RequireRole.
type UserHandler struct {
	user *userusecase.Usecase
}

func NewUserHandler(user *userusecase.Usecase) *UserHandler {
	return &UserHandler{user: user}
}

func (h *UserHandler) Me(c fiber.Ctx) error {
	id, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}

	usr, err := h.user.Me(c.Context(), id)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", toUserResponse(usr)))
}

func (h *UserHandler) UpdateMe(c fiber.Ctx) error {
	var req dto.UpdateProfileRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}

	id, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}

	usr, err := h.user.UpdateProfile(c.Context(), id, req.Name, req.Phone)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Profile updated", toUserResponse(usr)))
}

func (h *UserHandler) UpdatePassword(c fiber.Ctx) error {
	var req dto.ChangePasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}

	id, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}

	if err := h.user.ChangePassword(c.Context(), id, req.OldPassword, req.NewPassword); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Password changed", nil))
}

func (h *UserHandler) UpdateNotificationSettings(c fiber.Ctx) error {
	var req dto.UpdateNotificationSettingsRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}

	id, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}

	usr, err := h.user.UpdateNotificationSettings(c.Context(), id, *req.EmailMarketing)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Notification settings updated", toUserResponse(usr)))
}

func toUserResponse(usr *entity.User) dto.UserResponse {
	var verifiedAt *string
	if usr.EmailVerifiedAt != nil {
		s := usr.EmailVerifiedAt.Format(time.RFC3339)
		verifiedAt = &s
	}
	return dto.UserResponse{
		ID:              usr.ID.String(),
		Name:            usr.Name,
		Email:           usr.Email,
		Phone:           usr.Phone,
		Role:            entity.RoleName(usr.RoleID),
		Status:          usr.Status,
		EmailVerifiedAt: verifiedAt,
		EmailMarketing:  usr.EmailMarketing,
	}
}
