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

func newFeedRepositoryTestDB(t *testing.T) *gorm.DB {
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

		CREATE TABLE follows (
			follower_id text NOT NULL,
			following_id text NOT NULL,
			created_at datetime NOT NULL,
			PRIMARY KEY (follower_id, following_id)
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

func TestFindHomeFeedIncludesOwnAndFollowingPosts(t *testing.T) {
	db := newFeedRepositoryTestDB(t)
	repo := NewFeedRepository(db)

	currentUser := dbmodel.User{
		ID:           uuid.New(),
		Email:        "current@example.com",
		Username:     "current",
		PasswordHash: "hash",
		DisplayName:  "Current User",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	followedUser := dbmodel.User{
		ID:           uuid.New(),
		Email:        "followed@example.com",
		Username:     "followed",
		PasswordHash: "hash",
		DisplayName:  "Followed User",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	otherUser := dbmodel.User{
		ID:           uuid.New(),
		Email:        "other@example.com",
		Username:     "other",
		PasswordHash: "hash",
		DisplayName:  "Other User",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	require.NoError(t, db.Create(&currentUser).Error)
	require.NoError(t, db.Create(&followedUser).Error)
	require.NoError(t, db.Create(&otherUser).Error)

	require.NoError(t, db.Create(&dbmodel.Follow{
		FollowerID:  currentUser.ID,
		FollowingID: followedUser.ID,
		CreatedAt:   time.Now(),
	}).Error)

	ownPost := dbmodel.Post{
		ID:         uuid.New(),
		AuthorID:   currentUser.ID,
		Content:    "own post",
		Visibility: "public",
		CreatedAt:  time.Now().Add(-1 * time.Minute),
		UpdatedAt:  time.Now(),
	}

	followedPost := dbmodel.Post{
		ID:         uuid.New(),
		AuthorID:   followedUser.ID,
		Content:    "followed post",
		Visibility: "public",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	otherPost := dbmodel.Post{
		ID:         uuid.New(),
		AuthorID:   otherUser.ID,
		Content:    "other post",
		Visibility: "public",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	require.NoError(t, db.Create(&ownPost).Error)
	require.NoError(t, db.Create(&followedPost).Error)
	require.NoError(t, db.Create(&otherPost).Error)

	media := dbmodel.Media{
		ID:         uuid.New(),
		UploaderID: followedUser.ID,
		PostID:     &followedPost.ID,
		URL:        "https://example.com/media.png",
		StorageKey: "users/followed/uploads/media.png",
		MimeType:   "image/png",
		SizeBytes:  123,
		Status:     "attached",
		CreatedAt:  time.Now(),
	}

	require.NoError(t, db.Create(&media).Error)

	posts, nextCursor, err := repo.FindHomeFeed(context.Background(), currentUser.ID.String(), 20, "")

	require.NoError(t, err)
	assert.Empty(t, nextCursor)
	require.Len(t, posts, 2)

	postContents := []string{
		posts[0].Content,
		posts[1].Content,
	}

	assert.Contains(t, postContents, "own post")
	assert.Contains(t, postContents, "followed post")
	assert.NotContains(t, postContents, "other post")

	var foundFollowedPost dbmodel.Post
	for _, post := range posts {
		if post.ID == followedPost.ID {
			foundFollowedPost = post
		}
	}

	require.NotEqual(t, uuid.Nil, foundFollowedPost.ID)
	assert.Equal(t, "followed", foundFollowedPost.Author.Username)
	require.Len(t, foundFollowedPost.Media, 1)
	assert.Equal(t, "https://example.com/media.png", foundFollowedPost.Media[0].URL)
}

func TestFindHomeFeedExcludesReplies(t *testing.T) {
	db := newFeedRepositoryTestDB(t)
	repo := NewFeedRepository(db)

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

	parentPost := dbmodel.Post{
		ID:         uuid.New(),
		AuthorID:   user.ID,
		Content:    "parent",
		Visibility: "public",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	replyPost := dbmodel.Post{
		ID:            uuid.New(),
		AuthorID:      user.ID,
		ReplyToPostID: &parentPost.ID,
		Content:       "reply",
		Visibility:    "public",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	require.NoError(t, db.Create(&parentPost).Error)
	require.NoError(t, db.Create(&replyPost).Error)

	posts, _, err := repo.FindHomeFeed(context.Background(), user.ID.String(), 20, "")

	require.NoError(t, err)
	require.Len(t, posts, 1)
	assert.Equal(t, "parent", posts[0].Content)
}

func TestFindHomeFeedExcludesPrivatePosts(t *testing.T) {
	db := newFeedRepositoryTestDB(t)
	repo := NewFeedRepository(db)

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

	require.NoError(t, db.Create(&dbmodel.Post{
		ID:         uuid.New(),
		AuthorID:   user.ID,
		Content:    "public",
		Visibility: "public",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}).Error)

	require.NoError(t, db.Create(&dbmodel.Post{
		ID:         uuid.New(),
		AuthorID:   user.ID,
		Content:    "private",
		Visibility: "private",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}).Error)

	posts, _, err := repo.FindHomeFeed(context.Background(), user.ID.String(), 20, "")

	require.NoError(t, err)
	require.Len(t, posts, 1)
	assert.Equal(t, "public", posts[0].Content)
}

func TestFindHomeFeedCursor(t *testing.T) {
	db := newFeedRepositoryTestDB(t)
	repo := NewFeedRepository(db)

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

	older := time.Now().Add(-2 * time.Hour).UTC()
	newer := time.Now().Add(-1 * time.Hour).UTC()

	require.NoError(t, db.Create(&dbmodel.Post{
		ID:         uuid.New(),
		AuthorID:   user.ID,
		Content:    "older",
		Visibility: "public",
		CreatedAt:  older,
		UpdatedAt:  older,
	}).Error)

	require.NoError(t, db.Create(&dbmodel.Post{
		ID:         uuid.New(),
		AuthorID:   user.ID,
		Content:    "newer",
		Visibility: "public",
		CreatedAt:  newer,
		UpdatedAt:  newer,
	}).Error)

	posts, _, err := repo.FindHomeFeed(context.Background(), user.ID.String(), 20, newer.Format(time.RFC3339Nano))

	require.NoError(t, err)
	require.Len(t, posts, 1)
	assert.Equal(t, "older", posts[0].Content)
}

func TestFindHomeFeedInvalidCursor(t *testing.T) {
	db := newFeedRepositoryTestDB(t)
	repo := NewFeedRepository(db)

	posts, nextCursor, err := repo.FindHomeFeed(context.Background(), uuid.NewString(), 20, "bad-cursor")

	require.Error(t, err)
	assert.Nil(t, posts)
	assert.Empty(t, nextCursor)
}

func TestFindHomeFeedPaginationReturnsNextCursor(t *testing.T) {
	db := newFeedRepositoryTestDB(t)
	repo := NewFeedRepository(db)

	user := dbmodel.User{
		ID:           uuid.New(),
		Email:        "user-pagination@example.com",
		Username:     "userpagination",
		PasswordHash: "hash",
		DisplayName:  "User Pagination",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	require.NoError(t, db.Create(&user).Error)

	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		createdAt := now.Add(time.Duration(-i) * time.Minute)

		require.NoError(t, db.Create(&dbmodel.Post{
			ID:         uuid.New(),
			AuthorID:   user.ID,
			Content:    "post",
			Visibility: "public",
			CreatedAt:  createdAt,
			UpdatedAt:  createdAt,
		}).Error)
	}

	posts, nextCursor, err := repo.FindHomeFeed(context.Background(), user.ID.String(), 2, "")

	require.NoError(t, err)
	require.Len(t, posts, 2)
	assert.NotEmpty(t, nextCursor)
}

func TestFindHomeFeedDefaultsLimitWhenLimitIsZero(t *testing.T) {
	db := newFeedRepositoryTestDB(t)
	repo := NewFeedRepository(db)

	user := dbmodel.User{
		ID:           uuid.New(),
		Email:        "limit-zero@example.com",
		Username:     "limitzero",
		PasswordHash: "hash",
		DisplayName:  "Limit Zero",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	require.NoError(t, db.Create(&user).Error)

	now := time.Now().UTC()

	for i := 0; i < 21; i++ {
		createdAt := now.Add(time.Duration(-i) * time.Minute)

		require.NoError(t, db.Create(&dbmodel.Post{
			ID:         uuid.New(),
			AuthorID:   user.ID,
			Content:    "post",
			Visibility: "public",
			CreatedAt:  createdAt,
			UpdatedAt:  createdAt,
		}).Error)
	}

	posts, nextCursor, err := repo.FindHomeFeed(context.Background(), user.ID.String(), 0, "")

	require.NoError(t, err)
	require.Len(t, posts, 20)
	assert.NotEmpty(t, nextCursor)
}

func TestFindHomeFeedDefaultsLimitWhenLimitTooLarge(t *testing.T) {
	db := newFeedRepositoryTestDB(t)
	repo := NewFeedRepository(db)

	user := dbmodel.User{
		ID:           uuid.New(),
		Email:        "limit-large@example.com",
		Username:     "limitlarge",
		PasswordHash: "hash",
		DisplayName:  "Limit Large",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	require.NoError(t, db.Create(&user).Error)

	now := time.Now().UTC()

	for i := 0; i < 21; i++ {
		createdAt := now.Add(time.Duration(-i) * time.Minute)

		require.NoError(t, db.Create(&dbmodel.Post{
			ID:         uuid.New(),
			AuthorID:   user.ID,
			Content:    "post",
			Visibility: "public",
			CreatedAt:  createdAt,
			UpdatedAt:  createdAt,
		}).Error)
	}

	posts, nextCursor, err := repo.FindHomeFeed(context.Background(), user.ID.String(), 100, "")

	require.NoError(t, err)
	require.Len(t, posts, 20)
	assert.NotEmpty(t, nextCursor)
}
