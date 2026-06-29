package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/trungquantrannguyen/threadly/pkg/config"
)

func NewRedisClient(cfg config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
}

func Ping(ctx context.Context, client *redis.Client) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	return client.Ping(ctx).Err()
}
