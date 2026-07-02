package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/messaging"
	notificationpb "github.com/trungquantrannguyen/threadly/proto/notification"
	"github.com/trungquantrannguyen/threadly/services/notification-service/internal/dto"
)

type fakeNotificationService struct {
	notifications []dto.NotificationResponse
	count         int64
	updated       bool
	updatedCount  int64
	err           error

	getNotificationsCalled bool
	getNotificationsUserID string
	getNotificationsLimit  int

	countCalled bool
	countUserID string

	markReadCalled         bool
	markReadUserID         string
	markReadNotificationID string

	markAllCalled bool
	markAllUserID string
}

func (f *fakeNotificationService) CreateFromEvent(ctx context.Context, event messaging.Event) error {
	return f.err
}

func (f *fakeNotificationService) GetNotifications(ctx context.Context, userID string, limit int) ([]dto.NotificationResponse, error) {
	f.getNotificationsCalled = true
	f.getNotificationsUserID = userID
	f.getNotificationsLimit = limit

	if f.err != nil {
		return nil, f.err
	}

	return f.notifications, nil
}

func (f *fakeNotificationService) GetUnreadNotificationCount(ctx context.Context, userID string) (int64, error) {
	f.countCalled = true
	f.countUserID = userID

	if f.err != nil {
		return 0, f.err
	}

	return f.count, nil
}

func (f *fakeNotificationService) MarkNotificationRead(ctx context.Context, userID string, notificationID string) (bool, error) {
	f.markReadCalled = true
	f.markReadUserID = userID
	f.markReadNotificationID = notificationID

	if f.err != nil {
		return false, f.err
	}

	return f.updated, nil
}

func (f *fakeNotificationService) MarkAllNotificationsRead(ctx context.Context, userID string) (int64, error) {
	f.markAllCalled = true
	f.markAllUserID = userID

	if f.err != nil {
		return 0, f.err
	}

	return f.updatedCount, nil
}

func newTestNotificationServer(fakeSvc *fakeNotificationService) *NotificationServiceServer {
	return NewNotificationServiceServer(
		config.Config{
			ServiceName: "notification-service",
			AppEnv:      "test",
		},
		zerolog.Nop(),
		fakeSvc,
	)
}

func TestGetHealth(t *testing.T) {
	server := newTestNotificationServer(&fakeNotificationService{})

	res, err := server.GetHealth(context.Background(), &notificationpb.GetNotificationServiceHealthRequest{})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "ok", res.Status)
	assert.Equal(t, "notification-service", res.Service)
	assert.Equal(t, "test", res.Env)
	assert.NotEmpty(t, res.CheckedAt)
}

func TestGetNotificationsSuccess(t *testing.T) {
	fakeSvc := &fakeNotificationService{
		notifications: []dto.NotificationResponse{
			{
				ID:          "notification-1",
				RecipientID: "user-1",
				ActorID:     "actor-1",
				Type:        "post_liked",
				EntityType:  "post",
				EntityID:    "post-1",
				Payload:     `{"message":"liked your post"}`,
				ReadAt:      "",
				CreatedAt:   "2026-07-02T10:00:00Z",
				Actor: &dto.NotificationActorResponse{
					ID:          "actor-1",
					Username:    "actor",
					DisplayName: "Actor User",
					AvatarURL:   "https://example.com/avatar.png",
					IsVerified:  true,
				},
			},
		},
	}
	server := newTestNotificationServer(fakeSvc)

	res, err := server.GetNotifications(context.Background(), &notificationpb.GetNotificationsRequest{
		UserId: "user-1",
		Limit:  10,
	})

	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Notifications, 1)

	assert.True(t, fakeSvc.getNotificationsCalled)
	assert.Equal(t, "user-1", fakeSvc.getNotificationsUserID)
	assert.Equal(t, 10, fakeSvc.getNotificationsLimit)

	notification := res.Notifications[0]

	assert.Equal(t, "notification-1", notification.Id)
	assert.Equal(t, "user-1", notification.RecipientId)
	assert.Equal(t, "actor-1", notification.ActorId)
	assert.Equal(t, "post_liked", notification.Type)
	assert.Equal(t, "post", notification.EntityType)
	assert.Equal(t, "post-1", notification.EntityId)
	assert.Equal(t, `{"message":"liked your post"}`, notification.Payload)
	assert.Equal(t, "", notification.ReadAt)
	assert.Equal(t, "2026-07-02T10:00:00Z", notification.CreatedAt)

	require.NotNil(t, notification.Actor)
	assert.Equal(t, "actor-1", notification.Actor.Id)
	assert.Equal(t, "actor", notification.Actor.Username)
	assert.Equal(t, "Actor User", notification.Actor.DisplayName)
	assert.Equal(t, "https://example.com/avatar.png", notification.Actor.AvatarUrl)
	assert.True(t, notification.Actor.IsVerified)
}

func TestGetNotificationsSuccessWithoutActor(t *testing.T) {
	fakeSvc := &fakeNotificationService{
		notifications: []dto.NotificationResponse{
			{
				ID:          "notification-1",
				RecipientID: "user-1",
				Type:        "user_followed",
				EntityType:  "user",
				Payload:     `{"message":"started following you"}`,
				CreatedAt:   "2026-07-02T10:00:00Z",
				Actor:       nil,
			},
		},
	}
	server := newTestNotificationServer(fakeSvc)

	res, err := server.GetNotifications(context.Background(), &notificationpb.GetNotificationsRequest{
		UserId: "user-1",
		Limit:  5,
	})

	require.NoError(t, err)
	require.NotNil(t, res)
	require.Len(t, res.Notifications, 1)

	assert.Nil(t, res.Notifications[0].Actor)
}

func TestGetNotificationsReturnsError(t *testing.T) {
	expectedErr := errors.New("get notifications failed")

	fakeSvc := &fakeNotificationService{
		err: expectedErr,
	}
	server := newTestNotificationServer(fakeSvc)

	res, err := server.GetNotifications(context.Background(), &notificationpb.GetNotificationsRequest{
		UserId: "user-1",
		Limit:  10,
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestMarkNotificationReadSuccess(t *testing.T) {
	fakeSvc := &fakeNotificationService{
		updated: true,
	}
	server := newTestNotificationServer(fakeSvc)

	res, err := server.MarkNotificationRead(context.Background(), &notificationpb.MarkNotificationReadRequest{
		UserId:         "user-1",
		NotificationId: "notification-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeSvc.markReadCalled)
	assert.Equal(t, "user-1", fakeSvc.markReadUserID)
	assert.Equal(t, "notification-1", fakeSvc.markReadNotificationID)

	assert.True(t, res.Success)
	assert.Equal(t, "notification marked as read", res.Message)
}

func TestMarkNotificationReadNotUpdated(t *testing.T) {
	fakeSvc := &fakeNotificationService{
		updated: false,
	}
	server := newTestNotificationServer(fakeSvc)

	res, err := server.MarkNotificationRead(context.Background(), &notificationpb.MarkNotificationReadRequest{
		UserId:         "user-1",
		NotificationId: "notification-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.False(t, res.Success)
	assert.Equal(t, "notification not found or already read", res.Message)
}

func TestMarkNotificationReadReturnsError(t *testing.T) {
	expectedErr := errors.New("mark notification read failed")

	fakeSvc := &fakeNotificationService{
		err: expectedErr,
	}
	server := newTestNotificationServer(fakeSvc)

	res, err := server.MarkNotificationRead(context.Background(), &notificationpb.MarkNotificationReadRequest{
		UserId:         "user-1",
		NotificationId: "notification-1",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestMarkAllNotificationsReadSuccess(t *testing.T) {
	fakeSvc := &fakeNotificationService{
		updatedCount: 3,
	}
	server := newTestNotificationServer(fakeSvc)

	res, err := server.MarkAllNotificationsRead(context.Background(), &notificationpb.MarkAllNotificationsReadRequest{
		UserId: "user-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeSvc.markAllCalled)
	assert.Equal(t, "user-1", fakeSvc.markAllUserID)

	assert.True(t, res.Success)
	assert.Equal(t, "notifications marked as read", res.Message)
	assert.Equal(t, int32(3), res.UpdatedCount)
}

func TestMarkAllNotificationsReadReturnsError(t *testing.T) {
	expectedErr := errors.New("mark all failed")

	fakeSvc := &fakeNotificationService{
		err: expectedErr,
	}
	server := newTestNotificationServer(fakeSvc)

	res, err := server.MarkAllNotificationsRead(context.Background(), &notificationpb.MarkAllNotificationsReadRequest{
		UserId: "user-1",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestGetUnreadNotificationCountSuccess(t *testing.T) {
	fakeSvc := &fakeNotificationService{
		count: 7,
	}
	server := newTestNotificationServer(fakeSvc)

	res, err := server.GetUnreadNotificationCount(context.Background(), &notificationpb.GetUnreadNotificationCountRequest{
		UserId: "user-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeSvc.countCalled)
	assert.Equal(t, "user-1", fakeSvc.countUserID)
	assert.Equal(t, int64(7), res.Count)
}

func TestGetUnreadNotificationCountReturnsError(t *testing.T) {
	expectedErr := errors.New("count failed")

	fakeSvc := &fakeNotificationService{
		err: expectedErr,
	}
	server := newTestNotificationServer(fakeSvc)

	res, err := server.GetUnreadNotificationCount(context.Background(), &notificationpb.GetUnreadNotificationCountRequest{
		UserId: "user-1",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}
