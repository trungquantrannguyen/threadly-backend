package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSessionRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	_, err = sqlDB.Exec(`
		CREATE TABLE users (
			id text PRIMARY KEY,
			email text NOT NULL UNIQUE,
			username text NOT NULL UNIQUE,
			password_hash text NOT NULL,
			display_name text NOT NULL,
			bio text,
			avatar_url text,
			banner_url text,
			location text,
			website_url text,
			is_verified boolean NOT NULL DEFAULT false,
			follower_count integer NOT NULL DEFAULT 0,
			following_count integer NOT NULL DEFAULT 0,
			post_count integer NOT NULL DEFAULT 0,
			created_at datetime NOT NULL,
			updated_at datetime NOT NULL,
			deleted_at datetime,
			role text NOT NULL DEFAULT 'user'
		);

		CREATE TABLE sessions (
			id text PRIMARY KEY,
			user_id text NOT NULL,
			refresh_token_hash text NOT NULL UNIQUE,
			user_agent text,
			ip_address text,
			expires_at datetime NOT NULL,
			revoked_at datetime,
			created_at datetime NOT NULL
		);
	`)
	require.NoError(t, err)

	return db
}

func newSessionTestUser(email string, username string) dbmodel.User {
	now := time.Now().UTC()

	return dbmodel.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: "hashed-password",
		DisplayName:  "Test User",
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func newTestSession(userID uuid.UUID, refreshTokenHash string, expiresAt time.Time) dbmodel.Session {
	now := time.Now().UTC()
	userAgent := "Chrome"
	ipAddress := "127.0.0.1"

	return dbmodel.Session{
		ID:               uuid.New(),
		UserID:           userID,
		RefreshTokenHash: refreshTokenHash,
		UserAgent:        &userAgent,
		IPAddress:        &ipAddress,
		ExpiresAt:        expiresAt,
		CreatedAt:        now,
	}
}

func TestSessionRepositoryCreate(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	user := newSessionTestUser("session-create@example.com", "sessioncreate")
	require.NoError(t, db.Create(&user).Error)

	session := newTestSession(user.ID, "refresh-hash-create", time.Now().Add(time.Hour))

	err := repo.Create(context.Background(), &session)

	require.NoError(t, err)

	var saved dbmodel.Session
	require.NoError(t, db.First(&saved, "id = ?", session.ID).Error)

	assert.Equal(t, session.ID, saved.ID)
	assert.Equal(t, user.ID, saved.UserID)
	assert.Equal(t, "refresh-hash-create", saved.RefreshTokenHash)
	require.NotNil(t, saved.UserAgent)
	require.NotNil(t, saved.IPAddress)
	assert.Equal(t, "Chrome", *saved.UserAgent)
	assert.Equal(t, "127.0.0.1", *saved.IPAddress)
}

func TestSessionRepositoryFindActiveByRefreshTokenHash(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	user := newSessionTestUser("session-find@example.com", "sessionfind")
	require.NoError(t, db.Create(&user).Error)

	session := newTestSession(user.ID, "active-refresh-hash", time.Now().Add(time.Hour))
	require.NoError(t, db.Create(&session).Error)

	found, err := repo.FindActiveByRefreshTokenHash(context.Background(), "active-refresh-hash")

	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, session.ID, found.ID)
	assert.Equal(t, user.ID, found.UserID)
	assert.Equal(t, "active-refresh-hash", found.RefreshTokenHash)
}

func TestSessionRepositoryFindActiveByRefreshTokenHashNotFound(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	found, err := repo.FindActiveByRefreshTokenHash(context.Background(), "missing-hash")

	require.ErrorIs(t, err, ErrSessionNotFound)
	assert.Nil(t, found)
}

func TestSessionRepositoryFindActiveByRefreshTokenHashExpired(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	user := newSessionTestUser("session-expired@example.com", "sessionexpired")
	require.NoError(t, db.Create(&user).Error)

	session := newTestSession(user.ID, "expired-refresh-hash", time.Now().Add(-1*time.Hour))
	require.NoError(t, db.Create(&session).Error)

	found, err := repo.FindActiveByRefreshTokenHash(context.Background(), "expired-refresh-hash")

	require.ErrorIs(t, err, ErrSessionNotFound)
	assert.Nil(t, found)
}

func TestSessionRepositoryFindActiveByRefreshTokenHashRevoked(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	user := newSessionTestUser("session-revoked@example.com", "sessionrevoked")
	require.NoError(t, db.Create(&user).Error)

	revokedAt := time.Now().UTC()
	session := newTestSession(user.ID, "revoked-refresh-hash", time.Now().Add(time.Hour))
	session.RevokedAt = &revokedAt

	require.NoError(t, db.Create(&session).Error)

	found, err := repo.FindActiveByRefreshTokenHash(context.Background(), "revoked-refresh-hash")

	require.ErrorIs(t, err, ErrSessionNotFound)
	assert.Nil(t, found)
}

func TestSessionRepositoryRevokeByID(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	user := newSessionTestUser("session-revoke@example.com", "sessionrevoke")
	require.NoError(t, db.Create(&user).Error)

	session := newTestSession(user.ID, "revoke-refresh-hash", time.Now().Add(time.Hour))
	require.NoError(t, db.Create(&session).Error)

	err := repo.RevokeByID(context.Background(), session.ID.String())

	require.NoError(t, err)

	var saved dbmodel.Session
	require.NoError(t, db.First(&saved, "id = ?", session.ID).Error)

	require.NotNil(t, saved.RevokedAt)
}

func TestSessionRepositoryRevokeByIDNotFound(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	err := repo.RevokeByID(context.Background(), uuid.NewString())

	require.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionRepositoryRevokeByIDAlreadyRevoked(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	user := newSessionTestUser("session-already-revoked@example.com", "sessionalreadyrevoked")
	require.NoError(t, db.Create(&user).Error)

	revokedAt := time.Now().UTC()
	session := newTestSession(user.ID, "already-revoked-refresh-hash", time.Now().Add(time.Hour))
	session.RevokedAt = &revokedAt

	require.NoError(t, db.Create(&session).Error)

	err := repo.RevokeByID(context.Background(), session.ID.String())

	require.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSessionRepositoryRevokeAllByUserID(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	user := newSessionTestUser("session-revoke-all@example.com", "sessionrevokeall")
	otherUser := newSessionTestUser("session-other@example.com", "sessionother")

	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&otherUser).Error)

	sessionOne := newTestSession(user.ID, "revoke-all-hash-1", time.Now().Add(time.Hour))
	sessionTwo := newTestSession(user.ID, "revoke-all-hash-2", time.Now().Add(time.Hour))
	otherSession := newTestSession(otherUser.ID, "other-user-hash", time.Now().Add(time.Hour))

	require.NoError(t, db.Create(&sessionOne).Error)
	require.NoError(t, db.Create(&sessionTwo).Error)
	require.NoError(t, db.Create(&otherSession).Error)

	err := repo.RevokeAllByUserID(context.Background(), user.ID.String())

	require.NoError(t, err)

	var userSessions []dbmodel.Session
	require.NoError(t, db.Find(&userSessions, "user_id = ?", user.ID).Error)

	require.Len(t, userSessions, 2)
	for _, session := range userSessions {
		assert.NotNil(t, session.RevokedAt)
	}

	var savedOtherSession dbmodel.Session
	require.NoError(t, db.First(&savedOtherSession, "id = ?", otherSession.ID).Error)

	assert.Nil(t, savedOtherSession.RevokedAt)
}

func TestSessionRepositoryDeleteExpired(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	user := newSessionTestUser("session-delete-expired@example.com", "sessiondeleteexpired")
	require.NoError(t, db.Create(&user).Error)

	expiredSession := newTestSession(user.ID, "expired-delete-hash", time.Now().Add(-1*time.Hour))
	activeSession := newTestSession(user.ID, "active-keep-hash", time.Now().Add(time.Hour))

	require.NoError(t, db.Create(&expiredSession).Error)
	require.NoError(t, db.Create(&activeSession).Error)

	err := repo.DeleteExpired(context.Background())

	require.NoError(t, err)

	var expiredCount int64
	require.NoError(t, db.Model(&dbmodel.Session{}).
		Where("id = ?", expiredSession.ID).
		Count(&expiredCount).Error)

	assert.Equal(t, int64(0), expiredCount)

	var activeCount int64
	require.NoError(t, db.Model(&dbmodel.Session{}).
		Where("id = ?", activeSession.ID).
		Count(&activeCount).Error)

	assert.Equal(t, int64(1), activeCount)
}

func closeSessionRepositoryTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

func TestSessionRepositoryCreateDatabaseError(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	session := newTestSession(uuid.New(), "db-error-create-session", time.Now().Add(time.Hour))

	closeSessionRepositoryTestDB(t, db)

	err := repo.Create(context.Background(), &session)

	require.Error(t, err)
}

func TestSessionRepositoryFindActiveByRefreshTokenHashDatabaseError(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	closeSessionRepositoryTestDB(t, db)

	found, err := repo.FindActiveByRefreshTokenHash(context.Background(), "db-error-hash")

	require.Error(t, err)
	assert.Nil(t, found)
	assert.False(t, errors.Is(err, ErrSessionNotFound))
}

func TestSessionRepositoryRevokeByIDDatabaseError(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	closeSessionRepositoryTestDB(t, db)

	err := repo.RevokeByID(context.Background(), uuid.NewString())

	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrSessionNotFound))
}

func TestSessionRepositoryRevokeAllByUserIDDatabaseError(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	closeSessionRepositoryTestDB(t, db)

	err := repo.RevokeAllByUserID(context.Background(), uuid.NewString())

	require.Error(t, err)
}

func TestSessionRepositoryDeleteExpiredDatabaseError(t *testing.T) {
	db := newSessionRepositoryTestDB(t)
	repo := NewSessionRepository(db)

	closeSessionRepositoryTestDB(t, db)

	err := repo.DeleteExpired(context.Background())

	require.Error(t, err)
}
