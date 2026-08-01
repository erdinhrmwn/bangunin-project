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

type UserAddressRepository struct {
	db *pgxpool.Pool
}

func NewUserAddressRepository(db *pgxpool.Pool) *UserAddressRepository {
	return &UserAddressRepository{db: db}
}

func (r *UserAddressRepository) Create(ctx context.Context, a *entity.UserAddress) error {
	const q = `
		INSERT INTO user_addresses (
			id, user_id, label, recipient_name, recipient_phone,
			province_id, city_id, subdistrict, postal_code, address_detail, is_default
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Exec(ctx, q,
		a.ID, a.UserID, a.Label, a.RecipientName, a.RecipientPhone,
		a.ProvinceID, a.CityID, a.Subdistrict, a.PostalCode, a.AddressDetail, a.IsDefault,
	)
	if err != nil {
		return fmt.Errorf("postgres: create user address: %w", err)
	}
	return nil
}

func (r *UserAddressRepository) Update(ctx context.Context, a *entity.UserAddress) error {
	const q = `
		UPDATE user_addresses SET
			label = $2, recipient_name = $3, recipient_phone = $4,
			province_id = $5, city_id = $6, subdistrict = $7, postal_code = $8,
			address_detail = $9, is_default = $10
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q,
		a.ID, a.Label, a.RecipientName, a.RecipientPhone,
		a.ProvinceID, a.CityID, a.Subdistrict, a.PostalCode, a.AddressDetail, a.IsDefault,
	)
	if err != nil {
		return fmt.Errorf("postgres: update user address: %w", err)
	}
	return nil
}

func (r *UserAddressRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, err := r.db.Exec(ctx, `DELETE FROM user_addresses WHERE id = $1`, id); err != nil {
		return fmt.Errorf("postgres: delete user address: %w", err)
	}
	return nil
}

func (r *UserAddressRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.UserAddress, error) {
	a, err := scanUserAddress(r.db.QueryRow(ctx, selectUserAddressCols+`FROM user_addresses WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find user address: %w", err)
	}
	return a, nil
}

func (r *UserAddressRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.UserAddress, error) {
	rows, err := r.db.Query(ctx, selectUserAddressCols+`FROM user_addresses WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list user addresses: %w", err)
	}
	defer rows.Close()

	var out []*entity.UserAddress
	for rows.Next() {
		a, err := scanUserAddress(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan user address: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *UserAddressRepository) ClearDefault(ctx context.Context, userID uuid.UUID, keepID uuid.UUID) error {
	const q = `UPDATE user_addresses SET is_default = false WHERE user_id = $1 AND id != $2`
	if _, err := r.db.Exec(ctx, q, userID, keepID); err != nil {
		return fmt.Errorf("postgres: clear default address: %w", err)
	}
	return nil
}

const selectUserAddressCols = `
	SELECT id, user_id, label, recipient_name, recipient_phone,
		province_id, city_id, subdistrict, postal_code, address_detail, is_default, created_at
`

func scanUserAddress(row rowScanner) (*entity.UserAddress, error) {
	var a entity.UserAddress
	err := row.Scan(
		&a.ID, &a.UserID, &a.Label, &a.RecipientName, &a.RecipientPhone,
		&a.ProvinceID, &a.CityID, &a.Subdistrict, &a.PostalCode, &a.AddressDetail, &a.IsDefault, &a.CreatedAt,
	)
	return &a, err
}
