package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, u *entity.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Update(ctx context.Context, u *entity.User) error
	// CountCreatedBetween counts users created within [from, to] (FR-7.4).
	CountCreatedBetween(ctx context.Context, from, to time.Time) (int, error)
}
