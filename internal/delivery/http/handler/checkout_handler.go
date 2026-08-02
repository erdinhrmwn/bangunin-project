package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"

	checkoutusecase "erdinhrmwn/bangunin/internal/usecase/checkout"
)

// CheckoutHandler serves cart-to-order checkout (FR-5.2-5.4), mounted under /user/checkout.
type CheckoutHandler struct {
	checkout *checkoutusecase.Usecase
}

func NewCheckoutHandler(checkout *checkoutusecase.Usecase) *CheckoutHandler {
	return &CheckoutHandler{checkout: checkout}
}

func itemRequests(items []dto.CheckoutItemRequest) ([]checkoutusecase.ItemRequest, error) {
	out := make([]checkoutusecase.ItemRequest, len(items))
	for i, it := range items {
		variantID, err := uuid.Parse(it.VariantID)
		if err != nil {
			return nil, err
		}
		out[i] = checkoutusecase.ItemRequest{VariantID: variantID, Qty: it.Qty}
	}
	return out, nil
}

func (h *CheckoutHandler) Preview(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	var req dto.CheckoutPreviewRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	addressID, err := uuid.Parse(req.AddressID)
	if err != nil {
		return badRequest(c)
	}
	items, err := itemRequests(req.Items)
	if err != nil {
		return badRequest(c)
	}
	result, err := h.checkout.Preview(c.Context(), userID, addressID, items)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", result))
}

func (h *CheckoutHandler) Confirm(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	var req dto.CheckoutConfirmRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	addressID, err := uuid.Parse(req.AddressID)
	if err != nil {
		return badRequest(c)
	}
	choices := make([]checkoutusecase.ConfirmGroupChoice, len(req.Groups))
	for i, g := range req.Groups {
		supplierID, err := uuid.Parse(g.SupplierID)
		if err != nil {
			return badRequest(c)
		}
		items, err := itemRequests(g.Items)
		if err != nil {
			return badRequest(c)
		}
		choices[i] = checkoutusecase.ConfirmGroupChoice{SupplierID: supplierID, Items: items, ShippingCost: g.ShippingCost}
	}
	result, err := h.checkout.Confirm(c.Context(), userID, addressID, choices)
	if err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("Checkout confirmed", result))
}
