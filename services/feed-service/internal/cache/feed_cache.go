package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	feedpb "github.com/trungquantrannguyen/threadly/proto/feed"
)

type FeedCache interface {
	GetHomeFeed(ctx context.Context, userID string, limit int32, cursor string) (*feedpb.HomeFeedResponse, error)
	SetHomeFeed(ctx context.Context, userID string, limit int32, cursor string, feed *feedpb.HomeFeedResponse) error
	DeleteHomeFeedByUserID(ctx context.Context, userID string) error
}

type feedCache struct {
	redis *redis.Client
	ttl   time.Duration
}

func NewFeedCache(redisClient *redis.Client, ttl time.Duration) FeedCache {
	return &feedCache{
		redis: redisClient,
		ttl:   ttl,
	}
}

func (c *feedCache) GetHomeFeed(ctx context.Context, userID string, limit int32, cursor string) (*feedpb.HomeFeedResponse, error) {
	key := homeFeedKey(userID, limit, cursor)

	value, err := c.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var feed feedpb.HomeFeedResponse
	if err := json.Unmarshal([]byte(value), &feed); err != nil {
		return nil, err
	}

	return &feed, nil
}

func (c *feedCache) SetHomeFeed(ctx context.Context, userID string, limit int32, cursor string, feed *feedpb.HomeFeedResponse) error {
	key := homeFeedKey(userID, limit, cursor)

	bytes, err := json.Marshal(feed)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, key, bytes, c.ttl).Err()
}

func (c *feedCache) DeleteHomeFeedByUserID(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("feed:home:%s:*", userID)

	iter := c.redis.Scan(ctx, 0, pattern, 100).Iterator()
	for iter.Next(ctx) {
		if err := c.redis.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}

	return iter.Err()
}

func homeFeedKey(userID string, limit int32, cursor string) string {
	if cursor == "" {
		cursor = "first"
	}

	return fmt.Sprintf("feed:home:%s:limit:%d:cursor:%s", userID, limit, cursor)
}
