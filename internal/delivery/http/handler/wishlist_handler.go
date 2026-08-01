package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	"erdinhrmwn/bangunin/pkg/pagination"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"

	wishlistusecase "erdinhrmwn/bangunin/internal/usecase/wishlist"
)

// WishlistHandler serves user wishlist add/remove/list (FR-7.1).
type WishlistHandler struct {
	wishlist *wishlistusecase.Usecase
}

func NewWishlistHandler(wishlist *wishlistusecase.Usecase) *WishlistHandler {
	return &WishlistHandler{wishlist: wishlist}
}

func (h *WishlistHandler) Add(c fiber.Ctx) error {
	var req dto.ToggleWishlistRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}

	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	if err := h.wishlist.Add(c.Context(), userID, req.ProductID); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Added to wishlist", nil))
}

func (h *WishlistHandler) Remove(c fiber.Ctx) error {
	productID, err := parseUUIDParam(c, "product_id")
	if err != nil {
		return err
	}
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	if err := h.wishlist.Remove(c.Context(), userID, productID); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Removed from wishlist", nil))
}

func (h *WishlistHandler) ListMine(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	p := pagination.Parse(c.Query("page"), c.Query("per_page"))
	items, total, err := h.wishlist.ListMine(c.Context(), userID, p.Page, p.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", items, response.Meta{Page: p.Page, PerPage: p.PerPage, Total: total}))
}
