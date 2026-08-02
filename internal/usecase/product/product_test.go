package product_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	"erdinhrmwn/bangunin/pkg/apperr"

	productusecase "erdinhrmwn/bangunin/internal/usecase/product"
)

func newUsecase(t *testing.T) (*productusecase.Usecase, *mocks.MockProductRepository, *mocks.MockProductVariantRepository, *mocks.MockProductImageRepository) {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not reachable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	products := mocks.NewMockProductRepository(t)
	variants := mocks.NewMockProductVariantRepository(t)
	images := mocks.NewMockProductImageRepository(t)
	return productusecase.New(products, variants, images, rdb), products, variants, images
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

func TestCreateProduct_Success(t *testing.T) {
	uc, products, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySlug(mock.Anything, "semen-portland").Return(nil, errs.ErrNotFound)
	products.EXPECT().Create(mock.Anything, mock.MatchedBy(func(p *entity.Product) bool {
		return p.Slug == "semen-portland" && p.SupplierID == supplierID && p.Status == entity.ProductStatusDraft
	})).Return(nil)

	got, err := uc.CreateProduct(context.Background(), supplierID, productusecase.ProductInput{Name: "Semen Portland"})

	require.NoError(t, err)
	assert.Equal(t, "semen-portland", got.Slug)
}

func TestPublish_NoActiveVariant_Rejected(t *testing.T) {
	uc, products, variants, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)
	variants.EXPECT().CountActiveByProductID(mock.Anything, productID).Return(0, nil)

	_, err := uc.Publish(context.Background(), supplierID, productID)

	assert.Equal(t, "NO_ACTIVE_VARIANT", appErrCode(t, err))
}

func TestPublish_NoImage_Rejected(t *testing.T) {
	uc, products, variants, images := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)
	variants.EXPECT().CountActiveByProductID(mock.Anything, productID).Return(1, nil)
	images.EXPECT().CountByProductID(mock.Anything, productID).Return(0, nil)

	_, err := uc.Publish(context.Background(), supplierID, productID)

	assert.Equal(t, "NO_IMAGE", appErrCode(t, err))
}

func TestPublish_Success(t *testing.T) {
	uc, products, variants, images := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID, Status: entity.ProductStatusDraft}, nil)
	variants.EXPECT().CountActiveByProductID(mock.Anything, productID).Return(1, nil)
	images.EXPECT().CountByProductID(mock.Anything, productID).Return(1, nil)
	products.EXPECT().Update(mock.Anything, mock.MatchedBy(func(p *entity.Product) bool {
		return p.Status == entity.ProductStatusActive
	})).Return(nil)

	got, err := uc.Publish(context.Background(), supplierID, productID)

	require.NoError(t, err)
	assert.Equal(t, entity.ProductStatusActive, got.Status)
}

func TestPublish_NotOwner_NotFound(t *testing.T) {
	uc, products, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(nil, errs.ErrNotFound)

	_, err := uc.Publish(context.Background(), supplierID, productID)

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestCreateVariant_ZeroWeight_Rejected(t *testing.T) {
	uc, products, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)

	_, err := uc.CreateVariant(context.Background(), supplierID, productID, productusecase.VariantInput{SKU: "SKU-1", WeightGram: 0})

	assert.Equal(t, "VALIDATION_ERROR", appErrCode(t, err))
}

func TestCreateVariant_DuplicateSKU_Conflict(t *testing.T) {
	uc, products, variants, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)
	variants.EXPECT().Create(mock.Anything, mock.Anything).Return(errs.ErrConflict)

	_, err := uc.CreateVariant(context.Background(), supplierID, productID, productusecase.VariantInput{SKU: "SKU-1", WeightGram: 500})

	assert.Equal(t, "CONFLICT", appErrCode(t, err))
}

func TestAttachImage_FirstImage_AutoPrimary(t *testing.T) {
	uc, products, _, images := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)
	images.EXPECT().CountByProductID(mock.Anything, productID).Return(0, nil)
	images.EXPECT().UnsetPrimary(mock.Anything, productID).Return(nil)
	images.EXPECT().Create(mock.Anything, mock.MatchedBy(func(img *entity.ProductImage) bool {
		return img.IsPrimary
	})).Return(nil)

	got, err := uc.AttachImage(context.Background(), supplierID, productID, "https://example.com/a.jpg", false)

	require.NoError(t, err)
	assert.True(t, got.IsPrimary)
}

func TestAttachImage_TooManyImages_Rejected(t *testing.T) {
	uc, products, _, images := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)
	images.EXPECT().CountByProductID(mock.Anything, productID).Return(8, nil)

	_, err := uc.AttachImage(context.Background(), supplierID, productID, "https://example.com/a.jpg", false)

	assert.Equal(t, "TOO_MANY_IMAGES", appErrCode(t, err))
}

func TestDeleteImage_WrongProduct_NotFound(t *testing.T) {
	uc, products, _, images := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	otherProductID := uuid.Must(uuid.NewV7())
	imageID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)
	images.EXPECT().FindByID(mock.Anything, imageID).Return(&entity.ProductImage{ID: imageID, ProductID: otherProductID}, nil)

	err := uc.DeleteImage(context.Background(), supplierID, productID, imageID)

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestDeleteImage_Success(t *testing.T) {
	uc, products, _, images := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	imageID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)
	images.EXPECT().FindByID(mock.Anything, imageID).Return(&entity.ProductImage{ID: imageID, ProductID: productID}, nil)
	images.EXPECT().Delete(mock.Anything, imageID).Return(nil)

	err := uc.DeleteImage(context.Background(), supplierID, productID, imageID)

	assert.NoError(t, err)
}

func TestDeleteImage_NotOwner_NotFound(t *testing.T) {
	uc, products, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	imageID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(nil, errs.ErrNotFound)

	err := uc.DeleteImage(context.Background(), supplierID, productID, imageID)

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestGetProduct_Success(t *testing.T) {
	uc, products, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)

	got, err := uc.GetProduct(context.Background(), supplierID, productID)

	require.NoError(t, err)
	assert.Equal(t, productID, got.ID)
}

func TestGetProduct_NotFound(t *testing.T) {
	uc, products, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(nil, errs.ErrNotFound)

	_, err := uc.GetProduct(context.Background(), supplierID, productID)

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestListProducts_Success(t *testing.T) {
	uc, products, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	products.EXPECT().ListBySupplier(mock.Anything, supplierID, 1, 20).
		Return([]*entity.Product{{ID: uuid.Must(uuid.NewV7())}}, 1, nil)

	got, total, err := uc.ListProducts(context.Background(), supplierID, 1, 20)

	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, 1, total)
}

func TestUpdateVariant_ZeroWeight_Rejected(t *testing.T) {
	uc, _, variants, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variants.EXPECT().FindByIDAndSupplier(mock.Anything, variantID, supplierID).
		Return(&entity.ProductVariant{ID: variantID, SupplierID: supplierID}, nil)

	_, err := uc.UpdateVariant(context.Background(), supplierID, variantID, productusecase.VariantInput{WeightGram: 0})

	assert.Equal(t, "VALIDATION_ERROR", appErrCode(t, err))
}

func TestUpdateVariant_NotOwner_NotFound(t *testing.T) {
	uc, _, variants, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variants.EXPECT().FindByIDAndSupplier(mock.Anything, variantID, supplierID).Return(nil, errs.ErrNotFound)

	_, err := uc.UpdateVariant(context.Background(), supplierID, variantID, productusecase.VariantInput{WeightGram: 500})

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestUpdateVariant_Success_DefaultsMinOrderQty(t *testing.T) {
	uc, _, variants, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variants.EXPECT().FindByIDAndSupplier(mock.Anything, variantID, supplierID).
		Return(&entity.ProductVariant{ID: variantID, SupplierID: supplierID}, nil)
	variants.EXPECT().Update(mock.Anything, mock.MatchedBy(func(v *entity.ProductVariant) bool {
		return v.MinOrderQty == 1 && v.SKU == "SKU-2"
	})).Return(nil)

	got, err := uc.UpdateVariant(context.Background(), supplierID, variantID, productusecase.VariantInput{SKU: "SKU-2", WeightGram: 500, MinOrderQty: 0})

	require.NoError(t, err)
	assert.Equal(t, 1, got.MinOrderQty)
}

func TestListVariants_Success(t *testing.T) {
	uc, products, variants, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)
	variants.EXPECT().ListByProductID(mock.Anything, productID).Return([]*entity.ProductVariant{{ID: uuid.Must(uuid.NewV7())}}, nil)

	got, err := uc.ListVariants(context.Background(), supplierID, productID)

	require.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestListVariants_NotOwner_NotFound(t *testing.T) {
	uc, products, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(nil, errs.ErrNotFound)

	_, err := uc.ListVariants(context.Background(), supplierID, productID)

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestSetPrimaryImage_Success(t *testing.T) {
	uc, products, _, images := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	imageID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)
	images.EXPECT().FindByID(mock.Anything, imageID).Return(&entity.ProductImage{ID: imageID, ProductID: productID}, nil)
	images.EXPECT().UnsetPrimary(mock.Anything, productID).Return(nil)
	images.EXPECT().SetPrimary(mock.Anything, imageID).Return(nil)

	err := uc.SetPrimaryImage(context.Background(), supplierID, productID, imageID)

	assert.NoError(t, err)
}

func TestSetPrimaryImage_WrongProduct_NotFound(t *testing.T) {
	uc, products, _, images := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	otherProductID := uuid.Must(uuid.NewV7())
	imageID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(&entity.Product{ID: productID, SupplierID: supplierID}, nil)
	images.EXPECT().FindByID(mock.Anything, imageID).Return(&entity.ProductImage{ID: imageID, ProductID: otherProductID}, nil)

	err := uc.SetPrimaryImage(context.Background(), supplierID, productID, imageID)

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestSetPrimaryImage_NotOwner_NotFound(t *testing.T) {
	uc, products, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	imageID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).Return(nil, errs.ErrNotFound)

	err := uc.SetPrimaryImage(context.Background(), supplierID, productID, imageID)

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestUniqueSlug_CollisionThenFree(t *testing.T) {
	uc, products, _, _ := newUsecase(t)
	selfID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySlug(mock.Anything, "batu-bata").
		Return(&entity.Product{ID: uuid.Must(uuid.NewV7())}, nil)
	products.EXPECT().Create(mock.Anything, mock.MatchedBy(func(p *entity.Product) bool {
		return p.Slug == "batu-bata-2"
	})).Return(nil)
	products.EXPECT().FindBySlug(mock.Anything, "batu-bata-2").Return(nil, errs.ErrNotFound)

	got, err := uc.CreateProduct(context.Background(), selfID, productusecase.ProductInput{Name: "Batu Bata"})

	require.NoError(t, err)
	assert.Equal(t, "batu-bata-2", got.Slug)
}

func TestUpdateProduct_SlugUnchangedWhenNameSame(t *testing.T) {
	uc, products, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	products.EXPECT().FindBySupplierAndID(mock.Anything, supplierID, productID).
		Return(&entity.Product{ID: productID, SupplierID: supplierID, Name: "Semen", Slug: "semen"}, nil)
	products.EXPECT().Update(mock.Anything, mock.MatchedBy(func(p *entity.Product) bool {
		return p.Slug == "semen"
	})).Return(nil)

	got, err := uc.UpdateProduct(context.Background(), supplierID, productID, productusecase.ProductInput{Name: "Semen"})

	require.NoError(t, err)
	assert.Equal(t, "semen", got.Slug)
}
