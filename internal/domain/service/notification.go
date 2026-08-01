// Package service defines interfaces for external services (payment,
// shipping, notification, storage), implemented under internal/infra.
package service

import "context"

// NotificationService sends outbound notifications. Implemented by a
// log-only stub until notification-service (gRPC) lands; usecases depend on
// this interface only, so the swap needs no usecase changes (FR-2.10).
type NotificationService interface {
	SendEmail(ctx context.Context, to, subject, body string) error
}
