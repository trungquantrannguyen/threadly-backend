package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	dbmodel "github.com/trungquantrannguyen/threadly/db/models"
	"github.com/trungquantrannguyen/threadly/pkg/config"
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeFeedService struct {
	posts      []dbmodel.Post
	nextCursor string
	err        error

	called bool
}

func (f *fakeFeedService) GetHomeFeed(ctx context.Context, userID string, limit int, cursor string) ([]dbmodel.Post, string, error) {
	f.called = true

	if f.err != nil {
		return nil, "", f.err
	}

	return f.posts, f.nextCursor, nil
}

type fakeFeedCache struct {
	getResponse *feedpb.HomeFeedResponse
	getErr      error
	setErr      error

	getCalled    bool
	setCalled    bool
	deleteCalled bool

	setFeed *feedpb.HomeFeedResponse
}

func (f *fakeFeedCache) GetHomeFeed(ctx context.Context, userID string, limit int32, cursor string) (*feedpb.HomeFeedResponse, error) {
	f.getCalled = true

	if f.getErr != nil {
		return nil, f.getErr
	}

	return f.getResponse, nil
}

func (f *fakeFeedCache) SetHomeFeed(ctx context.Context, userID string, limit int32, cursor string, feed *feedpb.HomeFeedResponse) error {
	f.setCalled = true
	f.setFeed = feed

	if f.setErr != nil {
		return f.setErr
	}

	return nil
}

func (f *fakeFeedCache) DeleteHomeFeedByUserID(ctx context.Context, userID string) error {
	f.deleteCalled = true
	return nil
}

func TestGetHomeFeedReturnsCachedFeedOnCacheHit(t *testing.T) {
	cached := &feedpb.HomeFeedResponse{
		Posts: []*feedpb.FeedPostResponse{
			{
				Id:      uuid.NewString(),
				Content: "cached post",
			},
		},
		NextCursor: "cached-cursor",
	}

	feedSvc := &fakeFeedService{}
	feedCache := &fakeFeedCache{
		getResponse: cached,
	}

	server := NewFeedServiceServer(
		config.Config{ServiceName: "feed-service", AppEnv: "test"},
		zerolog.Nop(),
		feedSvc,
		feedCache,
	)

	res, err := server.GetHomeFeed(context.Background(), &feedpb.GetHomeFeedRequest{
		UserId: uuid.NewString(),
		Limit:  20,
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, feedCache.getCalled)
	assert.False(t, feedSvc.called)
	assert.False(t, feedCache.setCalled)

	assert.Equal(t, "cached post", res.GetPosts()[0].GetContent())
	assert.Equal(t, "cached-cursor", res.GetNextCursor())
}

func TestGetHomeFeedCacheMissCallsServiceAndCachesResult(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	mediaID := uuid.New()
	now := time.Now().UTC()

	avatarURL := "https://example.com/avatar.png"

	feedSvc := &fakeFeedService{
		posts: []dbmodel.Post{
			{
				ID:            postID,
				AuthorID:      userID,
				Content:       "fresh post",
				Visibility:    "public",
				LikeCount:     1,
				ReplyCount:    2,
				RepostCount:   3,
				BookmarkCount: 4,
				CreatedAt:     now,
				UpdatedAt:     now,
				Author: dbmodel.User{
					ID:          userID,
					Username:    "trungquan",
					DisplayName: "Trung Quan",
					AvatarURL:   &avatarURL,
					IsVerified:  true,
				},
				Media: []dbmodel.Media{
					{
						ID:        mediaID,
						URL:       "https://example.com/media.png",
						MimeType:  "image/png",
						SizeBytes: 123,
					},
				},
			},
		},
		nextCursor: "fresh-cursor",
	}

	feedCache := &fakeFeedCache{
		getErr: errors.New("cache miss"),
	}

	server := NewFeedServiceServer(
		config.Config{ServiceName: "feed-service", AppEnv: "test"},
		zerolog.Nop(),
		feedSvc,
		feedCache,
	)

	res, err := server.GetHomeFeed(context.Background(), &feedpb.GetHomeFeedRequest{
		UserId: userID.String(),
		Limit:  20,
		Cursor: "",
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, feedCache.getCalled)
	assert.True(t, feedSvc.called)
	assert.True(t, feedCache.setCalled)

	require.Len(t, res.GetPosts(), 1)

	post := res.GetPosts()[0]
	assert.Equal(t, postID.String(), post.GetId())
	assert.Equal(t, userID.String(), post.GetAuthorId())
	assert.Equal(t, "fresh post", post.GetContent())
	assert.Equal(t, "public", post.GetVisibility())
	assert.Equal(t, int32(1), post.GetLikeCount())
	assert.Equal(t, int32(2), post.GetReplyCount())
	assert.Equal(t, int32(3), post.GetRepostCount())
	assert.Equal(t, int32(4), post.GetBookmarkCount())
	assert.Equal(t, "fresh-cursor", res.GetNextCursor())

	require.NotNil(t, post.GetAuthor())
	assert.Equal(t, "trungquan", post.GetAuthor().GetUsername())
	assert.Equal(t, "Trung Quan", post.GetAuthor().GetDisplayName())
	assert.Equal(t, "https://example.com/avatar.png", post.GetAuthor().GetAvatarUrl())
	assert.True(t, post.GetAuthor().GetIsVerified())

	require.Len(t, post.GetMedia(), 1)
	assert.Equal(t, mediaID.String(), post.GetMedia()[0].GetId())
	assert.Equal(t, "https://example.com/media.png", post.GetMedia()[0].GetUrl())
	assert.Equal(t, "image/png", post.GetMedia()[0].GetMimeType())
	assert.Equal(t, int64(123), post.GetMedia()[0].GetSizeBytes())

	require.NotNil(t, feedCache.setFeed)
	require.Len(t, feedCache.setFeed.GetPosts(), 1)
}

func TestGetHomeFeedReturnsInternalErrorWhenServiceFails(t *testing.T) {
	feedSvc := &fakeFeedService{
		err: errors.New("database failed"),
	}

	feedCache := &fakeFeedCache{
		getErr: errors.New("cache miss"),
	}

	server := NewFeedServiceServer(
		config.Config{ServiceName: "feed-service", AppEnv: "test"},
		zerolog.Nop(),
		feedSvc,
		feedCache,
	)

	res, err := server.GetHomeFeed(context.Background(), &feedpb.GetHomeFeedRequest{
		UserId: uuid.NewString(),
		Limit:  20,
	})

	require.Error(t, err)
	assert.Nil(t, res)

	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestStringValueReturnsEmptyStringForNil(t *testing.T) {
	assert.Equal(t, "", stringValue(nil))
}

func TestStringValueReturnsStringValue(t *testing.T) {
	value := "hello"
	assert.Equal(t, "hello", stringValue(&value))
}

func TestGetHealth(t *testing.T) {
	server := NewFeedServiceServer(
		config.Config{
			ServiceName: "feed-service",
			AppEnv:      "test",
		},
		zerolog.Nop(),
		&fakeFeedService{},
		&fakeFeedCache{},
	)

	res, err := server.GetHealth(context.Background(), &feedpb.GetFeedServiceHealthRequest{})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.Equal(t, "ok", res.GetStatus())
	assert.Equal(t, "feed-service", res.GetService())
	assert.Equal(t, "test", res.GetEnv())
	assert.NotEmpty(t, res.GetCheckedAt())
}

func TestGetHomeFeedStillReturnsWhenCacheSetFails(t *testing.T) {
	userID := uuid.New()
	postID := uuid.New()
	now := time.Now().UTC()

	feedSvc := &fakeFeedService{
		posts: []dbmodel.Post{
			{
				ID:         postID,
				AuthorID:   userID,
				Content:    "fresh post",
				Visibility: "public",
				CreatedAt:  now,
				UpdatedAt:  now,
				Author: dbmodel.User{
					ID:          userID,
					Username:    "trungquan",
					DisplayName: "Trung Quan",
				},
			},
		},
		nextCursor: "",
	}

	feedCache := &fakeFeedCache{
		getErr: errors.New("cache miss"),
		setErr: errors.New("redis set failed"),
	}

	server := NewFeedServiceServer(
		config.Config{ServiceName: "feed-service", AppEnv: "test"},
		zerolog.Nop(),
		feedSvc,
		feedCache,
	)

	res, err := server.GetHomeFeed(context.Background(), &feedpb.GetHomeFeedRequest{
		UserId: userID.String(),
		Limit:  20,
	})

	require.NoError(t, err)
	require.NotNil(t, res)

	assert.True(t, feedCache.setCalled)
	require.Len(t, res.GetPosts(), 1)
	assert.Equal(t, "fresh post", res.GetPosts()[0].GetContent())
}

func TestToFeedMediaResponsesHandlesNilWidthAndHeight(t *testing.T) {
	mediaID := uuid.New()

	result := toFeedMediaResponses([]dbmodel.Media{
		{
			ID:        mediaID,
			URL:       "https://example.com/media.png",
			MimeType:  "image/png",
			SizeBytes: 123,
			Width:     nil,
			Height:    nil,
		},
	})

	require.Len(t, result, 1)

	assert.Equal(t, mediaID.String(), result[0].GetId())
	assert.Equal(t, "https://example.com/media.png", result[0].GetUrl())
	assert.Equal(t, "image/png", result[0].GetMimeType())
	assert.Equal(t, int64(123), result[0].GetSizeBytes())
	assert.Equal(t, int32(0), result[0].GetWidth())
	assert.Equal(t, int32(0), result[0].GetHeight())
}
