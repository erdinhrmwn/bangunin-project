package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type OrderStatusHistoryRepository struct {
	db *pgxpool.Pool
}

func NewOrderStatusHistoryRepository(db *pgxpool.Pool) *OrderStatusHistoryRepository {
	return &OrderStatusHistoryRepository{db: db}
}

func (r *OrderStatusHistoryRepository) Create(ctx context.Context, h *entity.OrderStatusHistory) error {
	const q = `
		INSERT INTO order_status_histories (id, order_id, from_status, to_status, actor_type, actor_id, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, q, h.ID, h.OrderID, h.FromStatus, h.ToStatus, h.ActorType, h.ActorID, h.Note)
	if err != nil {
		return fmt.Errorf("postgres: create order status history: %w", err)
	}
	return nil
}

func (r *OrderStatusHistoryRepository) ListByOrderID(ctx context.Context, orderID uuid.UUID) ([]*entity.OrderStatusHistory, error) {
	rows, err := r.db.Query(ctx, selectOrderStatusHistoryCols+`FROM order_status_histories WHERE order_id = $1 ORDER BY created_at`, orderID)
	if err != nil {
		return nil, fmt.Errorf("postgres: list order status histories: %w", err)
	}
	defer rows.Close()

	var out []*entity.OrderStatusHistory
	for rows.Next() {
		h, err := scanOrderStatusHistory(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres: scan order status history: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

const selectOrderStatusHistoryCols = `SELECT id, order_id, from_status, to_status, actor_type, actor_id, note, created_at `

func scanOrderStatusHistory(row rowScanner) (*entity.OrderStatusHistory, error) {
	var h entity.OrderStatusHistory
	err := row.Scan(&h.ID, &h.OrderID, &h.FromStatus, &h.ToStatus, &h.ActorType, &h.ActorID, &h.Note, &h.CreatedAt)
	return &h, err
}
