package http

import (
	"context"
	"fmt"
	"net/http"
	"net/netip"
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

// redisCallTimeout bounds the limiter's one Redis round trip. It is generous for
// a local INCR and short enough that a webhook sender with a tight timeout still
// sees the request succeed while the limiter fails open.
const redisCallTimeout = 300 * time.Millisecond

// ipv6ClientPrefixBits is how much of an IPv6 address identifies one client. A
// subscriber usually holds a whole /64 and can pick any address in it, so counting
// single addresses would let them step around every per-IP limit.
const ipv6ClientPrefixBits = 64

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
// Limits requests by client IP (IPv6 by /64) using a fixed-window counter in Redis.
// Uses a Lua script for atomic INCR + EXPIRE to prevent TTL-less keys on crashes.
func RateLimitMiddleware(rdb *redis.Client, cfg RateLimitConfig, logger *zap.Logger) gin.HandlerFunc {
	return rateLimitByKey(rdb, cfg, logger, func(c *gin.Context) string {
		return clientIPIdentity(c.ClientIP())
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
		return clientIPIdentity(c.ClientIP())
	})
}

// clientIPIdentity returns what a client IP is counted under: the /64 for IPv6, the
// address for IPv4. An IPv4-mapped IPv6 address is the same client as its IPv4 form,
// and a string that does not parse is kept so it still gets a counter of its own.
func clientIPIdentity(clientIP string) string {
	addr, err := netip.ParseAddr(clientIP)
	if err != nil {
		return clientIP
	}

	addr = addr.Unmap()
	if addr.Is4() {
		return addr.String()
	}

	prefix, err := addr.Prefix(ipv6ClientPrefixBits)
	if err != nil {
		return clientIP
	}
	return prefix.String()
}

func rateLimitByKey(rdb *redis.Client, cfg RateLimitConfig, logger *zap.Logger, keyFn func(*gin.Context) string) gin.HandlerFunc {
	// go-redis ignores a context deadline for network I/O unless the client opted
	// in, so the bound has to be on the client itself. WithTimeout returns a copy
	// with its own read/write timeout and leaves the shared client untouched.
	bounded := rdb.WithTimeout(redisCallTimeout)

	return func(c *gin.Context) {
		identity := keyFn(c)
		key := fmt.Sprintf("ratelimit:%s:%s", cfg.KeyPrefix, identity)

		// A slow or unreachable Redis must not hold every request for the
		// client's own timeout (and its retries) before failing open: the context
		// also stops the retry back-off.
		ctx, cancel := context.WithTimeout(c.Request.Context(), redisCallTimeout)
		defer cancel()

		windowSeconds := int(cfg.Window.Seconds())
		result, err := rateLimitScript.Run(ctx, bounded, []string{key}, windowSeconds).Int64()
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
