// Package product implements supplier-scoped product, variant, and image
// CRUD (FR-4.2–FR-4.4). Every mutation re-verifies ownership via a
// supplier-scoped lookup, not just role middleware.
package product

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/usecase/catalog"
	"erdinhrmwn/bangunin/pkg/apperr"
	"erdinhrmwn/bangunin/pkg/cache"
)

const maxImages = 8

type Usecase struct {
	products repository.ProductRepository
	variants repository.ProductVariantRepository
	images   repository.ProductImageRepository
	rdb      *redis.Client
}

func New(products repository.ProductRepository, variants repository.ProductVariantRepository, images repository.ProductImageRepository, rdb *redis.Client) *Usecase {
	return &Usecase{products: products, variants: variants, images: images, rdb: rdb}
}

// invalidateDetail deletes the public catalog cache entry for slug (FR-4.7,
// AC-4.d: edits must be visible without waiting for the TTL).
func (u *Usecase) invalidateDetail(ctx context.Context, slug string) {
	_ = cache.Delete(ctx, u.rdb, catalog.ProductDetailCacheKey(slug))
}

type ProductInput struct {
	CategoryID  int
	Name        string
	Description string
	Specs       map[string]any
}

func (u *Usecase) CreateProduct(ctx context.Context, supplierID uuid.UUID, in ProductInput) (*entity.Product, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("product: generate id: %w", err)
	}
	slug, err := u.uniqueSlug(ctx, in.Name, uuid.Nil)
	if err != nil {
		return nil, err
	}
	p := &entity.Product{
		ID:          id,
		SupplierID:  supplierID,
		CategoryID:  in.CategoryID,
		Name:        in.Name,
		Slug:        slug,
		Description: in.Description,
		Specs:       in.Specs,
		Status:      entity.ProductStatusDraft,
	}
	if err := u.products.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (u *Usecase) UpdateProduct(ctx context.Context, supplierID, id uuid.UUID, in ProductInput) (*entity.Product, error) {
	p, err := u.products.FindBySupplierAndID(ctx, supplierID, id)
	if err != nil {
		return nil, err
	}
	oldSlug := p.Slug
	if in.Name != p.Name {
		slug, err := u.uniqueSlug(ctx, in.Name, p.ID)
		if err != nil {
			return nil, err
		}
		p.Slug = slug
	}
	p.CategoryID = in.CategoryID
	p.Name = in.Name
	p.Description = in.Description
	p.Specs = in.Specs

	if err := u.products.Update(ctx, p); err != nil {
		return nil, err
	}
	u.invalidateDetail(ctx, oldSlug)
	if p.Slug != oldSlug {
		u.invalidateDetail(ctx, p.Slug)
	}
	return p, nil
}

func (u *Usecase) GetProduct(ctx context.Context, supplierID, id uuid.UUID) (*entity.Product, error) {
	return u.products.FindBySupplierAndID(ctx, supplierID, id)
}

func (u *Usecase) ListProducts(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.Product, int, error) {
	return u.products.ListBySupplier(ctx, supplierID, page, perPage)
}

// Publish moves a product from draft to active (FR-4.2). Requires at least
// one active variant and one image.
func (u *Usecase) Publish(ctx context.Context, supplierID, id uuid.UUID) (*entity.Product, error) {
	p, err := u.products.FindBySupplierAndID(ctx, supplierID, id)
	if err != nil {
		return nil, err
	}
	variants, err := u.variants.CountActiveByProductID(ctx, id)
	if err != nil {
		return nil, err
	}
	if variants == 0 {
		return nil, apperr.New("NO_ACTIVE_VARIANT", "Product needs at least one active variant to publish", 422)
	}
	images, err := u.images.CountByProductID(ctx, id)
	if err != nil {
		return nil, err
	}
	if images == 0 {
		return nil, apperr.New("NO_IMAGE", "Product needs at least one image to publish", 422)
	}
	p.Status = entity.ProductStatusActive
	if err := u.products.Update(ctx, p); err != nil {
		return nil, err
	}
	u.invalidateDetail(ctx, p.Slug)
	return p, nil
}

type VariantInput struct {
	SKU         string
	Name        string
	Unit        string
	Price       float64
	WeightGram  int
	LengthCM    int
	WidthCM     int
	HeightCM    int
	MinOrderQty int
	Stock       int
	IsActive    bool
}

func (u *Usecase) CreateVariant(ctx context.Context, supplierID, productID uuid.UUID, in VariantInput) (*entity.ProductVariant, error) {
	if _, err := u.products.FindBySupplierAndID(ctx, supplierID, productID); err != nil {
		return nil, err
	}
	if in.WeightGram <= 0 {
		return nil, apperr.New("VALIDATION_ERROR", "weight_gram must be greater than 0", 422)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("product: generate variant id: %w", err)
	}
	v := &entity.ProductVariant{
		ID:          id,
		ProductID:   productID,
		SupplierID:  supplierID,
		SKU:         in.SKU,
		Name:        in.Name,
		Unit:        in.Unit,
		Price:       in.Price,
		WeightGram:  in.WeightGram,
		LengthCM:    in.LengthCM,
		WidthCM:     in.WidthCM,
		HeightCM:    in.HeightCM,
		MinOrderQty: in.MinOrderQty,
		Stock:       in.Stock,
		IsActive:    in.IsActive,
	}
	if err := u.variants.Create(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (u *Usecase) UpdateVariant(ctx context.Context, supplierID, id uuid.UUID, in VariantInput) (*entity.ProductVariant, error) {
	v, err := u.variants.FindByIDAndSupplier(ctx, id, supplierID)
	if err != nil {
		return nil, err
	}
	if in.WeightGram <= 0 {
		return nil, apperr.New("VALIDATION_ERROR", "weight_gram must be greater than 0", 422)
	}
	v.SKU = in.SKU
	v.Name = in.Name
	v.Unit = in.Unit
	v.Price = in.Price
	v.WeightGram = in.WeightGram
	v.LengthCM = in.LengthCM
	v.WidthCM = in.WidthCM
	v.HeightCM = in.HeightCM
	v.MinOrderQty = in.MinOrderQty
	v.IsActive = in.IsActive

	if err := u.variants.Update(ctx, v); err != nil {
		return nil, err
	}
	return v, nil
}

func (u *Usecase) ListVariants(ctx context.Context, supplierID, productID uuid.UUID) ([]*entity.ProductVariant, error) {
	if _, err := u.products.FindBySupplierAndID(ctx, supplierID, productID); err != nil {
		return nil, err
	}
	return u.variants.ListByProductID(ctx, productID)
}

// AttachImage adds an image to a product (FR-4.4), capped at 8. The first
// image attached is automatically made primary.
func (u *Usecase) AttachImage(ctx context.Context, supplierID, productID uuid.UUID, url string, isPrimary bool) (*entity.ProductImage, error) {
	if _, err := u.products.FindBySupplierAndID(ctx, supplierID, productID); err != nil {
		return nil, err
	}
	count, err := u.images.CountByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if count >= maxImages {
		return nil, apperr.New("TOO_MANY_IMAGES", "Product can have at most 8 images", 422)
	}
	if count == 0 {
		isPrimary = true
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("product: generate image id: %w", err)
	}
	if isPrimary {
		if err := u.images.UnsetPrimary(ctx, productID); err != nil {
			return nil, err
		}
	}
	img := &entity.ProductImage{
		ID:        id,
		ProductID: productID,
		URL:       url,
		IsPrimary: isPrimary,
		SortOrder: count,
	}
	if err := u.images.Create(ctx, img); err != nil {
		return nil, err
	}
	return img, nil
}

// SetPrimaryImage marks the given image as the product's primary image,
// unsetting any previous primary.
func (u *Usecase) SetPrimaryImage(ctx context.Context, supplierID, productID, imageID uuid.UUID) error {
	if _, err := u.products.FindBySupplierAndID(ctx, supplierID, productID); err != nil {
		return err
	}
	img, err := u.images.FindByID(ctx, imageID)
	if err != nil {
		return err
	}
	if img.ProductID != productID {
		return errs.ErrNotFound
	}
	if err := u.images.UnsetPrimary(ctx, productID); err != nil {
		return err
	}
	return u.images.SetPrimary(ctx, imageID)
}

func (u *Usecase) DeleteImage(ctx context.Context, supplierID, productID, imageID uuid.UUID) error {
	if _, err := u.products.FindBySupplierAndID(ctx, supplierID, productID); err != nil {
		return err
	}
	img, err := u.images.FindByID(ctx, imageID)
	if err != nil {
		return err
	}
	if img.ProductID != productID {
		return errs.ErrNotFound
	}
	return u.images.Delete(ctx, imageID)
}

var slugNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(name string) string {
	s := slugNonAlnum.ReplaceAllString(strings.ToLower(name), "-")
	return strings.Trim(s, "-")
}

// uniqueSlug slugifies name and appends a short suffix on collision,
// excluding the product identified by selfID (uuid.Nil means "new product").
func (u *Usecase) uniqueSlug(ctx context.Context, name string, selfID uuid.UUID) (string, error) {
	base := slugify(name)
	slug := base
	for i := 2; ; i++ {
		existing, err := u.products.FindBySlug(ctx, slug)
		if errors.Is(err, errs.ErrNotFound) {
			return slug, nil
		}
		if err != nil {
			return "", err
		}
		if existing.ID == selfID {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}
