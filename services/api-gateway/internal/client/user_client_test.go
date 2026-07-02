package client

import (
	"context"
	"net"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	userpb "github.com/trungquantrannguyen/threadly/proto/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type fakeUserServiceServer struct {
	userpb.UnimplementedUserServiceServer

	healthCalled        bool
	registerCalled      bool
	loginCalled         bool
	refreshTokenCalled  bool
	logoutCalled        bool
	getMeCalled         bool
	updateProfileCalled bool
	deleteUserCalled    bool
}

func startFakeUserGRPCServer(t *testing.T, fakeServer *fakeUserServiceServer) (*UserClient, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	userpb.RegisterUserServiceServer(grpcServer, fakeServer)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		listener.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	client := &UserClient{
		conn:   conn,
		client: userpb.NewUserServiceClient(conn),
		log:    zerolog.Nop(),
	}

	cleanup := func() {
		_ = client.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}

	return client, cleanup
}

func fakeAuthUserResponse() *userpb.AuthUserResponse {
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

func fakeAuthResponse() *userpb.AuthResponse {
	return &userpb.AuthResponse{
		User:         fakeAuthUserResponse(),
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}
}

func (s *fakeUserServiceServer) GetHealth(ctx context.Context, req *userpb.GetUserServiceHealthRequest) (*userpb.GetUserServiceHealthResponse, error) {
	s.healthCalled = true

	return &userpb.GetUserServiceHealthResponse{
		Status:    "ok",
		Service:   "user-service",
		Env:       "test",
		CheckedAt: "now",
	}, nil
}

func (s *fakeUserServiceServer) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.AuthResponse, error) {
	s.registerCalled = true

	res := fakeAuthResponse()
	res.User.Email = req.GetEmail()
	res.User.Username = req.GetUsername()
	res.User.DisplayName = req.GetDisplayName()

	return res, nil
}

func (s *fakeUserServiceServer) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.AuthResponse, error) {
	s.loginCalled = true

	res := fakeAuthResponse()
	res.User.Username = req.GetEmailOrUsername()

	return res, nil
}

func (s *fakeUserServiceServer) RefreshToken(ctx context.Context, req *userpb.RefreshTokenRequest) (*userpb.AuthResponse, error) {
	s.refreshTokenCalled = true

	res := fakeAuthResponse()
	res.RefreshToken = req.GetRefreshToken()

	return res, nil
}

func (s *fakeUserServiceServer) Logout(ctx context.Context, req *userpb.LogoutRequest) (*userpb.LogoutResponse, error) {
	s.logoutCalled = true

	return &userpb.LogoutResponse{}, nil
}

func (s *fakeUserServiceServer) GetMe(ctx context.Context, req *userpb.GetMeRequest) (*userpb.AuthUserResponse, error) {
	s.getMeCalled = true

	user := fakeAuthUserResponse()
	user.Id = req.GetUserID()

	return user, nil
}

func (s *fakeUserServiceServer) UpdateProfile(ctx context.Context, req *userpb.UpdateProfileRequest) (*userpb.AuthUserResponse, error) {
	s.updateProfileCalled = true

	user := fakeAuthUserResponse()
	user.Id = req.GetUserID()
	user.DisplayName = req.GetDisplayName()
	user.Bio = req.GetBio()
	user.AvatarURL = req.GetAvatarURL()
	user.BannerURL = req.GetBannerURL()
	user.Location = req.GetLocation()
	user.WebsiteURL = req.GetWebsiteURL()

	return user, nil
}

func (s *fakeUserServiceServer) DeleteUser(ctx context.Context, req *userpb.DeleteUserRequest) (*userpb.DeleteUserResponse, error) {
	s.deleteUserCalled = true

	return &userpb.DeleteUserResponse{}, nil
}

func TestUserClientGetHealth(t *testing.T) {
	fakeServer := &fakeUserServiceServer{}
	client, cleanup := startFakeUserGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetHealth(context.Background())

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.healthCalled)
	assert.Equal(t, "ok", res.Status)
	assert.Equal(t, "user-service", res.Service)
}

func TestUserClientRegister(t *testing.T) {
	fakeServer := &fakeUserServiceServer{}
	client, cleanup := startFakeUserGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.Register(context.Background(), &userpb.RegisterRequest{
		Email:       "new@example.com",
		Username:    "newuser",
		Password:    "password123",
		DisplayName: "New User",
		UserAgent:   "test-agent",
		IpAddress:   "127.0.0.1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.registerCalled)
	assert.Equal(t, "new@example.com", res.User.Email)
	assert.Equal(t, "newuser", res.User.Username)
	assert.Equal(t, "New User", res.User.DisplayName)
	assert.Equal(t, "access-token", res.AccessToken)
	assert.Equal(t, "refresh-token", res.RefreshToken)
}

func TestUserClientLogin(t *testing.T) {
	fakeServer := &fakeUserServiceServer{}
	client, cleanup := startFakeUserGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.Login(context.Background(), &userpb.LoginRequest{
		EmailOrUsername: "testuser",
		Password:        "password123",
		UserAgent:       "test-agent",
		IpAddress:       "127.0.0.1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.loginCalled)
	assert.Equal(t, "testuser", res.User.Username)
	assert.Equal(t, "access-token", res.AccessToken)
	assert.Equal(t, "refresh-token", res.RefreshToken)
}

func TestUserClientRefreshToken(t *testing.T) {
	fakeServer := &fakeUserServiceServer{}
	client, cleanup := startFakeUserGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.RefreshToken(context.Background(), &userpb.RefreshTokenRequest{
		RefreshToken: "old-refresh-token",
		UserAgent:    "test-agent",
		IpAddress:    "127.0.0.1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.refreshTokenCalled)
	assert.Equal(t, "access-token", res.AccessToken)
	assert.Equal(t, "old-refresh-token", res.RefreshToken)
}

func TestUserClientLogout(t *testing.T) {
	fakeServer := &fakeUserServiceServer{}
	client, cleanup := startFakeUserGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.Logout(context.Background(), &userpb.LogoutRequest{
		RefreshToken: "refresh-token",
		UserID:       "user-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.logoutCalled)
}

func TestUserClientGetMe(t *testing.T) {
	fakeServer := &fakeUserServiceServer{}
	client, cleanup := startFakeUserGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.GetMe(context.Background(), &userpb.GetMeRequest{
		UserID: "user-123",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.getMeCalled)
	assert.Equal(t, "user-123", res.Id)
	assert.Equal(t, "test@example.com", res.Email)
	assert.Equal(t, "testuser", res.Username)
}

func TestUserClientUpdateProfile(t *testing.T) {
	fakeServer := &fakeUserServiceServer{}
	client, cleanup := startFakeUserGRPCServer(t, fakeServer)
	defer cleanup()

	displayName := "Updated User"
	bio := "Updated bio"
	avatarURL := "https://example.com/new-avatar.png"
	bannerURL := "https://example.com/new-banner.png"
	location := "Sydney"
	websiteURL := "https://updated.example.com"

	res, err := client.UpdateProfile(context.Background(), &userpb.UpdateProfileRequest{
		UserID:      "user-1",
		DisplayName: &displayName,
		Bio:         &bio,
		AvatarURL:   &avatarURL,
		BannerURL:   &bannerURL,
		Location:    &location,
		WebsiteURL:  &websiteURL,
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.updateProfileCalled)
	assert.Equal(t, "user-1", res.Id)
	assert.Equal(t, displayName, res.DisplayName)
	assert.Equal(t, bio, res.Bio)
	assert.Equal(t, avatarURL, res.AvatarURL)
	assert.Equal(t, bannerURL, res.BannerURL)
	assert.Equal(t, location, res.Location)
	assert.Equal(t, websiteURL, res.WebsiteURL)
}

func TestUserClientDeleteUser(t *testing.T) {
	fakeServer := &fakeUserServiceServer{}
	client, cleanup := startFakeUserGRPCServer(t, fakeServer)
	defer cleanup()

	res, err := client.DeleteUser(context.Background(), &userpb.DeleteUserRequest{
		UserID: "user-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeServer.deleteUserCalled)
}
