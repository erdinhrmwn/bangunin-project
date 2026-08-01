package inventory_test

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
	inventoryusecase "erdinhrmwn/bangunin/internal/usecase/inventory"
	"erdinhrmwn/bangunin/pkg/apperr"
)

func newUsecase(t *testing.T) (*inventoryusecase.Usecase, *mocks.MockProductVariantRepository, *mocks.MockStockMovementRepository, *mocks.MockSupplierRepository, *mocks.MockNotificationRepository) {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not reachable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	variants := mocks.NewMockProductVariantRepository(t)
	movements := mocks.NewMockStockMovementRepository(t)
	suppliers := mocks.NewMockSupplierRepository(t)
	notify := mocks.NewMockNotificationRepository(t)
	return inventoryusecase.New(variants, movements, suppliers, notify, rdb, 5), variants, movements, suppliers, notify
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

func TestAdjustStock_Success(t *testing.T) {
	uc, variants, _, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variants.EXPECT().FindByIDAndSupplier(mock.Anything, variantID, supplierID).Return(&entity.ProductVariant{ID: variantID, SupplierID: supplierID, Stock: 5}, nil)
	variants.EXPECT().AdjustStockWithMovement(mock.Anything, variantID, 10, mock.MatchedBy(func(m *entity.StockMovement) bool {
		return m.VariantID == variantID && m.Type == entity.MovementTypeAdjustment && m.Qty == 10 && m.RefType == entity.RefTypeManual
	})).Return(15, nil)

	got, err := uc.AdjustStock(context.Background(), supplierID, variantID, 10, "restock")

	require.NoError(t, err)
	assert.Equal(t, 15, got.Stock)
}

func TestAdjustStock_NegativeResult_Conflict(t *testing.T) {
	uc, variants, _, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variants.EXPECT().FindByIDAndSupplier(mock.Anything, variantID, supplierID).Return(&entity.ProductVariant{ID: variantID, SupplierID: supplierID, Stock: 5}, nil)
	variants.EXPECT().AdjustStockWithMovement(mock.Anything, variantID, -10, mock.Anything).Return(0, errs.ErrConflict)

	_, err := uc.AdjustStock(context.Background(), supplierID, variantID, -10, "")

	assert.Equal(t, "CONFLICT", appErrCode(t, err))
}

func TestAdjustStock_NotOwner_NotFound(t *testing.T) {
	uc, variants, _, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variants.EXPECT().FindByIDAndSupplier(mock.Anything, variantID, supplierID).Return(nil, errs.ErrNotFound)

	_, err := uc.AdjustStock(context.Background(), supplierID, variantID, 1, "")

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestListMovements_Success(t *testing.T) {
	uc, variants, movements, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variants.EXPECT().FindByIDAndSupplier(mock.Anything, variantID, supplierID).Return(&entity.ProductVariant{ID: variantID, SupplierID: supplierID}, nil)
	list := []*entity.StockMovement{{ID: uuid.Must(uuid.NewV7()), VariantID: variantID}}
	movements.EXPECT().ListByVariantID(mock.Anything, variantID, 1, 20).Return(list, 1, nil)

	got, total, err := uc.ListMovements(context.Background(), supplierID, variantID, 1, 20)

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, got, 1)
}

func TestListMovements_NotOwner_NotFound(t *testing.T) {
	uc, variants, _, _, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variants.EXPECT().FindByIDAndSupplier(mock.Anything, variantID, supplierID).Return(nil, errs.ErrNotFound)

	_, _, err := uc.ListMovements(context.Background(), supplierID, variantID, 1, 20)

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestCheckLowStock_NotifiesThenDedupesWithin24h(t *testing.T) {
	uc, variants, _, suppliers, notify := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variant := &entity.ProductVariant{ID: variantID, SupplierID: supplierID, SKU: "SKU-1", Name: "Semen 40kg", Stock: 2}
	userID := uuid.Must(uuid.NewV7())

	variants.EXPECT().ListBelowThreshold(mock.Anything, 5).Return([]*entity.ProductVariant{variant}, nil).Twice()
	suppliers.EXPECT().FindByID(mock.Anything, supplierID).Return(&entity.Supplier{ID: supplierID, UserID: userID}, nil).Once()
	notify.EXPECT().Create(mock.Anything, mock.MatchedBy(func(n *entity.Notification) bool {
		return n.UserID == userID && n.Type == entity.NotificationTypeLowStock
	})).Return(nil).Once()

	require.NoError(t, uc.CheckLowStock(context.Background()))
	// Second run within 24h must not re-notify (Redis SETNX dedupe).
	require.NoError(t, uc.CheckLowStock(context.Background()))
}
