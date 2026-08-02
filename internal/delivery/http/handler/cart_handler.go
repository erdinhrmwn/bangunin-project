package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"

	cartusecase "erdinhrmwn/bangunin/internal/usecase/cart"
)

// CartHandler serves the user's shopping cart (FR-5.1), mounted under /user/cart.
type CartHandler struct {
	cart *cartusecase.Usecase
}

func NewCartHandler(cart *cartusecase.Usecase) *CartHandler {
	return &CartHandler{cart: cart}
}

func (h *CartHandler) Get(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	items, err := h.cart.Get(c.Context(), userID)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", items))
}

func (h *CartHandler) AddItem(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	var req dto.AddCartItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	variantID, err := uuid.Parse(req.VariantID)
	if err != nil {
		return badRequest(c)
	}
	if err := h.cart.Add(c.Context(), userID, variantID, req.Qty); err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("Item added", nil))
}

func (h *CartHandler) UpdateItem(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	itemID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req dto.UpdateCartItemRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	if err := h.cart.UpdateQty(c.Context(), userID, itemID, req.Qty); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Item updated", nil))
}

func (h *CartHandler) RemoveItem(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	itemID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.cart.Remove(c.Context(), userID, itemID); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Item removed", nil))
}
