package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/dto"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/repository"
)

var (
	ErrInvalidPostContent = errors.New("post content is required")
	ErrInvalidPostID      = errors.New("invalid post id")
	ErrInvalidAuthorID    = errors.New("invalid author id")
)

type ContentService interface {
	CreatePost(ctx context.Context, req dto.CreatePostRequest) (*dto.PostResponse, error)
	GetPost(ctx context.Context, req dto.GetPostRequest) (*dto.PostResponse, error)
	DeletePost(ctx context.Context, req dto.DeletePostRequest) error
	CreateReply(ctx context.Context, req dto.CreateReplyRequest) (*dto.PostResponse, error)
	GetReplies(ctx context.Context, req dto.GetRepliesRequest) ([]dto.PostResponse, error)
}

type contentService struct {
	postRepo repository.PostRepository
}

func NewContentService(postRepo repository.PostRepository) ContentService {
	return &contentService{
		postRepo: postRepo,
	}
}

func (s *contentService) CreatePost(ctx context.Context, req dto.CreatePostRequest) (*dto.PostResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, ErrInvalidPostContent
	}

	authorID, err := uuid.Parse(req.AuthorID)
	if err != nil {
		return nil, ErrInvalidAuthorID
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = "public"
	}

	post := &dbmodel.Post{
		AuthorID:   authorID,
		Content:    content,
		Visibility: visibility,
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, err
	}

	return toPostResponse(post), nil
}

func (s *contentService) GetPost(ctx context.Context, req dto.GetPostRequest) (*dto.PostResponse, error) {
	if _, err := uuid.Parse(req.PostID); err != nil {
		return nil, ErrInvalidPostID
	}
	post, err := s.postRepo.FindByID(ctx, req.PostID)
	if err != nil {
		return nil, err
	}

	return toPostResponse(post), nil
}

func (s *contentService) DeletePost(ctx context.Context, req dto.DeletePostRequest) error {
	if _, err := uuid.Parse(req.PostID); err != nil {
		return ErrInvalidPostID
	}

	if _, err := uuid.Parse(req.RequesterID); err != nil {
		return ErrInvalidAuthorID
	}

	return s.postRepo.DeleteOwnPost(ctx, req.PostID, req.RequesterID)
}

func (s *contentService) CreateReply(ctx context.Context, req dto.CreateReplyRequest) (*dto.PostResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, ErrInvalidPostContent
	}

	authorID, err := uuid.Parse(req.AuthorID)
	if err != nil {
		return nil, ErrInvalidAuthorID
	}

	replyToPostID, err := uuid.Parse(req.ReplyToPostID)
	if err != nil {
		return nil, ErrInvalidPostID
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = "public"
	}

	post := &dbmodel.Post{
		AuthorID:      authorID,
		ReplyToPostID: &replyToPostID,
		Content:       content,
		Visibility:    visibility,
	}

	if err := s.postRepo.Create(ctx, post); err != nil {
		return nil, err
	}

	if err := s.postRepo.IncrementReplyCount(ctx, req.ReplyToPostID); err != nil {
		return nil, err
	}

	return toPostResponse(post), nil
}

func (s *contentService) GetReplies(ctx context.Context, req dto.GetRepliesRequest) ([]dto.PostResponse, error) {
	if _, err := uuid.Parse(req.PostID); err != nil {
		return nil, ErrInvalidPostID
	}

	replies, err := s.postRepo.FindReplies(ctx, req.PostID, req.Limit)
	if err != nil {
		return nil, err
	}

	responses := make([]dto.PostResponse, 0, len(replies))
	for _, reply := range replies {
		responses = append(responses, *toPostResponse(&reply))
	}

	return responses, nil
}

func toPostResponse(post *dbmodel.Post) *dto.PostResponse {
	replyToPostID := ""
	if post.ReplyToPostID != nil {
		replyToPostID = post.ReplyToPostID.String()
	}

	avatarURL := ""
	if post.Author.AvatarURL != nil {
		avatarURL = *post.Author.AvatarURL
	}

	return &dto.PostResponse{
		ID:            post.ID.String(),
		AuthorID:      post.AuthorID.String(),
		ReplyToPostID: replyToPostID,
		Content:       post.Content,
		Visibility:    post.Visibility,
		LikeCount:     post.LikeCount,
		ReplyCount:    post.ReplyCount,
		RepostCount:   post.RepostCount,
		BookmarkCount: post.BookmarkCount,
		CreatedAt:     post.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     post.UpdatedAt.Format(time.RFC3339),
		Author: dto.UserSummary{
			ID:          post.Author.ID.String(),
			Username:    post.Author.Username,
			DisplayName: post.Author.DisplayName,
			AvatarURL:   avatarURL,
			IsVerified:  post.Author.IsVerified,
		},
	}
}
