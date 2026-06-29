package grpc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
	"github.com/trungquantrannguyen/threadly/services/feed-service/internal/cache"
	"github.com/trungquantrannguyen/threadly/services/feed-service/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FeedServiceServer struct {
	feedpb.UnimplementedFeedServiceServer
	cfg         config.Config
	log         zerolog.Logger
	feedService service.FeedService
	feedCache   cache.FeedCache
}

func NewFeedServiceServer(cfg config.Config, log zerolog.Logger, feedService service.FeedService, feedCache cache.FeedCache) *FeedServiceServer {
	return &FeedServiceServer{
		cfg:         cfg,
		log:         log,
		feedService: feedService,
		feedCache:   feedCache,
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
	cachedFeed, err := s.feedCache.GetHomeFeed(ctx, req.GetUserId(), req.GetLimit(), req.GetCursor())
	if err == nil {
		s.log.Info().
			Str("user_id", req.GetUserId()).
			Msg("home feed cache hit")

		return cachedFeed, nil
	}

	s.log.Info().
		Str("user_id", req.GetUserId()).
		Msg("home feed cache miss")

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

	res := &feedpb.HomeFeedResponse{
		Posts:      resPosts,
		NextCursor: nextCursor,
	}

	if err := s.feedCache.SetHomeFeed(ctx, req.GetUserId(), req.GetLimit(), req.GetCursor(), res); err != nil {
		s.log.Warn().
			Err(err).
			Str("user_id", req.GetUserId()).
			Msg("failed to cache home feed")
	}

	return res, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
