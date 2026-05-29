package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

func RateLimit(redisClient *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "ratelimit:" + ip

		ctx := c.Request.Context()

		pipe := redisClient.Pipeline()
		incr := pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, window)

		_, err := pipe.Exec(ctx)
		if err != nil {
			c.Next()
			return
		}

		count := incr.Val()
		if count > int64(maxRequests) {
			utils.WriteError(c, http.StatusTooManyRequests, "Too many requests. Please try again later.", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
