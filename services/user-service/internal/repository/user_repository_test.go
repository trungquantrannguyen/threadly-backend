package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newUserRepositoryTestDB(t *testing.T) *gorm.DB {
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
	`)
	require.NoError(t, err)

	return db
}

func newTestUser(email string, username string) dbmodel.User {
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

func TestUserRepositoryCreate(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	user := newTestUser("user@example.com", "testuser")

	err := repo.Create(context.Background(), &user)

	require.NoError(t, err)

	var saved dbmodel.User
	require.NoError(t, db.First(&saved, "id = ?", user.ID).Error)

	assert.Equal(t, user.ID, saved.ID)
	assert.Equal(t, "user@example.com", saved.Email)
	assert.Equal(t, "testuser", saved.Username)
	assert.Equal(t, "Test User", saved.DisplayName)
	assert.Equal(t, "user", saved.Role)
}

func TestUserRepositoryCreateDuplicateReturnsError(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	first := newTestUser("duplicate@example.com", "duplicate")
	second := newTestUser("duplicate@example.com", "anotherusername")

	require.NoError(t, repo.Create(context.Background(), &first))

	err := repo.Create(context.Background(), &second)

	require.Error(t, err)
}

func TestUserRepositoryFindByID(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	user := newTestUser("findbyid@example.com", "findbyid")
	require.NoError(t, db.Create(&user).Error)

	found, err := repo.FindByID(context.Background(), user.ID.String())

	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "findbyid@example.com", found.Email)
	assert.Equal(t, "findbyid", found.Username)
}

func TestUserRepositoryFindByIDNotFound(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	found, err := repo.FindByID(context.Background(), uuid.NewString())

	require.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, found)
}

func TestUserRepositoryFindByEmailCaseInsensitive(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	user := newTestUser("CaseUser@Example.com", "caseemail")
	require.NoError(t, db.Create(&user).Error)

	found, err := repo.FindByEmail(context.Background(), "caseuser@example.com")

	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "CaseUser@Example.com", found.Email)
}

func TestUserRepositoryFindByEmailNotFound(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	found, err := repo.FindByEmail(context.Background(), "missing@example.com")

	require.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, found)
}

func TestUserRepositoryFindByUsernameCaseInsensitive(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	user := newTestUser("username@example.com", "TrungQuan")
	require.NoError(t, db.Create(&user).Error)

	found, err := repo.FindByUsername(context.Background(), "trungquan")

	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, user.ID, found.ID)
	assert.Equal(t, "TrungQuan", found.Username)
}

func TestUserRepositoryFindByUsernameNotFound(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	found, err := repo.FindByUsername(context.Background(), "missinguser")

	require.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, found)
}

func TestUserRepositoryUpdateProfile(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	user := newTestUser("update@example.com", "updateuser")
	require.NoError(t, db.Create(&user).Error)

	bio := "new bio"
	location := "Melbourne"
	avatarURL := "https://example.com/avatar.png"

	updates := map[string]interface{}{
		"bio":          bio,
		"location":     location,
		"avatar_url":   avatarURL,
		"display_name": "Updated Name",
	}

	err := repo.UpdateProfile(context.Background(), &user, updates)

	require.NoError(t, err)

	var saved dbmodel.User
	require.NoError(t, db.First(&saved, "id = ?", user.ID).Error)

	require.NotNil(t, saved.Bio)
	require.NotNil(t, saved.Location)
	require.NotNil(t, saved.AvatarURL)

	assert.Equal(t, "Updated Name", saved.DisplayName)
	assert.Equal(t, bio, *saved.Bio)
	assert.Equal(t, location, *saved.Location)
	assert.Equal(t, avatarURL, *saved.AvatarURL)
}

func TestUserRepositoryUpdateProfileWithEmptyUpdatesDoesNothing(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	user := newTestUser("emptyupdate@example.com", "emptyupdate")
	require.NoError(t, db.Create(&user).Error)

	err := repo.UpdateProfile(context.Background(), &user, map[string]interface{}{})

	require.NoError(t, err)

	var saved dbmodel.User
	require.NoError(t, db.First(&saved, "id = ?", user.ID).Error)

	assert.Equal(t, "Test User", saved.DisplayName)
	assert.Nil(t, saved.Bio)
}

func TestUserRepositoryDeleteByID(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	user := newTestUser("delete@example.com", "deleteuser")
	require.NoError(t, db.Create(&user).Error)

	err := repo.DeleteByID(context.Background(), user.ID.String())

	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), user.ID.String())

	require.ErrorIs(t, err, ErrUserNotFound)
	assert.Nil(t, found)
}

func TestUserRepositoryDeleteByIDNotFound(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	err := repo.DeleteByID(context.Background(), uuid.NewString())

	require.ErrorIs(t, err, ErrUserNotFound)
}

func TestIsDuplicateKeyErrorForPostgresUniqueViolation(t *testing.T) {
	err := &pgconn.PgError{
		Code: "23505",
	}

	assert.True(t, isDuplicateKeyError(err))
}

func TestIsDuplicateKeyErrorReturnsFalseForOtherErrors(t *testing.T) {
	assert.False(t, isDuplicateKeyError(errors.New("some other error")))
}

func closeUserRepositoryTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

func TestUserRepositoryCreateDatabaseError(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	closeUserRepositoryTestDB(t, db)

	user := newTestUser("db-error-create@example.com", "dberrorcreate")

	err := repo.Create(context.Background(), &user)

	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrDuplicateUser))
}

func TestUserRepositoryFindByIDDatabaseError(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	closeUserRepositoryTestDB(t, db)

	found, err := repo.FindByID(context.Background(), uuid.NewString())

	require.Error(t, err)
	assert.Nil(t, found)
	assert.False(t, errors.Is(err, ErrUserNotFound))
}

func TestUserRepositoryFindByEmailDatabaseError(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	closeUserRepositoryTestDB(t, db)

	found, err := repo.FindByEmail(context.Background(), "db-error@example.com")

	require.Error(t, err)
	assert.Nil(t, found)
	assert.False(t, errors.Is(err, ErrUserNotFound))
}

func TestUserRepositoryFindByUsernameDatabaseError(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	closeUserRepositoryTestDB(t, db)

	found, err := repo.FindByUsername(context.Background(), "dberroruser")

	require.Error(t, err)
	assert.Nil(t, found)
	assert.False(t, errors.Is(err, ErrUserNotFound))
}

func TestUserRepositoryUpdateProfileDatabaseError(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	user := newTestUser("db-error-update@example.com", "dberrorupdate")

	closeUserRepositoryTestDB(t, db)

	err := repo.UpdateProfile(context.Background(), &user, map[string]interface{}{
		"display_name": "Updated Name",
	})

	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrDuplicateUser))
}

func TestUserRepositoryDeleteByIDDatabaseError(t *testing.T) {
	db := newUserRepositoryTestDB(t)
	repo := NewUserRepository(db)

	closeUserRepositoryTestDB(t, db)

	err := repo.DeleteByID(context.Background(), uuid.NewString())

	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrUserNotFound))
}
