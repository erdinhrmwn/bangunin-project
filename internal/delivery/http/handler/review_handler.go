package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	reviewusecase "erdinhrmwn/bangunin/internal/usecase/review"
	"erdinhrmwn/bangunin/pkg/pagination"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"
)

// ReviewHandler serves review creation (FR-6.5), mounted under
// /user/orders/:order_item_id/reviews, /products/:slug/reviews, /user/reviews.
type ReviewHandler struct {
	review   *reviewusecase.Usecase
	products repository.ProductRepository
}

func NewReviewHandler(review *reviewusecase.Usecase, products repository.ProductRepository) *ReviewHandler {
	return &ReviewHandler{review: review, products: products}
}

func (h *ReviewHandler) Create(c fiber.Ctx) error {
	orderItemID, err := uuid.Parse(c.Params("order_item_id"))
	if err != nil {
		return badRequest(c)
	}
	var req dto.CreateReviewRequest
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
	rv, err := h.review.Create(c.Context(), userID, orderItemID, req.Rating, req.Comment)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Review created", rv))
}

func (h *ReviewHandler) ListByProduct(c fiber.Ctx) error {
	p, err := h.products.FindBySlug(c.Context(), c.Params("slug"))
	if err != nil {
		return errJSON(c, err)
	}
	rating := atoiDefault(c.Query("rating"), 0)
	limit := atoiDefault(c.Query("per_page"), 20)
	revs, nextCursor, err := h.review.ListByProduct(c.Context(), p.ID, rating, c.Query("cursor"), limit)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", revs, response.Meta{NextCursor: nextCursor}))
}

func (h *ReviewHandler) ListMine(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	pg := pagination.Parse(c.Query("page"), c.Query("per_page"))
	revs, total, err := h.review.ListByUser(c.Context(), userID, pg.Page, pg.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", revs, response.Meta{Page: pg.Page, PerPage: pg.PerPage, Total: total}))
}
