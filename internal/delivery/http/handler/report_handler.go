package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	"erdinhrmwn/bangunin/pkg/response"

	reportusecase "erdinhrmwn/bangunin/internal/usecase/report"
)

// ReportHandler serves the supplier sales summary + CSV export trigger (FR-7.3).
type ReportHandler struct {
	report    *reportusecase.Usecase
	suppliers repository.SupplierRepository
}

func NewReportHandler(report *reportusecase.Usecase, suppliers repository.SupplierRepository) *ReportHandler {
	return &ReportHandler{report: report, suppliers: suppliers}
}

func (h *ReportHandler) Summary(c fiber.Ctx) error {
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	from, to := parseAuditRange(c.Query("from"), c.Query("to"))
	s, err := h.report.GetSummary(c.Context(), supplierID, from, to)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", s))
}

func (h *ReportHandler) Export(c fiber.Ctx) error {
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	from, to := parseAuditRange(c.Query("from"), c.Query("to"))
	if err := h.report.Export(c.Context(), supplierID, from, to); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Report export queued", nil))
}

// AdminReportHandler serves the platform-wide summary + CSV export trigger (FR-7.4).
type AdminReportHandler struct {
	report *reportusecase.AdminUsecase
}

func NewAdminReportHandler(report *reportusecase.AdminUsecase) *AdminReportHandler {
	return &AdminReportHandler{report: report}
}

func (h *AdminReportHandler) Summary(c fiber.Ctx) error {
	from, to := parseAuditRange(c.Query("from"), c.Query("to"))
	s, err := h.report.GetSummary(c.Context(), from, to)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", s))
}

func (h *AdminReportHandler) Export(c fiber.Ctx) error {
	adminID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	from, to := parseAuditRange(c.Query("from"), c.Query("to"))
	if err := h.report.Export(c.Context(), adminID, from, to); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Report export queued", nil))
}
