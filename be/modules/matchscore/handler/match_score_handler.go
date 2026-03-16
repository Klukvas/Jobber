package handler

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/matchscore/model"
	"github.com/andreypavlenko/jobber/modules/matchscore/service"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
)

// MatchScoreHandler handles match score HTTP requests.
type MatchScoreHandler struct {
	service *service.MatchScoreService
}

// NewMatchScoreHandler creates a new match score handler.
func NewMatchScoreHandler(service *service.MatchScoreService) *MatchScoreHandler {
	return &MatchScoreHandler{service: service}
}

// CheckMatch godoc
// @Summary Check resume-job match score
// @Description Analyzes how well a resume matches a job posting using AI
// @Tags match-score
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.MatchScoreRequest true "Job ID and Resume ID"
// @Success 200 {object} model.MatchScoreResponse
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse
// @Failure 503 {object} httpPlatform.ErrorResponse
// @Router /match-score [post]
func (h *MatchScoreHandler) CheckMatch(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req model.MatchScoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(model.CodeValidationError), "Invalid request: job_id and resume_id are required")
		return
	}

	result, err := h.service.CheckMatch(c.Request.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, subModel.ErrLimitReached) {
			httpPlatform.RespondWithError(c, http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the AI usage limit for your current plan.")
			return
		}

		statusCode := http.StatusInternalServerError
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)

		switch {
		case errors.Is(err, model.ErrJobDescriptionEmpty), errors.Is(err, model.ErrResumeFileEmpty):
			statusCode = http.StatusBadRequest
		case errors.Is(err, jobModel.ErrJobNotFound):
			statusCode = http.StatusNotFound
			errorCode = model.CodeJobNotFound
			errorMessage = "Job not found"
		case errors.Is(err, resumeModel.ErrResumeNotFound):
			statusCode = http.StatusNotFound
			errorCode = model.CodeResumeNotFound
			errorMessage = "Resume not found"
		case errors.Is(err, model.ErrAINotConfigured):
			statusCode = http.StatusServiceUnavailable
		}

		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

// RegisterRoutes registers match score routes.
func (h *MatchScoreHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware, rateLimiter gin.HandlerFunc) {
	matchScore := router.Group("/match-score")
	matchScore.Use(authMiddleware)
	{
		matchScore.POST("", rateLimiter, h.CheckMatch)
	}
}
