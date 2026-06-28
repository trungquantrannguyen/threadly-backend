package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrUserNotFound = errors.New("User not found")

type InteractionRepository interface {
	UserExists(ctx context.Context, userID uuid.UUID) (bool, error)

	LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error)
	UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error)

	BookmarkPost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error)
	UnbookmarkPost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error)

	RepostPost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error)
	UndoRepost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error)

	FollowUser(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error)
	UnfollowUser(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error)
	GetFollowers(ctx context.Context, userID uuid.UUID, limit int) ([]dbmodel.User, error)
	GetFollowing(ctx context.Context, userID uuid.UUID, limit int) ([]dbmodel.User, error)
}

type interactionRepository struct {
	db *gorm.DB
}

func NewInteractionRepository(db *gorm.DB) InteractionRepository {
	return &interactionRepository{db: db}
}

func (r *interactionRepository) UserExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&dbmodel.User{}).
		Where("id = ?", userID).
		Count(&count).Error

	return count > 0, err
}

func (r *interactionRepository) LikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	return r.createPostInteraction(ctx, &dbmodel.Like{
		UserID: userID,
		PostID: postID,
	}, "like_count")
}

func (r *interactionRepository) UnlikePost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	return r.deletePostInteraction(ctx, &dbmodel.Like{}, userID, postID, "like_count")
}

func (r *interactionRepository) BookmarkPost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	return r.createPostInteraction(ctx, &dbmodel.Bookmark{
		UserID: userID,
		PostID: postID,
	}, "bookmark_count")
}

func (r *interactionRepository) UnbookmarkPost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	return r.deletePostInteraction(ctx, &dbmodel.Bookmark{}, userID, postID, "bookmark_count")
}

func (r *interactionRepository) RepostPost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	return r.createPostInteraction(ctx, &dbmodel.Repost{
		UserID: userID,
		PostID: postID,
	}, "repost_count")
}

func (r *interactionRepository) UndoRepost(ctx context.Context, userID uuid.UUID, postID uuid.UUID) (bool, error) {
	return r.deletePostInteraction(ctx, &dbmodel.Repost{}, userID, postID, "repost_count")
}

func (r *interactionRepository) createPostInteraction(ctx context.Context, model interface{}, counterColumn string) (bool, error) {
	var inserted bool

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "post_id"},
			},
			DoNothing: true,
		}).Create(model)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			inserted = false
			return nil
		}

		inserted = true

		postID, ok := getPostIDFromInteraction(model)
		if !ok {
			return errors.New("failed to extract post id from interaction")
		}

		return tx.Model(&dbmodel.Post{}).
			Where("id = ?", postID).
			UpdateColumn(counterColumn, gorm.Expr(counterColumn+" + 1")).
			Error
	})

	return inserted, err
}

func (r *interactionRepository) deletePostInteraction(ctx context.Context, model interface{}, userID uuid.UUID, postID uuid.UUID, counterColumn string) (bool, error) {
	var deleted bool

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where("user_id = ? AND post_id = ?", userID, postID).
			Delete(model)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			deleted = false
			return nil
		}

		deleted = true

		return tx.Model(&dbmodel.Post{}).
			Where("id = ? AND "+counterColumn+" > 0", postID).
			UpdateColumn(counterColumn, gorm.Expr(counterColumn+" - 1")).
			Error
	})

	return deleted, err
}

func getPostIDFromInteraction(model interface{}) (uuid.UUID, bool) {
	switch value := model.(type) {
	case *dbmodel.Like:
		return value.PostID, true
	case *dbmodel.Bookmark:
		return value.PostID, true
	case *dbmodel.Repost:
		return value.PostID, true
	default:
		return uuid.Nil, false
	}
}

func (r *interactionRepository) FollowUser(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error) {
	var inserted bool

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		follow := dbmodel.Follow{
			FollowerID:  followerID,
			FollowingID: followingID,
		}

		result := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "follower_id"},
				{Name: "following_id"},
			},
			DoNothing: true,
		}).Create(&follow)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			inserted = false
			return nil
		}

		inserted = true

		if err := tx.Model(&dbmodel.User{}).
			Where("id = ?", followerID).
			UpdateColumn("following_count", gorm.Expr("following_count + 1")).
			Error; err != nil {
			return err
		}

		return tx.Model(&dbmodel.User{}).
			Where("id = ?", followingID).
			UpdateColumn("follower_count", gorm.Expr("follower_count + 1")).
			Error
	})

	return inserted, err
}

func (r *interactionRepository) UnfollowUser(ctx context.Context, followerID uuid.UUID, followingID uuid.UUID) (bool, error) {
	var deleted bool

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where("follower_id = ? AND following_id = ?", followerID, followingID).
			Delete(&dbmodel.Follow{})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			deleted = false
			return nil
		}

		deleted = true

		if err := tx.Model(&dbmodel.User{}).
			Where("id = ? AND following_count > 0", followerID).
			UpdateColumn("following_count", gorm.Expr("following_count - 1")).
			Error; err != nil {
			return err
		}

		return tx.Model(&dbmodel.User{}).
			Where("id = ? AND follower_count > 0", followingID).
			UpdateColumn("follower_count", gorm.Expr("follower_count - 1")).
			Error
	})

	return deleted, err
}

func (r *interactionRepository) GetFollowers(ctx context.Context, userID uuid.UUID, limit int) ([]dbmodel.User, error) {
	var users []dbmodel.User

	err := r.db.WithContext(ctx).
		Model(&dbmodel.User{}).
		Select("users.*").
		Joins("JOIN follows ON follows.follower_id = users.id").
		Where("follows.following_id = ?", userID).
		Order("follows.created_at DESC").
		Limit(limit).
		Find(&users).Error

	return users, err
}

func (r *interactionRepository) GetFollowing(ctx context.Context, userID uuid.UUID, limit int) ([]dbmodel.User, error) {
	var users []dbmodel.User

	err := r.db.WithContext(ctx).
		Model(&dbmodel.User{}).
		Select("users.*").
		Joins("JOIN follows ON follows.following_id = users.id").
		Where("follows.follower_id = ?", userID).
		Order("follows.created_at DESC").
		Limit(limit).
		Find(&users).Error

	return users, err
}
