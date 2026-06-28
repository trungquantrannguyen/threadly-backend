package grpc

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	contentpb "github.com/trungquantrannguyen/threadly/proto/content"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/dto"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/repository"
	"github.com/trungquantrannguyen/threadly/services/content-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ContentServiceServer struct {
	contentpb.UnimplementedContentServiceServer
	cfg            config.Config
	log            zerolog.Logger
	contentService service.ContentService
}

func NewContentServiceServer(cfg config.Config, log zerolog.Logger, contentService service.ContentService) *ContentServiceServer {
	return &ContentServiceServer{
		cfg:            cfg,
		log:            log,
		contentService: contentService,
	}
}

func (s *ContentServiceServer) GetHealth(ctx context.Context, req *contentpb.GetContentServiceHealthRequest) (*contentpb.GetContentServiceHealthResponse, error) {
	s.log.Info().Msg("Content service is healthy")
	return &contentpb.GetContentServiceHealthResponse{
		Status:    "ok",
		Service:   s.cfg.ServiceName,
		Env:       s.cfg.AppEnv,
		CheckedAt: time.Now().String(),
	}, nil
}

func (s *ContentServiceServer) CreatePost(ctx context.Context, req *contentpb.CreatePostRequest) (*contentpb.PostResponse, error) {
	res, err := s.contentService.CreatePost(ctx, dto.CreatePostRequest{
		AuthorID:   req.GetAuthorId(),
		Content:    req.GetContent(),
		Visibility: req.GetVisibility(),
	})
	if err != nil {
		return nil, mapContentServiceError(err)
	}

	return toProtoPostResponse(res), nil
}

func (s *ContentServiceServer) GetPost(ctx context.Context, req *contentpb.GetPostRequest) (*contentpb.PostResponse, error) {
	res, err := s.contentService.GetPost(ctx, dto.GetPostRequest{
		PostID:   req.GetPostId(),
		ViewerID: req.GetViewerId(),
	})
	if err != nil {
		return nil, mapContentServiceError(err)
	}

	return toProtoPostResponse(res), nil
}

func (s *ContentServiceServer) DeletePost(ctx context.Context, req *contentpb.DeletePostRequest) (*contentpb.DeletePostResponse, error) {
	err := s.contentService.DeletePost(ctx, dto.DeletePostRequest{
		PostID:      req.GetPostId(),
		RequesterID: req.GetRequesterId(),
	})
	if err != nil {
		return nil, mapContentServiceError(err)
	}

	return &contentpb.DeletePostResponse{
		Success: true,
		Message: "post deleted successfully",
	}, nil
}

func (s *ContentServiceServer) CreateReply(ctx context.Context, req *contentpb.CreateReplyRequest) (*contentpb.PostResponse, error) {
	res, err := s.contentService.CreateReply(ctx, dto.CreateReplyRequest{
		AuthorID:      req.GetAuthorId(),
		ReplyToPostID: req.GetReplyToPostId(),
		Content:       req.GetContent(),
		Visibility:    req.GetVisibility(),
	})
	if err != nil {
		return nil, mapContentServiceError(err)
	}

	return toProtoPostResponse(res), nil
}

func (s *ContentServiceServer) GetReplies(ctx context.Context, req *contentpb.GetRepliesRequest) (*contentpb.PostListResponse, error) {
	res, err := s.contentService.GetReplies(ctx, dto.GetRepliesRequest{
		PostID: req.GetPostId(),
		Limit:  int(req.GetLimit()),
		Cursor: req.GetCursor(),
	})
	if err != nil {
		return nil, mapContentServiceError(err)
	}

	posts := make([]*contentpb.PostResponse, 0, len(res))

	for _, post := range res {
		posts = append(posts, toProtoPostResponse(&post))
	}

	return &contentpb.PostListResponse{
		Posts: posts,
	}, nil
}

func (s *ContentServiceServer) LikePost(ctx context.Context, req *contentpb.LikePostRequest) (*contentpb.ActionResponse, error) {
	res, err := s.contentService.LikePost(ctx, dto.LikePostRequest{
		UserID: req.GetUserId(),
		PostID: req.GetPostId(),
	})
	if err != nil {
		return nil, mapContentServiceError(err)
	}

	return &contentpb.ActionResponse{
		Success: res.Success,
		Message: res.Message,
	}, nil
}

func (s *ContentServiceServer) UnlikePost(ctx context.Context, req *contentpb.UnlikePostRequest) (*contentpb.ActionResponse, error) {
	res, err := s.contentService.UnlikePost(ctx, dto.UnlikePostRequest{
		UserID: req.GetUserId(),
		PostID: req.GetPostId(),
	})
	if err != nil {
		return nil, mapContentServiceError(err)
	}

	return &contentpb.ActionResponse{
		Success: res.Success,
		Message: res.Message,
	}, nil
}

func (s *ContentServiceServer) BookmarkPost(ctx context.Context, req *contentpb.BookmarkPostRequest) (*contentpb.ActionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "BookmarkPost not implemented yet")
}

func (s *ContentServiceServer) UnbookmarkPost(ctx context.Context, req *contentpb.UnbookmarkPostRequest) (*contentpb.ActionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "UnbookmarkPost not implemented yet")
}

func (s *ContentServiceServer) RepostPost(ctx context.Context, req *contentpb.RepostPostRequest) (*contentpb.ActionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "RepostPost not implemented yet")
}

func (s *ContentServiceServer) UndoRepost(ctx context.Context, req *contentpb.UndoRepostRequest) (*contentpb.ActionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "UndoRepost not implemented yet")
}

func (s *ContentServiceServer) FollowUser(ctx context.Context, req *contentpb.FollowUserRequest) (*contentpb.ActionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "FollowUser not implemented yet")
}

func (s *ContentServiceServer) UnfollowUser(ctx context.Context, req *contentpb.UnfollowUserRequest) (*contentpb.ActionResponse, error) {
	return nil, status.Error(codes.Unimplemented, "UnfollowUser not implemented yet")
}

func (s *ContentServiceServer) GetFollowers(ctx context.Context, req *contentpb.GetFollowersRequest) (*contentpb.UserListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetFollowers not implemented yet")
}

func (s *ContentServiceServer) GetFollowing(ctx context.Context, req *contentpb.GetFollowingRequest) (*contentpb.UserListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetFollowing not implemented yet")
}

func toProtoPostResponse(post *dto.PostResponse) *contentpb.PostResponse {
	return &contentpb.PostResponse{
		Id:            post.ID,
		AuthorId:      post.AuthorID,
		ReplyToPostId: post.ReplyToPostID,
		Content:       post.Content,
		Visibility:    post.Visibility,
		LikeCount:     int32(post.LikeCount),
		ReplyCount:    int32(post.ReplyCount),
		RepostCount:   int32(post.RepostCount),
		BookmarkCount: int32(post.BookmarkCount),
		CreatedAt:     post.CreatedAt,
		UpdatedAt:     post.UpdatedAt,
		Author: &contentpb.UserSummary{
			Id:          post.Author.ID,
			Username:    post.Author.Username,
			DisplayName: post.Author.DisplayName,
			AvatarUrl:   post.Author.AvatarURL,
			IsVerified:  post.Author.IsVerified,
		},
	}
}

func mapContentServiceError(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidPostContent),
		errors.Is(err, service.ErrInvalidPostID),
		errors.Is(err, service.ErrInvalidAuthorID):
		return status.Error(codes.InvalidArgument, err.Error())

	case errors.Is(err, repository.ErrPostNotFound):
		return status.Error(codes.NotFound, "post not found")

	case errors.Is(err, repository.ErrForbidden):
		return status.Error(codes.PermissionDenied, "you do not have permission to perform this action")

	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
