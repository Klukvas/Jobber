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
)

// ResumeBuilderHandler handles resume builder HTTP requests.
type ResumeBuilderHandler struct {
	service *service.ResumeBuilderService
}

// NewResumeBuilderHandler creates a new ResumeBuilderHandler.
func NewResumeBuilderHandler(service *service.ResumeBuilderService) *ResumeBuilderHandler {
	return &ResumeBuilderHandler{service: service}
}

// RegisterRoutes registers all resume builder routes.
func (h *ResumeBuilderHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := router.Group("/resume-builder")
	{
		group.POST("", authMiddleware, h.Create)
		group.GET("", authMiddleware, h.List)
		group.GET("/:id", authMiddleware, h.Get)
		group.PATCH("/:id", authMiddleware, h.Update)
		group.DELETE("/:id", authMiddleware, h.Delete)
		group.POST("/:id/duplicate", authMiddleware, h.Duplicate)

		// 1:1 sections
		group.PUT("/:id/contact", authMiddleware, h.UpsertContact)
		group.PUT("/:id/summary", authMiddleware, h.UpsertSummary)
		group.PUT("/:id/section-order", authMiddleware, h.UpdateSectionOrder)

		// 1:N sections
		registerSectionRoutes(group, authMiddleware, "experiences", h.CreateExperience, h.UpdateExperience, h.DeleteExperience)
		registerSectionRoutes(group, authMiddleware, "educations", h.CreateEducation, h.UpdateEducation, h.DeleteEducation)
		registerSectionRoutes(group, authMiddleware, "skills", h.CreateSkill, h.UpdateSkill, h.DeleteSkill)
		registerSectionRoutes(group, authMiddleware, "languages", h.CreateLanguage, h.UpdateLanguage, h.DeleteLanguage)
		registerSectionRoutes(group, authMiddleware, "certifications", h.CreateCertification, h.UpdateCertification, h.DeleteCertification)
		registerSectionRoutes(group, authMiddleware, "projects", h.CreateProject, h.UpdateProject, h.DeleteProject)
		registerSectionRoutes(group, authMiddleware, "volunteering", h.CreateVolunteering, h.UpdateVolunteering, h.DeleteVolunteering)
		registerSectionRoutes(group, authMiddleware, "custom-sections", h.CreateCustomSection, h.UpdateCustomSection, h.DeleteCustomSection)
	}
}

func registerSectionRoutes(group *gin.RouterGroup, authMiddleware gin.HandlerFunc, section string, create, update, del gin.HandlerFunc) {
	group.POST("/:id/"+section, authMiddleware, create)
	group.PATCH("/:id/"+section+"/:entryId", authMiddleware, update)
	group.DELETE("/:id/"+section+"/:entryId", authMiddleware, del)
}

func (h *ResumeBuilderHandler) Create(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req model.CreateResumeBuilderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

func (h *ResumeBuilderHandler) List(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	items, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, items)
}

func (h *ResumeBuilderHandler) Get(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	result, err := h.service.Get(c.Request.Context(), userID, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *ResumeBuilderHandler) Update(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.UpdateResumeBuilderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.Update(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *ResumeBuilderHandler) Delete(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	if err := h.service.Delete(c.Request.Context(), userID, id); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ResumeBuilderHandler) Duplicate(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	result, err := h.service.Duplicate(c.Request.Context(), userID, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

func (h *ResumeBuilderHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, subModel.ErrLimitReached) {
		httpPlatform.RespondWithError(c, http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the resume builder limit for your current plan.")
		return
	}

	errorCode := model.GetErrorCode(err)
	errorMessage := model.GetErrorMessage(err)

	statusCode := http.StatusInternalServerError
	switch errorCode {
	case model.CodeResumeBuilderNotFound, model.CodeSectionEntryNotFound:
		statusCode = http.StatusNotFound
	case model.CodeNotOwner:
		statusCode = http.StatusForbidden
	case model.CodeInvalidTemplate, model.CodeInvalidSpacing, model.CodeInvalidColor,
		model.CodeInvalidFont, model.CodeInvalidSectionKey, model.CodeInvalidSkillDisplay,
		model.CodeInvalidMargin, model.CodeInvalidSidebarWidth, model.CodeInvalidFontSize,
		model.CodeInvalidLayoutMode, model.CodeInvalidColumnValue:
		statusCode = http.StatusBadRequest
	}

	httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
}
