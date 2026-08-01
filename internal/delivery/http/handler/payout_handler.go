package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	payoutusecase "erdinhrmwn/bangunin/internal/usecase/payout"
	"erdinhrmwn/bangunin/pkg/pagination"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"
)

// PayoutHandler serves supplier withdraw requests + balance (FR-6.3) and
// admin approval/rejection (FR-6.4), mounted under /supplier and /admin.
type PayoutHandler struct {
	payout    *payoutusecase.Usecase
	suppliers repository.SupplierRepository
}

func NewPayoutHandler(payout *payoutusecase.Usecase, suppliers repository.SupplierRepository) *PayoutHandler {
	return &PayoutHandler{payout: payout, suppliers: suppliers}
}

func (h *PayoutHandler) Balance(c fiber.Ctx) error {
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	b, err := h.payout.Balance(c.Context(), supplierID)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", b))
}

func (h *PayoutHandler) Request(c fiber.Ctx) error {
	var req dto.RequestWithdrawRequest
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
	bankAccountID, err := uuid.Parse(req.BankAccountID)
	if err != nil {
		return badRequest(c)
	}
	w, err := h.payout.Request(c.Context(), supplierID, bankAccountID, req.Amount)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Withdraw requested", w))
}

func (h *PayoutHandler) ListMine(c fiber.Ctx) error {
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	p := pagination.Parse(c.Query("page"), c.Query("per_page"))
	ws, total, err := h.payout.ListBySupplier(c.Context(), supplierID, p.Page, p.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", ws, response.Meta{Page: p.Page, PerPage: p.PerPage, Total: total}))
}

func (h *PayoutHandler) GetMine(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	w, err := h.payout.GetForSupplier(c.Context(), id, supplierID)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", w))
}

func (h *PayoutHandler) ListAdmin(c fiber.Ctx) error {
	p := pagination.Parse(c.Query("page"), c.Query("per_page"))
	ws, total, err := h.payout.ListAll(c.Context(), c.Query("status"), p.Page, p.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", ws, response.Meta{Page: p.Page, PerPage: p.PerPage, Total: total}))
}

func (h *PayoutHandler) GetAdmin(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	w, err := h.payout.Get(c.Context(), id)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", w))
}

func (h *PayoutHandler) Approve(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	actorID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	w, err := h.payout.Approve(c.Context(), actorID, id, c.IP())
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Withdraw approved", w))
}

func (h *PayoutHandler) Reject(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	var req dto.RejectRequest
	if err := c.Bind().Body(&req); err != nil {
		return badRequest(c)
	}
	if verrs := validator.Struct(req); verrs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(response.Error("Invalid data", verrs))
	}
	actorID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	w, err := h.payout.Reject(c.Context(), actorID, id, req.Reason, c.IP())
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Withdraw rejected", w))
}
