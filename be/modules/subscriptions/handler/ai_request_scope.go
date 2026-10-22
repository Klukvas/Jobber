package handler

import (
	"context"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
)

// AIRequestScope bounds legacy calls below their durable quota lease lifetime.
// It must be installed before routes that can use the shared AI allowance.
func AIRequestScope(subscriptions *service.SubscriptionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		route := c.FullPath()
		isAI := strings.Contains(route, "/ai/") || strings.HasSuffix(route, "/match-score") || strings.HasSuffix(route, "/ats-check") || strings.HasSuffix(route, "/autofill-profile") || strings.HasSuffix(route, "/resume-tailoring")
		if !isAI {
			c.Next()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(service.AIRequestContext(ctx))
		defer subscriptions.FinishAIRequest(c.Request.Context())
		c.Next()
	}
}
