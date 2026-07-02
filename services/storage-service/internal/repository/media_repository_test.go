package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newMediaRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
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

		CREATE TABLE posts (
			id text PRIMARY KEY,
			author_id text NOT NULL,
			reply_to_post_id text,
			content text NOT NULL,
			visibility text NOT NULL DEFAULT 'public',
			like_count integer NOT NULL DEFAULT 0,
			reply_count integer NOT NULL DEFAULT 0,
			repost_count integer NOT NULL DEFAULT 0,
			bookmark_count integer NOT NULL DEFAULT 0,
			created_at datetime NOT NULL,
			updated_at datetime NOT NULL,
			deleted_at datetime
		);

		CREATE TABLE media (
			id text PRIMARY KEY,
			uploader_id text NOT NULL,
			post_id text,
			url text NOT NULL,
			storage_key text NOT NULL UNIQUE,
			mime_type text NOT NULL,
			size_bytes integer NOT NULL,
			width integer,
			height integer,
			status text NOT NULL DEFAULT 'uploaded',
			created_at datetime NOT NULL
		);
	`)
	require.NoError(t, err)

	return db
}

func TestCreateMedia(t *testing.T) {
	db := newMediaRepositoryTestDB(t)
	repo := NewMediaRepository(db)

	user := dbmodel.User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		Username:     "user",
		PasswordHash: "hash",
		DisplayName:  "User",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	require.NoError(t, db.Create(&user).Error)

	media := &dbmodel.Media{
		ID:         uuid.New(),
		UploaderID: user.ID,
		URL:        "https://example.com/media.png",
		StorageKey: "users/user/uploads/media.png",
		MimeType:   "image/png",
		SizeBytes:  123,
		Status:     "uploaded",
		CreatedAt:  time.Now(),
	}

	err := repo.Create(context.Background(), media)

	require.NoError(t, err)

	var saved dbmodel.Media
	require.NoError(t, db.First(&saved, "id = ?", media.ID).Error)

	assert.Equal(t, media.ID, saved.ID)
	assert.Equal(t, user.ID, saved.UploaderID)
	assert.Equal(t, "https://example.com/media.png", saved.URL)
	assert.Equal(t, "users/user/uploads/media.png", saved.StorageKey)
	assert.Equal(t, "image/png", saved.MimeType)
	assert.Equal(t, int64(123), saved.SizeBytes)
	assert.Equal(t, "uploaded", saved.Status)
}

func TestCreateMediaReturnsErrorForDuplicateStorageKey(t *testing.T) {
	db := newMediaRepositoryTestDB(t)
	repo := NewMediaRepository(db)

	user := dbmodel.User{
		ID:           uuid.New(),
		Email:        "user@example.com",
		Username:     "user",
		PasswordHash: "hash",
		DisplayName:  "User",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	require.NoError(t, db.Create(&user).Error)

	first := &dbmodel.Media{
		ID:         uuid.New(),
		UploaderID: user.ID,
		URL:        "https://example.com/first.png",
		StorageKey: "users/user/uploads/duplicate.png",
		MimeType:   "image/png",
		SizeBytes:  123,
		Status:     "uploaded",
		CreatedAt:  time.Now(),
	}

	second := &dbmodel.Media{
		ID:         uuid.New(),
		UploaderID: user.ID,
		URL:        "https://example.com/second.png",
		StorageKey: "users/user/uploads/duplicate.png",
		MimeType:   "image/png",
		SizeBytes:  456,
		Status:     "uploaded",
		CreatedAt:  time.Now(),
	}

	require.NoError(t, repo.Create(context.Background(), first))

	err := repo.Create(context.Background(), second)

	require.Error(t, err)
}
