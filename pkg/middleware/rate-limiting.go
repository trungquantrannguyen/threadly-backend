package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RedisRateLimiter(redisClient *redis.Client, limit int64, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		identifier := rateLimitIdentifier(c)
		key := fmt.Sprintf("rate_limit:%s:%s", c.FullPath(), identifier)

		ctx := c.Request.Context()

		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			_ = redisClient.Expire(ctx, key, window).Err()
		}

		if count > limit {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "Too many requests, please try again later",
			})
			return
		}

		c.Next()
	}
}

func rateLimitIdentifier(c *gin.Context) string {
	userID := GetUserID(c)
	if userID != "" {
		return "user:" + userID
	}

	return "ip:" + c.ClientIP()
}
