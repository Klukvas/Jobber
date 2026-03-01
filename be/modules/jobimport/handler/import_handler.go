package handler

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/jobimport/model"
	"github.com/andreypavlenko/jobber/modules/jobimport/service"
	"github.com/gin-gonic/gin"
)

// ImportHandler handles job import HTTP requests.
type ImportHandler struct {
	service *service.ImportService
}

// NewImportHandler creates a new import handler.
func NewImportHandler(service *service.ImportService) *ImportHandler {
	return &ImportHandler{service: service}
}

// ParseJobPage godoc
// @Summary Parse a job page using AI
// @Description Extracts structured job data from raw page text using Claude AI
// @Tags jobs
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.ParseJobRequest true "Page text and URL"
// @Success 200 {object} model.ParseJobResponse
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 503 {object} httpPlatform.ErrorResponse
// @Router /jobs/parse [post]
func (h *ImportHandler) ParseJobPage(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req model.ParseJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(model.CodeValidationError), "Invalid request: page_text (min 10 chars) and page_url (valid URL) are required")
		return
	}

	result, err := h.service.ParseJobPage(c.Request.Context(), userID, &req)
	if err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)

		statusCode := http.StatusInternalServerError
		if errors.Is(err, model.ErrAINotConfigured) {
			statusCode = http.StatusServiceUnavailable
		}

		if errorCode == "PLAN_LIMIT_REACHED" {
			statusCode = http.StatusForbidden
		}

		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

// RegisterRoutes registers import routes under the /jobs group.
func (h *ImportHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware, rateLimiter gin.HandlerFunc) {
	jobs := router.Group("/jobs")
	jobs.Use(authMiddleware)
	{
		jobs.POST("/parse", rateLimiter, h.ParseJobPage)
	}
}
