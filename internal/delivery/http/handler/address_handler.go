package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	userusecase "erdinhrmwn/bangunin/internal/usecase/user"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"
)

// AddressHandler serves user address book CRUD (FR-5.1), mounted under /user/addresses.
type AddressHandler struct {
	user *userusecase.Usecase
}

func NewAddressHandler(user *userusecase.Usecase) *AddressHandler {
	return &AddressHandler{user: user}
}

func addressInput(req dto.AddressRequest) userusecase.AddressInput {
	return userusecase.AddressInput{
		Label:          req.Label,
		RecipientName:  req.RecipientName,
		RecipientPhone: req.RecipientPhone,
		ProvinceID:     req.ProvinceID,
		CityID:         req.CityID,
		Subdistrict:    req.Subdistrict,
		PostalCode:     req.PostalCode,
		AddressDetail:  req.AddressDetail,
		IsDefault:      req.IsDefault,
	}
}

func (h *AddressHandler) List(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	addrs, err := h.user.ListAddresses(c.Context(), userID)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", addrs))
}

func (h *AddressHandler) Create(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	var req dto.AddressRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	a, err := h.user.CreateAddress(c.Context(), userID, addressInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("Address created", a))
}

func (h *AddressHandler) Update(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	var req dto.AddressRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	a, err := h.user.UpdateAddress(c.Context(), userID, id, addressInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Address updated", a))
}

func (h *AddressHandler) Delete(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	if err := h.user.DeleteAddress(c.Context(), userID, id); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Address deleted", nil))
}
