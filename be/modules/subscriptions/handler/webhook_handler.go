package handler

import (
	"io"
	"net/http"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WebhookHandler handles Paddle webhook HTTP requests (no auth).
type WebhookHandler struct {
	service *service.SubscriptionService
	logger  *zap.Logger
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(service *service.SubscriptionService, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{service: service, logger: logger}
}

// HandlePaddleWebhook processes incoming Paddle webhook events.
func (h *WebhookHandler) HandlePaddleWebhook(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "BAD_REQUEST", "Failed to read request body")
		return
	}

	signature := c.GetHeader("Paddle-Signature")

	if err := h.service.HandleWebhook(c.Request.Context(), body, signature); err != nil {
		h.logger.Error("Paddle webhook processing failed",
			zap.Error(err),
			zap.String("signature_present", func() string {
				if signature != "" {
					return "yes"
				}
				return "no"
			}()),
		)
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "WEBHOOK_ERROR", "invalid webhook payload")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// RegisterRoutes registers webhook routes (public, no auth).
func (h *WebhookHandler) RegisterRoutes(router *gin.RouterGroup) {
	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("/paddle", h.HandlePaddleWebhook)
	}
}
