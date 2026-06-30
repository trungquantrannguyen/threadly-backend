package repository

import (
	"context"
	"time"

	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/gorm"
)

type FeedRepository interface {
	FindHomeFeed(ctx context.Context, userID string, limit int, cursor string) ([]dbmodel.Post, string, error)
}

type feedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) FeedRepository {
	return &feedRepository{
		db: db,
	}
}

func (r *feedRepository) FindHomeFeed(ctx context.Context, userID string, limit int, cursor string) ([]dbmodel.Post, string, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	query := r.db.WithContext(ctx).
		Model(&dbmodel.Post{}).
		Preload("Author").
		Preload("Media").
		Joins(`
			LEFT JOIN follows
			ON follows.following_id = posts.author_id
			AND follows.follower_id = ?
		`, userID).
		Where("posts.reply_to_post_id IS NULL").
		Where("posts.visibility = ?", "public").
		Where("posts.author_id = ? OR follows.following_id IS NOT NULL", userID).
		Order("posts.created_at DESC").
		Limit(limit + 1)

	if cursor != "" {
		parsedCursor, err := time.Parse(time.RFC3339Nano, cursor)
		if err != nil {
			return nil, "", err
		}

		query = query.Where("posts.created_at < ?", parsedCursor)
	}

	var posts []dbmodel.Post
	if err := query.Find(&posts).Error; err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(posts) > limit {
		nextCursor = posts[limit-1].CreatedAt.Format(time.RFC3339Nano)
		posts = posts[:limit]
	}

	return posts, nextCursor, nil
}
