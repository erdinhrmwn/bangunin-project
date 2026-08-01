package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type AuditLogRepository struct {
	db *pgxpool.Pool
}

func NewAuditLogRepository(db *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, a *entity.AuditLog) error {
	const q = `
		INSERT INTO audit_logs (id, actor_id, action, entity_type, entity_id, metadata, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	metadata := a.Metadata
	if metadata == nil {
		metadata = map[string]any{} // metadata column is NOT NULL; nil map encodes as SQL NULL
	}
	_, err := r.db.Exec(ctx, q, a.ID, a.ActorID, a.Action, a.EntityType, a.EntityID, metadata, a.IPAddress)
	if err != nil {
		return fmt.Errorf("postgres: create audit log: %w", err)
	}
	return nil
}

func (r *AuditLogRepository) List(ctx context.Context, from, to time.Time, page, perPage int) ([]*entity.AuditLog, int, error) {
	var total int
	countQ := `SELECT count(*) FROM audit_logs WHERE created_at BETWEEN $1 AND $2`
	if err := r.db.QueryRow(ctx, countQ, from, to).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("postgres: count audit logs: %w", err)
	}

	const q = `
		SELECT id, actor_id, action, entity_type, entity_id, metadata, ip_address, created_at
		FROM audit_logs WHERE created_at BETWEEN $1 AND $2
		ORDER BY created_at DESC LIMIT $3 OFFSET $4
	`
	rows, err := r.db.Query(ctx, q, from, to, perPage, (page-1)*perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("postgres: list audit logs: %w", err)
	}
	defer rows.Close()

	var out []*entity.AuditLog
	for rows.Next() {
		var a entity.AuditLog
		if err := rows.Scan(&a.ID, &a.ActorID, &a.Action, &a.EntityType, &a.EntityID, &a.Metadata, &a.IPAddress, &a.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("postgres: scan audit log: %w", err)
		}
		out = append(out, &a)
	}
	return out, total, rows.Err()
}
