package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/andreypavlenko/jobber/modules/jobs/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
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

// respondWithMappedError maps a service error to an HTTP response
func respondWithMappedError(c *gin.Context, err error) {
	errorCode := model.GetErrorCode(err)
	errorMessage := model.GetErrorMessage(err)

	statusCode := http.StatusInternalServerError
	switch errorCode {
	case model.CodeJobNotFound, model.CodeCompanyNotFound,
		model.CodeStageTemplateNotFound, model.CodeJobStageNotFound,
		model.CodeResumeNotFound:
		statusCode = http.StatusNotFound
	case model.CodeJobTitleRequired, model.CodeInvalidJobStatus,
		model.CodeInvalidStatus, model.CodeStageNameRequired,
		model.CodeBothResumeTypesSet, model.CodeInvalidPhase,
		model.CodeInvalidMoveTarget:
		statusCode = http.StatusBadRequest
	case model.CodeStageTemplateInUse, model.CodeStageTemplateNameExists:
		statusCode = http.StatusConflict
	}

	httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
}

// Create godoc
// @Summary Create a new job
// @Description Create a new job card for the authenticated user. Without status/resume fields a "saved" (wishlist) card is created.
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
		if errors.Is(err, subModel.ErrLimitReached) {
			httpPlatform.RespondWithError(c, http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the job limit for your current plan.")
			return
		}
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, job)
}

// Get godoc
// @Summary Get a job
// @Description Get details of a specific job by ID, including pipeline state and comments
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
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, job)
}

// List godoc
// @Summary List jobs
// @Description Get a paginated list of job cards for the authenticated user with filtering and sorting
// @Tags jobs
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of items per page (default: 20, max: 100)"
// @Param offset query int false "Number of items to skip (default: 0)"
// @Param status query string false "Filter by status: saved, applied, on_hold, offer, rejected, archived, all. Empty and the legacy value 'active' mean everything except archived."
// @Param sort query string false "Sort format: field:order (e.g., last_activity:desc, created_at:desc, title:asc, company_name:asc, status:asc, applied_at:desc)"
// @Param search query string false "Case-insensitive search matched against job title and company name (max 100 chars)"
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

	// Parse and validate filter parameters.
	// "" and the legacy "active" mean "everything except archived" — backward
	// compatible with the deployed Chrome extension and pre-merge clients.
	status := c.Query("status")
	switch status {
	case "", "active", "all",
		string(model.StatusSaved), string(model.StatusApplied), string(model.StatusOnHold),
		string(model.StatusOffer), string(model.StatusRejected), string(model.StatusArchived):
		// valid
	default:
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "INVALID_STATUS", "Invalid status filter")
		return
	}

	// Parse and validate sort parameters
	sortParam := c.Query("sort")
	var sortBy, sortOrder string
	allowedSortFields := map[string]bool{
		"created_at": true, "title": true, "company_name": true,
		"last_activity": true, "status": true, "applied_at": true,
	}
	allowedSortOrders := map[string]bool{"asc": true, "desc": true}
	if sortParam != "" {
		// Parse format: "field:order" (e.g., "created_at:desc")
		parts := splitSort(sortParam)
		if len(parts) == 2 && allowedSortFields[parts[0]] && allowedSortOrders[parts[1]] {
			sortBy = parts[0]
			sortOrder = parts[1]
		} else {
			httpPlatform.RespondWithError(c, http.StatusBadRequest, "INVALID_SORT", "Invalid sort parameter")
			return
		}
	}

	// Optional case-insensitive search across job title and company name.
	// Cap the length to bound the LIKE pattern and reject abuse. Truncate by
	// rune (not byte) so a multibyte name — e.g. Cyrillic — is never split
	// into invalid UTF-8, which would make Postgres reject the ILIKE argument.
	search := strings.TrimSpace(c.Query("search"))
	if runes := []rune(search); len(runes) > 100 {
		search = string(runes[:100])
	}

	opts := &ports.ListOptions{
		Limit:   pagination.Limit,
		Offset:  pagination.Offset,
		SortBy:  sortBy,
		SortDir: sortOrder,
		Status:  status,
		Search:  search,
	}

	jobs, total, err := h.service.List(c.Request.Context(), userID, opts)
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
// @Description Update details or pipeline status of a specific job. Moving out of "saved" stamps applied_at; moving back clears it.
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
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, job)
}

// Move godoc
// @Summary Move a job on the unified board
// @Description Atomically reposition a card: a stage target sets the stage and derives the status from the stage's phase; a phase target completes the current stage and sets the derived status
// @Tags jobs
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Job ID"
// @Param request body model.MoveJobRequest true "Move target"
// @Success 200 {object} model.JobDTO
// @Failure 400 {object} httpPlatform.ErrorResponse "Invalid move target or phase"
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Job or stage template not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id}/move [post]
func (h *JobHandler) Move(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	jobID := c.Param("id")

	var req model.MoveJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	job, err := h.service.Move(c.Request.Context(), userID, jobID, &req)
	if err != nil {
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, job)
}

// Delete godoc
// @Summary Delete a job
// @Description Delete a specific job card by ID
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
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Job deleted successfully"})
}

// ToggleFavorite godoc
// @Summary Toggle job favorite status
// @Description Toggle the favorite status of a specific job
// @Tags jobs
// @Security BearerAuth
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} map[string]bool
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Job not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id}/favorite [post]
func (h *JobHandler) ToggleFavorite(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	jobID := c.Param("id")

	isFavorite, err := h.service.ToggleFavorite(c.Request.Context(), userID, jobID)
	if err != nil {
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"is_favorite": isFavorite})
}

// AddStage godoc
// @Summary Add a stage to a job
// @Description Append a new pipeline stage; the current active stage is completed automatically
// @Tags stages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Job ID"
// @Param request body model.AddStageRequest true "Stage details"
// @Success 201 {object} model.JobStageDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id}/stages [post]
func (h *JobHandler) AddStage(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	jobID := c.Param("id")

	var req model.AddStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	stage, err := h.service.AddStage(c.Request.Context(), userID, jobID, &req)
	if err != nil {
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, stage)
}

// ListStages godoc
// @Summary List job stages
// @Description Get the full stage history of a job
// @Tags stages
// @Security BearerAuth
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} []model.JobStageDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id}/stages [get]
func (h *JobHandler) ListStages(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	jobID := c.Param("id")

	stages, err := h.service.ListStages(c.Request.Context(), userID, jobID)
	if err != nil {
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, stages)
}

// UpdateStage godoc
// @Summary Update a job stage
// @Description Update the status of a specific stage
// @Tags stages
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Job ID"
// @Param stageId path string true "Stage ID"
// @Param request body model.UpdateStageRequest true "Stage update"
// @Success 200 {object} model.JobStageDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id}/stages/{stageId} [patch]
func (h *JobHandler) UpdateStage(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	jobID := c.Param("id")
	stageID := c.Param("stageId")

	var req model.UpdateStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	stage, err := h.service.UpdateStage(c.Request.Context(), userID, jobID, stageID, &req)
	if err != nil {
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, stage)
}

// DeleteStage godoc
// @Summary Delete a job stage
// @Description Delete a specific stage from a job's history
// @Tags stages
// @Security BearerAuth
// @Produce json
// @Param id path string true "Job ID"
// @Param stageId path string true "Stage ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id}/stages/{stageId} [delete]
func (h *JobHandler) DeleteStage(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	jobID := c.Param("id")
	stageID := c.Param("stageId")

	if err := h.service.DeleteStage(c.Request.Context(), userID, jobID, stageID); err != nil {
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Stage deleted successfully"})
}

// CreateStageTemplate godoc
// @Summary Create a stage template
// @Description Create a reusable stage definition
// @Tags stage-templates
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.CreateStageTemplateRequest true "Stage template details"
// @Success 201 {object} model.StageTemplateDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /stage-templates [post]
func (h *JobHandler) CreateStageTemplate(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req model.CreateStageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	template, err := h.service.CreateStageTemplate(c.Request.Context(), userID, &req)
	if err != nil {
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, template)
}

// ListStageTemplates godoc
// @Summary List stage templates
// @Description Get a paginated list of the user's stage templates
// @Tags stage-templates
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of items per page (default: 20, max: 100)"
// @Param offset query int false "Number of items to skip (default: 0)"
// @Success 200 {object} httpPlatform.PaginatedResponse{items=[]model.StageTemplateDTO}
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /stage-templates [get]
func (h *JobHandler) ListStageTemplates(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	pagination, err := httpPlatform.ParsePaginationParams(c)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "INVALID_PAGINATION_PARAMS", "Invalid pagination parameters")
		return
	}

	templates, total, err := h.service.ListStageTemplates(c.Request.Context(), userID, pagination.Limit, pagination.Offset)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list stage templates")
		return
	}

	httpPlatform.RespondWithPagination(c, http.StatusOK, templates, pagination.Limit, pagination.Offset, total)
}

// UpdateStageTemplate godoc
// @Summary Update a stage template
// @Description Update a reusable stage definition
// @Tags stage-templates
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param templateId path string true "Stage template ID"
// @Param request body model.UpdateStageTemplateRequest true "Stage template update"
// @Success 200 {object} model.StageTemplateDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /stage-templates/{templateId} [patch]
func (h *JobHandler) UpdateStageTemplate(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	templateID := c.Param("templateId")

	var req model.UpdateStageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	template, err := h.service.UpdateStageTemplate(c.Request.Context(), userID, templateID, &req)
	if err != nil {
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, template)
}

// DeleteStageTemplate godoc
// @Summary Delete a stage template
// @Description Delete a reusable stage definition. Fails with 409 if the template is used by any job stage.
// @Tags stage-templates
// @Security BearerAuth
// @Produce json
// @Param templateId path string true "Stage template ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse
// @Failure 409 {object} httpPlatform.ErrorResponse "Stage template in use"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /stage-templates/{templateId} [delete]
func (h *JobHandler) DeleteStageTemplate(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	templateID := c.Param("templateId")

	if err := h.service.DeleteStageTemplate(c.Request.Context(), userID, templateID); err != nil {
		respondWithMappedError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Stage template deleted successfully"})
}

// RegisterRoutes registers job, stage and stage-template routes
func (h *JobHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	jobs := router.Group("/jobs")
	jobs.Use(authMiddleware)
	{
		jobs.POST("", h.Create)
		jobs.GET("", h.List)
		jobs.GET("/:id", h.Get)
		jobs.PATCH("/:id", h.Update)
		jobs.DELETE("/:id", h.Delete)
		jobs.POST("/:id/favorite", h.ToggleFavorite)
		jobs.POST("/:id/move", h.Move)

		jobs.POST("/:id/stages", h.AddStage)
		jobs.GET("/:id/stages", h.ListStages)
		jobs.PATCH("/:id/stages/:stageId", h.UpdateStage)
		jobs.DELETE("/:id/stages/:stageId", h.DeleteStage)
	}

	templates := router.Group("/stage-templates")
	templates.Use(authMiddleware)
	{
		templates.POST("", h.CreateStageTemplate)
		templates.GET("", h.ListStageTemplates)
		templates.PATCH("/:templateId", h.UpdateStageTemplate)
		templates.DELETE("/:templateId", h.DeleteStageTemplate)
	}
}
