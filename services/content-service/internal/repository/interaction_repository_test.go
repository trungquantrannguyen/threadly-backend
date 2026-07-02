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
	"gorm.io/gorm/logger"
)

func newInteractionRepositoryTestDB(t *testing.T) *gorm.DB {
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

		CREATE TABLE likes (
			user_id text NOT NULL,
			post_id text NOT NULL,
			created_at datetime NOT NULL,
			PRIMARY KEY (user_id, post_id)
		);

		CREATE TABLE bookmarks (
			user_id text NOT NULL,
			post_id text NOT NULL,
			created_at datetime NOT NULL,
			PRIMARY KEY (user_id, post_id)
		);

		CREATE TABLE reposts (
			user_id text NOT NULL,
			post_id text NOT NULL,
			created_at datetime NOT NULL,
			PRIMARY KEY (user_id, post_id)
		);

		CREATE TABLE follows (
			follower_id text NOT NULL,
			following_id text NOT NULL,
			created_at datetime NOT NULL,
			PRIMARY KEY (follower_id, following_id)
		);
	`)
	require.NoError(t, err)

	return db
}

func closeInteractionRepositoryTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

func newInteractionRepoTestUser(username string) dbmodel.User {
	now := time.Now().UTC()

	return dbmodel.User{
		ID:           uuid.New(),
		Email:        username + "@example.com",
		Username:     username,
		PasswordHash: "hashed-password",
		DisplayName:  username,
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func newInteractionRepoTestPost(authorID uuid.UUID, content string) dbmodel.Post {
	now := time.Now().UTC()

	return dbmodel.Post{
		ID:         uuid.New(),
		AuthorID:   authorID,
		Content:    content,
		Visibility: "public",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func TestInteractionRepositoryUserExists(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("exists")
	require.NoError(t, db.Create(&user).Error)

	exists, err := repo.UserExists(context.Background(), user.ID)

	require.NoError(t, err)
	assert.True(t, exists)
}

func TestInteractionRepositoryUserExistsFalse(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	exists, err := repo.UserExists(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.False(t, exists)
}

func TestInteractionRepositoryLikePost(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("liker")
	require.NoError(t, db.Create(&user).Error)

	post := newInteractionRepoTestPost(user.ID, "post")
	require.NoError(t, db.Create(&post).Error)

	created, err := repo.LikePost(context.Background(), user.ID, post.ID)

	require.NoError(t, err)
	assert.True(t, created)

	var saved dbmodel.Post
	require.NoError(t, db.First(&saved, "id = ?", post.ID).Error)
	assert.Equal(t, 1, saved.LikeCount)
}

func TestInteractionRepositoryLikePostDuplicateReturnsFalse(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("duplicatelike")
	require.NoError(t, db.Create(&user).Error)

	post := newInteractionRepoTestPost(user.ID, "post")
	require.NoError(t, db.Create(&post).Error)

	created, err := repo.LikePost(context.Background(), user.ID, post.ID)
	require.NoError(t, err)
	assert.True(t, created)

	created, err = repo.LikePost(context.Background(), user.ID, post.ID)

	require.NoError(t, err)
	assert.False(t, created)

	var saved dbmodel.Post
	require.NoError(t, db.First(&saved, "id = ?", post.ID).Error)
	assert.Equal(t, 1, saved.LikeCount)
}

func TestInteractionRepositoryUnlikePost(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("unliker")
	require.NoError(t, db.Create(&user).Error)

	post := newInteractionRepoTestPost(user.ID, "post")
	post.LikeCount = 1
	require.NoError(t, db.Create(&post).Error)

	require.NoError(t, db.Create(&dbmodel.Like{
		UserID:    user.ID,
		PostID:    post.ID,
		CreatedAt: time.Now().UTC(),
	}).Error)

	deleted, err := repo.UnlikePost(context.Background(), user.ID, post.ID)

	require.NoError(t, err)
	assert.True(t, deleted)

	var saved dbmodel.Post
	require.NoError(t, db.First(&saved, "id = ?", post.ID).Error)
	assert.Equal(t, 0, saved.LikeCount)
}

func TestInteractionRepositoryUnlikePostMissingReturnsFalse(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	deleted, err := repo.UnlikePost(context.Background(), uuid.New(), uuid.New())

	require.NoError(t, err)
	assert.False(t, deleted)
}

func TestInteractionRepositoryBookmarkPost(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("bookmarker")
	require.NoError(t, db.Create(&user).Error)

	post := newInteractionRepoTestPost(user.ID, "post")
	require.NoError(t, db.Create(&post).Error)

	created, err := repo.BookmarkPost(context.Background(), user.ID, post.ID)

	require.NoError(t, err)
	assert.True(t, created)

	var saved dbmodel.Post
	require.NoError(t, db.First(&saved, "id = ?", post.ID).Error)
	assert.Equal(t, 1, saved.BookmarkCount)
}

func TestInteractionRepositoryBookmarkPostDuplicateReturnsFalse(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("duplicatebookmark")
	require.NoError(t, db.Create(&user).Error)

	post := newInteractionRepoTestPost(user.ID, "post")
	require.NoError(t, db.Create(&post).Error)

	created, err := repo.BookmarkPost(context.Background(), user.ID, post.ID)
	require.NoError(t, err)
	assert.True(t, created)

	created, err = repo.BookmarkPost(context.Background(), user.ID, post.ID)

	require.NoError(t, err)
	assert.False(t, created)

	var saved dbmodel.Post
	require.NoError(t, db.First(&saved, "id = ?", post.ID).Error)
	assert.Equal(t, 1, saved.BookmarkCount)
}

func TestInteractionRepositoryUnbookmarkPost(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("unbookmarker")
	require.NoError(t, db.Create(&user).Error)

	post := newInteractionRepoTestPost(user.ID, "post")
	post.BookmarkCount = 1
	require.NoError(t, db.Create(&post).Error)

	require.NoError(t, db.Create(&dbmodel.Bookmark{
		UserID:    user.ID,
		PostID:    post.ID,
		CreatedAt: time.Now().UTC(),
	}).Error)

	deleted, err := repo.UnbookmarkPost(context.Background(), user.ID, post.ID)

	require.NoError(t, err)
	assert.True(t, deleted)

	var saved dbmodel.Post
	require.NoError(t, db.First(&saved, "id = ?", post.ID).Error)
	assert.Equal(t, 0, saved.BookmarkCount)
}

func TestInteractionRepositoryRepostPost(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("reposter")
	require.NoError(t, db.Create(&user).Error)

	post := newInteractionRepoTestPost(user.ID, "post")
	require.NoError(t, db.Create(&post).Error)

	created, err := repo.RepostPost(context.Background(), user.ID, post.ID)

	require.NoError(t, err)
	assert.True(t, created)

	var saved dbmodel.Post
	require.NoError(t, db.First(&saved, "id = ?", post.ID).Error)
	assert.Equal(t, 1, saved.RepostCount)
}

func TestInteractionRepositoryRepostPostDuplicateReturnsFalse(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("duplicaterepost")
	require.NoError(t, db.Create(&user).Error)

	post := newInteractionRepoTestPost(user.ID, "post")
	require.NoError(t, db.Create(&post).Error)

	created, err := repo.RepostPost(context.Background(), user.ID, post.ID)
	require.NoError(t, err)
	assert.True(t, created)

	created, err = repo.RepostPost(context.Background(), user.ID, post.ID)

	require.NoError(t, err)
	assert.False(t, created)

	var saved dbmodel.Post
	require.NoError(t, db.First(&saved, "id = ?", post.ID).Error)
	assert.Equal(t, 1, saved.RepostCount)
}

func TestInteractionRepositoryUndoRepost(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("undorepost")
	require.NoError(t, db.Create(&user).Error)

	post := newInteractionRepoTestPost(user.ID, "post")
	post.RepostCount = 1
	require.NoError(t, db.Create(&post).Error)

	require.NoError(t, db.Create(&dbmodel.Repost{
		UserID:    user.ID,
		PostID:    post.ID,
		CreatedAt: time.Now().UTC(),
	}).Error)

	deleted, err := repo.UndoRepost(context.Background(), user.ID, post.ID)

	require.NoError(t, err)
	assert.True(t, deleted)

	var saved dbmodel.Post
	require.NoError(t, db.First(&saved, "id = ?", post.ID).Error)
	assert.Equal(t, 0, saved.RepostCount)
}

func TestInteractionRepositoryFollowUser(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	follower := newInteractionRepoTestUser("follower")
	following := newInteractionRepoTestUser("following")

	require.NoError(t, db.Create(&follower).Error)
	require.NoError(t, db.Create(&following).Error)

	created, err := repo.FollowUser(context.Background(), follower.ID, following.ID)

	require.NoError(t, err)
	assert.True(t, created)

	var savedFollower dbmodel.User
	require.NoError(t, db.First(&savedFollower, "id = ?", follower.ID).Error)
	assert.Equal(t, 1, savedFollower.FollowingCount)

	var savedFollowing dbmodel.User
	require.NoError(t, db.First(&savedFollowing, "id = ?", following.ID).Error)
	assert.Equal(t, 1, savedFollowing.FollowerCount)
}

func TestInteractionRepositoryFollowUserDuplicateReturnsFalse(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	follower := newInteractionRepoTestUser("duplicatefollower")
	following := newInteractionRepoTestUser("duplicatefollowing")

	require.NoError(t, db.Create(&follower).Error)
	require.NoError(t, db.Create(&following).Error)

	created, err := repo.FollowUser(context.Background(), follower.ID, following.ID)
	require.NoError(t, err)
	assert.True(t, created)

	created, err = repo.FollowUser(context.Background(), follower.ID, following.ID)

	require.NoError(t, err)
	assert.False(t, created)
}

func TestInteractionRepositoryUnfollowUser(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	follower := newInteractionRepoTestUser("unfollower")
	following := newInteractionRepoTestUser("unfollowing")

	follower.FollowingCount = 1
	following.FollowerCount = 1

	require.NoError(t, db.Create(&follower).Error)
	require.NoError(t, db.Create(&following).Error)

	require.NoError(t, db.Create(&dbmodel.Follow{
		FollowerID:  follower.ID,
		FollowingID: following.ID,
		CreatedAt:   time.Now().UTC(),
	}).Error)

	deleted, err := repo.UnfollowUser(context.Background(), follower.ID, following.ID)

	require.NoError(t, err)
	assert.True(t, deleted)

	var savedFollower dbmodel.User
	require.NoError(t, db.First(&savedFollower, "id = ?", follower.ID).Error)
	assert.Equal(t, 0, savedFollower.FollowingCount)

	var savedFollowing dbmodel.User
	require.NoError(t, db.First(&savedFollowing, "id = ?", following.ID).Error)
	assert.Equal(t, 0, savedFollowing.FollowerCount)
}

func TestInteractionRepositoryUnfollowUserMissingReturnsFalse(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	deleted, err := repo.UnfollowUser(context.Background(), uuid.New(), uuid.New())

	require.NoError(t, err)
	assert.False(t, deleted)
}

func TestInteractionRepositoryGetFollowers(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	target := newInteractionRepoTestUser("target")
	followerOne := newInteractionRepoTestUser("followerone")
	followerTwo := newInteractionRepoTestUser("followertwo")

	require.NoError(t, db.Create(&target).Error)
	require.NoError(t, db.Create(&followerOne).Error)
	require.NoError(t, db.Create(&followerTwo).Error)

	require.NoError(t, db.Create(&dbmodel.Follow{
		FollowerID:  followerOne.ID,
		FollowingID: target.ID,
		CreatedAt:   time.Now().UTC().Add(-2 * time.Hour),
	}).Error)

	require.NoError(t, db.Create(&dbmodel.Follow{
		FollowerID:  followerTwo.ID,
		FollowingID: target.ID,
		CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
	}).Error)

	users, err := repo.GetFollowers(context.Background(), target.ID, 10)

	require.NoError(t, err)
	require.Len(t, users, 2)

	assert.Equal(t, followerTwo.ID, users[0].ID)
	assert.Equal(t, followerOne.ID, users[1].ID)
}

func TestInteractionRepositoryGetFollowersUsesLimit(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	target := newInteractionRepoTestUser("limitfollowers")
	followerOne := newInteractionRepoTestUser("limitfollowerone")
	followerTwo := newInteractionRepoTestUser("limitfollowertwo")

	require.NoError(t, db.Create(&target).Error)
	require.NoError(t, db.Create(&followerOne).Error)
	require.NoError(t, db.Create(&followerTwo).Error)

	require.NoError(t, db.Create(&dbmodel.Follow{
		FollowerID:  followerOne.ID,
		FollowingID: target.ID,
		CreatedAt:   time.Now().UTC().Add(-2 * time.Hour),
	}).Error)

	require.NoError(t, db.Create(&dbmodel.Follow{
		FollowerID:  followerTwo.ID,
		FollowingID: target.ID,
		CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
	}).Error)

	users, err := repo.GetFollowers(context.Background(), target.ID, 1)

	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, followerTwo.ID, users[0].ID)
}

func TestInteractionRepositoryGetFollowing(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("userfollowing")
	followingOne := newInteractionRepoTestUser("followingone")
	followingTwo := newInteractionRepoTestUser("followingtwo")

	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&followingOne).Error)
	require.NoError(t, db.Create(&followingTwo).Error)

	require.NoError(t, db.Create(&dbmodel.Follow{
		FollowerID:  user.ID,
		FollowingID: followingOne.ID,
		CreatedAt:   time.Now().UTC().Add(-2 * time.Hour),
	}).Error)

	require.NoError(t, db.Create(&dbmodel.Follow{
		FollowerID:  user.ID,
		FollowingID: followingTwo.ID,
		CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
	}).Error)

	users, err := repo.GetFollowing(context.Background(), user.ID, 10)

	require.NoError(t, err)
	require.Len(t, users, 2)

	assert.Equal(t, followingTwo.ID, users[0].ID)
	assert.Equal(t, followingOne.ID, users[1].ID)
}

func TestInteractionRepositoryGetFollowingUsesLimit(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	user := newInteractionRepoTestUser("limitfollowing")
	followingOne := newInteractionRepoTestUser("limitfollowingone")
	followingTwo := newInteractionRepoTestUser("limitfollowingtwo")

	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&followingOne).Error)
	require.NoError(t, db.Create(&followingTwo).Error)

	require.NoError(t, db.Create(&dbmodel.Follow{
		FollowerID:  user.ID,
		FollowingID: followingOne.ID,
		CreatedAt:   time.Now().UTC().Add(-2 * time.Hour),
	}).Error)

	require.NoError(t, db.Create(&dbmodel.Follow{
		FollowerID:  user.ID,
		FollowingID: followingTwo.ID,
		CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
	}).Error)

	users, err := repo.GetFollowing(context.Background(), user.ID, 1)

	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, followingTwo.ID, users[0].ID)
}

func TestInteractionRepositoryUserExistsDatabaseError(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	closeInteractionRepositoryTestDB(t, db)

	exists, err := repo.UserExists(context.Background(), uuid.New())

	require.Error(t, err)
	assert.False(t, exists)
}

func TestInteractionRepositoryLikePostDatabaseError(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	closeInteractionRepositoryTestDB(t, db)

	created, err := repo.LikePost(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.False(t, created)
}

func TestInteractionRepositoryUnlikePostDatabaseError(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	closeInteractionRepositoryTestDB(t, db)

	deleted, err := repo.UnlikePost(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.False(t, deleted)
}

func TestInteractionRepositoryFollowUserDatabaseError(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	closeInteractionRepositoryTestDB(t, db)

	created, err := repo.FollowUser(context.Background(), uuid.New(), uuid.New())

	require.Error(t, err)
	assert.False(t, created)
}

func TestInteractionRepositoryGetFollowersDatabaseError(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	closeInteractionRepositoryTestDB(t, db)

	users, err := repo.GetFollowers(context.Background(), uuid.New(), 10)

	require.Error(t, err)
	assert.Nil(t, users)
}

func TestInteractionRepositoryGetFollowingDatabaseError(t *testing.T) {
	db := newInteractionRepositoryTestDB(t)
	repo := NewInteractionRepository(db)

	closeInteractionRepositoryTestDB(t, db)

	users, err := repo.GetFollowing(context.Background(), uuid.New(), 10)

	require.Error(t, err)
	assert.Nil(t, users)
}
