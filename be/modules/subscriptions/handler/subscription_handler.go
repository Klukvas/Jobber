package handler

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SubscriptionHandler handles subscription HTTP requests (auth required).
type SubscriptionHandler struct {
	service *service.SubscriptionService
	logger  *zap.Logger
}

// NewSubscriptionHandler creates a new SubscriptionHandler.
func NewSubscriptionHandler(svc *service.SubscriptionService, logger *zap.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{service: svc, logger: logger}
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
		h.logger.Error("failed to create portal session", zap.String("user_id", userID), zap.Error(err))
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "PORTAL_ERROR", "Failed to create portal session")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, model.PortalSessionDTO{URL: portalURL})
}

// ChangePlan changes the user's subscription to a different plan.
func (h *SubscriptionHandler) ChangePlan(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req model.ChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	if req.Plan != "pro" && req.Plan != "enterprise" {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "BAD_REQUEST", "Invalid plan")
		return
	}

	if err := h.service.ChangePlan(c.Request.Context(), userID, req.Plan); err != nil {
		h.logger.Error("failed to change plan", zap.String("user_id", userID), zap.Error(err))
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "CHANGE_PLAN_ERROR", "Failed to change plan")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"success": true})
}

// CancelSubscription schedules cancellation at the end of the billing period.
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	if err := h.service.CancelSubscription(c.Request.Context(), userID); err != nil {
		h.logger.Error("failed to cancel subscription", zap.String("user_id", userID), zap.Error(err))
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "CANCEL_ERROR", "Failed to cancel subscription")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"success": true})
}

// RegisterRoutes registers subscription routes (auth required).
// When paymentsEnabled is false, checkout and portal endpoints are not registered.
func (h *SubscriptionHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, paymentsEnabled bool) {
	sub := router.Group("/subscription")
	sub.Use(authMiddleware)
	{
		sub.GET("", h.GetSubscription)
		if paymentsEnabled {
			sub.GET("/checkout-config", h.GetCheckoutConfig)
			sub.POST("/portal", h.CreatePortalSession)
			sub.POST("/change-plan", h.ChangePlan)
			sub.POST("/cancel", h.CancelSubscription)
		}
	}
}
