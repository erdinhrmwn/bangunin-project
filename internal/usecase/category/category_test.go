package category_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
	"erdinhrmwn/bangunin/internal/domain/repository/mocks"
	"erdinhrmwn/bangunin/pkg/apperr"

	categoryusecase "erdinhrmwn/bangunin/internal/usecase/category"
)

func newUsecase(t *testing.T) (*categoryusecase.Usecase, *mocks.MockCategoryRepository) {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not reachable: %v", err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	require.NoError(t, rdb.Del(context.Background(), "category:tree").Err())

	categories := mocks.NewMockCategoryRepository(t)
	return categoryusecase.New(categories, rdb), categories
}

func appErrCode(t *testing.T, err error) string {
	t.Helper()
	return apperr.From(err).Code
}

func TestCreate_Success(t *testing.T) {
	uc, categories := newUsecase(t)
	categories.EXPECT().FindBySlug(mock.Anything, "semen").Return(nil, errs.ErrNotFound)
	categories.EXPECT().Create(mock.Anything, mock.MatchedBy(func(c *entity.Category) bool {
		return c.Slug == "semen" && c.Name == "Semen"
	})).RunAndReturn(func(_ context.Context, c *entity.Category) error {
		c.ID = 1
		return nil
	})

	got, err := uc.Create(context.Background(), categoryusecase.Input{Name: "Semen", IsActive: true})

	require.NoError(t, err)
	assert.Equal(t, "semen", got.Slug)
}

func TestCreate_ParentNotFound(t *testing.T) {
	uc, categories := newUsecase(t)
	parentID := 99
	categories.EXPECT().FindByID(mock.Anything, parentID).Return(nil, errs.ErrNotFound)

	_, err := uc.Create(context.Background(), categoryusecase.Input{Name: "Semen", ParentID: &parentID})

	assert.Equal(t, "PARENT_NOT_FOUND", appErrCode(t, err))
}

func TestCreate_MaxDepthExceeded(t *testing.T) {
	uc, categories := newUsecase(t)
	// chain: id 3 (requested parent) -> id 2 -> id 1 (root); adding a child
	// under 3 would create a 4th level, exceeding maxDepth=3.
	l1 := 1
	l2 := 2
	l3 := 3
	categories.EXPECT().FindByID(mock.Anything, 3).Return(&entity.Category{ID: 3, ParentID: &l2}, nil)
	categories.EXPECT().FindByID(mock.Anything, 2).Return(&entity.Category{ID: 2, ParentID: &l1}, nil)
	categories.EXPECT().FindByID(mock.Anything, 1).Return(&entity.Category{ID: 1, ParentID: nil}, nil)

	_, err := uc.Create(context.Background(), categoryusecase.Input{Name: "Too Deep", ParentID: &l3})

	assert.Equal(t, "MAX_DEPTH", appErrCode(t, err))
}

func TestDelete_HasChildren_Conflict(t *testing.T) {
	uc, categories := newUsecase(t)
	categories.EXPECT().FindByID(mock.Anything, 1).Return(&entity.Category{ID: 1}, nil)
	categories.EXPECT().CountChildren(mock.Anything, 1).Return(2, nil)

	err := uc.Delete(context.Background(), 1)

	assert.Equal(t, "HAS_CHILDREN", appErrCode(t, err))
}

func TestDelete_HasProducts_Conflict(t *testing.T) {
	uc, categories := newUsecase(t)
	categories.EXPECT().FindByID(mock.Anything, 1).Return(&entity.Category{ID: 1}, nil)
	categories.EXPECT().CountChildren(mock.Anything, 1).Return(0, nil)
	categories.EXPECT().CountProducts(mock.Anything, 1).Return(3, nil)

	err := uc.Delete(context.Background(), 1)

	assert.Equal(t, "HAS_PRODUCTS", appErrCode(t, err))
}

func TestDelete_Success(t *testing.T) {
	uc, categories := newUsecase(t)
	categories.EXPECT().FindByID(mock.Anything, 1).Return(&entity.Category{ID: 1}, nil)
	categories.EXPECT().CountChildren(mock.Anything, 1).Return(0, nil)
	categories.EXPECT().CountProducts(mock.Anything, 1).Return(0, nil)
	categories.EXPECT().Delete(mock.Anything, 1).Return(nil)

	err := uc.Delete(context.Background(), 1)

	assert.NoError(t, err)
}

func TestTree_CachesResult(t *testing.T) {
	uc, categories := newUsecase(t)
	list := []*entity.Category{{ID: 1, Name: "Semen", Slug: "semen"}}
	categories.EXPECT().ListTree(mock.Anything).Return(list, nil).Once()

	got1, err := uc.Tree(context.Background())
	require.NoError(t, err)
	assert.Len(t, got1, 1)

	// Second call must hit the cache, not the repository (mock has no
	// second ListTree expectation set, so a call here would fail the test).
	got2, err := uc.Tree(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "semen", got2[0].Slug)
}

func TestUpdate_SelfAsParent_Rejected(t *testing.T) {
	uc, categories := newUsecase(t)
	categories.EXPECT().FindByID(mock.Anything, 1).Return(&entity.Category{ID: 1, Name: "Semen"}, nil)

	selfID := 1
	_, err := uc.Update(context.Background(), 1, categoryusecase.Input{Name: "Semen", ParentID: &selfID})

	assert.Equal(t, "INVALID_PARENT", appErrCode(t, err))
}
