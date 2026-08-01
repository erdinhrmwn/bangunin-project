package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type SupplierBankAccountRepository interface {
	Create(ctx context.Context, a *entity.SupplierBankAccount) error
	Update(ctx context.Context, a *entity.SupplierBankAccount) error
	Delete(ctx context.Context, id uuid.UUID) error
	FindByID(ctx context.Context, id uuid.UUID) (*entity.SupplierBankAccount, error)
	FindBySupplierID(ctx context.Context, supplierID uuid.UUID) ([]*entity.SupplierBankAccount, error)
	UnsetDefault(ctx context.Context, supplierID uuid.UUID) error
}
