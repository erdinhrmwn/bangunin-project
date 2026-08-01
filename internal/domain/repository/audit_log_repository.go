package repository

import (
	"context"
	"time"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type AuditLogRepository interface {
	Create(ctx context.Context, a *entity.AuditLog) error
	List(ctx context.Context, from, to time.Time, page, perPage int) ([]*entity.AuditLog, int, error)
}
