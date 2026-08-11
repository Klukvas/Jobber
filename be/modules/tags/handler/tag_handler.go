package handler

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/tags/model"
	"github.com/andreypavlenko/jobber/modules/tags/service"
	"github.com/gin-gonic/gin"
)

type TagHandler struct {
	service *service.TagService
}

func NewTagHandler(service *service.TagService) *TagHandler {
	return &TagHandler{service: service}
}

// Create godoc
// @Summary Create a tag
// @Description Create a tag for the authenticated user
// @Tags tags
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.CreateTagRequest true "Tag details"
// @Success 201 {object} model.TagDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 409 {object} httpPlatform.ErrorResponse "Tag name already exists"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /tags [post]
func (h *TagHandler) Create(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	var req model.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(model.CodeValidationError), "Invalid request payload")
		return
	}

	tag, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondError(c, err, "Failed to create tag")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusCreated, tag)
}

// List godoc
// @Summary List tags
// @Description List all tags for the authenticated user
// @Tags tags
// @Security BearerAuth
// @Produce json
// @Success 200 {object} []model.TagDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /tags [get]
func (h *TagHandler) List(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}

	tags, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.CodeInternalError), "Failed to list tags")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, tags)
}

// Delete godoc
// @Summary Delete a tag
// @Description Delete a tag by ID (also removes its relations)
// @Tags tags
// @Security BearerAuth
// @Produce json
// @Param id path string true "Tag ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Tag not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /tags/{id} [delete]
func (h *TagHandler) Delete(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	tagID := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), userID, tagID); err != nil {
		h.respondError(c, err, "Failed to delete tag")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Tag deleted successfully"})
}

// Attach godoc
// @Summary Attach a tag to an entity
// @Description Attach a tag to a job or company owned by the user
// @Tags tags
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Tag ID"
// @Param request body model.AttachTagRequest true "Entity to attach the tag to"
// @Success 200 {object} map[string]string
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Tag or entity not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /tags/{id}/relations [post]
func (h *TagHandler) Attach(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	tagID := c.Param("id")

	var req model.AttachTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(model.CodeValidationError), "Invalid request payload")
		return
	}

	if err := h.service.Attach(c.Request.Context(), userID, tagID, &req); err != nil {
		h.respondError(c, err, "Failed to attach tag")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Tag attached successfully"})
}

// Detach godoc
// @Summary Detach a tag from an entity
// @Description Remove a tag from a job or company
// @Tags tags
// @Security BearerAuth
// @Produce json
// @Param id path string true "Tag ID"
// @Param entity_type query string true "Entity type: job or company"
// @Param entity_id query string true "Entity ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /tags/{id}/relations [delete]
func (h *TagHandler) Detach(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	tagID := c.Param("id")
	entityType := c.Query("entity_type")
	entityID := c.Query("entity_id")
	if entityID == "" {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(model.CodeValidationError), "entity_id is required")
		return
	}

	if err := h.service.Detach(c.Request.Context(), userID, tagID, entityType, entityID); err != nil {
		h.respondError(c, err, "Failed to detach tag")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Tag detached successfully"})
}

// ListByJob godoc
// @Summary List tags for a job
// @Tags tags
// @Security BearerAuth
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} []model.TagDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id}/tags [get]
func (h *TagHandler) ListByJob(c *gin.Context) {
	h.listByEntity(c, model.EntityTypeJob)
}

// ListByCompany godoc
// @Summary List tags for a company
// @Tags tags
// @Security BearerAuth
// @Produce json
// @Param id path string true "Company ID"
// @Success 200 {object} []model.TagDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /companies/{id}/tags [get]
func (h *TagHandler) ListByCompany(c *gin.Context) {
	h.listByEntity(c, model.EntityTypeCompany)
}

func (h *TagHandler) listByEntity(c *gin.Context, entityType string) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	entityID := c.Param("id")

	tags, err := h.service.ListByEntity(c.Request.Context(), userID, entityType, entityID)
	if err != nil {
		h.respondError(c, err, "Failed to list tags")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, tags)
}

// respondError maps domain errors to HTTP responses, defaulting to 500.
func (h *TagHandler) respondError(c *gin.Context, err error, fallbackMessage string) {
	statusCode := http.StatusInternalServerError
	errorCode := string(model.CodeInternalError)
	errorMessage := fallbackMessage

	switch {
	case errors.Is(err, model.ErrTagNameRequired):
		statusCode = http.StatusBadRequest
		errorCode = string(model.CodeTagNameRequired)
		errorMessage = "Tag name is required"
	case errors.Is(err, model.ErrInvalidColor):
		statusCode = http.StatusBadRequest
		errorCode = string(model.CodeInvalidColor)
		errorMessage = "Color must be a hex value like #2563eb"
	case errors.Is(err, model.ErrInvalidEntityType):
		statusCode = http.StatusBadRequest
		errorCode = string(model.CodeInvalidEntityType)
		errorMessage = "entity_type must be 'job' or 'company'"
	case errors.Is(err, model.ErrTagNameExists):
		statusCode = http.StatusConflict
		errorCode = string(model.CodeTagNameExists)
		errorMessage = "A tag with this name already exists"
	case errors.Is(err, model.ErrTagNotFound):
		statusCode = http.StatusNotFound
		errorCode = string(model.CodeTagNotFound)
		errorMessage = "Tag not found"
	case errors.Is(err, model.ErrEntityNotFound):
		statusCode = http.StatusNotFound
		errorCode = string(model.CodeEntityNotFound)
		errorMessage = "Tag or entity not found"
	}

	httpPlatform.RespondWithError(c, statusCode, errorCode, errorMessage)
}

func (h *TagHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	tags := router.Group("/tags")
	tags.Use(authMiddleware)
	{
		tags.POST("", h.Create)
		tags.GET("", h.List)
		tags.DELETE("/:id", h.Delete)
		tags.POST("/:id/relations", h.Attach)
		tags.DELETE("/:id/relations", h.Detach)
	}

	jobs := router.Group("/jobs")
	jobs.Use(authMiddleware)
	{
		jobs.GET("/:id/tags", h.ListByJob)
	}

	companies := router.Group("/companies")
	companies.Use(authMiddleware)
	{
		companies.GET("/:id/tags", h.ListByCompany)
	}
}
