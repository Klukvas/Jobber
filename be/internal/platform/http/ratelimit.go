package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// MaxRequests is the maximum number of requests allowed within the window
	MaxRequests int
	// Window is the time window for rate limiting
	Window time.Duration
	// KeyPrefix is prepended to the rate limit key (e.g., "auth", "api")
	KeyPrefix string
}

// rateLimitScript atomically increments the counter and sets TTL only on first access.
// Returns the current count after increment.
var rateLimitScript = redis.NewScript(`
local key = KEYS[1]
local count = redis.call("INCR", key)
if count == 1 then
    redis.call("EXPIRE", key, ARGV[1])
end
return count
`)

// RateLimitMiddleware creates a Redis-based rate limiting middleware.
// Limits requests by client IP using a fixed-window counter in Redis.
// Uses a Lua script for atomic INCR + EXPIRE to prevent TTL-less keys on crashes.
func RateLimitMiddleware(rdb *redis.Client, cfg RateLimitConfig, logger *zap.Logger) gin.HandlerFunc {
	return rateLimitByKey(rdb, cfg, logger, func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// UserRateLimitMiddleware creates a rate limiter keyed by authenticated user ID.
// Falls back to client IP if user_id is not set in context (e.g. unauthenticated request).
// Place this AFTER AuthMiddleware in the middleware chain.
func UserRateLimitMiddleware(rdb *redis.Client, cfg RateLimitConfig, logger *zap.Logger) gin.HandlerFunc {
	return rateLimitByKey(rdb, cfg, logger, func(c *gin.Context) string {
		if raw, exists := c.Get("user_id"); exists {
			if userID, ok := raw.(string); ok {
				return "user:" + userID
			}
		}
		return c.ClientIP()
	})
}

func rateLimitByKey(rdb *redis.Client, cfg RateLimitConfig, logger *zap.Logger, keyFn func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identity := keyFn(c)
		key := fmt.Sprintf("ratelimit:%s:%s", cfg.KeyPrefix, identity)

		ctx := c.Request.Context()

		windowSeconds := int(cfg.Window.Seconds())
		result, err := rateLimitScript.Run(ctx, rdb, []string{key}, windowSeconds).Int64()
		if err != nil {
			// On Redis error, allow the request (fail open) but log
			logger.Warn("rate limiter fail-open: redis error",
				zap.String("key", key),
				zap.Error(err),
			)
			c.Next()
			return
		}

		if result > int64(cfg.MaxRequests) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Too many requests, please try again later",
			})
			return
		}

		c.Next()
	}
}
