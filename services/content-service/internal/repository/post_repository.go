package repository

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"gorm.io/gorm"
)

var (
	ErrPostNotFound = errors.New("Post not found")
	ErrForbidden    = errors.New("Forbidden")
)

type UserTimelineItem struct {
	Type       string
	Post       dbmodel.Post
	RepostedAt *time.Time
	SortAt     time.Time
}

type PostRepository interface {
	Create(ctx context.Context, post *dbmodel.Post) error
	FindByID(ctx context.Context, postID string) (*dbmodel.Post, error)
	DeleteOwnPost(ctx context.Context, postID string, requesterID string) error
	FindReplies(ctx context.Context, postID string, limit int) ([]dbmodel.Post, error)
	IncrementReplyCount(ctx context.Context, postID string) error
	FindUserTimeline(ctx context.Context, userID uuid.UUID, limit int) ([]UserTimelineItem, error)
}

type postRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(ctx context.Context, post *dbmodel.Post) error {
	return r.db.WithContext(ctx).Create(post).Error
}

func (r *postRepository) FindByID(ctx context.Context, postID string) (*dbmodel.Post, error) {
	var post dbmodel.Post

	err := r.db.WithContext(ctx).
		Preload("Author").
		Where("id = ?", postID).
		First(&post).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	return &post, nil
}

func (r *postRepository) DeleteOwnPost(ctx context.Context, postID string, requesterID string) error {
	postIDUUID, err := ParseUUID(postID)
	if err != nil {
		return err
	}

	requesterIDUUID, err := ParseUUID(requesterID)
	if err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Where(&dbmodel.Post{ID: postIDUUID, AuthorID: requesterIDUUID}).Delete(&dbmodel.Post{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (r *postRepository) FindReplies(ctx context.Context, postID string, limit int) ([]dbmodel.Post, error) {
	var replies []dbmodel.Post

	if limit <= 0 || limit > 50 {
		limit = 20
	}

	err := r.db.WithContext(ctx).
		Preload("Author").
		Where("reply_to_post_id = ?", postID).
		Order("created_at ASC").
		Limit(limit).
		Find(&replies).Error
	if err != nil {
		return nil, err
	}

	return replies, nil
}

func (r *postRepository) IncrementReplyCount(ctx context.Context, postID string) error {
	return r.db.WithContext(ctx).
		Model(&dbmodel.Post{}).
		Where("id = ?", postID).
		UpdateColumn("reply_count", gorm.Expr("reply_count + 1")).
		Error
}

func (r *postRepository) FindUserTimeline(ctx context.Context, userID uuid.UUID, limit int) ([]UserTimelineItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var ownPosts []dbmodel.Post

	if err := r.db.WithContext(ctx).
		Preload("Author").
		Where("author_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&ownPosts).Error; err != nil {
		return nil, err
	}

	var reposts []dbmodel.Repost

	if err := r.db.WithContext(ctx).
		Preload("Post.Author").
		Joins("JOIN posts ON posts.id = reposts.post_id").
		Where("reposts.user_id = ?", userID).
		Where("posts.deleted_at IS NULL").
		Order("reposts.created_at DESC").
		Limit(limit).
		Find(&reposts).Error; err != nil {
		return nil, err
	}

	items := make([]UserTimelineItem, 0, len(ownPosts)+len(reposts))

	for _, post := range ownPosts {
		items = append(items, UserTimelineItem{
			Type:   "post",
			Post:   post,
			SortAt: post.CreatedAt,
		})
	}

	for _, repost := range reposts {
		repostedAt := repost.CreatedAt

		items = append(items, UserTimelineItem{
			Type:       "repost",
			Post:       repost.Post,
			RepostedAt: &repostedAt,
			SortAt:     repost.CreatedAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].SortAt.After(items[j].SortAt)
	})

	if len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

func ParseUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}
