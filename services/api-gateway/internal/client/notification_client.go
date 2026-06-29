package client

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	notificationpb "github.com/trungquantrannguyen/threadly/proto/notification"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type NotificationClient struct {
	conn   *grpc.ClientConn
	client notificationpb.NotificationServiceClient
	log    zerolog.Logger
}

func NewNotificationClient(cfg config.Config, log zerolog.Logger) (*NotificationClient, error) {
	conn, err := grpc.NewClient(
		cfg.NotificationServiceGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	client := notificationpb.NewNotificationServiceClient(conn)

	return &NotificationClient{
		conn:   conn,
		client: client,
		log:    log,
	}, nil
}

func (c *NotificationClient) GetHealth(ctx context.Context) (*notificationpb.GetNotificationServiceHealthResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetHealth(ctx, &notificationpb.GetNotificationServiceHealthRequest{})
}

func (c *NotificationClient) GetNotifications(ctx context.Context, userID string, limit int32) (*notificationpb.GetNotificationsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.GetNotifications(ctx, &notificationpb.GetNotificationsRequest{
		UserId: userID,
		Limit:  limit,
	})
}

func (c *NotificationClient) MarkNotificationRead(ctx context.Context, userID string, notificationID string) (*notificationpb.MarkNotificationReadResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.MarkNotificationRead(ctx, &notificationpb.MarkNotificationReadRequest{
		UserId:         userID,
		NotificationId: notificationID,
	})
}

func (c *NotificationClient) MarkAllNotificationsRead(ctx context.Context, userID string) (*notificationpb.MarkAllNotificationsReadResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return c.client.MarkAllNotificationsRead(ctx, &notificationpb.MarkAllNotificationsReadRequest{
		UserId: userID,
	})
}

func (c *NotificationClient) Close() error {
	return c.conn.Close()
}
