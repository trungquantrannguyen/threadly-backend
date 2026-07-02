package handlers

import (
	"bytes"
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
	contentpb "github.com/trungquantrannguyen/threadly/proto/content"
	userpb "github.com/trungquantrannguyen/threadly/proto/user"
	"github.com/trungquantrannguyen/threadly/services/api-gateway/internal/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeUserHandlerUserServiceServer struct {
	userpb.UnimplementedUserServiceServer

	err error

	healthCalled        bool
	registerCalled      bool
	loginCalled         bool
	refreshTokenCalled  bool
	logoutCalled        bool
	getMeCalled         bool
	updateProfileCalled bool
	deleteUserCalled    bool

	receivedEmail           string
	receivedUsername        string
	receivedDisplayName     string
	receivedEmailOrUsername string
	receivedRefreshToken    string
	receivedUserID          string
	receivedUserAgent       string
	receivedIPAddress       string
}

type fakeUserHandlerContentServiceServer struct {
	contentpb.UnimplementedContentServiceServer

	err error

	getUserTimelineCalled bool
	receivedUserID        string
	receivedLimit         int32
}

func startUserHandlerUserClient(t *testing.T, fakeServer *fakeUserHandlerUserServiceServer) (*client.UserClient, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, fakeServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	userClient, err := client.NewUserClient(config.Config{
		UserServiceGRPCAddr: listener.Addr().String(),
	}, zerolog.Nop())
	require.NoError(t, err)

	cleanup := func() {
		_ = userClient.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return userClient, cleanup
}

func startUserHandlerContentClient(t *testing.T, fakeServer *fakeUserHandlerContentServiceServer) (*client.ContentClient, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	contentpb.RegisterContentServiceServer(grpcServer, fakeServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	contentClient, err := client.NewContentClient(config.Config{
		ContentServiceGRPCAddr: listener.Addr().String(),
	}, zerolog.Nop())
	require.NoError(t, err)

	cleanup := func() {
		_ = contentClient.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return contentClient, cleanup
}

func fakeUserHandlerAuthUser() *userpb.AuthUserResponse {
	return &userpb.AuthUserResponse{
		Id:          "user-1",
		Email:       "test@example.com",
		Username:    "testuser",
		DisplayName: "Test User",
		Role:        "user",
		AvatarURL:   "https://example.com/avatar.png",
		Bio:         "hello",
		BannerURL:   "https://example.com/banner.png",
		Location:    "Melbourne",
		WebsiteURL:  "https://example.com",
	}
}

func fakeUserHandlerAuthResponse() *userpb.AuthResponse {
	return &userpb.AuthResponse{
		User:         fakeUserHandlerAuthUser(),
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}
}

func fakeUserHandlerTimelinePost() *contentpb.PostResponse {
	return &contentpb.PostResponse{
		Id:         "post-1",
		AuthorId:   "user-1",
		Content:    "hello timeline",
		Visibility: "public",
		CreatedAt:  "2026-07-02T10:00:00Z",
		UpdatedAt:  "2026-07-02T10:01:00Z",
		Author: &contentpb.UserSummary{
			Id:          "user-1",
			Username:    "testuser",
			DisplayName: "Test User",
			AvatarUrl:   "https://example.com/avatar.png",
		},
		Media: []*contentpb.MediaResponse{
			{
				Id:        "media-1",
				Url:       "https://example.com/media.png",
				MimeType:  "image/png",
				SizeBytes: 1024,
				Width:     640,
				Height:    480,
			},
		},
	}
}

func (s *fakeUserHandlerUserServiceServer) GetHealth(ctx context.Context, req *userpb.GetUserServiceHealthRequest) (*userpb.GetUserServiceHealthResponse, error) {
	s.healthCalled = true

	if s.err != nil {
		return nil, s.err
	}

	return &userpb.GetUserServiceHealthResponse{
		Service:   "user-service",
		Status:    "ok",
		Env:       "test",
		CheckedAt: "now",
	}, nil
}

func (s *fakeUserHandlerUserServiceServer) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.AuthResponse, error) {
	s.registerCalled = true
	s.receivedEmail = req.GetEmail()
	s.receivedUsername = req.GetUsername()
	s.receivedDisplayName = req.GetDisplayName()
	s.receivedUserAgent = req.GetUserAgent()
	s.receivedIPAddress = req.GetIpAddress()

	if s.err != nil {
		return nil, s.err
	}

	res := fakeUserHandlerAuthResponse()
	res.User.Email = req.GetEmail()
	res.User.Username = req.GetUsername()
	res.User.DisplayName = req.GetDisplayName()

	return res, nil
}

func (s *fakeUserHandlerUserServiceServer) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.AuthResponse, error) {
	s.loginCalled = true
	s.receivedEmailOrUsername = req.GetEmailOrUsername()
	s.receivedUserAgent = req.GetUserAgent()
	s.receivedIPAddress = req.GetIpAddress()

	if s.err != nil {
		return nil, s.err
	}

	res := fakeUserHandlerAuthResponse()
	res.User.Username = req.GetEmailOrUsername()

	return res, nil
}

func (s *fakeUserHandlerUserServiceServer) RefreshToken(ctx context.Context, req *userpb.RefreshTokenRequest) (*userpb.AuthResponse, error) {
	s.refreshTokenCalled = true
	s.receivedRefreshToken = req.GetRefreshToken()
	s.receivedUserAgent = req.GetUserAgent()
	s.receivedIPAddress = req.GetIpAddress()

	if s.err != nil {
		return nil, s.err
	}

	res := fakeUserHandlerAuthResponse()
	res.RefreshToken = req.GetRefreshToken()

	return res, nil
}

func (s *fakeUserHandlerUserServiceServer) Logout(ctx context.Context, req *userpb.LogoutRequest) (*userpb.LogoutResponse, error) {
	s.logoutCalled = true
	s.receivedRefreshToken = req.GetRefreshToken()
	s.receivedUserID = req.GetUserID()

	if s.err != nil {
		return nil, s.err
	}

	return &userpb.LogoutResponse{}, nil
}

func (s *fakeUserHandlerUserServiceServer) GetMe(ctx context.Context, req *userpb.GetMeRequest) (*userpb.AuthUserResponse, error) {
	s.getMeCalled = true
	s.receivedUserID = req.GetUserID()

	if s.err != nil {
		return nil, s.err
	}

	user := fakeUserHandlerAuthUser()
	user.Id = req.GetUserID()

	return user, nil
}

func (s *fakeUserHandlerUserServiceServer) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.AuthUserResponse, error) {
	s.updateProfileCalled = true
	s.receivedUserID = req.GetUserID()
	s.receivedDisplayName = req.GetDisplayName()

	if s.err != nil {
		return nil, s.err
	}

	user := fakeUserHandlerAuthUser()
	user.Id = req.GetUserID()
	user.DisplayName = req.GetDisplayName()
	user.Bio = req.GetBio()
	user.AvatarURL = req.GetAvatarURL()
	user.BannerURL = req.GetBannerURL()
	user.Location = req.GetLocation()
	user.WebsiteURL = req.GetWebsiteURL()

	return user, nil
}

func (s *fakeUserHandlerUserServiceServer) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	s.deleteUserCalled = true
	s.receivedUserID = req.GetUserID()

	if s.err != nil {
		return nil, s.err
	}

	return &userpb.DeleteUserResponse{}, nil
}

func (s *fakeUserHandlerContentServiceServer) GetUserTimeline(ctx context.Context, req *contentpb.GetUserTimelineRequest) (*contentpb.TimelineResponse, error) {
	s.getUserTimelineCalled = true
	s.receivedUserID = req.GetUserId()
	s.receivedLimit = req.GetLimit()

	if s.err != nil {
		return nil, s.err
	}

	return &contentpb.TimelineResponse{
		Items: []*contentpb.TimelineItemResponse{
			{
				Type: "post",
				Post: fakeUserHandlerTimelinePost(),
			},
			{
				Type:       "repost",
				Post:       fakeUserHandlerTimelinePost(),
				RepostedAt: "2026-07-02T11:00:00Z",
			},
		},
	}, nil
}

func newUserHandlerTestRouter(handler *UserHandler, withUser bool) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()

	if withUser {
		router.Use(func(c *gin.Context) {
			c.Set(middleware.UserIDKey, "user-1")
			c.Next()
		})
	}

	router.GET("/users/health", handler.GetHealth)
	router.POST("/users/register", handler.Register)
	router.POST("/users/login", handler.Login)
	router.POST("/users/refresh", handler.RefreshToken)
	router.POST("/users/logout", handler.Logout)
	router.GET("/users/me", handler.GetMe)
	router.PATCH("/users/me", handler.UpdateProfile)
	router.DELETE("/users/me", handler.DeleteUser)

	return router
}

func newUserHandlerForTest(t *testing.T, userServer *fakeUserHandlerUserServiceServer, contentServer *fakeUserHandlerContentServiceServer) (*UserHandler, func()) {
	t.Helper()

	userClient, cleanupUser := startUserHandlerUserClient(t, userServer)
	contentClient, cleanupContent := startUserHandlerContentClient(t, contentServer)

	handler := NewUserHandler(userClient, zerolog.Nop(), contentClient)

	cleanup := func() {
		cleanupUser()
		cleanupContent()
	}

	return handler, cleanup
}

func TestUserHandlerGetHealthSuccess(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, userServer.healthCalled)
	assert.Contains(t, w.Body.String(), "user service available")
	assert.Contains(t, w.Body.String(), "user-service")
}

func TestUserHandlerGetHealthUnavailable(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{
		err: errors.New("user service down"),
	}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/health", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.True(t, userServer.healthCalled)
	assert.Contains(t, w.Body.String(), "user service unavailable")
}

func TestUserHandlerRegisterSuccess(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	body := bytes.NewBufferString(`{
		"email": "new@example.com",
		"username": "newuser",
		"password": "password123",
		"display_name": "New User"
	}`)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/register", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.True(t, userServer.registerCalled)
	assert.Equal(t, "new@example.com", userServer.receivedEmail)
	assert.Equal(t, "newuser", userServer.receivedUsername)
	assert.Equal(t, "New User", userServer.receivedDisplayName)
	assert.Equal(t, "test-agent", userServer.receivedUserAgent)
	assert.Contains(t, w.Body.String(), "Register successfully")
	assert.Contains(t, w.Body.String(), "access-token")
	assert.Contains(t, w.Body.String(), "refresh-token")
}

func TestUserHandlerRegisterInvalidBody(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/register", bytes.NewBufferString(`bad-json`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, userServer.registerCalled)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestUserHandlerRegisterAlreadyExists(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{
		err: status.Error(codes.AlreadyExists, "user already exists"),
	}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/register", bytes.NewBufferString(`{
		"email": "new@example.com",
		"username": "newuser",
		"password": "password123",
		"display_name": "New User"
	}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.True(t, userServer.registerCalled)
	assert.Contains(t, w.Body.String(), "user already exists")
}

func TestUserHandlerLoginSuccess(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewBufferString(`{
		"email_or_username": "testuser",
  		"password": "password123"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, userServer.loginCalled)
	assert.Equal(t, "testuser", userServer.receivedEmailOrUsername)
	assert.Equal(t, "test-agent", userServer.receivedUserAgent)
	assert.Contains(t, w.Body.String(), "Login successfully")
	assert.Contains(t, w.Body.String(), "access-token")
	assert.Contains(t, w.Body.String(), "refresh-token")
}

func TestUserHandlerLoginInvalidBody(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewBufferString(`bad-json`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, userServer.loginCalled)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestUserHandlerLoginUnauthenticated(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{
		err: status.Error(codes.Unauthenticated, "invalid credentials"),
	}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewBufferString(`{
		"email_or_username": "testuser",
  		"password": "password123"
	}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, userServer.loginCalled)
	assert.Contains(t, w.Body.String(), "invalid credentials")
}

func TestUserHandlerRefreshTokenSuccess(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/refresh", bytes.NewBufferString(`{
		"refresh_token": "old-refresh-token"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, userServer.refreshTokenCalled)
	assert.Equal(t, "old-refresh-token", userServer.receivedRefreshToken)
	assert.Contains(t, w.Body.String(), "Token refreshed successfully")
	assert.Contains(t, w.Body.String(), "access-token")
}

func TestUserHandlerRefreshTokenInvalidBody(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/refresh", bytes.NewBufferString(`bad-json`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, userServer.refreshTokenCalled)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestUserHandlerLogoutSuccess(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/logout", bytes.NewBufferString(`{
		"refresh_token": "refresh-token"
	}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, userServer.logoutCalled)
	assert.Equal(t, "user-1", userServer.receivedUserID)
	assert.Equal(t, "refresh-token", userServer.receivedRefreshToken)
	assert.Contains(t, w.Body.String(), "Logged out successfully")
}

func TestUserHandlerLogoutUnauthorized(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/logout", bytes.NewBufferString(`{
		"refresh_token": "refresh-token"
	}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, userServer.logoutCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestUserHandlerGetMeSuccess(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, userServer.getMeCalled)
	assert.True(t, contentServer.getUserTimelineCalled)
	assert.Equal(t, "user-1", userServer.receivedUserID)
	assert.Equal(t, "user-1", contentServer.receivedUserID)
	assert.Equal(t, int32(20), contentServer.receivedLimit)
	assert.Contains(t, w.Body.String(), "Get user successfully")
	assert.Contains(t, w.Body.String(), "hello timeline")
}

func TestUserHandlerGetMeUnauthorized(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, userServer.getMeCalled)
	assert.False(t, contentServer.getUserTimelineCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestUserHandlerGetMeUserServiceNotFound(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{
		err: status.Error(codes.NotFound, "user not found"),
	}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, userServer.getMeCalled)
	assert.False(t, contentServer.getUserTimelineCalled)
	assert.Contains(t, w.Body.String(), "user not found")
}

func TestUserHandlerGetMeTimelineError(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{
		err: status.Error(codes.Internal, "timeline failed"),
	}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users/me", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.True(t, userServer.getMeCalled)
	assert.True(t, contentServer.getUserTimelineCalled)
	assert.Contains(t, w.Body.String(), "Internal server error")
}

func TestUserHandlerUpdateProfileSuccess(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewBufferString(`{
		"display_name": "Updated User",
		"bio": "Updated bio",
		"avatar_url": "https://example.com/avatar.png",
		"banner_url": "https://example.com/banner.png",
		"location": "Sydney",
		"website_url": "https://updated.example.com"
	}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, userServer.updateProfileCalled)
	assert.Equal(t, "user-1", userServer.receivedUserID)
	assert.Equal(t, "Updated User", userServer.receivedDisplayName)
	assert.Contains(t, w.Body.String(), "Profile updated successfully")
	assert.Contains(t, w.Body.String(), "Updated User")
}

func TestUserHandlerUpdateProfileInvalidBody(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewBufferString(`bad-json`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, userServer.updateProfileCalled)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestUserHandlerUpdateProfileUnauthorized(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewBufferString(`{"displayName":"Updated"}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, userServer.updateProfileCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestUserHandlerDeleteUserSuccess(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/users/me", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, userServer.deleteUserCalled)
	assert.Equal(t, "user-1", userServer.receivedUserID)
	assert.Contains(t, w.Body.String(), "User deleted successfully")
}

func TestUserHandlerDeleteUserUnauthorized(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/users/me", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, userServer.deleteUserCalled)
	assert.Contains(t, w.Body.String(), "Unauthorized")
}

func TestUserHandlerDeleteUserNotFound(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{
		err: status.Error(codes.NotFound, "user not found"),
	}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/users/me", nil)

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, userServer.deleteUserCalled)
	assert.Contains(t, w.Body.String(), "user not found")
}

func TestUserHandlerRefreshTokenGRPCError(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{
		err: status.Error(codes.Unauthenticated, "invalid refresh token"),
	}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/refresh", bytes.NewBufferString(`{
		"refresh_token": "bad-refresh-token"
	}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, userServer.refreshTokenCalled)
	assert.Contains(t, w.Body.String(), "invalid refresh token")
}

func TestUserHandlerLogoutGRPCError(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{
		err: status.Error(codes.Unauthenticated, "invalid refresh token"),
	}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/logout", bytes.NewBufferString(`{
		"refresh_token": "bad-refresh-token"
	}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
	assert.True(t, userServer.logoutCalled)
	assert.Contains(t, w.Body.String(), "invalid refresh token")
}

func TestUserHandlerUpdateProfileGRPCError(t *testing.T) {
	userServer := &fakeUserHandlerUserServiceServer{
		err: status.Error(codes.NotFound, "user not found"),
	}
	contentServer := &fakeUserHandlerContentServiceServer{}
	handler, cleanup := newUserHandlerForTest(t, userServer, contentServer)
	defer cleanup()

	router := newUserHandlerTestRouter(handler, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewBufferString(`{
		"display_name": "Updated User",
		"bio": "Updated bio"
	}`))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.True(t, userServer.updateProfileCalled)
	assert.Contains(t, w.Body.String(), "user not found")
}
