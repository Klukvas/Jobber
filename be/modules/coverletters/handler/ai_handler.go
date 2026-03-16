package handler

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/andreypavlenko/jobber/modules/coverletters/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AIHandler handles AI-powered cover letter generation.
type AIHandler struct {
	aiService *service.AIService
	logger    *zap.Logger
}

// NewAIHandler creates a new cover letter AIHandler.
func NewAIHandler(aiService *service.AIService, logger *zap.Logger) *AIHandler {
	return &AIHandler{aiService: aiService, logger: logger}
}

// RegisterRoutes registers AI routes for cover letters.
func (h *AIHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware, rateLimiter gin.HandlerFunc) {
	group := router.Group("/cover-letters")
	{
		group.POST("/ai/generate", authMiddleware, rateLimiter, h.Generate)
	}
}

type generateRequest struct {
	CoverLetterID  string `json:"cover_letter_id" binding:"required"`
	JobDescription string `json:"job_description" binding:"max=10000"`
}

// Generate generates cover letter content using AI.
func (h *AIHandler) Generate(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req generateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if _, err := uuid.Parse(req.CoverLetterID); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid cover_letter_id format")
		return
	}

	result, err := h.aiService.Generate(c.Request.Context(), userID, req.CoverLetterID, req.JobDescription)
	if err != nil {
		h.logger.Error("cover letter AI generation failed",
			zap.String("cover_letter_id", req.CoverLetterID),
			zap.Error(err),
		)
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *AIHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, subModel.ErrLimitReached):
		httpPlatform.RespondWithError(c, http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the AI usage limit for your current plan.")
	case errors.Is(err, model.ErrNotAuthorized):
		httpPlatform.RespondWithError(c, http.StatusForbidden, "NOT_AUTHORIZED", "You don't have access to this cover letter")
	case errors.Is(err, model.ErrCoverLetterNotFound):
		httpPlatform.RespondWithError(c, http.StatusNotFound, "COVER_LETTER_NOT_FOUND", "Cover letter not found")
	default:
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}
}
