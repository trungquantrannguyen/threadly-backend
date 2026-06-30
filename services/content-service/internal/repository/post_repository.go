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
	UpdateOwnPost(ctx context.Context, postID string, requesterID string, content string, visibility string) (*dbmodel.Post, error)
	DeleteOwnPost(ctx context.Context, postID string, requesterID string) error
	FindReplies(ctx context.Context, postID string, limit int) ([]dbmodel.Post, error)
	IncrementReplyCount(ctx context.Context, postID string) error
	FindUserTimeline(ctx context.Context, userID uuid.UUID, limit int) ([]UserTimelineItem, error)

	AttachMediaToPost(ctx context.Context, postID uuid.UUID, uploaderID uuid.UUID, mediaIDs []string) error
	ReplacePostMedia(ctx context.Context, postID uuid.UUID, uploaderID uuid.UUID, mediaIDs []string) error
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
		Preload("Media").
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
		Preload("Media").
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
		Preload("Media").
		Where("author_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&ownPosts).Error; err != nil {
		return nil, err
	}

	var reposts []dbmodel.Repost

	if err := r.db.WithContext(ctx).
		Preload("Post.Author").
		Preload("Post.Media").
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

func (r *postRepository) UpdateOwnPost(
	ctx context.Context,
	postID string,
	requesterID string,
	content string,
	visibility string,
) (*dbmodel.Post, error) {
	postIDUUID, err := uuid.Parse(postID)
	if err != nil {
		return nil, err
	}

	requesterIDUUID, err := uuid.Parse(requesterID)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{
		"content":    content,
		"visibility": visibility,
		"updated_at": time.Now().UTC(),
	}

	result := r.db.WithContext(ctx).
		Model(&dbmodel.Post{}).
		Where("id = ? AND author_id = ?", postIDUUID, requesterIDUUID).
		Updates(updates)

	if result.Error != nil {
		return nil, result.Error
	}

	if result.RowsAffected == 0 {
		return nil, ErrPostNotFound
	}

	return r.FindByID(ctx, postID)
}

func (r *postRepository) AttachMediaToPost(
	ctx context.Context,
	postID uuid.UUID,
	uploaderID uuid.UUID,
	mediaIDs []string,
) error {
	if len(mediaIDs) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(mediaIDs))
	for _, id := range mediaIDs {
		parsedID, err := uuid.Parse(id)
		if err != nil {
			return err
		}
		ids = append(ids, parsedID)
	}

	result := r.db.WithContext(ctx).
		Model(&dbmodel.Media{}).
		Where("id IN ?", ids).
		Where("uploader_id = ?", uploaderID).
		Where("post_id IS NULL OR post_id = ?", postID).
		Update("post_id", postID)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected != int64(len(ids)) {
		return ErrForbidden
	}

	return nil
}

func (r *postRepository) ReplacePostMedia(
	ctx context.Context,
	postID uuid.UUID,
	uploaderID uuid.UUID,
	mediaIDs []string,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&dbmodel.Media{}).
			Where("post_id = ? AND uploader_id = ?", postID, uploaderID).
			Update("post_id", nil).Error; err != nil {
			return err
		}

		if len(mediaIDs) == 0 {
			return nil
		}

		ids := make([]uuid.UUID, 0, len(mediaIDs))
		for _, id := range mediaIDs {
			parsedID, err := uuid.Parse(id)
			if err != nil {
				return err
			}
			ids = append(ids, parsedID)
		}

		result := tx.Model(&dbmodel.Media{}).
			Where("id IN ?", ids).
			Where("uploader_id = ?", uploaderID).
			Where("post_id IS NULL OR post_id = ?", postID).
			Update("post_id", postID)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected != int64(len(ids)) {
			return ErrForbidden
		}

		return nil
	})
}

func ParseUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}
