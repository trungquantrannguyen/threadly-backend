package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"github.com/trungquantrannguyen/threadly/pkg/messaging"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/dto"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/repository"
)

var (
	ErrInvalidPostContent = errors.New("post content is required")
	ErrInvalidPostID      = errors.New("invalid post id")
	ErrInvalidAuthorID    = errors.New("invalid author id")
	ErrInvalidUserID      = errors.New("invalid user id")
	ErrCannotFollowSelf   = errors.New("user cannot follow themselves")
)

type ContentService interface {
	CreatePost(ctx context.Context, req dto.CreatePostRequest) (*dto.PostResponse, error)
	GetPost(ctx context.Context, req dto.GetPostRequest) (*dto.PostResponse, error)
	UpdatePost(ctx context.Context, req dto.UpdatePostRequest) (*dto.PostResponse, error)
	DeletePost(ctx context.Context, req dto.DeletePostRequest) error
	CreateReply(ctx context.Context, req dto.CreateReplyRequest) (*dto.PostResponse, error)
	GetReplies(ctx context.Context, req dto.GetRepliesRequest) ([]dto.PostResponse, error)

	LikePost(ctx context.Context, req dto.LikePostRequest) (*dto.ActionResponse, error)
	UnlikePost(ctx context.Context, req dto.UnlikePostRequest) (*dto.ActionResponse, error)

	BookmarkPost(ctx context.Context, req dto.BookmarkPostRequest) (*dto.ActionResponse, error)
	UnbookmarkPost(ctx context.Context, req dto.UnbookmarkPostRequest) (*dto.ActionResponse, error)

	RepostPost(ctx context.Context, req dto.RepostPostRequest) (*dto.ActionResponse, error)
	UndoRepost(ctx context.Context, req dto.UndoRepostRequest) (*dto.ActionResponse, error)

	FollowUser(ctx context.Context, req dto.FollowUserRequest) (*dto.ActionResponse, error)
	UnfollowUser(ctx context.Context, req dto.UnfollowUserRequest) (*dto.ActionResponse, error)
	GetFollowers(ctx context.Context, req dto.GetFollowersRequest) ([]dto.UserSummary, error)
	GetFollowing(ctx context.Context, req dto.GetFollowingRequest) ([]dto.UserSummary, error)

	GetUserTimeline(ctx context.Context, req dto.GetUserTimelineRequest) ([]dto.TimelineItemResponse, error)
}

type contentService struct {
	postRepo        repository.PostRepository
	interactionRepo repository.InteractionRepository
	eventPublisher  messaging.Publisher
	log             zerolog.Logger
}

func NewContentService(postRepo repository.PostRepository, interactionRepo repository.InteractionRepository, eventPublisher messaging.Publisher, log zerolog.Logger) ContentService {
	return &contentService{
		postRepo:        postRepo,
		interactionRepo: interactionRepo,
		eventPublisher:  eventPublisher,
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

	if err := s.postRepo.AttachMediaToPost(ctx, post.ID, authorID, req.MediaIDs); err != nil {
		return nil, err
	}

	postWithMedia, err := s.postRepo.FindByID(ctx, post.ID.String())
	if err != nil {
		return nil, err
	}
	s.publishEvent(ctx, messaging.EventPostCreated, messaging.Event{
		EventID:   uuid.NewString(),
		Type:      messaging.EventPostCreated,
		ActorID:   req.AuthorID,
		PostID:    post.ID.String(),
		AuthorID:  post.AuthorID.String(),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})

	return toPostResponse(postWithMedia), nil
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

func (s *contentService) UpdatePost(ctx context.Context, req dto.UpdatePostRequest) (*dto.PostResponse, error) {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return nil, ErrInvalidPostContent
	}

	postID, err := uuid.Parse(req.PostID)
	if err != nil {
		return nil, ErrInvalidPostID
	}

	requesterID, err := uuid.Parse(req.RequesterID)
	if err != nil {
		return nil, ErrInvalidAuthorID
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = "public"
	}

	if _, err := s.postRepo.UpdateOwnPost(ctx, req.PostID, req.RequesterID, content, visibility); err != nil {
		return nil, err
	}

	if err := s.postRepo.ReplacePostMedia(ctx, postID, requesterID, req.MediaIDs); err != nil {
		return nil, err
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

	post, err := s.postRepo.FindByID(ctx, req.PostID)
	if err != nil {
		return err
	}

	if err := s.postRepo.DeleteOwnPost(ctx, req.PostID, req.RequesterID); err != nil {
		return err
	}

	s.publishEvent(ctx, messaging.EventPostDeleted, messaging.Event{
		EventID:   uuid.NewString(),
		Type:      messaging.EventPostDeleted,
		ActorID:   req.RequesterID,
		PostID:    req.PostID,
		AuthorID:  post.AuthorID.String(),
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})

	return nil
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

	parentPost, err := s.postRepo.FindByID(ctx, req.ReplyToPostID)
	if err != nil {
		return nil, err
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

	if parentPost.AuthorID.String() != req.AuthorID {
		s.publishEvent(ctx, messaging.EventReplyCreated, messaging.Event{
			EventID:      uuid.NewString(),
			Type:         messaging.EventReplyCreated,
			ActorID:      req.AuthorID,
			PostID:       post.ID.String(),
			AuthorID:     post.AuthorID.String(),
			TargetUserID: parentPost.AuthorID.String(),
			CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		})
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

func (s *contentService) LikePost(ctx context.Context, req dto.LikePostRequest) (*dto.ActionResponse, error) {
	userID, postID, err := parseUserAndPostID(req.UserID, req.PostID)
	if err != nil {
		return nil, err
	}

	if err := s.ensurePostExists(ctx, req.PostID); err != nil {
		return nil, err
	}

	post, err := s.postRepo.FindByID(ctx, req.PostID)
	if err != nil {
		return nil, err
	}

	created, err := s.interactionRepo.LikePost(ctx, userID, postID)
	if err != nil {
		return nil, err
	}

	if !created {
		return actionResponse("post already liked"), nil
	}

	if post.AuthorID.String() != req.UserID {
		s.publishEvent(ctx, messaging.EventPostLiked, messaging.Event{
			EventID:   uuid.NewString(),
			Type:      messaging.EventPostLiked,
			ActorID:   req.UserID,
			PostID:    req.PostID,
			AuthorID:  post.AuthorID.String(),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
	}

	return actionResponse("post liked successfully"), nil
}

func (s *contentService) UnlikePost(ctx context.Context, req dto.UnlikePostRequest) (*dto.ActionResponse, error) {
	userID, postID, err := parseUserAndPostID(req.UserID, req.PostID)
	if err != nil {
		return nil, err
	}

	if err := s.ensurePostExists(ctx, req.PostID); err != nil {
		return nil, err
	}

	deleted, err := s.interactionRepo.UnlikePost(ctx, userID, postID)
	if err != nil {
		return nil, err
	}

	if !deleted {
		return actionResponse("post was not liked"), nil
	}

	return actionResponse("post unliked successfully"), nil
}

func (s *contentService) BookmarkPost(ctx context.Context, req dto.BookmarkPostRequest) (*dto.ActionResponse, error) {
	userID, postID, err := parseUserAndPostID(req.UserID, req.PostID)
	if err != nil {
		return nil, err
	}

	if err := s.ensurePostExists(ctx, req.PostID); err != nil {
		return nil, err
	}

	created, err := s.interactionRepo.BookmarkPost(ctx, userID, postID)
	if err != nil {
		return nil, err
	}

	if !created {
		return actionResponse("post already bookmarked"), nil
	}

	return actionResponse("post bookmarked successfully"), nil
}

func (s *contentService) UnbookmarkPost(ctx context.Context, req dto.UnbookmarkPostRequest) (*dto.ActionResponse, error) {
	userID, postID, err := parseUserAndPostID(req.UserID, req.PostID)
	if err != nil {
		return nil, err
	}

	if err := s.ensurePostExists(ctx, req.PostID); err != nil {
		return nil, err
	}

	deleted, err := s.interactionRepo.UnbookmarkPost(ctx, userID, postID)
	if err != nil {
		return nil, err
	}

	if !deleted {
		return actionResponse("post was not bookmarked"), nil
	}

	return actionResponse("post unbookmarked successfully"), nil
}

func (s *contentService) RepostPost(ctx context.Context, req dto.RepostPostRequest) (*dto.ActionResponse, error) {
	userID, postID, err := parseUserAndPostID(req.UserID, req.PostID)
	if err != nil {
		return nil, err
	}

	post, err := s.postRepo.FindByID(ctx, req.PostID)
	if err != nil {
		return nil, err
	}

	created, err := s.interactionRepo.RepostPost(ctx, userID, postID)
	if err != nil {
		return nil, err
	}

	if !created {
		return actionResponse("post already reposted"), nil
	}

	if post.AuthorID.String() != req.UserID {
		s.publishEvent(ctx, messaging.EventPostReposted, messaging.Event{
			EventID:   uuid.NewString(),
			Type:      messaging.EventPostReposted,
			ActorID:   req.UserID,
			PostID:    req.PostID,
			AuthorID:  post.AuthorID.String(),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
	}

	return actionResponse("post reposted successfully"), nil
}

func (s *contentService) UndoRepost(ctx context.Context, req dto.UndoRepostRequest) (*dto.ActionResponse, error) {
	userID, postID, err := parseUserAndPostID(req.UserID, req.PostID)
	if err != nil {
		return nil, err
	}

	if err := s.ensurePostExists(ctx, req.PostID); err != nil {
		return nil, err
	}

	deleted, err := s.interactionRepo.UndoRepost(ctx, userID, postID)
	if err != nil {
		return nil, err
	}

	if !deleted {
		return actionResponse("post was not reposted"), nil
	}

	return actionResponse("repost removed successfully"), nil
}

func (s *contentService) FollowUser(ctx context.Context, req dto.FollowUserRequest) (*dto.ActionResponse, error) {
	followerID, err := uuid.Parse(req.FollowerID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	followingID, err := uuid.Parse(req.FollowingID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	if followerID == followingID {
		return nil, ErrCannotFollowSelf
	}

	if err := s.ensureUserExists(ctx, followerID); err != nil {
		return nil, err
	}

	if err := s.ensureUserExists(ctx, followingID); err != nil {
		return nil, err
	}

	created, err := s.interactionRepo.FollowUser(ctx, followerID, followingID)
	if err != nil {
		return nil, err
	}

	if !created {
		return actionResponse("user already followed"), nil
	}

	s.publishEvent(ctx, messaging.EventUserFollowed, messaging.Event{
		EventID:      uuid.NewString(),
		Type:         messaging.EventUserFollowed,
		ActorID:      followerID.String(),
		TargetUserID: followingID.String(),
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	})

	return actionResponse("user followed successfully"), nil
}

func (s *contentService) UnfollowUser(ctx context.Context, req dto.UnfollowUserRequest) (*dto.ActionResponse, error) {
	followerID, err := uuid.Parse(req.FollowerID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	followingID, err := uuid.Parse(req.FollowingID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	if followerID == followingID {
		return nil, ErrCannotFollowSelf
	}

	if err := s.ensureUserExists(ctx, followerID); err != nil {
		return nil, err
	}

	if err := s.ensureUserExists(ctx, followingID); err != nil {
		return nil, err
	}

	deleted, err := s.interactionRepo.UnfollowUser(ctx, followerID, followingID)
	if err != nil {
		return nil, err
	}

	if !deleted {
		return actionResponse("user was not followed"), nil
	}
	s.publishEvent(ctx, messaging.EventUserUnfollowed, messaging.Event{
		EventID:      uuid.NewString(),
		Type:         messaging.EventUserUnfollowed,
		ActorID:      followerID.String(),
		TargetUserID: followingID.String(),
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	})

	return actionResponse("user unfollowed successfully"), nil
}

func (s *contentService) GetFollowers(ctx context.Context, req dto.GetFollowersRequest) ([]dto.UserSummary, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	if err := s.ensureUserExists(ctx, userID); err != nil {
		return nil, err
	}

	users, err := s.interactionRepo.GetFollowers(ctx, userID, normalizeLimit(req.Limit))
	if err != nil {
		return nil, err
	}

	return toUserSummaryResponses(users), nil
}

func (s *contentService) GetFollowing(ctx context.Context, req dto.GetFollowingRequest) ([]dto.UserSummary, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	if err := s.ensureUserExists(ctx, userID); err != nil {
		return nil, err
	}

	users, err := s.interactionRepo.GetFollowing(ctx, userID, normalizeLimit(req.Limit))
	if err != nil {
		return nil, err
	}

	return toUserSummaryResponses(users), nil
}

func parseUserAndPostID(userIDValue string, postIDValue string) (uuid.UUID, uuid.UUID, error) {
	userID, err := uuid.Parse(userIDValue)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidUserID
	}

	postID, err := uuid.Parse(postIDValue)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidPostID
	}

	return userID, postID, nil
}

func (s *contentService) ensurePostExists(ctx context.Context, postID string) error {
	_, err := s.postRepo.FindByID(ctx, postID)
	return err
}

func (s *contentService) ensureUserExists(ctx context.Context, userID uuid.UUID) error {
	exists, err := s.interactionRepo.UserExists(ctx, userID)
	if err != nil {
		return err
	}

	if !exists {
		return repository.ErrUserNotFound
	}

	return nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > 50 {
		return 20
	}

	return limit
}

func actionResponse(message string) *dto.ActionResponse {
	return &dto.ActionResponse{
		Success: true,
		Message: message,
	}
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

	mediaResponses := make([]dto.MediaResponse, 0, len(post.Media))
	for _, media := range post.Media {
		mediaResponses = append(mediaResponses, toMediaResponse(media))
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
		Media: mediaResponses,
	}
}

func toUserSummaryResponses(users []dbmodel.User) []dto.UserSummary {
	responses := make([]dto.UserSummary, 0, len(users))

	for _, user := range users {
		responses = append(responses, toUserSummaryResponse(&user))
	}

	return responses
}

func toUserSummaryResponse(user *dbmodel.User) dto.UserSummary {
	avatarURL := ""
	if user.AvatarURL != nil {
		avatarURL = *user.AvatarURL
	}

	return dto.UserSummary{
		ID:          user.ID.String(),
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   avatarURL,
		IsVerified:  user.IsVerified,
	}
}

func (s *contentService) GetUserTimeline(ctx context.Context, req dto.GetUserTimelineRequest) ([]dto.TimelineItemResponse, error) {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, ErrInvalidUserID
	}

	if err := s.ensureUserExists(ctx, userID); err != nil {
		return nil, err
	}

	items, err := s.postRepo.FindUserTimeline(ctx, userID, normalizeLimit(req.Limit))
	if err != nil {
		return nil, err
	}

	responses := make([]dto.TimelineItemResponse, 0, len(items))

	for _, item := range items {
		repostedAt := ""
		if item.RepostedAt != nil {
			repostedAt = item.RepostedAt.Format(time.RFC3339)
		}

		responses = append(responses, dto.TimelineItemResponse{
			Type:       item.Type,
			Post:       *toPostResponse(&item.Post),
			RepostedAt: repostedAt,
		})
	}

	return responses, nil
}

func (s *contentService) publishEvent(ctx context.Context, routingKey string, event messaging.Event) {
	if s.eventPublisher == nil {
		return
	}

	if err := s.eventPublisher.Publish(ctx, routingKey, event); err != nil {
		s.log.Warn().
			Err(err).
			Str("event_type", routingKey).
			Msg("failed to publish content event")
	}
}

func toMediaResponse(media dbmodel.Media) dto.MediaResponse {
	width := 0
	if media.Width != nil {
		width = *media.Width
	}

	height := 0
	if media.Height != nil {
		height = *media.Height
	}

	return dto.MediaResponse{
		ID:        media.ID.String(),
		URL:       media.URL,
		MimeType:  media.MimeType,
		SizeBytes: media.SizeBytes,
		Width:     width,
		Height:    height,
	}
}
