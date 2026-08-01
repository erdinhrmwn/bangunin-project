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

type SupplierRepository struct {
	db *pgxpool.Pool
}

func NewSupplierRepository(db *pgxpool.Pool) *SupplierRepository {
	return &SupplierRepository{db: db}
}

func (r *SupplierRepository) Create(ctx context.Context, s *entity.Supplier) error {
	const q = `
		INSERT INTO suppliers (
			id, user_id, store_name, slug, description, status, origin_city_id,
			pickup_address, own_fleet_enabled, fleet_coverage_km, fleet_flat_rate
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.db.Exec(ctx, q,
		s.ID, s.UserID, s.StoreName, s.Slug, s.Description, s.Status, s.OriginCityID,
		s.PickupAddress, s.OwnFleetEnabled, s.FleetCoverageKM, s.FleetFlatRate,
	)
	if err != nil {
		return fmt.Errorf("postgres: create supplier: %w", err)
	}
	return nil
}

func (r *SupplierRepository) Update(ctx context.Context, s *entity.Supplier) error {
	const q = `
		UPDATE suppliers
		SET store_name = $2, slug = $3, description = $4, status = $5, origin_city_id = $6,
			pickup_address = $7, own_fleet_enabled = $8, fleet_coverage_km = $9,
			fleet_flat_rate = $10, verified_at = $11
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q,
		s.ID, s.StoreName, s.Slug, s.Description, s.Status, s.OriginCityID,
		s.PickupAddress, s.OwnFleetEnabled, s.FleetCoverageKM, s.FleetFlatRate, s.VerifiedAt,
	)
	if err != nil {
		return fmt.Errorf("postgres: update supplier: %w", err)
	}
	return nil
}

func (r *SupplierRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Supplier, error) {
	const q = selectSupplierCols + `FROM suppliers WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

func (r *SupplierRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*entity.Supplier, error) {
	const q = selectSupplierCols + `FROM suppliers WHERE user_id = $1`
	return r.scanOne(ctx, q, userID)
}

func (r *SupplierRepository) FindBySlug(ctx context.Context, slug string) (*entity.Supplier, error) {
	const q = selectSupplierCols + `FROM suppliers WHERE slug = $1`
	return r.scanOne(ctx, q, slug)
}

func (r *SupplierRepository) List(ctx context.Context, status, q string, page, perPage int) ([]*entity.Supplier, int, error) {
	where := "WHERE ($1 = '' OR status = $1) AND ($2 = '' OR store_name ILIKE '%' || $2 || '%')"

	var total int
	countQ := `SELECT count(*) FROM suppliers ` + where
	if err := r.db.QueryRow(ctx, countQ, status, q).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count suppliers: %w", err)
	}

	listQ := selectSupplierCols + `FROM suppliers ` + where + ` ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	rows, err := r.db.Query(ctx, listQ, status, q, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list suppliers: %w", err)
	}
	defer rows.Close()

	var out []*entity.Supplier
	for rows.Next() {
		s, err := scanSupplier(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("postgres: scan supplier: %w", err)
		}
		out = append(out, s)
	}
	return out, total, rows.Err()
}

const selectSupplierCols = `
	SELECT id, user_id, store_name, slug, description, status, origin_city_id,
		pickup_address, own_fleet_enabled, fleet_coverage_km, fleet_flat_rate,
		verified_at, created_at
`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSupplier(row rowScanner) (*entity.Supplier, error) {
	var s entity.Supplier
	err := row.Scan(
		&s.ID, &s.UserID, &s.StoreName, &s.Slug, &s.Description, &s.Status, &s.OriginCityID,
		&s.PickupAddress, &s.OwnFleetEnabled, &s.FleetCoverageKM, &s.FleetFlatRate,
		&s.VerifiedAt, &s.CreatedAt,
	)
	return &s, err
}

func (r *SupplierRepository) scanOne(ctx context.Context, query string, arg any) (*entity.Supplier, error) {
	s, err := scanSupplier(r.db.QueryRow(ctx, query, arg))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find supplier: %w", err)
	}
	return s, nil
}
