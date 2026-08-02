// Package notify provides a shared "create notification, optionally email"
// helper, replacing the near-identical create+email sequence that was
// duplicated across the order, adminsupplier, inventory, and payout
// usecases.
package notify

import (
	"context"

	"github.com/google/uuid"

	"erdinhrmwn/bangunin/internal/domain/entity"
	"erdinhrmwn/bangunin/internal/domain/repository"
	"erdinhrmwn/bangunin/internal/domain/service"
)

// Send creates a notification of notifType for userID.
func Send(ctx context.Context, repo repository.NotificationRepository, userID uuid.UUID, notifType, title, body string) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return repo.Create(ctx, &entity.Notification{
		ID: id, UserID: userID, Type: notifType, Title: title, Body: body,
	})
}

// SendWithEmail creates a notification then also emails the user.
func SendWithEmail(ctx context.Context, repo repository.NotificationRepository, users repository.UserRepository, email service.EmailEnqueuer, userID uuid.UUID, notifType, title, body string) error {
	if err := Send(ctx, repo, userID, notifType, title, body); err != nil {
		return err
	}
	usr, err := users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	return email.EnqueueEmail(usr.Email, title, body)
}
