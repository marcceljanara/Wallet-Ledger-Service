package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

var rateLimitScript = redis.NewScript(`
	local current = redis.call("INCR", KEYS[1])
	if current == 1 then
		redis.call("EXPIRE", KEYS[1], ARGV[1])
	end
	return current
`)

func RateLimit(redisClient *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "ratelimit:" + ip

		ctx := c.Request.Context()

		count, err := rateLimitScript.Run(ctx, redisClient, []string{key}, int(window.Seconds())).Int64()
		if err != nil {
			c.Next()
			return
		}

		if count > int64(maxRequests) {
			utils.WriteError(c, http.StatusTooManyRequests, "Too many requests. Please try again later.", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
