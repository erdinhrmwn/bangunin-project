package grpcclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	notificationv1 "erdinhrmwn/bangunin/proto/notification/v1"
)

// NotificationClient implements domain/service.NotificationService against
// the standalone services/notification gRPC server (FR-6.6).
type NotificationClient struct {
	client notificationv1.NotificationServiceClient
}

// NewNotificationClient dials addr, using TLS unless env is "development".
func NewNotificationClient(addr, env string) (*NotificationClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(transportCredentials(env)))
	if err != nil {
		return nil, fmt.Errorf("grpcclient: dial notification-service: %w", err)
	}
	return &NotificationClient{client: notificationv1.NewNotificationServiceClient(conn)}, nil
}

func (c *NotificationClient) SendEmail(ctx context.Context, to, subject, body string) error {
	_, err := c.client.SendEmail(ctx, &notificationv1.SendEmailRequest{To: to, Subject: subject, Body: body})
	if err != nil {
		return fmt.Errorf("grpcclient: send email: %w", err)
	}
	return nil
}
