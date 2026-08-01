// Package grpcclient holds clients for external gRPC services
// (payment-service, notification-service).
package grpcclient

import (
	"context"

	"github.com/rs/zerolog"
)

// NotificationStub logs email content instead of sending it, until
// notification-service is available (FR-2.10).
type NotificationStub struct {
	log zerolog.Logger
}

func NewNotificationStub(log zerolog.Logger) *NotificationStub {
	return &NotificationStub{log: log}
}

func (s *NotificationStub) SendEmail(ctx context.Context, to, subject, body string) error {
	s.log.Info().
		Str("to", to).
		Str("subject", subject).
		Str("body", body).
		Msg("notification stub: email")
	return nil
}
