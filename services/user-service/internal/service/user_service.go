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
	ErrUnauthorized        = errors.New("Unauthorized")
	ErrInvalidProfile      = errors.New("invalid profile data")
)

type UserService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (*dto.AuthResponse, error)
	Logout(ctx context.Context, req dto.LogoutRequest) error
	GetMe(ctx context.Context, userID string) (*dto.AuthUserResponse, error)
	UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) (*dto.AuthUserResponse, error)
	DeleteUser(ctx context.Context, userID string) error
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
			return nil, ErrInvalidRefreshToken
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
	if req.RefreshToken == "" {
		return ErrInvalidRefreshToken
	}

	refreshTokenHash := auth.HashRefreshToken(req.RefreshToken)

	session, err := s.sessionRepo.FindActiveByRefreshTokenHash(ctx, refreshTokenHash)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			return nil
		}

		return err
	}

	if session.UserID.String() != req.UserID {
		return ErrUnauthorized
	}

	if err := s.sessionRepo.RevokeByID(ctx, session.ID.String()); err != nil {
		return err
	}

	return nil
}

func (s *userService) GetMe(ctx context.Context, userID string) (*dto.AuthUserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return toAuthUserResponse(user), nil
}

func (s *userService) UpdateProfile(ctx context.Context, req dto.UpdateProfileRequest) (*dto.AuthUserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}

	if req.DisplayName != nil {
		displayName := strings.TrimSpace(*req.DisplayName)
		if displayName == "" || len(displayName) > 80 {
			return nil, ErrInvalidProfile
		}
		updates["display_name"] = displayName
	}

	if value, ok, err := nullableProfileField(req.Bio, 280); err != nil {
		return nil, err
	} else if ok {
		updates["bio"] = value
	}

	if value, ok, err := nullableProfileField(req.AvatarURL, 2048); err != nil {
		return nil, err
	} else if ok {
		updates["avatar_url"] = value
	}

	if value, ok, err := nullableProfileField(req.BannerURL, 2048); err != nil {
		return nil, err
	} else if ok {
		updates["banner_url"] = value
	}

	if value, ok, err := nullableProfileField(req.Location, 100); err != nil {
		return nil, err
	} else if ok {
		updates["location"] = value
	}

	if value, ok, err := nullableProfileField(req.WebsiteURL, 2048); err != nil {
		return nil, err
	} else if ok {
		updates["website_url"] = value
	}

	if err := s.userRepo.UpdateProfile(ctx, user, updates); err != nil {
		return nil, err
	}

	updatedUser, err := s.userRepo.FindByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	return toAuthUserResponse(updatedUser), nil
}

func (s *userService) DeleteUser(ctx context.Context, userID string) error {
	if _, err := s.userRepo.FindByID(ctx, userID); err != nil {
		return err
	}

	if err := s.userRepo.DeleteByID(ctx, userID); err != nil {
		return err
	}

	return s.sessionRepo.RevokeAllByUserID(ctx, userID)
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
	returnUser := toAuthUserResponse(user)

	return &dto.AuthResponse{
		User:         *returnUser,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func nullableProfileField(value *string, max int) (interface{}, bool, error) {
	if value == nil {
		return nil, false, nil
	}

	trimmed := strings.TrimSpace(*value)
	if len(trimmed) > max {
		return nil, false, ErrInvalidProfile
	}

	if trimmed == "" {
		return nil, true, nil
	}

	return trimmed, true, nil
}

func toAuthUserResponse(user *dbmodel.User) *dto.AuthUserResponse {
	return &dto.AuthUserResponse{
		ID:          user.ID.String(),
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
		AvatarURL:   stringValue(user.AvatarURL),
		Bio:         stringValue(user.Bio),
		BannerURL:   stringValue(user.BannerURL),
		Location:    stringValue(user.Location),
		WebsiteURL:  stringValue(user.WebsiteURL),
	}
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
