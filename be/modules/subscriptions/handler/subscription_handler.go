package handler

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
)

// SubscriptionHandler handles subscription HTTP requests (auth required).
type SubscriptionHandler struct {
	service *service.SubscriptionService
}

// NewSubscriptionHandler creates a new SubscriptionHandler.
func NewSubscriptionHandler(service *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: service}
}

// GetSubscription returns the current user's subscription.
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	dto, err := h.service.GetSubscription(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, model.ErrSubscriptionNotFound) {
			// Auto-create free subscription for users who don't have one yet
			if ensureErr := h.service.EnsureFreeSubscription(c.Request.Context(), userID); ensureErr != nil {
				httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to create subscription")
				return
			}
			dto, err = h.service.GetSubscription(c.Request.Context(), userID)
			if err != nil {
				httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get subscription")
				return
			}
		} else {
			httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get subscription")
			return
		}
	}

	httpPlatform.RespondWithData(c, http.StatusOK, dto)
}

// GetCheckoutConfig returns Paddle checkout configuration for the frontend.
func (h *SubscriptionHandler) GetCheckoutConfig(c *gin.Context) {
	config := h.service.GetCheckoutConfig()
	httpPlatform.RespondWithData(c, http.StatusOK, config)
}

// CreatePortalSession creates a Paddle customer portal session.
func (h *SubscriptionHandler) CreatePortalSession(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	portalURL, err := h.service.CreatePortalSession(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "PORTAL_ERROR", "Failed to create portal session")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, model.PortalSessionDTO{URL: portalURL})
}

// RegisterRoutes registers subscription routes (auth required).
func (h *SubscriptionHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	sub := router.Group("/subscription")
	sub.Use(authMiddleware)
	{
		sub.GET("", h.GetSubscription)
		sub.GET("/checkout-config", h.GetCheckoutConfig)
		sub.POST("/portal", h.CreatePortalSession)
	}
}
