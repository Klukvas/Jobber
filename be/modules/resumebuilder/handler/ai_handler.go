package handler

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AIHandler handles AI-powered resume suggestions.
type AIHandler struct {
	aiService *service.AIService
}

// NewAIHandler creates a new AIHandler.
func NewAIHandler(aiService *service.AIService) *AIHandler {
	return &AIHandler{aiService: aiService}
}

// RegisterRoutes registers AI routes.
func (h *AIHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware, rateLimiter gin.HandlerFunc) {
	group := router.Group("/resume-builder")
	{
		group.POST("/ai/suggest-bullets", authMiddleware, rateLimiter, h.SuggestBullets)
		group.POST("/ai/suggest-summary", authMiddleware, rateLimiter, h.SuggestSummary)
		group.POST("/ai/improve-text", authMiddleware, rateLimiter, h.ImproveText)
		group.POST("/:id/ats-check", authMiddleware, rateLimiter, h.ATSCheck)
	}
}

type suggestBulletsRequest struct {
	JobTitle           string `json:"job_title" binding:"required,max=255"`
	Company            string `json:"company" binding:"required,max=255"`
	CurrentDescription string `json:"current_description" binding:"max=10000"`
}

// SuggestBullets generates bullet point suggestions.
func (h *AIHandler) SuggestBullets(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req suggestBulletsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	result, err := h.aiService.SuggestBulletPoints(c.Request.Context(), userID, req.JobTitle, req.Company, req.CurrentDescription)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

type suggestSummaryRequest struct {
	ResumeID string `json:"resume_id" binding:"required"`
}

// SuggestSummary generates a professional summary.
func (h *AIHandler) SuggestSummary(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req suggestSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	if _, err := uuid.Parse(req.ResumeID); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid resume_id format")
		return
	}

	summary, err := h.aiService.SuggestSummary(c.Request.Context(), userID, req.ResumeID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, map[string]string{"summary": summary})
}

type improveTextRequest struct {
	Text        string `json:"text" binding:"required,max=10000"`
	Instruction string `json:"instruction" binding:"required,max=500"`
}

// ImproveText improves text based on an instruction.
func (h *AIHandler) ImproveText(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req improveTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	improved, err := h.aiService.ImproveText(c.Request.Context(), userID, req.Text, req.Instruction)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, map[string]string{"improved": improved})
}

type atsCheckRequest struct {
	Locale string `json:"locale"`
}

// ATSCheck analyzes a resume for ATS compatibility.
func (h *AIHandler) ATSCheck(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	resumeID := c.Param("id")
	if _, err := uuid.Parse(resumeID); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid resume ID format")
		return
	}

	var req atsCheckRequest
	// Ignore bind error — locale is optional
	_ = c.ShouldBindJSON(&req)

	// Validate locale against supported set
	allowedLocales := map[string]bool{"": true, "en": true, "ru": true, "ua": true, "uk": true, "de": true, "fr": true, "es": true, "pt": true, "it": true, "pl": true}
	if !allowedLocales[req.Locale] {
		req.Locale = ""
	}

	result, err := h.aiService.ATSCheck(c.Request.Context(), userID, resumeID, req.Locale)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *AIHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, subModel.ErrLimitReached) {
		httpPlatform.RespondWithError(c, http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the AI usage limit for your current plan.")
		return
	}

	errorCode := model.GetErrorCode(err)
	errorMessage := model.GetErrorMessage(err)

	statusCode := http.StatusInternalServerError
	switch errorCode {
	case model.CodeResumeBuilderNotFound:
		statusCode = http.StatusNotFound
	case model.CodeNotOwner:
		statusCode = http.StatusForbidden
	}

	httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
}
