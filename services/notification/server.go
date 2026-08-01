package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	notificationv1 "erdinhrmwn/bangunin/proto/notification/v1"
)

type notificationServer struct {
	notificationv1.UnimplementedNotificationServiceServer
	mailjet *mailjetClient
	log     zerolog.Logger
}

func (s *notificationServer) SendEmail(ctx context.Context, req *notificationv1.SendEmailRequest) (*notificationv1.SendEmailResponse, error) {
	id, err := s.mailjet.SendEmail(ctx, req.To, req.Subject, req.Body)
	if err != nil {
		s.log.Error().Err(err).Str("to", req.To).Msg("send email failed")
		return nil, fmt.Errorf("send email: %w", err)
	}
	return &notificationv1.SendEmailResponse{MessageId: id}, nil
}
