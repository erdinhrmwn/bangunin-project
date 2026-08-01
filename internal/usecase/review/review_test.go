package review_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	reviewusecase "erdinhrmwn/bangunin/internal/usecase/review"
	"erdinhrmwn/bangunin/pkg/apperr"
)

type deps struct {
	reviews  *mocks.MockReviewRepository
	orders   *mocks.MockOrderRepository
	variants *mocks.MockProductVariantRepository
	products *mocks.MockProductRepository
}

func newUsecase(t *testing.T) (*reviewusecase.Usecase, *deps) {
	t.Helper()
	d := &deps{
		reviews:  mocks.NewMockReviewRepository(t),
		orders:   mocks.NewMockOrderRepository(t),
		variants: mocks.NewMockProductVariantRepository(t),
		products: mocks.NewMockProductRepository(t),
	}
	uc := reviewusecase.New(d.reviews, d.orders, d.variants, d.products)
	return uc, d
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

func TestCreate_NotOwner_Returns403(t *testing.T) {
	uc, d := newUsecase(t)
	userID, otherUserID, itemID, orderID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindItemByID(mock.Anything, itemID).Return(&entity.OrderItem{ID: itemID, OrderID: orderID}, nil)
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, UserID: otherUserID, Status: entity.OrderStatusCompleted}, nil)

	_, err := uc.Create(context.Background(), userID, itemID, 5, "great")

	require.Equal(t, "FORBIDDEN", appErrCode(t, err))
}

func TestCreate_OrderNotCompleted_Returns422(t *testing.T) {
	uc, d := newUsecase(t)
	userID, itemID, orderID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindItemByID(mock.Anything, itemID).Return(&entity.OrderItem{ID: itemID, OrderID: orderID}, nil)
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, UserID: userID, Status: entity.OrderStatusShipped}, nil)

	_, err := uc.Create(context.Background(), userID, itemID, 5, "great")

	require.Equal(t, "VALIDATION_ERROR", appErrCode(t, err))
}

func TestCreate_AlreadyReviewed_Returns409(t *testing.T) {
	uc, d := newUsecase(t)
	userID, itemID, orderID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindItemByID(mock.Anything, itemID).Return(&entity.OrderItem{ID: itemID, OrderID: orderID}, nil)
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, UserID: userID, Status: entity.OrderStatusCompleted}, nil)
	d.reviews.EXPECT().ExistsByOrderItemID(mock.Anything, itemID).Return(true, nil)

	_, err := uc.Create(context.Background(), userID, itemID, 5, "great")

	require.Equal(t, "CONFLICT", appErrCode(t, err))
}

func TestCreate_Success_RecomputesRating(t *testing.T) {
	uc, d := newUsecase(t)
	userID, itemID, orderID, variantID, productID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.orders.EXPECT().FindItemByID(mock.Anything, itemID).Return(&entity.OrderItem{ID: itemID, OrderID: orderID, VariantID: variantID}, nil)
	d.orders.EXPECT().FindByID(mock.Anything, orderID).Return(&entity.Order{ID: orderID, UserID: userID, Status: entity.OrderStatusCompleted}, nil)
	d.reviews.EXPECT().ExistsByOrderItemID(mock.Anything, itemID).Return(false, nil)
	d.variants.EXPECT().FindByID(mock.Anything, variantID).Return(&entity.ProductVariant{ID: variantID, ProductID: productID}, nil)
	d.reviews.EXPECT().Create(mock.Anything, mock.MatchedBy(func(r *entity.Review) bool {
		return r.OrderItemID == itemID && r.ProductID == productID && r.UserID == userID && r.Rating == 5 && r.Comment == "great"
	})).Return(nil)
	d.reviews.EXPECT().AverageRating(mock.Anything, productID).Return(4.5, 2, nil)
	d.products.EXPECT().UpdateRating(mock.Anything, productID, 4.5, 2).Return(nil)

	rv, err := uc.Create(context.Background(), userID, itemID, 5, "great")

	require.NoError(t, err)
	require.Equal(t, productID, rv.ProductID)
}

func TestListByProduct_Passthrough(t *testing.T) {
	uc, d := newUsecase(t)
	productID := uuid.Must(uuid.NewV7())
	d.reviews.EXPECT().ListByProduct(mock.Anything, productID, 5, "cursor", 20).Return([]*entity.Review{{ID: uuid.Must(uuid.NewV7())}}, "next", nil)

	revs, next, err := uc.ListByProduct(context.Background(), productID, 5, "cursor", 20)

	require.NoError(t, err)
	require.Len(t, revs, 1)
	require.Equal(t, "next", next)
}

func TestListByUser_Passthrough(t *testing.T) {
	uc, d := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	d.reviews.EXPECT().ListByUser(mock.Anything, userID, 1, 10).Return([]*entity.Review{{ID: uuid.Must(uuid.NewV7())}}, 1, nil)

	revs, total, err := uc.ListByUser(context.Background(), userID, 1, 10)

	require.NoError(t, err)
	require.Len(t, revs, 1)
	require.Equal(t, 1, total)
}
