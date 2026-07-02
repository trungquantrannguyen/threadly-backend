package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
)

func newTestFeedCache(t *testing.T) (*miniredis.Miniredis, FeedCache) {
	t.Helper()

	server := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{
		Addr: server.Addr(),
	})

	return server, NewFeedCache(client, time.Minute)
}

func TestHomeFeedKeyWithoutCursor(t *testing.T) {
	key := homeFeedKey("user-1", 20, "")

	assert.Equal(t, "feed:home:user-1:limit:20:cursor:first", key)
}

func TestHomeFeedKeyWithCursor(t *testing.T) {
	key := homeFeedKey("user-1", 20, "cursor-123")

	assert.Equal(t, "feed:home:user-1:limit:20:cursor:cursor-123", key)
}

func TestSetAndGetHomeFeed(t *testing.T) {
	_, feedCache := newTestFeedCache(t)

	expected := &feedpb.HomeFeedResponse{
		Posts: []*feedpb.FeedPostResponse{
			{
				Id:      "post-1",
				Content: "hello",
				Media: []*feedpb.FeedMediaResponse{
					{
						Id:       "media-1",
						Url:      "https://example.com/image.png",
						MimeType: "image/png",
					},
				},
			},
		},
		NextCursor: "next-cursor",
	}

	err := feedCache.SetHomeFeed(context.Background(), "user-1", 20, "", expected)
	require.NoError(t, err)

	actual, err := feedCache.GetHomeFeed(context.Background(), "user-1", 20, "")
	require.NoError(t, err)
	require.NotNil(t, actual)

	assert.Equal(t, expected.GetNextCursor(), actual.GetNextCursor())
	require.Len(t, actual.GetPosts(), 1)
	assert.Equal(t, "post-1", actual.GetPosts()[0].GetId())
	assert.Equal(t, "hello", actual.GetPosts()[0].GetContent())

	require.Len(t, actual.GetPosts()[0].GetMedia(), 1)
	assert.Equal(t, "media-1", actual.GetPosts()[0].GetMedia()[0].GetId())
}

func TestGetHomeFeedReturnsErrorWhenMissing(t *testing.T) {
	_, feedCache := newTestFeedCache(t)

	res, err := feedCache.GetHomeFeed(context.Background(), "missing-user", 20, "")

	require.Error(t, err)
	assert.Nil(t, res)
}

func TestDeleteHomeFeedByUserID(t *testing.T) {
	redisServer, feedCache := newTestFeedCache(t)

	feed := &feedpb.HomeFeedResponse{
		Posts: []*feedpb.FeedPostResponse{
			{Id: "post-1"},
		},
	}

	require.NoError(t, feedCache.SetHomeFeed(context.Background(), "user-1", 20, "", feed))
	require.NoError(t, feedCache.SetHomeFeed(context.Background(), "user-1", 50, "cursor-1", feed))
	require.NoError(t, feedCache.SetHomeFeed(context.Background(), "user-2", 20, "", feed))

	err := feedCache.DeleteHomeFeedByUserID(context.Background(), "user-1")
	require.NoError(t, err)

	assert.False(t, redisServer.Exists("feed:home:user-1:limit:20:cursor:first"))
	assert.False(t, redisServer.Exists("feed:home:user-1:limit:50:cursor:cursor-1"))
	assert.True(t, redisServer.Exists("feed:home:user-2:limit:20:cursor:first"))
}

func TestGetHomeFeedReturnsErrorForInvalidJSON(t *testing.T) {
	redisServer, feedCache := newTestFeedCache(t)

	redisServer.Set(
		"feed:home:user-1:limit:20:cursor:first",
		"{bad-json",
	)

	res, err := feedCache.GetHomeFeed(context.Background(), "user-1", 20, "")

	require.Error(t, err)
	assert.Nil(t, res)
}
