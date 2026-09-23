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
		// Most skips are routine: a duplicate, a replay, an event type Jobber
		// does not act on, or another product's subscription in the shared
		// store. Five are not. Four of them drop the event for good:
		//
		//   - an environment mismatch means this deployment is pointed at the
		//     wrong billing environment;
		//   - an unproven order tag means an order named a Jobber user without
		//     the proof this server mints, which is either a forged tag or a
		//     rotated secret — never a normal purchase;
		//   - a tagged-owner conflict means an order claimed a user who is
		//     already paying for a different subscription, which a legitimate
		//     first checkout cannot produce;
		//   - a link conflict means the atomic write refused to repoint a user
		//     at a second subscription while their first one still bills, so
		//     somebody may be paying twice and only one of the two is cancellable
		//     from Jobber.
		//
		// The fifth is a refund: nothing was dropped and nothing was wrong, but
		// money left the account and no subscription moved because of it, which
		// somebody should see rather than discover in a payout.
		switch {
		case errors.Is(skipped.Err, service.ErrEnvironmentMismatch):
			h.logger.Warn("FastSpring webhook event dropped: billing environment mismatch", fields...)
		case errors.Is(skipped.Err, service.ErrUnprovenOrderTag):
			h.logger.Warn("FastSpring webhook event dropped: order tag names a user it cannot prove", fields...)
		case errors.Is(skipped.Err, service.ErrTaggedOwnerConflict):
			h.logger.Warn("FastSpring webhook event dropped: order tag contradicts the subscription it names", fields...)
		case errors.Is(skipped.Err, service.ErrSubscriptionLinkConflict):
			h.logger.Warn("FastSpring webhook event dropped: user is already linked to another live subscription", fields...)
		case errors.Is(skipped.Err, service.ErrRefundNeedsReview):
			h.logger.Warn("FastSpring refund observed, no subscription changed by it", fields...)
		default:
			h.logger.Info("FastSpring webhook event acknowledged without changes", fields...)
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
