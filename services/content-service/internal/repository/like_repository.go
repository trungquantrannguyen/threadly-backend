package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	likepb "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrLikeNotFound = errors.New("Like not found")

type LikeRepository interface {
	LikePost(ctx context.Context, postID uuid.UUID, userID uuid.UUID) (bool, error)
	UnlikePost(ctx context.Context, postID uuid.UUID, userID uuid.UUID) (bool, error)
}

type likeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) LikeRepository {
	return &likeRepository{db: db}
}

func (r *likeRepository) LikePost(ctx context.Context, postID uuid.UUID, userID uuid.UUID) (bool, error) {
	var inserted bool

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		like := likepb.Like{
			UserID: userID,
			PostID: postID,
		}

		result := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "post_id"},
			},
			DoNothing: true,
		}).Create(&like)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			inserted = false
			return nil
		}

		inserted = true

		return tx.Model(&dbmodel.Post{}).
			Where("id = ?", postID).
			UpdateColumn("like_count", gorm.Expr("like_count + 1")).
			Error
	})

	return inserted, err
}

func (r *likeRepository) UnlikePost(ctx context.Context, postID uuid.UUID, userID uuid.UUID) (bool, error) {
	var deleted bool

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.
			Where("user_id = ? AND post_id = ?", userID, postID).
			Delete(&dbmodel.Like{})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			deleted = false
			return nil
		}

		deleted = true

		return tx.Model(&dbmodel.Post{}).
			Where("id = ? AND like_count > 0", postID).
			UpdateColumn("like_count", gorm.Expr("like_count - 1")).
			Error
	})

	return deleted, err
}
