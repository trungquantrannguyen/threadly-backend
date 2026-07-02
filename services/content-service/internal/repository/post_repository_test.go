package repository

import (
	"context"
	"errors"
	"fmt"
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

func newPostRepositoryTestDB(t *testing.T) *gorm.DB {
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

		CREATE TABLE reposts (
			user_id text NOT NULL,
			post_id text NOT NULL,
			created_at datetime NOT NULL,
			PRIMARY KEY (user_id, post_id)
		);
	`)
	require.NoError(t, err)

	return db
}

func closePostRepositoryTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

func newPostRepoTestUser(username string) dbmodel.User {
	now := time.Now().UTC()
	id := uuid.New()

	return dbmodel.User{
		ID:           id,
		Email:        username + "@example.com",
		Username:     username,
		PasswordHash: "hashed-password",
		DisplayName:  username,
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func newPostRepoTestPost(authorID uuid.UUID, content string, createdAt time.Time) dbmodel.Post {
	return dbmodel.Post{
		ID:         uuid.New(),
		AuthorID:   authorID,
		Content:    content,
		Visibility: "public",
		CreatedAt:  createdAt,
		UpdatedAt:  createdAt,
	}
}

func newPostRepoTestMedia(uploaderID uuid.UUID, postID *uuid.UUID, storageKey string) dbmodel.Media {
	now := time.Now().UTC()

	return dbmodel.Media{
		ID:         uuid.New(),
		UploaderID: uploaderID,
		PostID:     postID,
		URL:        "https://example.com/" + storageKey,
		StorageKey: storageKey,
		MimeType:   "image/png",
		SizeBytes:  1024,
		Status:     "uploaded",
		CreatedAt:  now,
	}
}

func TestPostRepositoryCreateAndFindByID(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("author")
	require.NoError(t, db.Create(&user).Error)

	post := newPostRepoTestPost(user.ID, "hello threadly", time.Now().UTC())

	err := repo.Create(context.Background(), &post)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), post.ID.String())

	require.NoError(t, err)
	require.NotNil(t, found)

	assert.Equal(t, post.ID, found.ID)
	assert.Equal(t, user.ID, found.AuthorID)
	assert.Equal(t, "hello threadly", found.Content)
	assert.Equal(t, "author", found.Author.Username)
}

func TestPostRepositoryFindByIDPreloadsMedia(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("mediaauthor")
	require.NoError(t, db.Create(&user).Error)

	post := newPostRepoTestPost(user.ID, "post with media", time.Now().UTC())
	require.NoError(t, db.Create(&post).Error)

	media := newPostRepoTestMedia(user.ID, &post.ID, "media-1.png")
	media.Status = "attached"
	require.NoError(t, db.Create(&media).Error)

	found, err := repo.FindByID(context.Background(), post.ID.String())

	require.NoError(t, err)
	require.NotNil(t, found)
	require.Len(t, found.Media, 1)

	assert.Equal(t, media.ID, found.Media[0].ID)
	assert.Equal(t, "attached", found.Media[0].Status)
}

func TestPostRepositoryFindByIDNotFound(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	found, err := repo.FindByID(context.Background(), uuid.NewString())

	require.ErrorIs(t, err, ErrPostNotFound)
	assert.Nil(t, found)
}

func TestPostRepositoryUpdateOwnPost(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("updater")
	require.NoError(t, db.Create(&user).Error)

	post := newPostRepoTestPost(user.ID, "old content", time.Now().UTC())
	require.NoError(t, db.Create(&post).Error)

	updated, err := repo.UpdateOwnPost(
		context.Background(),
		post.ID.String(),
		user.ID.String(),
		"new content",
		"followers",
	)

	require.NoError(t, err)
	require.NotNil(t, updated)

	assert.Equal(t, "new content", updated.Content)
	assert.Equal(t, "followers", updated.Visibility)
}

func TestPostRepositoryUpdateOwnPostNotOwner(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	owner := newPostRepoTestUser("owner")
	other := newPostRepoTestUser("other")

	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&other).Error)

	post := newPostRepoTestPost(owner.ID, "old content", time.Now().UTC())
	require.NoError(t, db.Create(&post).Error)

	updated, err := repo.UpdateOwnPost(
		context.Background(),
		post.ID.String(),
		other.ID.String(),
		"bad update",
		"public",
	)

	require.ErrorIs(t, err, ErrPostNotFound)
	assert.Nil(t, updated)
}

func TestPostRepositoryDeleteOwnPost(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("deleter")
	require.NoError(t, db.Create(&user).Error)

	post := newPostRepoTestPost(user.ID, "delete me", time.Now().UTC())
	require.NoError(t, db.Create(&post).Error)

	media := newPostRepoTestMedia(user.ID, &post.ID, "delete-media.png")
	media.Status = "attached"
	require.NoError(t, db.Create(&media).Error)

	err := repo.DeleteOwnPost(context.Background(), post.ID.String(), user.ID.String())

	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), post.ID.String())
	require.ErrorIs(t, err, ErrPostNotFound)
	assert.Nil(t, found)

	var savedMedia dbmodel.Media
	require.NoError(t, db.First(&savedMedia, "id = ?", media.ID).Error)
	assert.Equal(t, "deleted", savedMedia.Status)
}

func TestPostRepositoryDeleteOwnPostNotOwner(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	owner := newPostRepoTestUser("deleteowner")
	other := newPostRepoTestUser("deleteother")

	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&other).Error)

	post := newPostRepoTestPost(owner.ID, "delete me", time.Now().UTC())
	require.NoError(t, db.Create(&post).Error)

	err := repo.DeleteOwnPost(context.Background(), post.ID.String(), other.ID.String())

	require.ErrorIs(t, err, ErrPostNotFound)
}

func TestPostRepositoryFindRepliesOrdersByCreatedAtAscAndUsesLimit(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("replyauthor")
	require.NoError(t, db.Create(&user).Error)

	parent := newPostRepoTestPost(user.ID, "parent", time.Now().UTC())
	require.NoError(t, db.Create(&parent).Error)

	older := newPostRepoTestPost(user.ID, "older reply", time.Now().UTC().Add(-3*time.Hour))
	older.ReplyToPostID = &parent.ID

	middle := newPostRepoTestPost(user.ID, "middle reply", time.Now().UTC().Add(-2*time.Hour))
	middle.ReplyToPostID = &parent.ID

	newer := newPostRepoTestPost(user.ID, "newer reply", time.Now().UTC().Add(-1*time.Hour))
	newer.ReplyToPostID = &parent.ID

	require.NoError(t, db.Create(&newer).Error)
	require.NoError(t, db.Create(&older).Error)
	require.NoError(t, db.Create(&middle).Error)

	replies, err := repo.FindReplies(context.Background(), parent.ID.String(), 2)

	require.NoError(t, err)
	require.Len(t, replies, 2)

	assert.Equal(t, older.ID, replies[0].ID)
	assert.Equal(t, middle.ID, replies[1].ID)
}

func TestPostRepositoryFindRepliesDefaultsLimit(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("defaultreply")
	require.NoError(t, db.Create(&user).Error)

	parent := newPostRepoTestPost(user.ID, "parent", time.Now().UTC())
	require.NoError(t, db.Create(&parent).Error)

	for i := 0; i < 25; i++ {
		reply := newPostRepoTestPost(user.ID, fmt.Sprintf("reply %d", i), time.Now().UTC().Add(time.Duration(i)*time.Minute))
		reply.ReplyToPostID = &parent.ID
		require.NoError(t, db.Create(&reply).Error)
	}

	replies, err := repo.FindReplies(context.Background(), parent.ID.String(), 0)

	require.NoError(t, err)
	require.Len(t, replies, 20)
}

func TestPostRepositoryIncrementReplyCount(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("counter")
	require.NoError(t, db.Create(&user).Error)

	post := newPostRepoTestPost(user.ID, "count replies", time.Now().UTC())
	require.NoError(t, db.Create(&post).Error)

	err := repo.IncrementReplyCount(context.Background(), post.ID.String())

	require.NoError(t, err)

	var saved dbmodel.Post
	require.NoError(t, db.First(&saved, "id = ?", post.ID).Error)
	assert.Equal(t, 1, saved.ReplyCount)
}

func TestPostRepositoryAttachMediaToPost(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("mediauploader")
	require.NoError(t, db.Create(&user).Error)

	post := newPostRepoTestPost(user.ID, "attach media", time.Now().UTC())
	require.NoError(t, db.Create(&post).Error)

	mediaOne := newPostRepoTestMedia(user.ID, nil, "attach-one.png")
	mediaTwo := newPostRepoTestMedia(user.ID, nil, "attach-two.png")

	require.NoError(t, db.Create(&mediaOne).Error)
	require.NoError(t, db.Create(&mediaTwo).Error)

	err := repo.AttachMediaToPost(context.Background(), post.ID, user.ID, []string{
		mediaOne.ID.String(),
		mediaTwo.ID.String(),
	})

	require.NoError(t, err)

	var saved []dbmodel.Media
	require.NoError(t, db.Where("id IN ?", []uuid.UUID{mediaOne.ID, mediaTwo.ID}).Find(&saved).Error)
	require.Len(t, saved, 2)

	for _, item := range saved {
		require.NotNil(t, item.PostID)
		assert.Equal(t, post.ID, *item.PostID)
		assert.Equal(t, "attached", item.Status)
	}
}

func TestPostRepositoryAttachMediaToPostEmptyList(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	err := repo.AttachMediaToPost(context.Background(), uuid.New(), uuid.New(), nil)

	require.NoError(t, err)
}

func TestPostRepositoryAttachMediaToPostInvalidMediaID(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	err := repo.AttachMediaToPost(context.Background(), uuid.New(), uuid.New(), []string{
		"bad-media-id",
	})

	require.Error(t, err)
}

func TestPostRepositoryAttachMediaToPostForbiddenForWrongUploader(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	owner := newPostRepoTestUser("ownerattach")
	wrongUploader := newPostRepoTestUser("wrongattach")

	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&wrongUploader).Error)

	post := newPostRepoTestPost(owner.ID, "attach forbidden", time.Now().UTC())
	require.NoError(t, db.Create(&post).Error)

	media := newPostRepoTestMedia(wrongUploader.ID, nil, "wrong-uploader.png")
	require.NoError(t, db.Create(&media).Error)

	err := repo.AttachMediaToPost(context.Background(), post.ID, owner.ID, []string{
		media.ID.String(),
	})

	require.ErrorIs(t, err, ErrForbidden)
}

func TestPostRepositoryReplacePostMedia(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("replaceuploader")
	require.NoError(t, db.Create(&user).Error)

	post := newPostRepoTestPost(user.ID, "replace media", time.Now().UTC())
	require.NoError(t, db.Create(&post).Error)

	oldMedia := newPostRepoTestMedia(user.ID, &post.ID, "old-media.png")
	oldMedia.Status = "attached"

	newMedia := newPostRepoTestMedia(user.ID, nil, "new-media.png")

	require.NoError(t, db.Create(&oldMedia).Error)
	require.NoError(t, db.Create(&newMedia).Error)

	err := repo.ReplacePostMedia(context.Background(), post.ID, user.ID, []string{
		newMedia.ID.String(),
	})

	require.NoError(t, err)

	var savedOld dbmodel.Media
	require.NoError(t, db.First(&savedOld, "id = ?", oldMedia.ID).Error)

	assert.Nil(t, savedOld.PostID)
	assert.Equal(t, "uploaded", savedOld.Status)

	var savedNew dbmodel.Media
	require.NoError(t, db.First(&savedNew, "id = ?", newMedia.ID).Error)

	require.NotNil(t, savedNew.PostID)
	assert.Equal(t, post.ID, *savedNew.PostID)
	assert.Equal(t, "attached", savedNew.Status)
}

func TestPostRepositoryReplacePostMediaEmptyListDetachesExistingMedia(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("detachmedia")
	require.NoError(t, db.Create(&user).Error)

	post := newPostRepoTestPost(user.ID, "detach media", time.Now().UTC())
	require.NoError(t, db.Create(&post).Error)

	oldMedia := newPostRepoTestMedia(user.ID, &post.ID, "detach-old.png")
	oldMedia.Status = "attached"
	require.NoError(t, db.Create(&oldMedia).Error)

	err := repo.ReplacePostMedia(context.Background(), post.ID, user.ID, nil)

	require.NoError(t, err)

	var savedOld dbmodel.Media
	require.NoError(t, db.First(&savedOld, "id = ?", oldMedia.ID).Error)

	assert.Nil(t, savedOld.PostID)
	assert.Equal(t, "uploaded", savedOld.Status)
}

func TestPostRepositoryReplacePostMediaForbidden(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	owner := newPostRepoTestUser("replaceowner")
	other := newPostRepoTestUser("replaceother")

	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&other).Error)

	post := newPostRepoTestPost(owner.ID, "replace forbidden", time.Now().UTC())
	require.NoError(t, db.Create(&post).Error)

	media := newPostRepoTestMedia(other.ID, nil, "replace-wrong-uploader.png")
	require.NoError(t, db.Create(&media).Error)

	err := repo.ReplacePostMedia(context.Background(), post.ID, owner.ID, []string{
		media.ID.String(),
	})

	require.ErrorIs(t, err, ErrForbidden)
}

func TestPostRepositoryFindUserTimeline(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("timelineuser")
	other := newPostRepoTestUser("timelineother")

	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&other).Error)

	ownPost := newPostRepoTestPost(user.ID, "own post", time.Now().UTC().Add(-2*time.Hour))
	require.NoError(t, db.Create(&ownPost).Error)

	otherPost := newPostRepoTestPost(other.ID, "reposted post", time.Now().UTC().Add(-3*time.Hour))
	require.NoError(t, db.Create(&otherPost).Error)

	repostTime := time.Now().UTC().Add(-1 * time.Hour)
	repost := dbmodel.Repost{
		UserID:    user.ID,
		PostID:    otherPost.ID,
		CreatedAt: repostTime,
	}
	require.NoError(t, db.Create(&repost).Error)

	items, err := repo.FindUserTimeline(context.Background(), user.ID, 10)

	require.NoError(t, err)
	require.Len(t, items, 2)

	assert.Equal(t, "repost", items[0].Type)
	assert.Equal(t, otherPost.ID, items[0].Post.ID)
	require.NotNil(t, items[0].RepostedAt)
	assert.Equal(t, repostTime.Format(time.RFC3339), items[0].RepostedAt.Format(time.RFC3339))

	assert.Equal(t, "post", items[1].Type)
	assert.Equal(t, ownPost.ID, items[1].Post.ID)
}

func TestPostRepositoryFindUserTimelineDefaultsLimit(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	user := newPostRepoTestUser("timelinelimit")
	require.NoError(t, db.Create(&user).Error)

	for i := 0; i < 25; i++ {
		post := newPostRepoTestPost(user.ID, fmt.Sprintf("post %d", i), time.Now().UTC().Add(time.Duration(i)*time.Minute))
		require.NoError(t, db.Create(&post).Error)
	}

	items, err := repo.FindUserTimeline(context.Background(), user.ID, 0)

	require.NoError(t, err)
	require.Len(t, items, 20)
}

func TestPostRepositoryCreateDatabaseError(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	closePostRepositoryTestDB(t, db)

	post := newPostRepoTestPost(uuid.New(), "db error", time.Now().UTC())

	err := repo.Create(context.Background(), &post)

	require.Error(t, err)
}

func TestPostRepositoryFindByIDDatabaseError(t *testing.T) {
	db := newPostRepositoryTestDB(t)
	repo := NewPostRepository(db)

	closePostRepositoryTestDB(t, db)

	found, err := repo.FindByID(context.Background(), uuid.NewString())

	require.Error(t, err)
	assert.Nil(t, found)
	assert.False(t, errors.Is(err, ErrPostNotFound))
}
