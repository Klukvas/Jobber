package handler

import (
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/gin-gonic/gin"
)

// --- Contact ---

func (h *ResumeBuilderHandler) UpsertContact(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.UpsertContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.UpsertContact(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

// --- Summary ---

func (h *ResumeBuilderHandler) UpsertSummary(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.UpsertSummaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.UpsertSummary(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

// --- Section Order ---

func (h *ResumeBuilderHandler) UpdateSectionOrder(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.BatchUpdateSectionOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.UpdateSectionOrder(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

// --- Experiences ---

func (h *ResumeBuilderHandler) CreateExperience(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.CreateExperienceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.CreateExperience(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

func (h *ResumeBuilderHandler) UpdateExperience(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	var req model.UpdateExperienceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.UpdateExperience(c.Request.Context(), userID, id, entryID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *ResumeBuilderHandler) DeleteExperience(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	if err := h.service.DeleteExperience(c.Request.Context(), userID, id, entryID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Educations ---

func (h *ResumeBuilderHandler) CreateEducation(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.CreateEducationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.CreateEducation(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

func (h *ResumeBuilderHandler) UpdateEducation(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	var req model.UpdateEducationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.UpdateEducation(c.Request.Context(), userID, id, entryID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *ResumeBuilderHandler) DeleteEducation(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	if err := h.service.DeleteEducation(c.Request.Context(), userID, id, entryID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Skills ---

func (h *ResumeBuilderHandler) CreateSkill(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.CreateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.CreateSkill(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

func (h *ResumeBuilderHandler) UpdateSkill(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	var req model.UpdateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.UpdateSkill(c.Request.Context(), userID, id, entryID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *ResumeBuilderHandler) DeleteSkill(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	if err := h.service.DeleteSkill(c.Request.Context(), userID, id, entryID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Languages ---

func (h *ResumeBuilderHandler) CreateLanguage(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.CreateLanguageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.CreateLanguage(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

func (h *ResumeBuilderHandler) UpdateLanguage(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	var req model.UpdateLanguageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.UpdateLanguage(c.Request.Context(), userID, id, entryID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *ResumeBuilderHandler) DeleteLanguage(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	if err := h.service.DeleteLanguage(c.Request.Context(), userID, id, entryID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Certifications ---

func (h *ResumeBuilderHandler) CreateCertification(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.CreateCertificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.CreateCertification(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

func (h *ResumeBuilderHandler) UpdateCertification(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	var req model.UpdateCertificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.UpdateCertification(c.Request.Context(), userID, id, entryID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *ResumeBuilderHandler) DeleteCertification(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	if err := h.service.DeleteCertification(c.Request.Context(), userID, id, entryID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Projects ---

func (h *ResumeBuilderHandler) CreateProject(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.CreateProject(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

func (h *ResumeBuilderHandler) UpdateProject(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	var req model.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.UpdateProject(c.Request.Context(), userID, id, entryID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *ResumeBuilderHandler) DeleteProject(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	if err := h.service.DeleteProject(c.Request.Context(), userID, id, entryID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Volunteering ---

func (h *ResumeBuilderHandler) CreateVolunteering(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.CreateVolunteeringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.CreateVolunteering(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

func (h *ResumeBuilderHandler) UpdateVolunteering(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	var req model.UpdateVolunteeringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.UpdateVolunteering(c.Request.Context(), userID, id, entryID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *ResumeBuilderHandler) DeleteVolunteering(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	if err := h.service.DeleteVolunteering(c.Request.Context(), userID, id, entryID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// --- Custom Sections ---

func (h *ResumeBuilderHandler) CreateCustomSection(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	var req model.CreateCustomSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.CreateCustomSection(c.Request.Context(), userID, id, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, result)
}

func (h *ResumeBuilderHandler) UpdateCustomSection(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	var req model.UpdateCustomSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	result, err := h.service.UpdateCustomSection(c.Request.Context(), userID, id, entryID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *ResumeBuilderHandler) DeleteCustomSection(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	id := c.Param("id")
	entryID := c.Param("entryId")
	if err := h.service.DeleteCustomSection(c.Request.Context(), userID, id, entryID); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
