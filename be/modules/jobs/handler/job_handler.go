package handler

import (
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/service"
	"github.com/gin-gonic/gin"
)

// JobHandler handles job HTTP requests
type JobHandler struct {
	service *service.JobService
}

// NewJobHandler creates a new job handler
func NewJobHandler(service *service.JobService) *JobHandler {
	return &JobHandler{service: service}
}

// Create godoc
// @Summary Create a new job
// @Description Create a new job posting for the authenticated user
// @Tags jobs
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.CreateJobRequest true "Job details"
// @Success 201 {object} model.JobDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs [post]
func (h *JobHandler) Create(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req model.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	job, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)
		
		statusCode := http.StatusInternalServerError
		if errorCode == model.CodeJobTitleRequired {
			statusCode = http.StatusBadRequest
		}
		
		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, job)
}

// Get godoc
// @Summary Get a job
// @Description Get details of a specific job by ID
// @Tags jobs
// @Security BearerAuth
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} model.JobDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Job not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id} [get]
func (h *JobHandler) Get(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	jobID := c.Param("id")

	job, err := h.service.GetByID(c.Request.Context(), userID, jobID)
	if err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)
		
		statusCode := http.StatusInternalServerError
		if errorCode == model.CodeJobNotFound {
			statusCode = http.StatusNotFound
		}
		
		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, job)
}

// List godoc
// @Summary List jobs
// @Description Get a paginated list of job postings for the authenticated user with filtering and sorting
// @Tags jobs
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of items per page (default: 20, max: 100)"
// @Param offset query int false "Number of items to skip (default: 0)"
// @Param status query string false "Filter by status: active, archived, all (default: active)"
// @Param sort query string false "Sort format: field:order (e.g., created_at:desc, title:asc, company_name:asc)"
// @Success 200 {object} httpPlatform.PaginatedResponse{items=[]model.JobDTO}
// @Failure 400 {object} httpPlatform.ErrorResponse "Invalid pagination parameters"
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs [get]
func (h *JobHandler) List(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	// Parse pagination parameters
	pagination, err := httpPlatform.ParsePaginationParams(c)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "INVALID_PAGINATION_PARAMS", "Invalid pagination parameters")
		return
	}

	// Parse filter parameters
	status := c.DefaultQuery("status", "active")

	// Parse sort parameters
	sortParam := c.Query("sort")
	var sortBy, sortOrder string
	if sortParam != "" {
		// Parse format: "field:order" (e.g., "created_at:desc")
		parts := splitSort(sortParam)
		if len(parts) == 2 {
			sortBy = parts[0]
			sortOrder = parts[1]
		}
	}

	jobs, total, err := h.service.List(c.Request.Context(), userID, pagination.Limit, pagination.Offset, status, sortBy, sortOrder)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list jobs")
		return
	}

	httpPlatform.RespondWithPagination(c, http.StatusOK, jobs, pagination.Limit, pagination.Offset, total)
}

// splitSort splits a sort parameter like "created_at:desc" into [field, order]
func splitSort(sort string) []string {
	for i := 0; i < len(sort); i++ {
		if sort[i] == ':' {
			return []string{sort[:i], sort[i+1:]}
		}
	}
	return []string{sort}
}

// Update godoc
// @Summary Update a job
// @Description Update details of a specific job posting
// @Tags jobs
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Job ID"
// @Param request body model.UpdateJobRequest true "Updated job details"
// @Success 200 {object} model.JobDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Job not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id} [patch]
func (h *JobHandler) Update(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	jobID := c.Param("id")

	var req model.UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	job, err := h.service.Update(c.Request.Context(), userID, jobID, &req)
	if err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)
		
		statusCode := http.StatusInternalServerError
		if errorCode == model.CodeJobNotFound {
			statusCode = http.StatusNotFound
		} else if errorCode == model.CodeJobTitleRequired {
			statusCode = http.StatusBadRequest
		}
		
		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, job)
}

// Delete godoc
// @Summary Delete a job
// @Description Delete a specific job posting by ID
// @Tags jobs
// @Security BearerAuth
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Job not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id} [delete]
func (h *JobHandler) Delete(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	jobID := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), userID, jobID); err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)
		
		statusCode := http.StatusInternalServerError
		if errorCode == model.CodeJobNotFound {
			statusCode = http.StatusNotFound
		}
		
		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Job deleted successfully"})
}

// RegisterRoutes registers job routes
func (h *JobHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	jobs := router.Group("/jobs")
	jobs.Use(authMiddleware)
	{
		jobs.POST("", h.Create)
		jobs.GET("", h.List)
		jobs.GET("/:id", h.Get)
		jobs.PATCH("/:id", h.Update)
		jobs.DELETE("/:id", h.Delete)
	}
}
