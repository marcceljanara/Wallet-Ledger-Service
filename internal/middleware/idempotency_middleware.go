package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"marcceljanara/wallet-ledger-service/internal/utils"
)

// cachedResponse represents the structure stored in Redis to cache successful
// HTTP responses for subsequent identical idempotent requests.
type cachedResponse struct {
	StatusCode int    `json:"status_code"`
	Body       string `json:"body"`
}

// responseCapture is a custom wrapper around gin.ResponseWriter used to intercept
// and buffer HTTP response bodies and status codes during processing.
type responseCapture struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

// Write intercepts bytes written to the response and buffers them.
func (w *responseCapture) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteString intercepts string content written to the response and buffers it.
func (w *responseCapture) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// WriteHeader intercepts and stores the HTTP status code.
func (w *responseCapture) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Idempotency implements a middleware that prevents duplicate execution of mutation requests (e.g. TopUp, Transfer).
// It checks for a unique "Idempotency-Key" header, checks Redis for cached results, and caches successful (2xx) responses.
func Idempotency(redisClient *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only enforce idempotency for POST requests as they perform mutations.
		if c.Request.Method != http.MethodPost {
			c.Next()
			return
		}

		// Ensure the client provided the Idempotency-Key header.
		key := c.GetHeader("Idempotency-Key")
		if key == "" {
			utils.WriteError(c, http.StatusBadRequest, "Idempotency-Key header is required", nil)
			c.Abort()
			return
		}

		redisKey := "idempotency:" + key

		// Attempt to fetch previous response from Redis (Cache Hit scenario).
		val, err := redisClient.Get(c, redisKey).Result()
		if err == nil {
			var res cachedResponse
			if err := json.Unmarshal([]byte(val), &res); err == nil {
				c.Writer.Header().Set("Content-Type", "application/json")
				c.Writer.Header().Set("X-Cache", "HIT")
				c.Writer.WriteHeader(res.StatusCode)
				_, _ = c.Writer.Write([]byte(res.Body))
				c.Abort()
				return
			}
		}

		// Cache Miss scenario: wrap response writer to capture output.
		capture := &responseCapture{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
			statusCode:     http.StatusOK,
		}
		c.Writer = capture

		c.Next()

		// Cache successful responses (2xx) in Redis for 24 hours.
		if capture.statusCode >= 200 && capture.statusCode < 300 {
			res := cachedResponse{
				StatusCode: capture.statusCode,
				Body:       capture.body.String(),
			}
			data, err := json.Marshal(res)
			if err == nil {
				_ = redisClient.Set(c, redisKey, data, 24*time.Hour).Err()
			}
		}
	}
}
