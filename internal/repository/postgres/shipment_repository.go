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

type ShipmentRepository struct {
	db *pgxpool.Pool
}

func NewShipmentRepository(db *pgxpool.Pool) *ShipmentRepository {
	return &ShipmentRepository{db: db}
}

func (r *ShipmentRepository) Create(ctx context.Context, s *entity.Shipment) error {
	const q = `
		INSERT INTO shipments (id, order_id, method, courier_code, courier_service, tracking_number, cost, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, q, s.ID, s.OrderID, s.Method, s.CourierCode, s.CourierService, s.TrackingNumber, s.Cost, s.Status)
	if err != nil {
		return fmt.Errorf("postgres: create shipment: %w", err)
	}
	return nil
}

func (r *ShipmentRepository) Update(ctx context.Context, s *entity.Shipment) error {
	const q = `
		UPDATE shipments SET
			courier_code = $2, courier_service = $3, tracking_number = $4, cost = $5,
			status = $6, shipped_at = $7, delivered_at = $8
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, q, s.ID, s.CourierCode, s.CourierService, s.TrackingNumber, s.Cost, s.Status, s.ShippedAt, s.DeliveredAt)
	if err != nil {
		return fmt.Errorf("postgres: update shipment: %w", err)
	}
	return nil
}

func (r *ShipmentRepository) FindByOrderID(ctx context.Context, orderID uuid.UUID) (*entity.Shipment, error) {
	s, err := scanShipment(r.db.QueryRow(ctx, selectShipmentCols+`FROM shipments WHERE order_id = $1`, orderID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("postgres: find shipment: %w", err)
	}
	return s, nil
}

const selectShipmentCols = `
	SELECT id, order_id, method, courier_code, courier_service, COALESCE(tracking_number, ''), cost, status, shipped_at, delivered_at, created_at
`

func scanShipment(row rowScanner) (*entity.Shipment, error) {
	var s entity.Shipment
	err := row.Scan(
		&s.ID, &s.OrderID, &s.Method, &s.CourierCode, &s.CourierService, &s.TrackingNumber,
		&s.Cost, &s.Status, &s.ShippedAt, &s.DeliveredAt, &s.CreatedAt,
	)
	return &s, err
}
