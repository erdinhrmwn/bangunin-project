package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type WithdrawRequestRepository interface {
	Create(ctx context.Context, w *entity.WithdrawRequest) error
	Update(ctx context.Context, w *entity.WithdrawRequest) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.WithdrawRequest, error)
	ListBySupplier(ctx context.Context, supplierID uuid.UUID, page, perPage int) ([]*entity.WithdrawRequest, int, error)
	ListAll(ctx context.Context, status string, page, perPage int) ([]*entity.WithdrawRequest, int, error)
}
