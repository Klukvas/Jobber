package handler

import (
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
		h.logger.Error("failed to submit support request",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.CodeTelegramError), "Failed to send support request. Please try again later.")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Support request sent successfully"})
}

// RegisterRoutes registers support routes on the given router group.
func (h *SupportHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, rateLimiter gin.HandlerFunc) {
	support := router.Group("/support")
	support.Use(authMiddleware, rateLimiter)
	{
		support.POST("", h.Create)
	}
}
