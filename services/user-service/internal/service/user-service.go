package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/rs/zerolog"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"github.com/trungquantrannguyen/threadly/pkg/auth"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/services/user-service/internal/dto"
	"github.com/trungquantrannguyen/threadly/services/user-service/internal/repository"
)

var (
	ErrInvalidCredential   = errors.New("Invalid email/username or password")
	ErrInvalidRefreshToken = errors.New("Invalid refresh token")
)

type UserService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (*dto.AuthResponse, error)
	Logout(ctx context.Context, req dto.LogoutRequest) error
	GetMe(ctx context.Context, userID string) (*dto.AuthUserResponse, error)
}

type userService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	cfg         config.Config
	log         zerolog.Logger
}

func NewUserService(
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	cfg config.Config,
	log zerolog.Logger,
) UserService {
	return &userService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		cfg:         cfg,
	}
}

func (s *userService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	username := strings.ToLower(strings.TrimSpace(req.Username))

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &dbmodel.User{
		Email:        email,
		Username:     username,
		PasswordHash: passwordHash,
		DisplayName:  req.DisplayName,
		Role:         "user",
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.createAuthResponse(ctx, user, req.UserAgent, req.IPAddress)
}

func (s *userService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	identifier := strings.ToLower(strings.TrimSpace(req.EmailOrUsername))

	var user *dbmodel.User
	var err error
	if strings.Contains(identifier, "@") {
		user, err = s.userRepo.FindByEmail(ctx, identifier)
	} else {
		user, err = s.userRepo.FindByUsername(ctx, identifier)
	}

	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredential
		}
		return nil, err
	}

	if err := auth.CheckPassword(req.Password, user.PasswordHash); err != nil {
		return nil, ErrInvalidCredential
	}

	return s.createAuthResponse(ctx, user, req.UserAgent, req.IPAddress)
}

func (s *userService) RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (*dto.AuthResponse, error) {
	refreshHashToken := auth.HashRefreshToken(req.RefreshToken)

	session, err := s.sessionRepo.FindActiveByRefreshTokenHash(ctx, refreshHashToken)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return nil, ErrInvalidCredential
		}
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, session.UserID.String())
	if err != nil {
		return nil, err
	}

	if err := s.sessionRepo.RevokeByID(ctx, session.ID.String()); err != nil {
		return nil, err
	}

	return s.createAuthResponse(ctx, user, req.UserAgent, req.IPAddress)
}

func (s *userService) Logout(ctx context.Context, req dto.LogoutRequest) error {
	refreshTokenHash := auth.HashRefreshToken(req.RefreshToken)

	session, err := s.sessionRepo.FindActiveByRefreshTokenHash(ctx, refreshTokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrSessionNotFound) {
			return nil
		}

		return err
	}

	return s.sessionRepo.RevokeByID(ctx, session.ID.String())
}

func (s *userService) GetMe(ctx context.Context, userID string) (*dto.AuthUserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dto.AuthUserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		AvatarURL:   stringValue(user.AvatarURL),
	}, nil
}

func (s *userService) createAuthResponse(
	ctx context.Context,
	user *dbmodel.User,
	userAgent string,
	ipAddress string,
) (*dto.AuthResponse, error) {
	accessToken, err := auth.GenerateAccessToken(
		user.ID.String(),
		user.Email,
		user.Username,
		user.Role,
		s.cfg.JWTSecret,
		s.cfg.AccessTokenTTL,
	)
	if err != nil {
		return nil, err
	}

	refreshToken, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshTokenHash := auth.HashRefreshToken(refreshToken)

	session := &dbmodel.Session{
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		UserAgent:        &userAgent,
		IPAddress:        &ipAddress,
		ExpiresAt:        time.Now().Add(s.cfg.RefreshTokenTTL),
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		User: dto.AuthUserResponse{
			ID:          user.ID.String(),
			Email:       user.Email,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        user.Role,
			AvatarURL:   stringValue(user.AvatarURL),
		},
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
