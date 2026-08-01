package catalog_test

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
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	catalogusecase "erdinhrmwn/bangunin/internal/usecase/catalog"
	"erdinhrmwn/bangunin/pkg/apperr"
)

func newUsecase(t *testing.T) (*catalogusecase.Usecase, *mocks.MockProductRepository, *mocks.MockSupplierRepository) {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not reachable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	products := mocks.NewMockProductRepository(t)
	suppliers := mocks.NewMockSupplierRepository(t)
	return catalogusecase.New(products, suppliers, rdb), products, suppliers
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

func TestSearch_PassesThroughToRepo(t *testing.T) {
	uc, products, _ := newUsecase(t)
	params := repository.SearchParams{Query: "semen"}
	products.EXPECT().Search(mock.Anything, params).Return([]*repository.ProductSummary{{Name: "Semen Portland"}}, "cursor1", nil)

	got, next, err := uc.Search(context.Background(), params)

	require.NoError(t, err)
	assert.Equal(t, "cursor1", next)
	assert.Len(t, got, 1)
}

func TestDetail_CacheMiss_FetchesAndCaches(t *testing.T) {
	uc, products, _ := newUsecase(t)
	require.NoError(t, redis.NewClient(&redis.Options{Addr: "localhost:6379"}).Del(context.Background(), "catalog:product:semen-portland").Err())
	products.EXPECT().FindBySlug(mock.Anything, "semen-portland").Return(&entity.Product{Slug: "semen-portland", Status: entity.ProductStatusActive}, nil)

	got, err := uc.Detail(context.Background(), "semen-portland")

	require.NoError(t, err)
	assert.Equal(t, "semen-portland", got.Slug)
}

func TestDetail_NotActive_NotFound(t *testing.T) {
	uc, products, _ := newUsecase(t)
	require.NoError(t, redis.NewClient(&redis.Options{Addr: "localhost:6379"}).Del(context.Background(), "catalog:product:draft-item").Err())
	products.EXPECT().FindBySlug(mock.Anything, "draft-item").Return(&entity.Product{Slug: "draft-item", Status: entity.ProductStatusDraft}, nil)

	_, err := uc.Detail(context.Background(), "draft-item")

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestSupplierStore_Success(t *testing.T) {
	uc, products, suppliers := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	suppliers.EXPECT().FindBySlug(mock.Anything, "toko-jaya").Return(&entity.Supplier{ID: supplierID, Slug: "toko-jaya", Status: entity.SupplierStatusApproved}, nil)
	products.EXPECT().ListBySupplierPublic(mock.Anything, supplierID, 1, 20).Return([]*entity.Product{{ID: uuid.Must(uuid.NewV7())}}, 1, nil)

	s, list, total, err := uc.SupplierStore(context.Background(), "toko-jaya", 1, 20)

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, list, 1)
	assert.Equal(t, "toko-jaya", s.Slug)
}

func TestSupplierStore_NotApproved_NotFound(t *testing.T) {
	uc, _, suppliers := newUsecase(t)
	suppliers.EXPECT().FindBySlug(mock.Anything, "pending-shop").Return(&entity.Supplier{Slug: "pending-shop", Status: entity.SupplierStatusPending}, nil)

	_, _, _, err := uc.SupplierStore(context.Background(), "pending-shop", 1, 20)

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestSupplierStore_NotFound(t *testing.T) {
	uc, _, suppliers := newUsecase(t)
	suppliers.EXPECT().FindBySlug(mock.Anything, "ghost-shop").Return(nil, errs.ErrNotFound)

	_, _, _, err := uc.SupplierStore(context.Background(), "ghost-shop", 1, 20)

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}
