package cart_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	"erdinhrmwn/bangunin/pkg/apperr"

	cartusecase "erdinhrmwn/bangunin/internal/usecase/cart"
)

func newUsecase(t *testing.T) (*cartusecase.Usecase, *mocks.MockCartRepository, *mocks.MockProductVariantRepository, *mocks.MockProductRepository) {
	t.Helper()
	carts := mocks.NewMockCartRepository(t)
	variants := mocks.NewMockProductVariantRepository(t)
	products := mocks.NewMockProductRepository(t)
	return cartusecase.New(carts, variants, products), carts, variants, products
}

func TestAdd_Success(t *testing.T) {
	uc, carts, variants, _ := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	cartID := uuid.Must(uuid.NewV7())

	variants.EXPECT().FindByID(mock.Anything, variantID).Return(&entity.ProductVariant{ID: variantID}, nil)
	carts.EXPECT().FindOrCreateByUserID(mock.Anything, userID).Return(&entity.Cart{ID: cartID, UserID: userID}, nil)
	carts.EXPECT().UpsertItem(mock.Anything, mock.MatchedBy(func(i *entity.CartItem) bool {
		return i.CartID == cartID && i.VariantID == variantID && i.Qty == 2
	})).Return(nil)
	carts.EXPECT().Touch(mock.Anything, cartID).Return(nil)

	require.NoError(t, uc.Add(context.Background(), userID, variantID, 2))
}

func TestAdd_VariantNotFound(t *testing.T) {
	uc, _, variants, _ := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())

	variants.EXPECT().FindByID(mock.Anything, variantID).Return(nil, errs.ErrNotFound)

	err := uc.Add(context.Background(), userID, variantID, 1)
	require.Error(t, err)
	require.Equal(t, "NOT_FOUND", apperr.From(err).Code)
}

func TestRemove_ForbiddenWhenNotOwner(t *testing.T) {
	uc, carts, _, _ := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	itemID := uuid.Must(uuid.NewV7())
	otherCartID := uuid.Must(uuid.NewV7())
	myCartID := uuid.Must(uuid.NewV7())

	carts.EXPECT().FindItemByID(mock.Anything, itemID).Return(&entity.CartItem{ID: itemID, CartID: otherCartID}, nil)
	carts.EXPECT().FindOrCreateByUserID(mock.Anything, userID).Return(&entity.Cart{ID: myCartID, UserID: userID}, nil)

	err := uc.Remove(context.Background(), userID, itemID)
	require.Error(t, err)
	require.Equal(t, "FORBIDDEN", apperr.From(err).Code)
}

func TestGet_ComputesFlags(t *testing.T) {
	uc, carts, variants, products := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	cartID := uuid.Must(uuid.NewV7())
	variantID := uuid.Must(uuid.NewV7())
	productID := uuid.Must(uuid.NewV7())
	itemID := uuid.Must(uuid.NewV7())

	carts.EXPECT().FindOrCreateByUserID(mock.Anything, userID).Return(&entity.Cart{ID: cartID, UserID: userID}, nil)
	carts.EXPECT().ListItemsByCartID(mock.Anything, cartID).Return([]*entity.CartItem{
		{ID: itemID, CartID: cartID, VariantID: variantID, Qty: 5},
	}, nil)
	variants.EXPECT().FindByID(mock.Anything, variantID).Return(&entity.ProductVariant{
		ID: variantID, ProductID: productID, Stock: 2, MinOrderQty: 10, IsActive: true,
	}, nil)
	products.EXPECT().FindByID(mock.Anything, productID).Return(&entity.Product{
		ID: productID, Status: entity.ProductStatusInactive,
	}, nil)

	items, err := uc.Get(context.Background(), userID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.True(t, items[0].OutOfStock)
	require.True(t, items[0].BelowMinOrder)
	require.True(t, items[0].ProductInactive)
}
