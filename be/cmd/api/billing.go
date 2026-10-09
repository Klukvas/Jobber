package main

import (
	"time"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// newBillingWebhookRateLimiter bounds the one public route with no auth
// middleware in front of it.
//
// The HMAC is what authenticates a delivery, but verifying it means reading and
// hashing the whole body first, so an unsigned flood still buys real work. The
// limit sits far above anything Creem produces and the limiter fails open on a
// Redis error, deliberately: a throttled delivery is a lifecycle event that has
// to come back through the retry schedule, and one that never comes back is a
// subscriber left on the wrong plan.
func newBillingWebhookRateLimiter(rdb *redis.Client, log *zap.Logger) gin.HandlerFunc {
	return httpPlatform.RateLimitMiddleware(rdb, httpPlatform.RateLimitConfig{
		MaxRequests: 600,
		Window:      1 * time.Minute,
		KeyPrefix:   "billing-webhook",
	}, log)
}
