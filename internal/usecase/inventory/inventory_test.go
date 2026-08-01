package inventory_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	inventoryusecase "erdinhrmwn/bangunin/internal/usecase/inventory"
	"erdinhrmwn/bangunin/pkg/apperr"
)

func newUsecase(t *testing.T) (*inventoryusecase.Usecase, *mocks.MockProductVariantRepository, *mocks.MockStockMovementRepository) {
	t.Helper()
	variants := mocks.NewMockProductVariantRepository(t)
	movements := mocks.NewMockStockMovementRepository(t)
	return inventoryusecase.New(variants, movements), variants, movements
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

func TestAdjustStock_Success(t *testing.T) {
	uc, variants, _ := newUsecase(t)
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
	uc, variants, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variants.EXPECT().FindByIDAndSupplier(mock.Anything, variantID, supplierID).Return(&entity.ProductVariant{ID: variantID, SupplierID: supplierID, Stock: 5}, nil)
	variants.EXPECT().AdjustStockWithMovement(mock.Anything, variantID, -10, mock.Anything).Return(0, errs.ErrConflict)

	_, err := uc.AdjustStock(context.Background(), supplierID, variantID, -10, "")

	assert.Equal(t, "CONFLICT", appErrCode(t, err))
}

func TestAdjustStock_NotOwner_NotFound(t *testing.T) {
	uc, variants, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variants.EXPECT().FindByIDAndSupplier(mock.Anything, variantID, supplierID).Return(nil, errs.ErrNotFound)

	_, err := uc.AdjustStock(context.Background(), supplierID, variantID, 1, "")

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}

func TestListMovements_Success(t *testing.T) {
	uc, variants, movements := newUsecase(t)
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
	uc, variants, _ := newUsecase(t)
	supplierID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	variants.EXPECT().FindByIDAndSupplier(mock.Anything, variantID, supplierID).Return(nil, errs.ErrNotFound)

	_, _, err := uc.ListMovements(context.Background(), supplierID, variantID, 1, 20)

	assert.Equal(t, "NOT_FOUND", appErrCode(t, err))
}
