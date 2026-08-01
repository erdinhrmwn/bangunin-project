package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	supplierusecase "erdinhrmwn/bangunin/internal/usecase/supplier"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"
)

// SupplierHandler serves the caller's own store profile, documents, bank
// accounts, and submission (FR-3.2-3.5), mounted under /supplier.
type SupplierHandler struct {
	supplier *supplierusecase.Usecase
}

func NewSupplierHandler(supplier *supplierusecase.Usecase) *SupplierHandler {
	return &SupplierHandler{supplier: supplier}
}

func profileInput(req dto.UpsertProfileRequest) supplierusecase.ProfileInput {
	return supplierusecase.ProfileInput{
		StoreName:       req.StoreName,
		Description:     req.Description,
		OriginCityID:    req.OriginCityID,
		PickupAddress:   req.PickupAddress,
		OwnFleetEnabled: req.OwnFleetEnabled,
		FleetCoverageKM: req.FleetCoverageKM,
		FleetFlatRate:   req.FleetFlatRate,
	}
}

func (h *SupplierHandler) CreateProfile(c fiber.Ctx) error {
	var req dto.UpsertProfileRequest
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
	s, err := h.supplier.CreateProfile(c.Context(), userID, profileInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("Profile created", s))
}

func (h *SupplierHandler) UpdateProfile(c fiber.Ctx) error {
	var req dto.UpsertProfileRequest
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
	s, err := h.supplier.UpdateProfile(c.Context(), userID, profileInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Profile updated", s))
}

func (h *SupplierHandler) Profile(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	s, err := h.supplier.Profile(c.Context(), userID)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", s))
}

func (h *SupplierHandler) UploadDocument(c fiber.Ctx) error {
	var req dto.UploadDocumentRequest
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
	d, err := h.supplier.UploadDocument(c.Context(), userID, req.DocType, req.FileKey)
	if err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("Document uploaded", d))
}

func (h *SupplierHandler) ListDocuments(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	docs, err := h.supplier.ListDocuments(c.Context(), userID)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", docs))
}

func bankAccountInput(req dto.BankAccountRequest) supplierusecase.BankAccountInput {
	return supplierusecase.BankAccountInput{
		BankCode:      req.BankCode,
		AccountNumber: req.AccountNumber,
		AccountName:   req.AccountName,
		IsDefault:     req.IsDefault,
	}
}

func (h *SupplierHandler) CreateBankAccount(c fiber.Ctx) error {
	var req dto.BankAccountRequest
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
	a, err := h.supplier.UpsertBankAccount(c.Context(), userID, nil, bankAccountInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("Bank account created", a))
}

func (h *SupplierHandler) UpdateBankAccount(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	var req dto.BankAccountRequest
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
	a, err := h.supplier.UpsertBankAccount(c.Context(), userID, &id, bankAccountInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Bank account updated", a))
}

func (h *SupplierHandler) ListBankAccounts(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	accounts, err := h.supplier.ListBankAccounts(c.Context(), userID)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", accounts))
}

func (h *SupplierHandler) DeleteBankAccount(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return badRequest(c)
	}
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	if err := h.supplier.DeleteBankAccount(c.Context(), userID, id); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Bank account deleted", nil))
}

func (h *SupplierHandler) Submit(c fiber.Ctx) error {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return errJSON(c, err)
	}
	if err := h.supplier.Submit(c.Context(), userID); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Submitted for review", nil))
}
