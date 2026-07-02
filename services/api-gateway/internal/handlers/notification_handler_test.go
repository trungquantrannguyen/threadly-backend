package handlers

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/pkg/middleware"
	notificationpb "github.com/trungquantrannguyen/threadly/proto/notification"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
	"google.golang.org/grpc"
)

type fakeNotificationHandlerServiceServer struct {
	notificationpb.UnimplementedNotificationServiceServer

	err error

	healthCalled                     bool
	getNotificationsCalled           bool
	markNotificationReadCalled       bool
	markAllNotificationsReadCalled   bool
	getUnreadNotificationCountCalled bool

	receivedUserID         string
	receivedLimit          int32
	receivedNotificationID string
}

func startNotificationHandlerTestClient(t *testing.T, fakeServer *fakeNotificationHandlerServiceServer) (*client.NotificationClient, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	notificationpb.RegisterNotificationServiceServer(grpcServer, fakeServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	notificationClient, err := client.NewNotificationClient(config.Config{
		NotificationServiceGRPCAddr: listener.Addr().String(),
	}, zerolog.Nop())
	require.NoError(t, err)

	cleanup := func() {
		_ = notificationClient.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return notificationClient, cleanup
}

func (s *fakeNotificationHandlerServiceServer) GetHealth(ctx context.Context, req *notificationpb.GetNotificationServiceHealthRequest) (*notificationpb.GetNotificationServiceHealthResponse, error) {
	s.healthCalled = true

	if s.err != nil {
		return nil, s.err
	}

	return &notificationpb.GetNotificationServiceHealthResponse{
		Service:   "notification-service",
		Status:    "ok",
		Env:       "test",
		CheckedAt: "now",
	}, nil
}

func (s *fakeNotificationHandlerServiceServer) GetNotifications(ctx context.Context, req *notificationpb.GetNotificationsRequest) (*notificationpb.GetNotificationsResponse, error) {
	s.getNotificationsCalled = true
	s.receivedUserID = req.GetUserId()
	s.receivedLimit = req.GetLimit()

	if s.err != nil {
		return nil, s.err
	}

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

func (s *fakeNotificationHandlerServiceServer) MarkNotificationRead(ctx context.Context, req *notificationpb.MarkNotificationReadRequest) (*notificationpb.MarkNotificationReadResponse, error) {
	s.markNotificationReadCalled = true
	s.receivedUserID = req.GetUserId()
	s.receivedNotificationID = req.GetNotificationId()

	if s.err != nil {
		return nil, s.err
	}

	return &notificationpb.MarkNotificationReadResponse{
		Success: true,
		Message: "notification marked as read",
	}, nil
}

func (s *fakeNotificationHandlerServiceServer) MarkAllNotificationsRead(ctx context.Context, req *notificationpb.MarkAllNotificationsReadRequest) (*notificationpb.MarkAllNotificationsReadResponse, error) {
	s.markAllNotificationsReadCalled = true
	s.receivedUserID = req.GetUserId()

	if s.err != nil {
		return nil, s.err
	}

	return &notificationpb.MarkAllNotificationsReadResponse{
		Success:      true,
		Message:      "all notifications marked as read",
		UpdatedCount: 3,
	}, nil
}

func (s *fakeNotificationHandlerServiceServer) GetUnreadNotificationCount(ctx context.Context, req *notificationpb.GetUnreadNotificationCountRequest) (*notificationpb.GetUnreadNotificationCountResponse, error) {
	s.getUnreadNotificationCountCalled = true
	s.receivedUserID = req.GetUserId()

	if s.err != nil {
		return nil, s.err
	}

	return &notificationpb.GetUnreadNotificationCountResponse{
		Count: 5,
	}, nil
}

func newNotificationHandlerTestRouter(handler *NotificationHandler, withUser bool) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	if withUser {
		router.Use(func(c *gin.Context) {
			c.Set(middleware.UserIDKey, "user-1")
			c.Next()
		})
	}

	router.GET("/notifications/health", handler.GetHealth)
	router.GET("/notifications", handler.GetNotifications)
	router.PATCH("/notifications/:id/read", handler.MarkNotificationRead)
	router.PATCH("/notifications/read-all", handler.MarkAllNotificationsRead)
	router.GET("/notifications/unread-count", handler.GetUnreadNotificationCount)

	return router
}

func TestNotificationHandlerGetHealthSuccess(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.healthCalled)
	assert.Contains(t, w.Body.String(), "Notification service available")
	assert.Contains(t, w.Body.String(), "notification-service")
}

func TestNotificationHandlerGetHealthServiceUnavailable(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{
		err: errors.New("notification service down"),
	}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, fakeServer.healthCalled)
	assert.Contains(t, w.Body.String(), "Notification service unavailable")
}

func TestNotificationHandlerGetNotificationsSuccess(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications?limit=10", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.getNotificationsCalled)
	assert.Equal(t, "user-1", fakeServer.receivedUserID)
	assert.Equal(t, int32(10), fakeServer.receivedLimit)
	assert.Contains(t, w.Body.String(), "Notifications fetched successfully")
	assert.Contains(t, w.Body.String(), "notification-1")
}

func TestNotificationHandlerGetNotificationsDefaultLimit(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.getNotificationsCalled)
	assert.Equal(t, int32(20), fakeServer.receivedLimit)
}

func TestNotificationHandlerGetNotificationsInvalidLimitKeepsDefault(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications?limit=abc", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.getNotificationsCalled)
	assert.Equal(t, int32(20), fakeServer.receivedLimit)
}

func TestNotificationHandlerGetNotificationsUnauthorized(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, fakeServer.getNotificationsCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestNotificationHandlerGetNotificationsServiceUnavailable(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{
		err: errors.New("notification query failed"),
	}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, fakeServer.getNotificationsCalled)
	assert.Contains(t, w.Body.String(), "Notification service unavailable")
}

func TestNotificationHandlerMarkNotificationReadSuccess(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/notifications/notification-1/read", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.markNotificationReadCalled)
	assert.Equal(t, "user-1", fakeServer.receivedUserID)
	assert.Equal(t, "notification-1", fakeServer.receivedNotificationID)
	assert.Contains(t, w.Body.String(), "notification marked as read")
}

func TestNotificationHandlerMarkNotificationReadUnauthorized(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/notifications/notification-1/read", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, fakeServer.markNotificationReadCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestNotificationHandlerMarkNotificationReadServiceUnavailable(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{
		err: errors.New("mark read failed"),
	}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/notifications/notification-1/read", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, fakeServer.markNotificationReadCalled)
	assert.Contains(t, w.Body.String(), "Notification service unavailable")
}

func TestNotificationHandlerMarkAllNotificationsReadSuccess(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/notifications/read-all", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.markAllNotificationsReadCalled)
	assert.Equal(t, "user-1", fakeServer.receivedUserID)
	assert.Contains(t, w.Body.String(), "all notifications marked as read")
	assert.Contains(t, w.Body.String(), "3")
}

func TestNotificationHandlerMarkAllNotificationsReadUnauthorized(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/notifications/read-all", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, fakeServer.markAllNotificationsReadCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestNotificationHandlerMarkAllNotificationsReadServiceUnavailable(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{
		err: errors.New("mark all read failed"),
	}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/notifications/read-all", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, fakeServer.markAllNotificationsReadCalled)
	assert.Contains(t, w.Body.String(), "Notification service unavailable")
}

func TestNotificationHandlerGetUnreadNotificationCountSuccess(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.getUnreadNotificationCountCalled)
	assert.Equal(t, "user-1", fakeServer.receivedUserID)
	assert.Contains(t, w.Body.String(), "Unread notification count fetched successfully")
	assert.Contains(t, w.Body.String(), "5")
}

func TestNotificationHandlerGetUnreadNotificationCountUnauthorized(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, fakeServer.getUnreadNotificationCountCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestNotificationHandlerGetUnreadNotificationCountServiceUnavailable(t *testing.T) {
	fakeServer := &fakeNotificationHandlerServiceServer{
		err: errors.New("count failed"),
	}
	notificationClient, cleanup := startNotificationHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewNotificationHandler(notificationClient, zerolog.Nop())
	router := newNotificationHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/notifications/unread-count", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, fakeServer.getUnreadNotificationCountCalled)
	assert.Contains(t, w.Body.String(), "Notification service unavailable")
}
