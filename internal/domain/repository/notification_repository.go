package repository

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
)

type NotificationRepository interface {
	Create(ctx context.Context, n *entity.Notification) error
	FindByUserID(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Notification, int, error)
	MarkRead(ctx context.Context, id, userID uuid.UUID) error
}
