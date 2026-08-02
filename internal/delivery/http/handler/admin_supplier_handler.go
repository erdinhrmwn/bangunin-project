package handler

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	"erdinhrmwn/bangunin/pkg/pagination"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"

	adminsupplierusecase "erdinhrmwn/bangunin/internal/usecase/adminsupplier"
)

// AdminSupplierHandler serves admin review of supplier applications
// (FR-3.6-3.7), mounted under /admin.
type AdminSupplierHandler struct {
	admin *adminsupplierusecase.Usecase
}

func NewAdminSupplierHandler(admin *adminsupplierusecase.Usecase) *AdminSupplierHandler {
	return &AdminSupplierHandler{admin: admin}
}

func (h *AdminSupplierHandler) List(c fiber.Ctx) error {
	p := pagination.Parse(c.Query("page"), c.Query("per_page"))
	suppliers, total, err := h.admin.List(c.Context(), c.Query("status"), c.Query("q"), p.Page, p.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", suppliers, response.Meta{Page: p.Page, PerPage: p.PerPage, Total: total}))
}

func (h *AdminSupplierHandler) Get(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	detail, err := h.admin.Get(c.Context(), id)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", detail))
}

func (h *AdminSupplierHandler) Approve(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	actorID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	s, err := h.admin.Approve(c.Context(), actorID, id, c.IP())
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Supplier approved", s))
}

func (h *AdminSupplierHandler) Reject(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
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
	s, err := h.admin.Reject(c.Context(), actorID, id, req.Reason, c.IP())
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Supplier rejected", s))
}

func (h *AdminSupplierHandler) Suspend(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req dto.SuspendRequest
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
	s, err := h.admin.Suspend(c.Context(), actorID, id, req.Reason, c.IP())
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Supplier suspended", s))
}

func (h *AdminSupplierHandler) AuditLogs(c fiber.Ctx) error {
	p := pagination.Parse(c.Query("page"), c.Query("per_page"))
	from, to := parseAuditRange(c.Query("from"), c.Query("to"))
	logs, total, err := h.admin.AuditLogs(c.Context(), from, to, p.Page, p.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", logs, response.Meta{Page: p.Page, PerPage: p.PerPage, Total: total}))
}

// parseAuditRange defaults to the last 30 days when from/to aren't given
// or fail to parse (YYYY-MM-DD).
func parseAuditRange(fromStr, toStr string) (from, to time.Time) {
	to = time.Now()
	from = to.AddDate(0, 0, -30)
	if t, err := time.Parse("2006-01-02", fromStr); err == nil {
		from = t
	}
	if t, err := time.Parse("2006-01-02", toStr); err == nil {
		to = t
	}
	return from, to
}
