package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/domain/repository"
	inventoryusecase "erdinhrmwn/bangunin/internal/usecase/inventory"
	"erdinhrmwn/bangunin/pkg/pagination"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"
)

// InventoryHandler serves supplier stock adjustment and movement history
// (FR-4.5), mounted under /supplier/variants.
type InventoryHandler struct {
	inventory *inventoryusecase.Usecase
	suppliers repository.SupplierRepository
}

func NewInventoryHandler(inventory *inventoryusecase.Usecase, suppliers repository.SupplierRepository) *InventoryHandler {
	return &InventoryHandler{inventory: inventory, suppliers: suppliers}
}

func (h *InventoryHandler) AdjustStock(c fiber.Ctx) error {
	variantID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	var req dto.StockAdjustmentRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	v, err := h.inventory.AdjustStock(c.Context(), supplierID, variantID, req.Qty, req.Note)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Stock adjusted", v))
}

func (h *InventoryHandler) ListMovements(c fiber.Ctx) error {
	variantID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	p := pagination.Parse(c.Query("page"), c.Query("per_page"))
	movements, total, err := h.inventory.ListMovements(c.Context(), supplierID, variantID, p.Page, p.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", movements, response.Meta{Page: p.Page, PerPage: p.PerPage, Total: total}))
}
