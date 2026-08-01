package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"

	categoryusecase "erdinhrmwn/bangunin/internal/usecase/category"
)

// CategoryHandler serves admin category CRUD and the public tree (FR-4.1).
type CategoryHandler struct {
	category *categoryusecase.Usecase
}

func NewCategoryHandler(category *categoryusecase.Usecase) *CategoryHandler {
	return &CategoryHandler{category: category}
}

func categoryInput(req dto.CategoryRequest) categoryusecase.Input {
	return categoryusecase.Input{
		ParentID:  req.ParentID,
		Name:      req.Name,
		IsActive:  req.IsActive,
		SortOrder: req.SortOrder,
	}
}

func (h *CategoryHandler) Create(c fiber.Ctx) error {
	var req dto.CategoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	cat, err := h.category.Create(c.Context(), categoryInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("Category created", cat))
}

func (h *CategoryHandler) Update(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	var req dto.CategoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	cat, err := h.category.Update(c.Context(), id, categoryInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Category updated", cat))
}

func (h *CategoryHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	if err := h.category.Delete(c.Context(), id); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Category deleted", nil))
}

func (h *CategoryHandler) Tree(c fiber.Ctx) error {
	tree, err := h.category.Tree(c.Context())
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", tree))
}
