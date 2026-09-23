package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// maxWebhookBodyBytes caps the body on this unauthenticated endpoint so a huge
// payload cannot exhaust memory. FastSpring batches stay well under 1 MB.
const maxWebhookBodyBytes = 1 << 20

// WebhookHandler handles FastSpring webhook HTTP requests (no auth — the HMAC
// signature is the authentication).
type WebhookHandler struct {
	service *service.SubscriptionService
	logger  *zap.Logger
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(service *service.SubscriptionService, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{service: service, logger: logger}
}

// HandleFastSpringWebhook processes an incoming FastSpring webhook batch.
//
// Acknowledgement follows the documented contract: 200 when every event in the
// batch is processed, 202 with the processed event IDs (one per line) when only
// some are, and 503 when none are, so FastSpring retries the rest.
//
// https://developer.fastspring.com/docs/processed-and-unprocessed-webhook-events
func (h *WebhookHandler) HandleFastSpringWebhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBodyBytes))
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "BAD_REQUEST", "Failed to read request body")
		return
	}

	signature := c.GetHeader(fastspring.SignatureHeader)

	result, err := h.service.HandleWebhook(c.Request.Context(), body, signature)
	if err != nil {
		// A rejected signature or an unparseable body means nothing was written.
		// The reason stays in the logs; the response body reveals nothing.
		h.logger.Warn("FastSpring webhook rejected",
			zap.Error(err),
			zap.Bool("signature_present", signature != ""),
			zap.Int("body_bytes", len(body)),
		)
		status := http.StatusBadRequest
		if errors.Is(err, fastspring.ErrSecretMissing) {
			// Misconfiguration on our side — let FastSpring retry once it is fixed.
			status = http.StatusServiceUnavailable
		}
		httpPlatform.RespondWithError(c, status, "WEBHOOK_ERROR", "invalid webhook payload")
		return
	}

	h.logOutcomes(result)

	switch {
	case result.AllProcessed():
		c.String(http.StatusOK, "OK")
	case len(result.Processed) > 0:
		c.String(http.StatusAccepted, strings.Join(result.Processed, "\n"))
	default:
		c.String(http.StatusServiceUnavailable, "")
	}
}

func (h *WebhookHandler) logOutcomes(result service.WebhookResult) {
	for _, skipped := range result.Skipped {
		fields := []zap.Field{
			zap.String("event_id", skipped.EventID),
			zap.String("event_type", skipped.EventType),
			zap.Error(skipped.Err),
		}
		// Which of these is worth a person's attention is decided in the
		// service, beside the sentinels, because the reconciliation sweep has to
		// reach the same verdict for an event it recovers.
		message, needsAttention := service.SkipReport(skipped.Err)
		if needsAttention {
			h.logger.Warn(message, fields...)
		} else {
			h.logger.Info(message, fields...)
		}
	}
	for _, failed := range result.Failed {
		h.logger.Error("FastSpring webhook event failed, will be retried",
			zap.String("event_id", failed.EventID),
			zap.String("event_type", failed.EventType),
			zap.Error(failed.Err),
		)
	}
}

// RegisterRoutes registers webhook routes (public, no auth).
//
// The HMAC is what authenticates a delivery, but verifying it means reading and
// hashing the whole body first — so an unsigned flood still buys real work on
// the one route with no auth in front of it. The limiter bounds that.
//
// It has to be generous, and it fails open on a Redis error, because the cost
// of throttling the provider is worse than the cost of the flood: a rejected
// delivery is a lifecycle event that has to come back through the retry
// schedule, and one that never comes back is a subscriber on the wrong plan.
func (h *WebhookHandler) RegisterRoutes(router *gin.RouterGroup, rateLimiter gin.HandlerFunc) {
	webhooks := router.Group("/webhooks")
	if rateLimiter != nil {
		webhooks.Use(rateLimiter)
	}
	{
		webhooks.POST("/fastspring", h.HandleFastSpringWebhook)
	}
}
