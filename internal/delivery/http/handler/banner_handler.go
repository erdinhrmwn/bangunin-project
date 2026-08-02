package handler

import (
	"github.com/gofiber/fiber/v3"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"

	bannerusecase "erdinhrmwn/bangunin/internal/usecase/banner"
)

// BannerHandler serves admin banner CRUD and the public active list (FR-7.2).
type BannerHandler struct {
	banner *bannerusecase.Usecase
}

func NewBannerHandler(banner *bannerusecase.Usecase) *BannerHandler {
	return &BannerHandler{banner: banner}
}

func bannerInput(req dto.BannerRequest) bannerusecase.Input {
	return bannerusecase.Input{
		ImageURL:  req.ImageURL,
		Link:      req.Link,
		StartsAt:  req.StartsAt,
		EndsAt:    req.EndsAt,
		SortOrder: req.SortOrder,
		IsActive:  req.IsActive,
	}
}

func (h *BannerHandler) Create(c fiber.Ctx) error {
	var req dto.BannerRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	b, err := h.banner.Create(c.Context(), bannerInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("Banner created", b))
}

func (h *BannerHandler) Update(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req dto.BannerRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	b, err := h.banner.Update(c.Context(), id, bannerInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Banner updated", b))
}

func (h *BannerHandler) Delete(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	if err := h.banner.Delete(c.Context(), id); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Banner deleted", nil))
}

func (h *BannerHandler) ListAdmin(c fiber.Ctx) error {
	list, err := h.banner.ListAll(c.Context())
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", list))
}

func (h *BannerHandler) ListActive(c fiber.Ctx) error {
	list, err := h.banner.ListActive(c.Context())
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", list))
}
