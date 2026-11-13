package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const aiRequestTimeout = 90 * time.Second

// AIRequestScope bounds legacy calls below their durable quota lease lifetime.
// It must be installed before routes that can use the shared AI allowance.
func AIRequestScope(subscriptions *service.SubscriptionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		route := c.FullPath()
		kind := creditOperationKind(route)
		isAI := kind != ""
		if !isAI {
			c.Next()
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), aiRequestTimeout)
		defer cancel()
		key := c.GetHeader("Idempotency-Key")
		if key != "" {
			if _, err := uuid.Parse(key); err != nil {
				httpPlatform.RespondWithError(c, 400, "BAD_REQUEST", "Invalid Idempotency-Key")
				c.Abort()
				return
			}
		}
		var reader io.Reader = bytes.NewReader(nil)
		if c.Request.Body != nil {
			reader = c.Request.Body
		}
		body, err := io.ReadAll(io.LimitReader(reader, 11_534_337))
		if err != nil || len(body) > 11_534_336 {
			httpPlatform.RespondWithError(c, 413, "BAD_REQUEST", "AI input is too large")
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))
		sum := sha256.Sum256(append([]byte(c.Request.Method+":"+c.Request.URL.Path+":"), body...))
		var finisher AIRequestFinisher
		if subscriptions != nil {
			finisher = subscriptions
		}
		ctx, finish := BeginAIRequestScope(ctx, func() string { owner, _ := auth.GetUserID(c); return owner }, key, kind, hex.EncodeToString(sum[:]), finisher)
		defer finish()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// AIRequestFinisher releases unsuccessful reservations after a bounded AI call.
type AIRequestFinisher interface{ FinishAIRequest(context.Context) }

// BeginAIRequestScope is shared by HTTP routes and individual MCP tool calls.
// The owner callback permits HTTP authentication to run after global middleware.
func BeginAIRequestScope(parent context.Context, owner func() string, key, kind, fingerprint string, subscriptions AIRequestFinisher) (context.Context, func()) {
	ctx, cancel := context.WithTimeout(parent, aiRequestTimeout)
	ctx = ai.CreditContext(service.AIRequestContext(ctx), owner, key, kind)
	ai.SetCreditFingerprint(ctx, fingerprint)
	return ctx, func() {
		defer cancel()
		if subscriptions != nil {
			subscriptions.FinishAIRequest(ctx)
		}
	}
}

func creditOperationKind(route string) string {
	switch {
	case strings.Contains(route, "/interview-prep/"):
		return "interview_plan"
	case strings.Contains(route, "/resume-tailoring/import"):
		return "resume_import"
	case strings.HasSuffix(route, "/resume-tailoring"):
		return "tailor"
	case strings.HasSuffix(route, "/autofill-profile"):
		return "autofill_import"
	case strings.HasSuffix(route, "/jobs/parse"):
		return "job_parse"
	case strings.HasSuffix(route, "/match-score"):
		return "match"
	case strings.HasSuffix(route, "/ats-check"):
		return "ats"
	case strings.Contains(route, "/cover-letters/ai/"):
		return "cover_letter"
	case strings.HasSuffix(route, "/suggest-bullets"):
		return "bullet"
	case strings.HasSuffix(route, "/suggest-summary"):
		return "summary"
	case strings.Contains(route, "/ai/"):
		return "improve"
	case strings.Contains(route, "/resume-builder/import"):
		return "resume_import"
	default:
		return ""
	}
}
