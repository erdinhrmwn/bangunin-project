package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/errs"
)

type SupplierBankAccountRepository struct {
	db *pgxpool.Pool
}

func NewSupplierBankAccountRepository(db *pgxpool.Pool) *SupplierBankAccountRepository {
	return &SupplierBankAccountRepository{db: db}
}

func (r *SupplierBankAccountRepository) Create(ctx context.Context, a *entity.SupplierBankAccount) error {
	const q = `
		INSERT INTO supplier_bank_accounts (id, supplier_id, bank_code, account_number, account_name, is_default)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(ctx, q, a.ID, a.SupplierID, a.BankCode, a.AccountNumber, a.AccountName, a.IsDefault)
	if err != nil {
		return fmt.Errorf("postgres: create supplier bank account: %w", err)
	}
	return nil
}

func (r *SupplierBankAccountRepository) Update(ctx context.Context, a *entity.SupplierBankAccount) error {
	const q = `
		UPDATE supplier_bank_accounts
		SET bank_code = $2, account_number = $3, account_name = $4, is_default = $5
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q, a.ID, a.BankCode, a.AccountNumber, a.AccountName, a.IsDefault)
	if err != nil {
		return fmt.Errorf("postgres: update supplier bank account: %w", err)
	}
	return nil
}

func (r *SupplierBankAccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM supplier_bank_accounts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("postgres: delete supplier bank account: %w", err)
	}
	return nil
}

func (r *SupplierBankAccountRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.SupplierBankAccount, error) {
	const q = `
		SELECT id, supplier_id, bank_code, account_number, account_name, is_default, created_at
		FROM supplier_bank_accounts WHERE id = $1
	`
	var a entity.SupplierBankAccount
	err := r.db.QueryRow(ctx, q, id).Scan(
		&a.ID, &a.SupplierID, &a.BankCode, &a.AccountNumber, &a.AccountName, &a.IsDefault, &a.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find supplier bank account: %w", err)
	}
	return &a, nil
}

func (r *SupplierBankAccountRepository) FindBySupplierID(ctx context.Context, supplierID uuid.UUID) ([]*entity.SupplierBankAccount, error) {
	const q = `
		SELECT id, supplier_id, bank_code, account_number, account_name, is_default, created_at
		FROM supplier_bank_accounts WHERE supplier_id = $1 ORDER BY created_at
	`
	rows, err := r.db.Query(ctx, q, supplierID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list supplier bank accounts: %w", err)
	}
	defer rows.Close()

	var out []*entity.SupplierBankAccount
	for rows.Next() {
		var a entity.SupplierBankAccount
		if err := rows.Scan(&a.ID, &a.SupplierID, &a.BankCode, &a.AccountNumber, &a.AccountName, &a.IsDefault, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("postgres: scan supplier bank account: %w", err)
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

func (r *SupplierBankAccountRepository) UnsetDefault(ctx context.Context, supplierID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE supplier_bank_accounts SET is_default = false WHERE supplier_id = $1`, supplierID)
	if err != nil {
		return fmt.Errorf("postgres: unset default supplier bank account: %w", err)
	}
	return nil
}
