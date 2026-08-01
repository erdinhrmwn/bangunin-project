package banner_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	bannerusecase "erdinhrmwn/bangunin/internal/usecase/banner"
)

func newUsecase(t *testing.T) (*bannerusecase.Usecase, *mocks.MockBannerRepository) {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not reachable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	require.NoError(t, rdb.Del(context.Background(), "banner:active").Err())

	banners := mocks.NewMockBannerRepository(t)
	return bannerusecase.New(banners, rdb), banners
}

func TestCreate_Success(t *testing.T) {
	uc, banners := newUsecase(t)
	in := bannerusecase.Input{
		ImageURL: "https://example.com/a.png",
		StartsAt: time.Now(),
		EndsAt:   time.Now().Add(time.Hour),
		IsActive: true,
	}
	banners.EXPECT().Create(mock.Anything, mock.MatchedBy(func(b *entity.Banner) bool {
		return b.ImageURL == in.ImageURL
	})).Return(nil)

	b, err := uc.Create(context.Background(), in)

	require.NoError(t, err)
	require.Equal(t, in.ImageURL, b.ImageURL)
}

func TestUpdate_NotFound_ReturnsError(t *testing.T) {
	uc, banners := newUsecase(t)
	id := uuid.Must(uuid.NewV7())
	banners.EXPECT().FindByID(mock.Anything, id).Return(nil, errs.ErrNotFound)

	_, err := uc.Update(context.Background(), id, bannerusecase.Input{})

	require.ErrorIs(t, err, errs.ErrNotFound)
}

func TestListActive_CachesResult(t *testing.T) {
	uc, banners := newUsecase(t)
	want := []*entity.Banner{{ID: uuid.Must(uuid.NewV7())}}
	banners.EXPECT().ListActive(mock.Anything).Return(want, nil).Once()

	got1, err := uc.ListActive(context.Background())
	require.NoError(t, err)
	require.Equal(t, want, got1)

	got2, err := uc.ListActive(context.Background())
	require.NoError(t, err)
	require.Equal(t, want, got2)
}
