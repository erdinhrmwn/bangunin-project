// Package notification lists a user's in-app notifications and marks them
// read.
package notification

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
)

type Usecase struct {
	notify repository.NotificationRepository
}

func New(notify repository.NotificationRepository) *Usecase {
	return &Usecase{notify: notify}
}

func (u *Usecase) List(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.Notification, int, error) {
	return u.notify.FindByUserID(ctx, userID, page, perPage)
}

func (u *Usecase) MarkRead(ctx context.Context, id, userID uuid.UUID) error {
	return u.notify.MarkRead(ctx, id, userID)
}
