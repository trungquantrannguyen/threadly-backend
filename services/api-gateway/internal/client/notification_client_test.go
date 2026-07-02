package client

import (
	"context"
	"net"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	notificationpb "github.com/trungquantrannguyen/threadly/proto/notification"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type fakeNotificationServiceServer struct {
	notificationpb.UnimplementedNotificationServiceServer

	healthCalled                     bool
	getNotificationsCalled           bool
	markNotificationReadCalled       bool
	markAllNotificationsReadCalled   bool
	getUnreadNotificationCountCalled bool
}

func startFakeNotificationGRPCServer(t *testing.T, fakeServer *fakeNotificationServiceServer) (*NotificationClient, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	notificationpb.RegisterNotificationServiceServer(grpcServer, fakeServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := &NotificationClient{
		conn:   conn,
		client: notificationpb.NewNotificationServiceClient(conn),
		log:    zerolog.Nop(),
	}

	cleanup := func() {
		_ = client.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return client, cleanup
}

func (s *fakeNotificationServiceServer) GetHealth(ctx context.Context, req *notificationpb.GetNotificationServiceHealthRequest) (*notificationpb.GetNotificationServiceHealthResponse, error) {
	s.healthCalled = true

	return &notificationpb.GetNotificationServiceHealthResponse{
		Status:    "ok",
		Service:   "notification-service",
		Env:       "test",
		CheckedAt: "now",
	}, nil
}

func (s *fakeNotificationServiceServer) GetNotifications(ctx context.Context, req *notificationpb.GetNotificationsRequest) (*notificationpb.GetNotificationsResponse, error) {
	s.getNotificationsCalled = true

	return &notificationpb.GetNotificationsResponse{
		Notifications: []*notificationpb.NotificationResponse{
			{
				Id:          "notification-1",
				RecipientId: req.GetUserId(),
				ActorId:     "actor-1",
				Type:        "post_liked",
				EntityType:  "post",
				EntityId:    "post-1",
				Payload:     `{"message":"liked your post"}`,
				ReadAt:      "",
				CreatedAt:   "2026-07-02T10:00:00Z",
				Actor: &notificationpb.NotificationActor{
					Id:          "actor-1",
					Username:    "actor",
					DisplayName: "Actor User",
					AvatarUrl:   "https://example.com/avatar.png",
					IsVerified:  true,
				},
			},
		},
	}, nil
}

func (s *fakeNotificationServiceServer) MarkNotificationRead(ctx context.Context, req *notificationpb.MarkNotificationReadRequest) (*notificationpb.MarkNotificationReadResponse, error) {
	s.markNotificationReadCalled = true

	return &notificationpb.MarkNotificationReadResponse{
		Success: true,
		Message: "notification marked as read",
	}, nil
}

func (s *fakeNotificationServiceServer) MarkAllNotificationsRead(ctx context.Context, req *notificationpb.MarkAllNotificationsReadRequest) (*notificationpb.MarkAllNotificationsReadResponse, error) {
	s.markAllNotificationsReadCalled = true

	return &notificationpb.MarkAllNotificationsReadResponse{
		Success:      true,
		Message:      "all notifications marked as read",
		UpdatedCount: 3,
	}, nil
}

func (s *fakeNotificationServiceServer) GetUnreadNotificationCount(ctx context.Context, req *notificationpb.GetUnreadNotificationCountRequest) (*notificationpb.GetUnreadNotificationCountResponse, error) {
	s.getUnreadNotificationCountCalled = true

	return &notificationpb.GetUnreadNotificationCountResponse{
		Count: 5,
	}, nil
}

func TestNotificationClientGetHealth(t *testing.T) {
	fakeServer := &fakeNotificationServiceServer{}
	client, cleanup := startFakeNotificationGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetHealth(context.Background())

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.healthCalled)
	assert.Equal(t, "ok", res.Status)
	assert.Equal(t, "notification-service", res.Service)
	assert.Equal(t, "test", res.Env)
}

func TestNotificationClientGetNotifications(t *testing.T) {
	fakeServer := &fakeNotificationServiceServer{}
	client, cleanup := startFakeNotificationGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetNotifications(context.Background(), "user-1", 20)

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.getNotificationsCalled)
	require.Len(t, res.Notifications, 1)

	notification := res.Notifications[0]
	assert.Equal(t, "notification-1", notification.Id)
	assert.Equal(t, "user-1", notification.RecipientId)
	assert.Equal(t, "actor-1", notification.ActorId)
	assert.Equal(t, "post_liked", notification.Type)
	assert.Equal(t, "post", notification.EntityType)
	assert.Equal(t, "post-1", notification.EntityId)
	require.NotNil(t, notification.Actor)
	assert.Equal(t, "actor", notification.Actor.Username)
	assert.True(t, notification.Actor.IsVerified)
}

func TestNotificationClientMarkNotificationRead(t *testing.T) {
	fakeServer := &fakeNotificationServiceServer{}
	client, cleanup := startFakeNotificationGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.MarkNotificationRead(context.Background(), "user-1", "notification-1")

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.markNotificationReadCalled)
	assert.True(t, res.Success)
	assert.Equal(t, "notification marked as read", res.Message)
}

func TestNotificationClientMarkAllNotificationsRead(t *testing.T) {
	fakeServer := &fakeNotificationServiceServer{}
	client, cleanup := startFakeNotificationGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.MarkAllNotificationsRead(context.Background(), "user-1")

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.markAllNotificationsReadCalled)
	assert.True(t, res.Success)
	assert.Equal(t, "all notifications marked as read", res.Message)
	assert.Equal(t, int32(3), res.UpdatedCount)
}

func TestNotificationClientGetUnreadNotificationCount(t *testing.T) {
	fakeServer := &fakeNotificationServiceServer{}
	client, cleanup := startFakeNotificationGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetUnreadNotificationCount(context.Background(), "user-1")

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.getUnreadNotificationCountCalled)
	assert.Equal(t, int64(5), res.Count)
}
