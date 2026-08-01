package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	catalogusecase "erdinhrmwn/bangunin/internal/usecase/catalog"
	wishlistusecase "erdinhrmwn/bangunin/internal/usecase/wishlist"
	"erdinhrmwn/bangunin/pkg/pagination"
	"erdinhrmwn/bangunin/pkg/response"
)

// CatalogHandler serves the public storefront: search, product detail,
// supplier store page (FR-4.6-4.9).
type CatalogHandler struct {
	catalog  *catalogusecase.Usecase
	wishlist *wishlistusecase.Usecase
}

func NewCatalogHandler(catalog *catalogusecase.Usecase, wishlist *wishlistusecase.Usecase) *CatalogHandler {
	return &CatalogHandler{catalog: catalog, wishlist: wishlist}
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func atofDefault(s string, def float64) float64 {
	if s == "" {
		return def
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return f
}

func (h *CatalogHandler) Search(c fiber.Ctx) error {
	params := repository.SearchParams{
		Query:      c.Query("q"),
		CategoryID: atoiDefault(c.Query("category"), 0),
		CityID:     atoiDefault(c.Query("city"), 0),
		MinPrice:   atofDefault(c.Query("min_price"), 0),
		MaxPrice:   atofDefault(c.Query("max_price"), 0),
		Sort:       c.Query("sort", repository.SortNewest),
		Cursor:     c.Query("cursor"),
		Limit:      atoiDefault(c.Query("per_page"), 20),
	}
	products, nextCursor, err := h.catalog.Search(c.Context(), params)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", products, response.Meta{NextCursor: nextCursor}))
}

func (h *CatalogHandler) Detail(c fiber.Ctx) error {
	p, err := h.catalog.Detail(c.Context(), c.Params("slug"))
	if err != nil {
		return errJSON(c, err)
	}

	resp := dto.ProductDetailResponse{Product: p}
	if userID, err := uuid.Parse(ctxutil.UserID(c)); err == nil {
		resp.IsWishlisted, err = h.wishlist.IsWishlisted(c.Context(), userID, p.ID)
		if err != nil {
			return errJSON(c, err)
		}
	}
	return c.JSON(response.Success("OK", resp))
}

func (h *CatalogHandler) SupplierStore(c fiber.Ctx) error {
	p := pagination.Parse(c.Query("page"), c.Query("per_page"))
	s, products, total, err := h.catalog.SupplierStore(c.Context(), c.Params("slug"), p.Page, p.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", fiber.Map{"supplier": s, "products": products}, response.Meta{Page: p.Page, PerPage: p.PerPage, Total: total}))
}
