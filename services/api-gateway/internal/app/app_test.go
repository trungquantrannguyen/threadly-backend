package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/config"
)

func newTestConfig() config.Config {
	return config.Config{
		AppEnv:      "test",
		ServiceName: "api-gateway",

		UserServiceGRPCAddr:         "127.0.0.1:1",
		ContentServiceGRPCAddr:      "127.0.0.1:2",
		FeedServiceGRPCAddr:         "127.0.0.1:3",
		NotificationServiceGRPCAddr: "127.0.0.1:4",
		StorageServiceGRPCAddr:      "127.0.0.1:5",

		RedisAddr: "127.0.0.1:6379",
		JWTSecret: "test-secret",
	}
}

func TestNewRouterCreatesRouterAndCleanup(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, cleanup := NewRouter(newTestConfig(), zerolog.Nop())

	require.NotNil(t, router)
	require.NotNil(t, cleanup)

	err := cleanup()
	require.NoError(t, err)
}

func TestNewRouterRootHealthRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, cleanup := NewRouter(newTestConfig(), zerolog.Nop())
	defer func() {
		require.NoError(t, cleanup())
	}()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "api-gateway")
}

func TestNewRouterRegistersExpectedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, cleanup := NewRouter(newTestConfig(), zerolog.Nop())
	defer func() {
		require.NoError(t, cleanup())
	}()

	routes := router.Routes()
	routeMap := make(map[string]bool)

	for _, route := range routes {
		routeMap[route.Method+" "+route.Path] = true
	}

	expectedRoutes := []string{
		"GET /health",
		"GET /api/health",

		"GET /api/users/health",
		"POST /api/users/register",
		"POST /api/users/login",
		"POST /api/users/refresh",
		"POST /api/users/logout",
		"GET /api/users/me",
		"PATCH /api/users/me",
		"DELETE /api/users/me",

		"GET /api/contents/health",
		"POST /api/contents/posts",
		"GET /api/contents/posts/:postID",
		"PATCH /api/contents/posts/:postID",
		"DELETE /api/contents/posts/:postID",
		"POST /api/contents/posts/:postID/replies",
		"GET /api/contents/posts/:postID/replies",
		"POST /api/contents/posts/:postID/likes",
		"DELETE /api/contents/posts/:postID/likes",
		"POST /api/contents/posts/:postID/bookmarks",
		"DELETE /api/contents/posts/:postID/bookmarks",
		"POST /api/contents/posts/:postID/reposts",
		"DELETE /api/contents/posts/:postID/reposts",
		"POST /api/contents/users/:userID/follow",
		"DELETE /api/contents/users/:userID/follow",
		"GET /api/contents/users/:userID/followers",
		"GET /api/contents/users/:userID/following",

		"GET /api/feeds/health",
		"GET /api/feeds/home",

		"GET /api/notifications/health",
		"GET /api/notifications",
		"GET /api/notifications/unread-count",
		"PATCH /api/notifications/:id/read",
		"PATCH /api/notifications/read-all",

		"GET /api/storages/health",
		"POST /api/storages/upload",
	}

	for _, expectedRoute := range expectedRoutes {
		assert.True(t, routeMap[expectedRoute], "missing route: %s", expectedRoute)
	}
}

func TestNewRouterProductionMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := newTestConfig()
	cfg.AppEnv = "production"

	router, cleanup := NewRouter(cfg, zerolog.Nop())

	require.NotNil(t, router)
	require.NoError(t, cleanup())
	assert.Equal(t, gin.ReleaseMode, gin.Mode())
}
