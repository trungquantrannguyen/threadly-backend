package grpc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
	"github.com/trungquantrannguyen/threadly/services/feed-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FeedServiceServer struct {
	feedpb.UnimplementedFeedServiceServer
	cfg         config.Config
	log         zerolog.Logger
	feedService service.FeedService
}

func NewFeedServiceServer(cfg config.Config, log zerolog.Logger, feedService service.FeedService) *FeedServiceServer {
	return &FeedServiceServer{
		cfg:         cfg,
		log:         log,
		feedService: feedService,
	}
}

func (s *FeedServiceServer) GetHealth(ctx context.Context, req *feedpb.GetFeedServiceHealthRequest) (*feedpb.GetFeedServiceHealthResponse, error) {
	s.log.Info().Msg("Feed service is healthy")
	return &feedpb.GetFeedServiceHealthResponse{
		Status:    "ok",
		Service:   s.cfg.ServiceName,
		Env:       s.cfg.AppEnv,
		CheckedAt: time.Now().String(),
	}, nil
}

func (s *FeedServiceServer) GetHomeFeed(ctx context.Context, req *feedpb.GetHomeFeedRequest) (*feedpb.HomeFeedResponse, error) {
	posts, nextCursor, err := s.feedService.GetHomeFeed(
		ctx,
		req.GetUserId(),
		int(req.GetLimit()),
		req.GetCursor(),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get home feed")
	}

	resPosts := make([]*feedpb.FeedPostResponse, 0, len(posts))

	for _, post := range posts {
		author := post.Author

		resPosts = append(resPosts, &feedpb.FeedPostResponse{
			Id:            post.ID.String(),
			AuthorId:      post.AuthorID.String(),
			Content:       post.Content,
			Visibility:    post.Visibility,
			LikeCount:     int32(post.LikeCount),
			ReplyCount:    int32(post.ReplyCount),
			RepostCount:   int32(post.RepostCount),
			BookmarkCount: int32(post.BookmarkCount),
			CreatedAt:     post.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt:     post.UpdatedAt.Format(time.RFC3339Nano),
			Author: &feedpb.FeedUserSummary{
				Id:          author.ID.String(),
				Username:    author.Username,
				DisplayName: author.DisplayName,
				AvatarUrl:   stringValue(author.AvatarURL),
				IsVerified:  author.IsVerified,
			},
		})
	}

	return &feedpb.HomeFeedResponse{
		Posts:      resPosts,
		NextCursor: nextCursor,
	}, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
