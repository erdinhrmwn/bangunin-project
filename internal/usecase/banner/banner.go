// Package banner implements admin CRUD and the public active-banner list
// (FR-7.2), cached for 5 minutes and invalidated on any mutation.
package banner

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/pkg/cache"
)

const (
	activeCacheKey = "banner:active"
	activeCacheTTL = 5 * time.Minute
)

type Usecase struct {
	banners repository.BannerRepository
	rdb     *redis.Client
}

func New(banners repository.BannerRepository, rdb *redis.Client) *Usecase {
	return &Usecase{banners: banners, rdb: rdb}
}

type Input struct {
	ImageURL  string
	Link      string
	StartsAt  time.Time
	EndsAt    time.Time
	SortOrder int
	IsActive  bool
}

func (u *Usecase) Create(ctx context.Context, in Input) (*entity.Banner, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	b := &entity.Banner{
		ID:        id,
		ImageURL:  in.ImageURL,
		Link:      in.Link,
		StartsAt:  in.StartsAt,
		EndsAt:    in.EndsAt,
		SortOrder: in.SortOrder,
		IsActive:  in.IsActive,
	}
	if err := u.banners.Create(ctx, b); err != nil {
		return nil, err
	}
	u.invalidate(ctx)
	return b, nil
}

func (u *Usecase) Update(ctx context.Context, id uuid.UUID, in Input) (*entity.Banner, error) {
	b, err := u.banners.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	b.ImageURL = in.ImageURL
	b.Link = in.Link
	b.StartsAt = in.StartsAt
	b.EndsAt = in.EndsAt
	b.SortOrder = in.SortOrder
	b.IsActive = in.IsActive

	if err := u.banners.Update(ctx, b); err != nil {
		return nil, err
	}
	u.invalidate(ctx)
	return b, nil
}

func (u *Usecase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := u.banners.Delete(ctx, id); err != nil {
		return err
	}
	u.invalidate(ctx)
	return nil
}

func (u *Usecase) ListAll(ctx context.Context) ([]*entity.Banner, error) {
	return u.banners.ListAll(ctx)
}

// ListActive returns banners active & within schedule, cache-aside for 5m.
func (u *Usecase) ListActive(ctx context.Context) ([]*entity.Banner, error) {
	if cached, ok, err := cache.GetJSON[[]*entity.Banner](ctx, u.rdb, activeCacheKey); err == nil && ok {
		return cached, nil
	}
	list, err := u.banners.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	_ = cache.SetJSON(ctx, u.rdb, activeCacheKey, list, activeCacheTTL)
	return list, nil
}

func (u *Usecase) invalidate(ctx context.Context) {
	_ = cache.Delete(ctx, u.rdb, activeCacheKey)
}
