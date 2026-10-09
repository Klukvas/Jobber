package handler

import (
	"errors"
	"io"
	"net/http"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/subscriptions/creem"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// maxWebhookBodyBytes caps the body on this unauthenticated endpoint so a huge
// payload cannot exhaust memory. Creem events stay well under 1 MB.
const maxWebhookBodyBytes = 1 << 20

// WebhookHandler handles Creem webhook HTTP requests (no auth — the HMAC
// signature is the authentication).
type WebhookHandler struct {
	service *service.SubscriptionService
	logger  *zap.Logger
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(service *service.SubscriptionService, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{service: service, logger: logger}
}

// HandleCreemWebhook processes one incoming Creem webhook delivery.
//
// Creem treats a 200 as "delivered" and retries anything else on a schedule that
// ends after 24 hours, so a delivery that failed on our side answers 500 and
// everything we are done with — applied, duplicate, deliberately skipped —
// answers 200.
//
// https://docs.creem.io/code/webhooks
func (h *WebhookHandler) HandleCreemWebhook(c *gin.Context) {
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpPlatform.RespondWithError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Request body too large")
			return
		}
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "BAD_REQUEST", "Failed to read request body")
		return
	}

	signature := c.GetHeader(creem.SignatureHeader)

	result, err := h.service.HandleWebhook(c.Request.Context(), body, signature)
	switch {
	case err == nil:
		h.logOutcome(result)
		c.String(http.StatusOK, "OK")
	case errors.Is(err, service.ErrEventFailed):
		h.logger.Error("Creem webhook event failed, will be retried",
			zap.String("event_id", result.EventID),
			zap.String("event_type", result.EventType),
			zap.Error(err),
		)
		c.String(http.StatusInternalServerError, "")
	default:
		// A rejected signature or an unparseable body means nothing was written.
		// The reason stays in the logs; the response body reveals nothing.
		h.logger.Warn("Creem webhook rejected",
			zap.Error(err),
			zap.Bool("signature_present", signature != ""),
			zap.Int("body_bytes", len(body)),
		)
		status := http.StatusBadRequest
		if errors.Is(err, creem.ErrSecretMissing) {
			// Misconfiguration on our side — let Creem retry once it is fixed.
			status = http.StatusServiceUnavailable
		}
		httpPlatform.RespondWithError(c, status, "WEBHOOK_ERROR", "invalid webhook payload")
	}
}

func (h *WebhookHandler) logOutcome(result service.WebhookResult) {
	if result.Skipped == nil {
		return
	}
	fields := []zap.Field{
		zap.String("event_id", result.EventID),
		zap.String("event_type", result.EventType),
		zap.Error(result.Skipped),
	}
	// How loudly a skip is reported is decided in the service, beside the
	// sentinels. An event that was dropped for good is an error: Creem will never
	// redeliver it, so this line is the only trace that a paying user may be on
	// the wrong plan.
	message, severity := service.SkipReport(result.Skipped)
	switch severity {
	case service.SkipLostEvent:
		h.logger.Error(message, fields...)
	case service.SkipNeedsReview:
		h.logger.Warn(message, fields...)
	default:
		h.logger.Info(message, fields...)
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
		webhooks.POST("/creem", h.HandleCreemWebhook)
	}
}
