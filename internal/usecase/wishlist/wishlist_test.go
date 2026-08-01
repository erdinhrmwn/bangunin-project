package wishlist_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	wishlistusecase "erdinhrmwn/bangunin/internal/usecase/wishlist"
)

type deps struct {
	wishlists *mocks.MockWishlistRepository
	products  *mocks.MockProductRepository
}

func newUsecase(t *testing.T) (*wishlistusecase.Usecase, *deps) {
	t.Helper()
	d := &deps{
		wishlists: mocks.NewMockWishlistRepository(t),
		products:  mocks.NewMockProductRepository(t),
	}
	uc := wishlistusecase.New(d.wishlists, d.products)
	return uc, d
}

func TestAdd_ProductNotFound_ReturnsError(t *testing.T) {
	uc, d := newUsecase(t)
	userID, productID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.products.EXPECT().FindByID(mock.Anything, productID).Return(nil, errs.ErrNotFound)

	err := uc.Add(context.Background(), userID, productID)

	require.ErrorIs(t, err, errs.ErrNotFound)
}

func TestAdd_Success(t *testing.T) {
	uc, d := newUsecase(t)
	userID, productID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.products.EXPECT().FindByID(mock.Anything, productID).Return(&entity.Product{ID: productID}, nil)
	d.wishlists.EXPECT().Create(mock.Anything, mock.MatchedBy(func(w *entity.Wishlist) bool {
		return w.UserID == userID && w.ProductID == productID
	})).Return(nil)

	err := uc.Add(context.Background(), userID, productID)

	require.NoError(t, err)
}

func TestRemove_DelegatesToRepo(t *testing.T) {
	uc, d := newUsecase(t)
	userID, productID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.wishlists.EXPECT().Delete(mock.Anything, userID, productID).Return(nil)

	err := uc.Remove(context.Background(), userID, productID)

	require.NoError(t, err)
}

func TestIsWishlisted_DelegatesToRepo(t *testing.T) {
	uc, d := newUsecase(t)
	userID, productID := uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())
	d.wishlists.EXPECT().ExistsByUserAndProduct(mock.Anything, userID, productID).Return(true, nil)

	ok, err := uc.IsWishlisted(context.Background(), userID, productID)

	require.NoError(t, err)
	require.True(t, ok)
}

func TestListMine_DelegatesToRepo(t *testing.T) {
	uc, d := newUsecase(t)
	userID := uuid.Must(uuid.NewV7())
	want := []*entity.Wishlist{{ID: uuid.Must(uuid.NewV7()), UserID: userID}}
	d.wishlists.EXPECT().ListByUser(mock.Anything, userID, 1, 20).Return(want, 1, nil)

	got, total, err := uc.ListMine(context.Background(), userID, 1, 20)

	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, want, got)
}
