package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	orderusecase "erdinhrmwn/bangunin/internal/usecase/order"
	"erdinhrmwn/bangunin/pkg/apperr"
	"erdinhrmwn/bangunin/pkg/pagination"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"
)

// OrderHandler serves order listing/detail and status transitions (FR-5.5-5.8),
// mounted under /user/orders, /supplier/orders, /admin/orders.
type OrderHandler struct {
	order     *orderusecase.Usecase
	suppliers repository.SupplierRepository
}

func NewOrderHandler(order *orderusecase.Usecase, suppliers repository.SupplierRepository) *OrderHandler {
	return &OrderHandler{order: order, suppliers: suppliers}
}

func (h *OrderHandler) ListMine(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	p := pagination.Parse(c.Query("page"), c.Query("per_page"))
	orders, total, err := h.order.ListByUser(c.Context(), userID, p.Page, p.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", orders, response.Meta{Page: p.Page, PerPage: p.PerPage, Total: total}))
}

func (h *OrderHandler) GetMine(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	o, err := h.order.Get(c.Context(), id)
	if err != nil {
		return errJSON(c, err)
	}
	if o.UserID != userID {
		return errJSON(c, apperr.New("FORBIDDEN", "Order does not belong to you", 403))
	}
	return c.JSON(response.Success("OK", o))
}

func (h *OrderHandler) Cancel(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	var req dto.CancelOrderRequest
	_ = c.Bind().Body(&req)
	if err := h.order.Cancel(c.Context(), id, userID, entity.ActorTypeUser); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Order cancelled", nil))
}

func (h *OrderHandler) ListSupplier(c fiber.Ctx) error {
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	p := pagination.Parse(c.Query("page"), c.Query("per_page"))
	orders, total, err := h.order.ListBySupplier(c.Context(), supplierID, p.Page, p.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", orders, response.Meta{Page: p.Page, PerPage: p.PerPage, Total: total}))
}

func (h *OrderHandler) GetSupplier(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	o, err := h.order.Get(c.Context(), id)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", o))
}

func (h *OrderHandler) Process(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	if err := h.order.Process(c.Context(), id, supplierID); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Order processing", nil))
}

func (h *OrderHandler) Ship(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	var req dto.ShipOrderRequest
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
	if err := h.order.Ship(c.Context(), id, supplierID, req.Method, req.CourierCode, req.TrackingNumber); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Order shipped", nil))
}

func (h *OrderHandler) Deliver(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	if err := h.order.Deliver(c.Context(), id, supplierID); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Order delivered", nil))
}

func (h *OrderHandler) ListAdmin(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err == nil {
		o, err := h.order.Get(c.Context(), id)
		if err != nil {
			return errJSON(c, err)
		}
		return c.JSON(response.Success("OK", o))
	}
	return badRequest(c)
}

func (h *OrderHandler) ForceStatus(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	adminID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return badRequest(c)
	}
	var req dto.ForceStatusRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	if err := h.order.ForceStatus(c.Context(), id, adminID, req.Status, req.Reason); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Order status forced", nil))
}
