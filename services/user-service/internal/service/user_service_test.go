package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	"github.com/trungquantrannguyen/threadly/services/user-service/internal/dto"
	"github.com/trungquantrannguyen/threadly/services/user-service/internal/repository"
)

type fakeUserRepo struct {
	usersByID       map[string]*dbmodel.User
	usersByEmail    map[string]*dbmodel.User
	usersByUsername map[string]*dbmodel.User

	createErr         error
	findByIDErr       error
	findByEmailErr    error
	findByUsernameErr error
	updateErr         error
	deleteErr         error

	createdUser *dbmodel.User
	updatedUser *dbmodel.User
	updates     map[string]interface{}
	deletedID   string
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		usersByID:       make(map[string]*dbmodel.User),
		usersByEmail:    make(map[string]*dbmodel.User),
		usersByUsername: make(map[string]*dbmodel.User),
	}
}

func (f *fakeUserRepo) Create(ctx context.Context, user *dbmodel.User) error {
	if f.createErr != nil {
		return f.createErr
	}

	if _, exists := f.usersByEmail[strings.ToLower(user.Email)]; exists {
		return repository.ErrDuplicateUser
	}

	if _, exists := f.usersByUsername[strings.ToLower(user.Username)]; exists {
		return repository.ErrDuplicateUser
	}

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	if user.Role == "" {
		user.Role = "user"
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}

	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Now().UTC()
	}

	copied := *user
	f.createdUser = &copied

	f.usersByID[user.ID.String()] = &copied
	f.usersByEmail[strings.ToLower(user.Email)] = &copied
	f.usersByUsername[strings.ToLower(user.Username)] = &copied

	return nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id string) (*dbmodel.User, error) {
	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}

	user, exists := f.usersByID[id]
	if !exists {
		return nil, repository.ErrUserNotFound
	}

	return user, nil
}

func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*dbmodel.User, error) {
	if f.findByEmailErr != nil {
		return nil, f.findByEmailErr
	}

	user, exists := f.usersByEmail[strings.ToLower(email)]
	if !exists {
		return nil, repository.ErrUserNotFound
	}

	return user, nil
}

func (f *fakeUserRepo) FindByUsername(ctx context.Context, username string) (*dbmodel.User, error) {
	if f.findByUsernameErr != nil {
		return nil, f.findByUsernameErr
	}

	user, exists := f.usersByUsername[strings.ToLower(username)]
	if !exists {
		return nil, repository.ErrUserNotFound
	}

	return user, nil
}

func (f *fakeUserRepo) UpdateProfile(ctx context.Context, user *dbmodel.User, updates map[string]interface{}) error {
	if f.updateErr != nil {
		return f.updateErr
	}

	f.updatedUser = user
	f.updates = updates

	if value, ok := updates["display_name"].(string); ok {
		user.DisplayName = value
	}

	if value, ok := updates["bio"].(string); ok {
		user.Bio = &value
	}

	if value, ok := updates["avatar_url"].(string); ok {
		user.AvatarURL = &value
	}

	if value, ok := updates["banner_url"].(string); ok {
		user.BannerURL = &value
	}

	if value, ok := updates["location"].(string); ok {
		user.Location = &value
	}

	if value, ok := updates["website_url"].(string); ok {
		user.WebsiteURL = &value
	}

	return nil
}

func (f *fakeUserRepo) DeleteByID(ctx context.Context, id string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}

	f.deletedID = id

	if _, exists := f.usersByID[id]; !exists {
		return repository.ErrUserNotFound
	}

	delete(f.usersByID, id)

	return nil
}

type fakeSessionRepo struct {
	sessionsByID   map[string]*dbmodel.Session
	sessionsByHash map[string]*dbmodel.Session

	latestSession *dbmodel.Session

	createErr        error
	findActiveErr    error
	revokeErr        error
	revokeAllErr     error
	deleteExpiredErr error

	createdSessions []*dbmodel.Session
	revokedID       string
	revokedUserID   string
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{
		sessionsByID:   make(map[string]*dbmodel.Session),
		sessionsByHash: make(map[string]*dbmodel.Session),
	}
}

func (f *fakeSessionRepo) Create(ctx context.Context, session *dbmodel.Session) error {
	if f.createErr != nil {
		return f.createErr
	}

	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now().UTC()
	}

	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)
	}

	copied := *session

	f.latestSession = &copied
	f.createdSessions = append(f.createdSessions, &copied)
	f.sessionsByID[session.ID.String()] = &copied
	f.sessionsByHash[session.RefreshTokenHash] = &copied

	return nil
}

func (f *fakeSessionRepo) FindActiveByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*dbmodel.Session, error) {
	if f.findActiveErr != nil {
		return nil, f.findActiveErr
	}

	session, exists := f.sessionsByHash[refreshTokenHash]
	if !exists && f.latestSession != nil {
		session = f.latestSession
		exists = true
	}

	if !exists {
		return nil, repository.ErrSessionNotFound
	}

	if session.RevokedAt != nil || session.ExpiresAt.Before(time.Now()) {
		return nil, repository.ErrSessionNotFound
	}

	return session, nil
}

func (f *fakeSessionRepo) RevokeByID(ctx context.Context, sessionID string) error {
	if f.revokeErr != nil {
		return f.revokeErr
	}

	session, exists := f.sessionsByID[sessionID]
	if !exists {
		return repository.ErrSessionNotFound
	}

	if session.RevokedAt != nil {
		return repository.ErrSessionNotFound
	}

	now := time.Now()
	session.RevokedAt = &now
	f.revokedID = sessionID

	return nil
}

func (f *fakeSessionRepo) RevokeAllByUserID(ctx context.Context, userID string) error {
	if f.revokeAllErr != nil {
		return f.revokeAllErr
	}

	now := time.Now()
	f.revokedUserID = userID

	for _, session := range f.sessionsByID {
		if session.UserID.String() == userID && session.RevokedAt == nil {
			session.RevokedAt = &now
		}
	}

	return nil
}

func (f *fakeSessionRepo) DeleteExpired(ctx context.Context) error {
	return f.deleteExpiredErr
}

func newTestUserService(userRepo *fakeUserRepo, sessionRepo *fakeSessionRepo) UserService {
	return NewUserService(
		userRepo,
		sessionRepo,
		config.Config{
			ServiceName:     "user-service",
			AppEnv:          "test",
			JWTSecret:       "test-secret",
			AccessTokenTTL:  60 * time.Minute,
			RefreshTokenTTL: 7 * 24 * time.Hour,
		},
		zerolog.Nop(),
	)
}

func TestRegisterSuccess(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	res, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "user@example.com",
		Username:    "trungquan",
		Password:    "Password123!",
		DisplayName: "Trung Quan",
		UserAgent:   "Chrome",
		IPAddress:   "127.0.0.1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "user@example.com", res.User.Email)
	assert.Equal(t, "trungquan", res.User.Username)
	assert.Equal(t, "Trung Quan", res.User.DisplayName)
	assert.Equal(t, "user", res.User.Role)
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)

	require.NotNil(t, userRepo.createdUser)
	assert.NotEmpty(t, userRepo.createdUser.PasswordHash)
	assert.NotEqual(t, "Password123!", userRepo.createdUser.PasswordHash)

	require.Len(t, sessionRepo.createdSessions, 1)
	assert.NotEmpty(t, sessionRepo.createdSessions[0].RefreshTokenHash)
	assert.NotNil(t, sessionRepo.createdSessions[0].UserAgent)
	assert.NotNil(t, sessionRepo.createdSessions[0].IPAddress)
}

func TestRegisterDuplicateUserReturnsError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "duplicate@example.com",
		Username:    "duplicate",
		Password:    "Password123!",
		DisplayName: "Duplicate",
	})

	require.NoError(t, err)

	res, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "duplicate@example.com",
		Username:    "another",
		Password:    "Password123!",
		DisplayName: "Another",
	})

	require.ErrorIs(t, err, repository.ErrDuplicateUser)
	assert.Nil(t, res)
}

func TestLoginSuccessWithEmail(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "login@example.com",
		Username:    "loginuser",
		Password:    "Password123!",
		DisplayName: "Login User",
	})
	require.NoError(t, err)

	res, err := svc.Login(context.Background(), dto.LoginRequest{
		EmailOrUsername: "login@example.com",
		Password:        "Password123!",
		UserAgent:       "Chrome",
		IPAddress:       "127.0.0.1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "login@example.com", res.User.Email)
	assert.Equal(t, "loginuser", res.User.Username)
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)
}

func TestLoginSuccessWithUsername(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "username-login@example.com",
		Username:    "usernameuser",
		Password:    "Password123!",
		DisplayName: "Username User",
	})
	require.NoError(t, err)

	res, err := svc.Login(context.Background(), dto.LoginRequest{
		EmailOrUsername: "usernameuser",
		Password:        "Password123!",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "usernameuser", res.User.Username)
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)
}

func TestLoginWrongPasswordReturnsInvalidCredential(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "wrong-password@example.com",
		Username:    "wrongpassword",
		Password:    "Password123!",
		DisplayName: "Wrong Password",
	})
	require.NoError(t, err)

	res, err := svc.Login(context.Background(), dto.LoginRequest{
		EmailOrUsername: "wrong-password@example.com",
		Password:        "WrongPassword123!",
	})

	require.ErrorIs(t, err, ErrInvalidCredential)
	assert.Nil(t, res)
}

func TestLoginMissingUserReturnsInvalidCredential(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	res, err := svc.Login(context.Background(), dto.LoginRequest{
		EmailOrUsername: "missing@example.com",
		Password:        "Password123!",
	})

	require.ErrorIs(t, err, ErrInvalidCredential)
	assert.Nil(t, res)
}

func TestRefreshTokenSuccess(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "refresh@example.com",
		Username:    "refreshuser",
		Password:    "Password123!",
		DisplayName: "Refresh User",
	})
	require.NoError(t, err)

	res, err := svc.RefreshToken(context.Background(), dto.RefreshTokenRequest{
		RefreshToken: registerRes.RefreshToken,
		UserAgent:    "Chrome",
		IPAddress:    "127.0.0.1",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "refresh@example.com", res.User.Email)
	assert.NotEmpty(t, res.AccessToken)
	assert.NotEmpty(t, res.RefreshToken)
}

func TestRefreshTokenInvalidReturnsInvalidRefreshToken(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	res, err := svc.RefreshToken(context.Background(), dto.RefreshTokenRequest{
		RefreshToken: "invalid-token",
	})

	require.ErrorIs(t, err, ErrInvalidRefreshToken)
	assert.Nil(t, res)
}

func TestLogoutSuccess(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "logout@example.com",
		Username:    "logoutuser",
		Password:    "Password123!",
		DisplayName: "Logout User",
	})
	require.NoError(t, err)

	err = svc.Logout(context.Background(), dto.LogoutRequest{
		UserID:       registerRes.User.ID,
		RefreshToken: registerRes.RefreshToken,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, sessionRepo.revokedID)
}

func TestLogoutInvalidTokenReturnsSessionNotFound(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	err := svc.Logout(context.Background(), dto.LogoutRequest{
		UserID:       uuid.NewString(),
		RefreshToken: "bad-token",
	})

	require.ErrorIs(t, err, repository.ErrSessionNotFound)
}

func TestGetMeSuccess(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "me@example.com",
		Username:    "meuser",
		Password:    "Password123!",
		DisplayName: "Me User",
	})
	require.NoError(t, err)

	res, err := svc.GetMe(context.Background(), registerRes.User.ID)

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "me@example.com", res.Email)
	assert.Equal(t, "meuser", res.Username)
}

func TestGetMeUserNotFound(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	res, err := svc.GetMe(context.Background(), uuid.NewString())

	require.ErrorIs(t, err, repository.ErrUserNotFound)
	assert.Nil(t, res)
}

func TestUpdateProfileSuccess(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "profile@example.com",
		Username:    "profileuser",
		Password:    "Password123!",
		DisplayName: "Profile User",
	})
	require.NoError(t, err)

	displayName := "Updated User"
	bio := "new bio"
	location := "Melbourne"

	res, err := svc.UpdateProfile(context.Background(), dto.UpdateProfileRequest{
		UserID:      registerRes.User.ID,
		DisplayName: &displayName,
		Bio:         &bio,
		Location:    &location,
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "Updated User", res.DisplayName)
	assert.Equal(t, "new bio", res.Bio)
	assert.Equal(t, "Melbourne", res.Location)

	require.NotNil(t, userRepo.updates)
	assert.Equal(t, "Updated User", userRepo.updates["display_name"])
	assert.Equal(t, "new bio", userRepo.updates["bio"])
	assert.Equal(t, "Melbourne", userRepo.updates["location"])
}

func TestUpdateProfileUserNotFound(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	displayName := "Missing User"

	res, err := svc.UpdateProfile(context.Background(), dto.UpdateProfileRequest{
		UserID:      uuid.NewString(),
		DisplayName: &displayName,
	})

	require.ErrorIs(t, err, repository.ErrUserNotFound)
	assert.Nil(t, res)
}

func TestDeleteUserSuccess(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "delete@example.com",
		Username:    "deleteuser",
		Password:    "Password123!",
		DisplayName: "Delete User",
	})
	require.NoError(t, err)

	err = svc.DeleteUser(context.Background(), registerRes.User.ID)

	require.NoError(t, err)
	assert.Equal(t, registerRes.User.ID, userRepo.deletedID)
}

func TestDeleteUserNotFound(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	err := svc.DeleteUser(context.Background(), uuid.NewString())

	require.ErrorIs(t, err, repository.ErrUserNotFound)
}

func TestRegisterUserRepositoryError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()

	expectedErr := errors.New("user create failed")
	userRepo.createErr = expectedErr

	svc := newTestUserService(userRepo, sessionRepo)

	res, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "create-error@example.com",
		Username:    "createerror",
		Password:    "Password123!",
		DisplayName: "Create Error",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestRegisterSessionCreateError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()

	expectedErr := errors.New("session create failed")
	sessionRepo.createErr = expectedErr

	svc := newTestUserService(userRepo, sessionRepo)

	res, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "session-error@example.com",
		Username:    "sessionerror",
		Password:    "Password123!",
		DisplayName: "Session Error",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestLoginFindByEmailUnexpectedError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()

	expectedErr := errors.New("email lookup failed")
	userRepo.findByEmailErr = expectedErr

	svc := newTestUserService(userRepo, sessionRepo)

	res, err := svc.Login(context.Background(), dto.LoginRequest{
		EmailOrUsername: "error@example.com",
		Password:        "Password123!",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestLoginFindByUsernameUnexpectedError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()

	expectedErr := errors.New("username lookup failed")
	userRepo.findByUsernameErr = expectedErr

	svc := newTestUserService(userRepo, sessionRepo)

	res, err := svc.Login(context.Background(), dto.LoginRequest{
		EmailOrUsername: "erroruser",
		Password:        "Password123!",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestLoginSessionCreateError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	_, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "login-session-error@example.com",
		Username:    "loginsessionerror",
		Password:    "Password123!",
		DisplayName: "Login Session Error",
	})
	require.NoError(t, err)

	expectedErr := errors.New("login session create failed")
	sessionRepo.createErr = expectedErr

	res, err := svc.Login(context.Background(), dto.LoginRequest{
		EmailOrUsername: "login-session-error@example.com",
		Password:        "Password123!",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestRefreshTokenFindSessionUnexpectedError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()

	expectedErr := errors.New("session lookup failed")
	sessionRepo.findActiveErr = expectedErr

	svc := newTestUserService(userRepo, sessionRepo)

	res, err := svc.RefreshToken(context.Background(), dto.RefreshTokenRequest{
		RefreshToken: "refresh-token",
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestRefreshTokenUserNotFound(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "refresh-user-missing@example.com",
		Username:    "refreshusermissing",
		Password:    "Password123!",
		DisplayName: "Refresh User Missing",
	})
	require.NoError(t, err)

	delete(userRepo.usersByID, registerRes.User.ID)

	res, err := svc.RefreshToken(context.Background(), dto.RefreshTokenRequest{
		RefreshToken: registerRes.RefreshToken,
	})

	require.ErrorIs(t, err, repository.ErrUserNotFound)
	assert.Nil(t, res)
}

func TestRefreshTokenRevokeSessionError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "refresh-revoke-error@example.com",
		Username:    "refreshrevokeerror",
		Password:    "Password123!",
		DisplayName: "Refresh Revoke Error",
	})
	require.NoError(t, err)

	expectedErr := errors.New("revoke session failed")
	sessionRepo.revokeErr = expectedErr

	res, err := svc.RefreshToken(context.Background(), dto.RefreshTokenRequest{
		RefreshToken: registerRes.RefreshToken,
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestRefreshTokenNewSessionCreateError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "refresh-create-error@example.com",
		Username:    "refreshcreateerror",
		Password:    "Password123!",
		DisplayName: "Refresh Create Error",
	})
	require.NoError(t, err)

	expectedErr := errors.New("new session create failed")
	sessionRepo.createErr = expectedErr

	res, err := svc.RefreshToken(context.Background(), dto.RefreshTokenRequest{
		RefreshToken: registerRes.RefreshToken,
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestLogoutEmptyRefreshTokenReturnsInvalidRefreshToken(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	err := svc.Logout(context.Background(), dto.LogoutRequest{
		UserID:       uuid.NewString(),
		RefreshToken: "",
	})

	require.ErrorIs(t, err, ErrInvalidRefreshToken)
}

func TestLogoutUserMismatchReturnsUnauthorized(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "logout-mismatch@example.com",
		Username:    "logoutmismatch",
		Password:    "Password123!",
		DisplayName: "Logout Mismatch",
	})
	require.NoError(t, err)

	err = svc.Logout(context.Background(), dto.LogoutRequest{
		UserID:       uuid.NewString(),
		RefreshToken: registerRes.RefreshToken,
	})

	require.ErrorIs(t, err, ErrUnauthorized)
}

func TestLogoutRevokeSessionError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "logout-revoke-error@example.com",
		Username:    "logoutrevokeerror",
		Password:    "Password123!",
		DisplayName: "Logout Revoke Error",
	})
	require.NoError(t, err)

	expectedErr := errors.New("logout revoke failed")
	sessionRepo.revokeErr = expectedErr

	err = svc.Logout(context.Background(), dto.LogoutRequest{
		UserID:       registerRes.User.ID,
		RefreshToken: registerRes.RefreshToken,
	})

	require.ErrorIs(t, err, expectedErr)
}

func TestUpdateProfileAllOptionalFields(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "profile-all@example.com",
		Username:    "profileall",
		Password:    "Password123!",
		DisplayName: "Profile All",
	})
	require.NoError(t, err)

	displayName := "Updated All"
	bio := "updated bio"
	avatarURL := "https://example.com/avatar.png"
	bannerURL := "https://example.com/banner.png"
	location := "Melbourne"
	websiteURL := "https://example.com"

	res, err := svc.UpdateProfile(context.Background(), dto.UpdateProfileRequest{
		UserID:      registerRes.User.ID,
		DisplayName: &displayName,
		Bio:         &bio,
		AvatarURL:   &avatarURL,
		BannerURL:   &bannerURL,
		Location:    &location,
		WebsiteURL:  &websiteURL,
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, displayName, res.DisplayName)
	assert.Equal(t, bio, res.Bio)
	assert.Equal(t, avatarURL, res.AvatarURL)
	assert.Equal(t, bannerURL, res.BannerURL)
	assert.Equal(t, location, res.Location)
	assert.Equal(t, websiteURL, res.WebsiteURL)

	assert.Equal(t, displayName, userRepo.updates["display_name"])
	assert.Equal(t, bio, userRepo.updates["bio"])
	assert.Equal(t, avatarURL, userRepo.updates["avatar_url"])
	assert.Equal(t, bannerURL, userRepo.updates["banner_url"])
	assert.Equal(t, location, userRepo.updates["location"])
	assert.Equal(t, websiteURL, userRepo.updates["website_url"])
}

func TestUpdateProfileRepositoryUpdateError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "profile-update-error@example.com",
		Username:    "profileupdateerror",
		Password:    "Password123!",
		DisplayName: "Profile Update Error",
	})
	require.NoError(t, err)

	expectedErr := errors.New("profile update failed")
	userRepo.updateErr = expectedErr

	displayName := "Updated Name"

	res, err := svc.UpdateProfile(context.Background(), dto.UpdateProfileRequest{
		UserID:      registerRes.User.ID,
		DisplayName: &displayName,
	})

	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, res)
}

func TestDeleteUserRepositoryError(t *testing.T) {
	userRepo := newFakeUserRepo()
	sessionRepo := newFakeSessionRepo()
	svc := newTestUserService(userRepo, sessionRepo)

	registerRes, err := svc.Register(context.Background(), dto.RegisterRequest{
		Email:       "delete-repo-error@example.com",
		Username:    "deleterepoerror",
		Password:    "Password123!",
		DisplayName: "Delete Repo Error",
	})
	require.NoError(t, err)

	expectedErr := errors.New("delete user failed")
	userRepo.deleteErr = expectedErr

	err = svc.DeleteUser(context.Background(), registerRes.User.ID)

	require.ErrorIs(t, err, expectedErr)
}
