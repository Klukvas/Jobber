package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TailoringService interface {
	Generate(context.Context, string, string, string, string) (*model.TailoringProposalDTO, error)
	Apply(context.Context, string, string, string, *model.ApplyTailoringRequest) (*model.ApplyTailoringResult, bool, error)
}

type TailoringHandler struct{ service TailoringService }

func NewTailoringHandler(service TailoringService) *TailoringHandler {
	return &TailoringHandler{service: service}
}

func (h *TailoringHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware, rateLimiter gin.HandlerFunc) {
	router.POST("/jobs/:id/resume-tailoring", authMiddleware, rateLimiter, h.Generate)
	router.POST("/jobs/:id/resume-tailoring/:proposalId/apply", authMiddleware, rateLimiter, h.Apply)
}

func (h *TailoringHandler) Generate(c *gin.Context) {
	userID, ok := tailoringUser(c)
	if !ok {
		return
	}
	var req struct {
		ResumeBuilderID string `json:"resume_builder_id" binding:"required,uuid"`
		RequestKey      string `json:"request_key" binding:"omitempty,uuid"`
		Locale          string `json:"locale" binding:"omitempty,oneof=en ru uk"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if err := c.ShouldBindJSON(&req); err != nil {
		tailoringError(c, model.ErrTailoringValidation)
		return
	}
	if req.Locale == "" {
		req.Locale = "en"
	}
	var result *model.TailoringProposalDTO
	var err error
	if durable, ok := h.service.(interface {
		GenerateIdempotent(context.Context, string, string, string, string, string) (*model.TailoringProposalDTO, error)
	}); ok && req.RequestKey != "" {
		result, err = durable.GenerateIdempotent(c.Request.Context(), userID, c.Param("id"), req.ResumeBuilderID, req.Locale, req.RequestKey)
	} else {
		result, err = h.service.Generate(c.Request.Context(), userID, c.Param("id"), req.ResumeBuilderID, req.Locale)
	}
	if err != nil {
		tailoringError(c, err)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, result)
}

func (h *TailoringHandler) Apply(c *gin.Context) {
	userID, ok := tailoringUser(c)
	if !ok {
		return
	}
	if _, err := uuid.Parse(c.Param("proposalId")); err != nil {
		tailoringError(c, model.ErrTailoringValidation)
		return
	}
	var req model.ApplyTailoringRequest
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16384)
	if err := c.ShouldBindJSON(&req); err != nil {
		tailoringError(c, model.ErrTailoringValidation)
		return
	}
	result, replayed, err := h.service.Apply(c.Request.Context(), userID, c.Param("id"), c.Param("proposalId"), &req)
	if err != nil {
		tailoringError(c, err)
		return
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	httpPlatform.RespondWithData(c, status, result)
}

func tailoringUser(c *gin.Context) (string, bool) {
	userID, ok := auth.GetUserID(c)
	if !ok {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return "", false
	}
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		tailoringError(c, model.ErrTailoringValidation)
		return "", false
	}
	return userID, true
}

func tailoringError(c *gin.Context, err error) {
	status, code, message := http.StatusInternalServerError, "INTERNAL_ERROR", "Unable to adapt resume"
	switch {
	case errors.Is(err, model.ErrRecognitionRequired):
		status, code, message = http.StatusConflict, "RECOGNITION_REQUIRED", "Review and confirm the editable copy before adapting it"
	case errors.Is(err, model.ErrPDFUnreadable):
		status, code, message = http.StatusUnprocessableEntity, "PDF_UNREADABLE", "Could not read this PDF. Scanned PDFs are not supported. Paste your resume text instead"
	case errors.Is(err, model.ErrOperationConflict):
		status, code, message = http.StatusConflict, "REQUEST_KEY_CONFLICT", "This request key was used with different input"
	case errors.Is(err, model.ErrOperationPending):
		status, code, message = http.StatusConflict, "OPERATION_PENDING", "This request is still running. Retry with the same request key"
	case errors.Is(err, model.ErrOperationInterrupted):
		status, code, message = http.StatusConflict, "OPERATION_INTERRUPTED", "The previous request was interrupted. It will not run again with this key"
	case errors.Is(err, model.ErrOperationDeleted):
		status, code, message = http.StatusGone, "OPERATION_RESULT_DELETED", "The result was deleted. This request cannot recreate it"
	case errors.Is(err, model.ErrTailoringValidation):
		status, code, message = http.StatusBadRequest, "VALIDATION_ERROR", "Select valid changes and provide a title and a saved Job description"
	case errors.Is(err, model.ErrTailoringNotFound), errors.Is(err, model.ErrResumeBuilderNotFound), errors.Is(err, model.ErrNotOwner):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "Job or Builder Resume not found"
	case errors.Is(err, subModel.ErrLimitReached):
		status, code, message = http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the usage limit for your current plan"
	case errors.Is(err, model.ErrSourceChanged):
		status, code, message = http.StatusConflict, "SOURCE_CHANGED", "The source Builder Resume changed. Generate a new proposal"
	case errors.Is(err, model.ErrJobChanged):
		status, code, message = http.StatusConflict, "JOB_CHANGED", "The Job changed. Generate a new proposal"
	case errors.Is(err, model.ErrProposalAlreadyApplied):
		status, code, message = http.StatusConflict, "PROPOSAL_ALREADY_APPLIED", "This proposal has already been applied"
	case errors.Is(err, model.ErrProposalExpired):
		status, code, message = http.StatusGone, "PROPOSAL_EXPIRED", "The proposal expired. Generate a new proposal"
	case errors.Is(err, model.ErrAIUnavailable):
		status, code, message = http.StatusServiceUnavailable, "AI_UNAVAILABLE", "AI resume tailoring is currently unavailable"
	case errors.Is(err, model.ErrProposalStoreUnavailable):
		status, code, message = http.StatusServiceUnavailable, "PROPOSAL_STORE_UNAVAILABLE", "Resume tailoring is temporarily unavailable"
	case errors.Is(err, model.ErrAIInvalidResponse):
		status, code, message = http.StatusBadGateway, "AI_INVALID_RESPONSE", "AI returned unsupported changes. Please try again"
	case errors.Is(err, model.ErrAIRequestFailed):
		status, code, message = http.StatusBadGateway, "AI_REQUEST_FAILED", "AI could not generate suggestions. Please try again"
	case errors.Is(err, model.ErrAITimeout):
		status, code, message = http.StatusGatewayTimeout, "AI_TIMEOUT", "AI took too long. Please try again"
	}
	httpPlatform.RespondWithError(c, status, code, message)
}
