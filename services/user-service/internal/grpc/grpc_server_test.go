package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	userpb "github.com/trungquantrannguyen/threadly/proto/user"
	"github.com/trungquantrannguyen/threadly/services/user-service/internal/dto"
	"github.com/trungquantrannguyen/threadly/services/user-service/internal/repository"
	"github.com/trungquantrannguyen/threadly/services/user-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeUserService struct {
	authRes *dto.AuthResponse
	userRes *dto.AuthUserResponse
	err     error

	registerCalled bool
	loginCalled    bool
	refreshCalled  bool
	logoutCalled   bool
	getMeCalled    bool
	updateCalled   bool
	deleteCalled   bool

	registerReq dto.RegisterRequest
	loginReq    dto.LoginRequest
	refreshReq  dto.RefreshTokenRequest
	logoutReq   dto.LogoutRequest
	updateReq   dto.UpdateProfileRequest
	getMeUserID string
	deleteID    string
}

func (f *fakeUserService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	f.registerCalled = true
	f.registerReq = req

	if f.err != nil {
		return nil, f.err
	}

	return f.authRes, nil
}

func (f *fakeUserService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	f.loginCalled = true
	f.loginReq = req

	if f.err != nil {
		return nil, f.err
	}

	return f.authRes, nil
}

func (f *fakeUserService) RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (*dto.AuthResponse, error) {
	f.refreshCalled = true
	f.refreshReq = req

	if f.err != nil {
		return nil, f.err
	}

	return f.authRes, nil
}

func (f *fakeUserService) Logout(ctx context.Context, req dto.LogoutRequest) error {
	f.logoutCalled = true
	f.logoutReq = req

	return f.err
}

func (f *fakeUserService) GetMe(ctx context.Context, userID string) (*dto.AuthUserResponse, error) {
	f.getMeCalled = true
	f.getMeUserID = userID

	if f.err != nil {
		return nil, f.err
	}

	return f.userRes, nil
}

func (f *fakeUserService) UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) (*dto.AuthUserResponse, error) {
	f.updateCalled = true
	f.updateReq = req

	if f.err != nil {
		return nil, f.err
	}

	return f.userRes, nil
}

func (f *fakeUserService) DeleteUser(ctx context.Context, userID string) error {
	f.deleteCalled = true
	f.deleteID = userID

	return f.err
}

func newTestUserServer(fakeSvc *fakeUserService) *UserServiceServer {
	return NewUserServiceServer(
		config.Config{
			ServiceName: "user-service",
			AppEnv:      "test",
		},
		zerolog.Nop(),
		fakeSvc,
	)
}

func testAuthResponse() *dto.AuthResponse {
	return &dto.AuthResponse{
		User: dto.AuthUserResponse{
			ID:          "user-1",
			Email:       "user@example.com",
			Username:    "trungquan",
			DisplayName: "Trung Quan",
			AvatarURL:   "https://example.com/avatar.png",
			Role:        "user",
			Bio:         "hello",
			BannerURL:   "https://example.com/banner.png",
			Location:    "Melbourne",
			WebsiteURL:  "https://example.com",
		},
		AccessToken:  "access-token",
		RefreshToken: "refresh-token",
	}
}

func testUserResponse() *dto.AuthUserResponse {
	return &dto.AuthUserResponse{
		ID:          "user-1",
		Email:       "user@example.com",
		Username:    "trungquan",
		DisplayName: "Trung Quan",
		AvatarURL:   "https://example.com/avatar.png",
		Role:        "user",
		Bio:         "hello",
		BannerURL:   "https://example.com/banner.png",
		Location:    "Melbourne",
		WebsiteURL:  "https://example.com",
	}
}

func TestGetHealth(t *testing.T) {
	server := newTestUserServer(&fakeUserService{})

	res, err := server.GetHealth(context.Background(), &userpb.GetUserServiceHealthRequest{})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "ok", res.GetStatus())
	assert.Equal(t, "user-service", res.GetService())
	assert.Equal(t, "test", res.GetEnv())
	assert.NotEmpty(t, res.GetCheckedAt())
}

func TestRegisterSuccess(t *testing.T) {
	fakeSvc := &fakeUserService{
		authRes: testAuthResponse(),
	}

	server := newTestUserServer(fakeSvc)

	res, err := server.Register(context.Background(), &userpb.RegisterRequest{
		Email:       "user@example.com",
		Username:    "trungquan",
		Password:    "password123",
		DisplayName: "Trung Quan",
		UserAgent:   "Chrome",
		IpAddress:   "127.0.0.1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeSvc.registerCalled)
	assert.Equal(t, "user@example.com", fakeSvc.registerReq.Email)
	assert.Equal(t, "trungquan", fakeSvc.registerReq.Username)
	assert.Equal(t, "password123", fakeSvc.registerReq.Password)
	assert.Equal(t, "Trung Quan", fakeSvc.registerReq.DisplayName)
	assert.Equal(t, "Chrome", fakeSvc.registerReq.UserAgent)
	assert.Equal(t, "127.0.0.1", fakeSvc.registerReq.IPAddress)

	assert.Equal(t, "access-token", res.GetAccessToken())
	assert.Equal(t, "refresh-token", res.GetRefreshToken())
	require.NotNil(t, res.GetUser())
	assert.Equal(t, "user-1", res.GetUser().GetId())
	assert.Equal(t, "trungquan", res.GetUser().GetUsername())
	assert.Equal(t, "user", res.GetUser().GetRole())
}

func TestLoginSuccess(t *testing.T) {
	fakeSvc := &fakeUserService{
		authRes: testAuthResponse(),
	}

	server := newTestUserServer(fakeSvc)

	res, err := server.Login(context.Background(), &userpb.LoginRequest{
		EmailOrUsername: "trungquan",
		Password:        "password123",
		UserAgent:       "Chrome",
		IpAddress:       "127.0.0.1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeSvc.loginCalled)
	assert.Equal(t, "trungquan", fakeSvc.loginReq.EmailOrUsername)
	assert.Equal(t, "password123", fakeSvc.loginReq.Password)
	assert.Equal(t, "Chrome", fakeSvc.loginReq.UserAgent)
	assert.Equal(t, "127.0.0.1", fakeSvc.loginReq.IPAddress)

	assert.Equal(t, "access-token", res.GetAccessToken())
	assert.Equal(t, "refresh-token", res.GetRefreshToken())
}

func TestRefreshTokenSuccess(t *testing.T) {
	fakeSvc := &fakeUserService{
		authRes: testAuthResponse(),
	}

	server := newTestUserServer(fakeSvc)

	res, err := server.RefreshToken(context.Background(), &userpb.RefreshTokenRequest{
		RefreshToken: "old-refresh-token",
		UserAgent:    "Chrome",
		IpAddress:    "127.0.0.1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeSvc.refreshCalled)
	assert.Equal(t, "old-refresh-token", fakeSvc.refreshReq.RefreshToken)
	assert.Equal(t, "Chrome", fakeSvc.refreshReq.UserAgent)
	assert.Equal(t, "127.0.0.1", fakeSvc.refreshReq.IPAddress)

	assert.Equal(t, "access-token", res.GetAccessToken())
	assert.Equal(t, "refresh-token", res.GetRefreshToken())
}

func TestLogoutSuccess(t *testing.T) {
	fakeSvc := &fakeUserService{}
	server := newTestUserServer(fakeSvc)

	res, err := server.Logout(context.Background(), &userpb.LogoutRequest{
		UserID:       "user-1",
		RefreshToken: "refresh-token",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeSvc.logoutCalled)
	assert.Equal(t, "user-1", fakeSvc.logoutReq.UserID)
	assert.Equal(t, "refresh-token", fakeSvc.logoutReq.RefreshToken)
}

func TestGetMeSuccess(t *testing.T) {
	fakeSvc := &fakeUserService{
		userRes: testUserResponse(),
	}

	server := newTestUserServer(fakeSvc)

	res, err := server.GetMe(context.Background(), &userpb.GetMeRequest{
		UserID: "user-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeSvc.getMeCalled)
	assert.Equal(t, "user-1", fakeSvc.getMeUserID)

	assert.Equal(t, "user-1", res.GetId())
	assert.Equal(t, "user@example.com", res.GetEmail())
	assert.Equal(t, "trungquan", res.GetUsername())
	assert.Equal(t, "Trung Quan", res.GetDisplayName())
	assert.Equal(t, "user", res.GetRole())
	assert.Equal(t, "hello", res.GetBio())
}

func TestUpdateProfileSuccess(t *testing.T) {
	fakeSvc := &fakeUserService{
		userRes: testUserResponse(),
	}

	server := newTestUserServer(fakeSvc)

	displayName := "New Display Name"
	bio := "new bio"
	location := "Melbourne"

	res, err := server.UpdateProfile(context.Background(), &userpb.UpdateProfileRequest{
		UserID:      "user-1",
		DisplayName: &displayName,
		Bio:         &bio,
		Location:    &location,
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeSvc.updateCalled)
	assert.Equal(t, "user-1", fakeSvc.updateReq.UserID)
	require.NotNil(t, fakeSvc.updateReq.DisplayName)
	assert.Equal(t, "New Display Name", *fakeSvc.updateReq.DisplayName)
	require.NotNil(t, fakeSvc.updateReq.Bio)
	assert.Equal(t, "new bio", *fakeSvc.updateReq.Bio)
	require.NotNil(t, fakeSvc.updateReq.Location)
	assert.Equal(t, "Melbourne", *fakeSvc.updateReq.Location)

	assert.Equal(t, "user-1", res.GetId())
	assert.Equal(t, "Trung Quan", res.GetDisplayName())
}

func TestDeleteUserSuccess(t *testing.T) {
	fakeSvc := &fakeUserService{}
	server := newTestUserServer(fakeSvc)

	res, err := server.DeleteUser(context.Background(), &userpb.DeleteUserRequest{
		UserID: "user-1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, fakeSvc.deleteCalled)
	assert.Equal(t, "user-1", fakeSvc.deleteID)
}

func TestLoginInvalidCredentialMapsToUnauthenticated(t *testing.T) {
	server := newTestUserServer(&fakeUserService{
		err: service.ErrInvalidCredential,
	})

	res, err := server.Login(context.Background(), &userpb.LoginRequest{
		EmailOrUsername: "wrong",
		Password:        "wrong",
	})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "Invalid email/username or password", st.Message())
}

func TestRefreshTokenInvalidMapsToUnauthenticated(t *testing.T) {
	server := newTestUserServer(&fakeUserService{
		err: service.ErrInvalidRefreshToken,
	})

	res, err := server.RefreshToken(context.Background(), &userpb.RefreshTokenRequest{
		RefreshToken: "bad-token",
	})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Unauthenticated, st.Code())
	assert.Equal(t, "invalid or expired refresh token", st.Message())
}

func TestRegisterDuplicateUserMapsToAlreadyExists(t *testing.T) {
	server := newTestUserServer(&fakeUserService{
		err: repository.ErrDuplicateUser,
	})

	res, err := server.Register(context.Background(), &userpb.RegisterRequest{})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.AlreadyExists, st.Code())
	assert.Equal(t, "Email or username already exists", st.Message())
}

func TestGetMeUserNotFoundMapsToNotFound(t *testing.T) {
	server := newTestUserServer(&fakeUserService{
		err: repository.ErrUserNotFound,
	})

	res, err := server.GetMe(context.Background(), &userpb.GetMeRequest{
		UserID: "missing-user",
	})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "User not found", st.Message())
}

func TestLogoutSessionNotFoundMapsToNotFound(t *testing.T) {
	server := newTestUserServer(&fakeUserService{
		err: repository.ErrSessionNotFound,
	})

	res, err := server.Logout(context.Background(), &userpb.LogoutRequest{})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, "Session not found", st.Message())
}

func TestDeleteUserUnauthorizedMapsToPermissionDenied(t *testing.T) {
	server := newTestUserServer(&fakeUserService{
		err: service.ErrUnauthorized,
	})

	res, err := server.DeleteUser(context.Background(), &userpb.DeleteUserRequest{
		UserID: "user-1",
	})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.PermissionDenied, st.Code())
	assert.Equal(t, "You do not have permission to perform this action", st.Message())
}

func TestUpdateProfileInvalidProfileMapsToInvalidArgument(t *testing.T) {
	server := newTestUserServer(&fakeUserService{
		err: service.ErrInvalidProfile,
	})

	res, err := server.UpdateProfile(context.Background(), &userpb.UpdateProfileRequest{
		UserID: "user-1",
	})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Equal(t, "Invalid profile data", st.Message())
}

func TestUnknownErrorMapsToInternal(t *testing.T) {
	server := newTestUserServer(&fakeUserService{
		err: errors.New("database down"),
	})

	res, err := server.GetMe(context.Background(), &userpb.GetMeRequest{
		UserID: "user-1",
	})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
	assert.Equal(t, "Internal server error", st.Message())
}

func TestMapUserServiceErrorNil(t *testing.T) {
	assert.Nil(t, mapUserServiceError(nil))
}
