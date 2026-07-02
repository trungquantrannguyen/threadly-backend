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
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeFeedHandlerServiceServer struct {
	feedpb.UnimplementedFeedServiceServer

	err error

	healthCalled      bool
	getHomeFeedCalled bool

	receivedUserID string
	receivedLimit  int32
	receivedCursor string
}

func startFeedHandlerTestClient(t *testing.T, fakeServer *fakeFeedHandlerServiceServer) (*client.FeedClient, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	feedpb.RegisterFeedServiceServer(grpcServer, fakeServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	feedClient, err := client.NewFeedClient(config.Config{
		FeedServiceGRPCAddr: listener.Addr().String(),
	}, zerolog.Nop())
	require.NoError(t, err)

	cleanup := func() {
		_ = feedClient.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return feedClient, cleanup
}

func (s *fakeFeedHandlerServiceServer) GetHealth(ctx context.Context, req *feedpb.GetFeedServiceHealthRequest) (*feedpb.GetFeedServiceHealthResponse, error) {
	s.healthCalled = true

	if s.err != nil {
		return nil, s.err
	}

	return &feedpb.GetFeedServiceHealthResponse{
		Service:   "feed-service",
		Status:    "ok",
		Env:       "test",
		CheckedAt: "now",
	}, nil
}

func (s *fakeFeedHandlerServiceServer) GetHomeFeed(ctx context.Context, req *feedpb.GetHomeFeedRequest) (*feedpb.HomeFeedResponse, error) {
	s.getHomeFeedCalled = true
	s.receivedUserID = req.GetUserId()
	s.receivedLimit = req.GetLimit()
	s.receivedCursor = req.GetCursor()

	if s.err != nil {
		return nil, s.err
	}

	return &feedpb.HomeFeedResponse{
		Posts: []*feedpb.FeedPostResponse{
			{
				Id:         "post-1",
				AuthorId:   "author-1",
				Content:    "hello feed",
				Visibility: "public",
				CreatedAt:  "2026-07-02T10:00:00Z",
				UpdatedAt:  "2026-07-02T10:01:00Z",
				Author: &feedpb.FeedUserSummary{
					Id:          "author-1",
					Username:    "author",
					DisplayName: "Author User",
				},
			},
		},
		NextCursor: "next-cursor",
	}, nil
}

func newFeedHandlerTestRouter(handler *FeedHandler, withUser bool) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	if withUser {
		router.Use(func(c *gin.Context) {
			c.Set(middleware.UserIDKey, "user-1")
			c.Next()
		})
	}

	router.GET("/feeds/health", handler.GetHealth)
	router.GET("/feeds/home", handler.GetHomeFeed)

	return router
}

func TestFeedHandlerGetHealthSuccess(t *testing.T) {
	fakeServer := &fakeFeedHandlerServiceServer{}
	feedClient, cleanup := startFeedHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewFeedHandler(feedClient, zerolog.Nop())
	router := newFeedHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feeds/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.healthCalled)
	assert.Contains(t, w.Body.String(), "Feed service available")
	assert.Contains(t, w.Body.String(), "feed-service")
}

func TestFeedHandlerGetHealthServiceUnavailable(t *testing.T) {
	fakeServer := &fakeFeedHandlerServiceServer{
		err: errors.New("feed service down"),
	}
	feedClient, cleanup := startFeedHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewFeedHandler(feedClient, zerolog.Nop())
	router := newFeedHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feeds/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, fakeServer.healthCalled)
	assert.Contains(t, w.Body.String(), "Feed service unavailable")
}

func TestFeedHandlerGetHomeFeedSuccess(t *testing.T) {
	fakeServer := &fakeFeedHandlerServiceServer{}
	feedClient, cleanup := startFeedHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewFeedHandler(feedClient, zerolog.Nop())
	router := newFeedHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feeds/home?limit=10&cursor=abc", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.getHomeFeedCalled)

	assert.Equal(t, "user-1", fakeServer.receivedUserID)
	assert.Equal(t, int32(10), fakeServer.receivedLimit)
	assert.Equal(t, "abc", fakeServer.receivedCursor)

	assert.Contains(t, w.Body.String(), "Get home feed successfully")
	assert.Contains(t, w.Body.String(), "hello feed")
	assert.Contains(t, w.Body.String(), "next-cursor")
}

func TestFeedHandlerGetHomeFeedDefaultLimit(t *testing.T) {
	fakeServer := &fakeFeedHandlerServiceServer{}
	feedClient, cleanup := startFeedHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewFeedHandler(feedClient, zerolog.Nop())
	router := newFeedHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feeds/home", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, fakeServer.getHomeFeedCalled)
	assert.Equal(t, int32(20), fakeServer.receivedLimit)
}

func TestFeedHandlerGetHomeFeedUnauthorized(t *testing.T) {
	fakeServer := &fakeFeedHandlerServiceServer{}
	feedClient, cleanup := startFeedHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewFeedHandler(feedClient, zerolog.Nop())
	router := newFeedHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feeds/home", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, fakeServer.getHomeFeedCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestFeedHandlerGetHomeFeedInvalidLimitString(t *testing.T) {
	fakeServer := &fakeFeedHandlerServiceServer{}
	feedClient, cleanup := startFeedHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewFeedHandler(feedClient, zerolog.Nop())
	router := newFeedHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feeds/home?limit=abc", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, fakeServer.getHomeFeedCalled)
	assert.Contains(t, w.Body.String(), "Invalid limit")
}

func TestFeedHandlerGetHomeFeedInvalidLimitZero(t *testing.T) {
	fakeServer := &fakeFeedHandlerServiceServer{}
	feedClient, cleanup := startFeedHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewFeedHandler(feedClient, zerolog.Nop())
	router := newFeedHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feeds/home?limit=0", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, fakeServer.getHomeFeedCalled)
	assert.Contains(t, w.Body.String(), "Invalid limit")
}

func TestFeedHandlerGetHomeFeedInvalidLimitTooLarge(t *testing.T) {
	fakeServer := &fakeFeedHandlerServiceServer{}
	feedClient, cleanup := startFeedHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewFeedHandler(feedClient, zerolog.Nop())
	router := newFeedHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feeds/home?limit=51", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, fakeServer.getHomeFeedCalled)
	assert.Contains(t, w.Body.String(), "Invalid limit")
}

func TestFeedHandlerGetHomeFeedGRPCError(t *testing.T) {
	fakeServer := &fakeFeedHandlerServiceServer{
		err: status.Error(codes.Internal, "feed failed"),
	}
	feedClient, cleanup := startFeedHandlerTestClient(t, fakeServer)
	defer cleanup()

	handler := NewFeedHandler(feedClient, zerolog.Nop())
	router := newFeedHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/feeds/home", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.True(t, fakeServer.getHomeFeedCalled)
	assert.Contains(t, w.Body.String(), "Internal server error")
}
