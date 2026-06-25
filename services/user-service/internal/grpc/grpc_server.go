package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	userpb "github.com/trungquantrannguyen/threadly/proto/user"
	"github.com/trungquantrannguyen/threadly/services/user-service/internal/dto"
	"github.com/trungquantrannguyen/threadly/services/user-service/internal/repository"
	"github.com/trungquantrannguyen/threadly/services/user-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserServiceServer struct {
	userpb.UnimplementedUserServiceServer
	cfg         config.Config
	log         zerolog.Logger
	userService service.UserService
}

func NewUserServiceServer(cfg config.Config, log zerolog.Logger, userService service.UserService) *UserServiceServer {
	return &UserServiceServer{
		cfg:         cfg,
		log:         log,
		userService: userService,
	}
}

func (s *UserServiceServer) GetHealth(ctx context.Context, req *userpb.GetUserServiceHealthRequest) (*userpb.GetUserServiceHealthResponse, error) {
	s.log.Info().Msg("User service is healthy")
	return &userpb.GetUserServiceHealthResponse{
		Status:    "ok",
		Service:   s.cfg.ServiceName,
		Env:       s.cfg.AppEnv,
		CheckedAt: time.Now().String(),
	}, nil
}

func (s *UserServiceServer) Register(ctx context.Context, req *userpb.RegisterRequest) (*userpb.AuthResponse, error) {
	res, err := s.userService.Register(ctx, dto.RegisterRequest{
		Email:       req.GetEmail(),
		Username:    req.GetUsername(),
		Password:    req.GetPassword(),
		DisplayName: req.GetDisplayName(),
		UserAgent:   req.GetUserAgent(),
		IPAddress:   req.GetIpAddress(),
	})
	if err != nil {
		return nil, mapUserServiceError(err)
	}

	return toProtoAuthResponse(res), nil
}

func (s *UserServiceServer) Login(ctx context.Context, req *userpb.LoginRequest) (*userpb.AuthResponse, error) {
	res, err := s.userService.Login(ctx, dto.LoginRequest{
		EmailOrUsername: req.GetEmailOrUsername(),
		Password:        req.GetPassword(),
		UserAgent:       req.GetUserAgent(),
		IPAddress:       req.GetIpAddress(),
	})
	if err != nil {
		return nil, mapUserServiceError(err)
	}

	return toProtoAuthResponse(res), nil
}

func (s *UserServiceServer) RefreshToken(ctx context.Context, req *userpb.RefreshTokenRequest) (*userpb.AuthResponse, error) {
	res, err := s.userService.RefreshToken(ctx, dto.RefreshTokenRequest{
		RefreshToken: req.GetRefreshToken(),
		UserAgent:    req.GetUserAgent(),
		IPAddress:    req.GetIpAddress(),
	})
	if err != nil {
		return nil, mapUserServiceError(err)
	}

	return toProtoAuthResponse(res), nil
}

func (s *UserServiceServer) Logout(ctx context.Context, req *userpb.LogoutRequest) (*userpb.LogoutResponse, error) {
	err := s.userService.Logout(ctx, dto.LogoutRequest{
		RefreshToken: req.GetRefreshToken(),
		UserID:       req.GetUserID(),
	})
	if err != nil {
		return nil, mapUserServiceError(err)
	}

	return &userpb.LogoutResponse{}, nil
}

func (s *UserServiceServer) GetMe(ctx context.Context, req *userpb.GetMeRequest) (*userpb.AuthUserResponse, error) {
	res, err := s.userService.GetMe(ctx, req.GetUserID())
	if err != nil {
		return nil, mapUserServiceError(err)
	}

	return toProtoUserResponse(*res), nil
}

func toProtoAuthResponse(res *dto.AuthResponse) *userpb.AuthResponse {
	return &userpb.AuthResponse{
		User:         toProtoUserResponse(res.User),
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
	}
}

func toProtoUserResponse(user dto.AuthUserResponse) *userpb.AuthUserResponse {
	return &userpb.AuthUserResponse{
		Id:          user.ID,
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Role:        user.Role,
	}
}

func mapUserServiceError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, service.ErrInvalidCredential):
		return status.Error(codes.Unauthenticated, "Invalid email/username or password")
	case errors.Is(err, service.ErrInvalidRefreshToken):
		return status.Error(codes.Unauthenticated, "invalid or expired refresh token")

	case errors.Is(err, repository.ErrDuplicateUser):
		return status.Error(codes.AlreadyExists, "Email or username already exists")

	case errors.Is(err, repository.ErrUserNotFound):
		return status.Error(codes.NotFound, "User not found")

	case errors.Is(err, repository.ErrSessionNotFound):
		return status.Error(codes.NotFound, "Session not found")

	case errors.Is(err, service.ErrUnauthorized):
		return status.Error(codes.PermissionDenied, "You do not have permission to perform this action")
	default:
		return status.Error(codes.Internal, "Internal server error")
	}
}
