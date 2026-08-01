package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/delivery/http/dto"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/pkg/ctxutil"
	"erdinhrmwn/bangunin/pkg/pagination"
	"erdinhrmwn/bangunin/pkg/response"
	"erdinhrmwn/bangunin/pkg/validator"

	productusecase "erdinhrmwn/bangunin/internal/usecase/product"
)

// resolveSupplierID looks up the calling user's supplier profile ID.
// Shared by all supplier-scoped catalog handlers.
func resolveSupplierID(c fiber.Ctx, suppliers repository.SupplierRepository) (uuid.UUID, error) {
	userID, err := uuid.Parse(ctxutil.UserID(c))
	if err != nil {
		return uuid.UUID{}, err
	}
	s, err := suppliers.FindByUserID(c.Context(), userID)
	if err != nil {
		return uuid.UUID{}, err
	}
	return s.ID, nil
}

// ProductHandler serves supplier product/variant/image CRUD (FR-4.2-4.4),
// mounted under /supplier/products.
type ProductHandler struct {
	product   *productusecase.Usecase
	suppliers repository.SupplierRepository
}

func NewProductHandler(product *productusecase.Usecase, suppliers repository.SupplierRepository) *ProductHandler {
	return &ProductHandler{product: product, suppliers: suppliers}
}

func productInput(req dto.ProductRequest) productusecase.ProductInput {
	return productusecase.ProductInput{
		CategoryID:  req.CategoryID,
		Name:        req.Name,
		Description: req.Description,
		Specs:       req.Specs,
	}
}

func variantInput(req dto.VariantRequest) productusecase.VariantInput {
	return productusecase.VariantInput{
		SKU:         req.SKU,
		Name:        req.Name,
		Unit:        req.Unit,
		Price:       req.Price,
		WeightGram:  req.WeightGram,
		LengthCM:    req.LengthCM,
		WidthCM:     req.WidthCM,
		HeightCM:    req.HeightCM,
		MinOrderQty: req.MinOrderQty,
		Stock:       req.Stock,
		IsActive:    req.IsActive,
	}
}

func (h *ProductHandler) Create(c fiber.Ctx) error {
	var req dto.ProductRequest
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
	p, err := h.product.CreateProduct(c.Context(), supplierID, productInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("Product created", p))
}

func (h *ProductHandler) Update(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req dto.ProductRequest
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
	p, err := h.product.UpdateProduct(c.Context(), supplierID, id, productInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Product updated", p))
}

func (h *ProductHandler) Get(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	p, err := h.product.GetProduct(c.Context(), supplierID, id)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", p))
}

func (h *ProductHandler) List(c fiber.Ctx) error {
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	p := pagination.Parse(c.Query("page"), c.Query("per_page"))
	products, total, err := h.product.ListProducts(c.Context(), supplierID, p.Page, p.PerPage)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.SuccessPaginated("OK", products, response.Meta{Page: p.Page, PerPage: p.PerPage, Total: total}))
}

func (h *ProductHandler) Publish(c fiber.Ctx) error {
	id, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	p, err := h.product.Publish(c.Context(), supplierID, id)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Product published", p))
}

func (h *ProductHandler) CreateVariant(c fiber.Ctx) error {
	productID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req dto.VariantRequest
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
	v, err := h.product.CreateVariant(c.Context(), supplierID, productID, variantInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("Variant created", v))
}

func (h *ProductHandler) UpdateVariant(c fiber.Ctx) error {
	variantID, err := parseUUIDParam(c, "variantId")
	if err != nil {
		return err
	}
	var req dto.VariantRequest
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
	v, err := h.product.UpdateVariant(c.Context(), supplierID, variantID, variantInput(req))
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Variant updated", v))
}

func (h *ProductHandler) ListVariants(c fiber.Ctx) error {
	productID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	variants, err := h.product.ListVariants(c.Context(), supplierID, productID)
	if err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("OK", variants))
}

func (h *ProductHandler) AttachImage(c fiber.Ctx) error {
	productID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	var req dto.AttachImageRequest
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
	img, err := h.product.AttachImage(c.Context(), supplierID, productID, req.URL, req.IsPrimary)
	if err != nil {
		return errJSON(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(response.Success("Image attached", img))
}

func (h *ProductHandler) SetPrimaryImage(c fiber.Ctx) error {
	productID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	imageID, err := parseUUIDParam(c, "imageId")
	if err != nil {
		return err
	}
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	if err := h.product.SetPrimaryImage(c.Context(), supplierID, productID, imageID); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Primary image set", nil))
}

func (h *ProductHandler) DeleteImage(c fiber.Ctx) error {
	productID, err := parseUUIDParam(c, "id")
	if err != nil {
		return err
	}
	imageID, err := parseUUIDParam(c, "imageId")
	if err != nil {
		return err
	}
	supplierID, err := resolveSupplierID(c, h.suppliers)
	if err != nil {
		return errJSON(c, err)
	}
	if err := h.product.DeleteImage(c.Context(), supplierID, productID, imageID); err != nil {
		return errJSON(c, err)
	}
	return c.JSON(response.Success("Image deleted", nil))
}
