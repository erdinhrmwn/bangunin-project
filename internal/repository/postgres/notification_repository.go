package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) Create(ctx context.Context, n *entity.Notification) error {
	const q = `
		INSERT INTO notifications (id, user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	data := n.Data
	if data == nil {
		data = map[string]any{} // data column is NOT NULL; nil map encodes as SQL NULL
	}
	_, err := r.db.Exec(ctx, q, n.ID, n.UserID, n.Type, n.Title, n.Body, data)
	if err != nil {
		return fmt.Errorf("postgres: create notification: %w", err)
	}
	return nil
}

func (r *NotificationRepository) FindByUserID(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Notification, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count notifications: %w", err)
	}

	const q = `
		SELECT id, user_id, type, title, body, data, read_at, created_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, q, userID, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list notifications: %w", err)
	}
	defer rows.Close()

	var out []*entity.Notification
	for rows.Next() {
		var n entity.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Data, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("postgres: scan notification: %w", err)
		}
		out = append(out, &n)
	}
	return out, total, rows.Err()
}

func (r *NotificationRepository) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	const q = `UPDATE notifications SET read_at = now() WHERE id = $1 AND user_id = $2 AND read_at IS NULL`
	_, err := r.db.Exec(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("postgres: mark notification read: %w", err)
	}
	return nil
}
