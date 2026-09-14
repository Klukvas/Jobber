package handler

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/support/model"
	"github.com/andreypavlenko/jobber/modules/support/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SupportHandler handles support-related HTTP requests.
type SupportHandler struct {
	service *service.SupportService
	logger  *zap.Logger
}

// NewSupportHandler creates a new support handler.
func NewSupportHandler(service *service.SupportService, logger *zap.Logger) *SupportHandler {
	return &SupportHandler{service: service, logger: logger}
}

// Create godoc
// @Summary Submit a support request
// @Description Send a support message that will be forwarded to the support team via Telegram
// @Tags support
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.CreateSupportRequest true "Support request details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Failure 503 {object} httpPlatform.ErrorResponse "SUPPORT_UNAVAILABLE — this deployment has no support channel configured"
// @Router /support [post]
func (h *SupportHandler) Create(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}

	var req model.CreateSupportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(model.CodeValidationError), "Subject (min 3 chars) and message (min 10 chars) are required")
		return
	}

	if err := h.service.Submit(c.Request.Context(), userID, req.Subject, req.Message, req.Page); err != nil {
		if errors.Is(err, model.ErrSupportUnavailable) {
			// Not a fault — this deployment simply has no support channel.
			// Answered as a first-class state so the UI can explain it instead
			// of showing a 404 with the API URL in it.
			httpPlatform.RespondWithError(c, http.StatusServiceUnavailable, string(model.CodeSupportUnavailable),
				"In-app support is unavailable right now. Please email us instead.")
			return
		}

		h.logger.Error("failed to submit support request",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.CodeTelegramError), "Failed to send support request. Please try again later.")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Support request sent successfully"})
}

// Status godoc
// @Summary Report whether in-app support is available
// @Description Lets the UI hide or disable the support form when no support channel is configured
// @Tags support
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]bool
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Router /support/status [get]
func (h *SupportHandler) Status(c *gin.Context) {
	if _, ok := auth.MustGetUserID(c); !ok {
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"available": h.service.Available()})
}

// RegisterRoutes registers support routes on the given router group.
func (h *SupportHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, rateLimiter gin.HandlerFunc) {
	support := router.Group("/support")
	support.Use(authMiddleware)
	{
		// The status probe is a cheap read the UI calls on mount, so it stays
		// outside the 3-per-5-minutes submission limit.
		support.GET("/status", h.Status)
		support.POST("", rateLimiter, h.Create)
	}
}
