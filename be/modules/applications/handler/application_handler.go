package handler

import (
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/applications/model"
	"github.com/andreypavlenko/jobber/modules/applications/service"
	"github.com/gin-gonic/gin"
)

type ApplicationHandler struct {
	service *service.ApplicationService
}

func NewApplicationHandler(service *service.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{service: service}
}

// Create godoc
// @Summary Create a new application
// @Description Create a new job application linking a job and resume
// @Tags applications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.CreateApplicationRequest true "Application details"
// @Success 201 {object} model.ApplicationDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /applications [post]
func (h *ApplicationHandler) Create(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	var req model.CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	app, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusCreated, app)
}

// Get godoc
// @Summary Get an application
// @Description Get details of a specific application by ID
// @Tags applications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Application ID"
// @Success 200 {object} model.ApplicationDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Application not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /applications/{id} [get]
func (h *ApplicationHandler) Get(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	appID := c.Param("id")

	app, err := h.service.GetByID(c.Request.Context(), userID, appID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if model.GetErrorCode(err) == model.CodeApplicationNotFound {
			statusCode = http.StatusNotFound
		}
		httpPlatform.RespondWithError(c, statusCode, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, app)
}

// List godoc
// @Summary List applications
// @Description Get a paginated list of job applications for the authenticated user
// @Tags applications
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of items per page (default: 20, max: 100)"
// @Param offset query int false "Number of items to skip (default: 0)"
// @Param sort_by query string false "Sort field: last_activity, status, applied_at (default: last_activity)"
// @Param sort_dir query string false "Sort direction: asc, desc (default: desc)"
// @Success 200 {object} httpPlatform.PaginatedResponse{items=[]model.ApplicationDTO}
// @Failure 400 {object} httpPlatform.ErrorResponse "Invalid pagination parameters"
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /applications [get]
func (h *ApplicationHandler) List(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	
	// Parse pagination parameters
	pagination, err := httpPlatform.ParsePaginationParams(c)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "INVALID_PAGINATION_PARAMS", "Invalid pagination parameters")
		return
	}

	// Parse sorting parameters
	sortBy := c.DefaultQuery("sort_by", "last_activity")
	sortDir := c.DefaultQuery("sort_dir", "desc")

	apps, total, err := h.service.List(c.Request.Context(), userID, sortBy, sortDir, pagination.Limit, pagination.Offset)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list applications")
		return
	}
	httpPlatform.RespondWithPagination(c, http.StatusOK, apps, pagination.Limit, pagination.Offset, total)
}

// Update godoc
// @Summary Update an application
// @Description Update status of a specific application
// @Tags applications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param request body model.UpdateApplicationRequest true "Updated application details"
// @Success 200 {object} model.ApplicationDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Application not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /applications/{id} [patch]
func (h *ApplicationHandler) Update(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	appID := c.Param("id")
	var req model.UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	app, err := h.service.Update(c.Request.Context(), userID, appID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if model.GetErrorCode(err) == model.CodeApplicationNotFound {
			statusCode = http.StatusNotFound
		}
		httpPlatform.RespondWithError(c, statusCode, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, app)
}

// Delete godoc
// @Summary Delete an application
// @Description Delete a specific application by ID
// @Tags applications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Application ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Application not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /applications/{id} [delete]
func (h *ApplicationHandler) Delete(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	appID := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), userID, appID); err != nil {
		statusCode := http.StatusInternalServerError
		if model.GetErrorCode(err) == model.CodeApplicationNotFound {
			statusCode = http.StatusNotFound
		}
		httpPlatform.RespondWithError(c, statusCode, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Application deleted successfully"})
}

// AddStage godoc
// @Summary Add a stage to an application
// @Description Add a new stage to an application's timeline
// @Tags applications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param request body model.AddStageRequest true "Stage template ID"
// @Success 201 {object} model.ApplicationStageDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Application not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /applications/{id}/stages [post]
func (h *ApplicationHandler) AddStage(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	appID := c.Param("id")
	var req model.AddStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	stage, err := h.service.AddStage(c.Request.Context(), userID, appID, &req)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusCreated, stage)
}

// UpdateStage godoc
// @Summary Update an application stage
// @Description Update status and other fields of a specific stage
// @Tags applications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param stageId path string true "Stage ID"
// @Param request body model.UpdateStageRequest true "Stage update details"
// @Success 200 {object} model.ApplicationStageDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Application or stage not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /applications/{id}/stages/{stageId} [patch]
func (h *ApplicationHandler) UpdateStage(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	appID := c.Param("id")
	stageID := c.Param("stageId")
	var req model.UpdateStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	stage, err := h.service.UpdateStage(c.Request.Context(), userID, appID, stageID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if model.GetErrorCode(err) == model.CodeApplicationStageNotFound {
			statusCode = http.StatusNotFound
		} else if model.GetErrorCode(err) == model.CodeInvalidStatus {
			statusCode = http.StatusBadRequest
		}
		httpPlatform.RespondWithError(c, statusCode, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, stage)
}

// CompleteStage godoc
// @Summary Complete an application stage
// @Description Mark a specific stage as completed
// @Tags applications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Application ID"
// @Param stageId path string true "Stage ID"
// @Param request body model.CompleteStageRequest false "Completion details"
// @Success 200 {object} model.ApplicationStageDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Application or stage not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /applications/{id}/stages/{stageId}/complete [patch]
func (h *ApplicationHandler) CompleteStage(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	appID := c.Param("id")
	stageID := c.Param("stageId")
	var req model.CompleteStageRequest
	// Body is optional; ignore EOF/empty body errors but reject malformed JSON
	if err := c.ShouldBindJSON(&req); err != nil && c.Request.ContentLength > 0 {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	stage, err := h.service.CompleteStage(c.Request.Context(), userID, appID, stageID, &req)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, stage)
}

// ListStages godoc
// @Summary List application stages
// @Description Get all stages for a specific application
// @Tags applications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Application ID"
// @Success 200 {object} []model.ApplicationStageDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Application not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /applications/{id}/stages [get]
func (h *ApplicationHandler) ListStages(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	appID := c.Param("id")

	stages, err := h.service.ListStages(c.Request.Context(), userID, appID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, stages)
}

// DeleteStage godoc
// @Summary Delete an application stage
// @Description Delete a specific stage from an application
// @Tags applications
// @Security BearerAuth
// @Produce json
// @Param id path string true "Application ID"
// @Param stageId path string true "Stage ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Application or stage not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /applications/{id}/stages/{stageId} [delete]
func (h *ApplicationHandler) DeleteStage(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	appID := c.Param("id")
	stageID := c.Param("stageId")

	if err := h.service.DeleteStage(c.Request.Context(), userID, appID, stageID); err != nil {
		statusCode := http.StatusInternalServerError
		if model.GetErrorCode(err) == model.CodeApplicationStageNotFound {
			statusCode = http.StatusNotFound
		}
		httpPlatform.RespondWithError(c, statusCode, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Stage deleted successfully"})
}

// CreateStageTemplate godoc
// @Summary Create a stage template
// @Description Create a reusable stage template for the authenticated user
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
func (h *ApplicationHandler) CreateStageTemplate(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	var req model.CreateStageTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	template, err := h.service.CreateStageTemplate(c.Request.Context(), userID, &req)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusCreated, template)
}

// ListStageTemplates godoc
// @Summary List stage templates
// @Description Get a paginated list of stage templates for the authenticated user
// @Tags stage-templates
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of items per page (default: 20, max: 100)"
// @Param offset query int false "Number of items to skip (default: 0)"
// @Success 200 {object} httpPlatform.PaginatedResponse{items=[]model.StageTemplateDTO}
// @Failure 400 {object} httpPlatform.ErrorResponse "Invalid pagination parameters"
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /stage-templates [get]
func (h *ApplicationHandler) ListStageTemplates(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	
	// Parse pagination parameters
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
// @Description Update details of a specific stage template
// @Tags stage-templates
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param templateId path string true "Stage Template ID"
// @Param request body model.UpdateStageTemplateRequest true "Updated stage template details"
// @Success 200 {object} model.StageTemplateDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Stage template not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /stage-templates/{templateId} [patch]
func (h *ApplicationHandler) UpdateStageTemplate(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
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
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, template)
}

// DeleteStageTemplate godoc
// @Summary Delete a stage template
// @Description Delete a specific stage template by ID
// @Tags stage-templates
// @Security BearerAuth
// @Produce json
// @Param templateId path string true "Stage Template ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Stage template not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /stage-templates/{templateId} [delete]
func (h *ApplicationHandler) DeleteStageTemplate(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	templateID := c.Param("templateId")

	if err := h.service.DeleteStageTemplate(c.Request.Context(), userID, templateID); err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Stage template deleted successfully"})
}

func (h *ApplicationHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	apps := router.Group("/applications")
	apps.Use(authMiddleware)
	{
		apps.POST("", h.Create)
		apps.GET("", h.List)
		apps.GET("/:id", h.Get)
		apps.PATCH("/:id", h.Update)
		apps.DELETE("/:id", h.Delete)
		
		// Stages
		apps.POST("/:id/stages", h.AddStage)
		apps.GET("/:id/stages", h.ListStages)
		apps.PATCH("/:id/stages/:stageId", h.UpdateStage)
		apps.PATCH("/:id/stages/:stageId/complete", h.CompleteStage)
		apps.DELETE("/:id/stages/:stageId", h.DeleteStage)
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
