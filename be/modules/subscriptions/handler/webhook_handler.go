package handler

import (
	"io"
	"log"
	"net/http"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
)

// WebhookHandler handles Paddle webhook HTTP requests (no auth).
type WebhookHandler struct {
	service *service.SubscriptionService
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(service *service.SubscriptionService) *WebhookHandler {
	return &WebhookHandler{service: service}
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
		// Log internal details but return a generic message to the caller
		log.Printf("[WARN] Paddle webhook error: %v", err)
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
