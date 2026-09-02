package handler

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AutofillProfileHandler serves AI-extracted Autofill Profiles of Uploaded
// Resumes. Registered only when the AI client is configured (same nil-gating
// as the match-score handler), so the route is absent when AI is disabled.
type AutofillProfileHandler struct {
	service *service.AutofillProfileService
	logger  *zap.Logger
}

func NewAutofillProfileHandler(service *service.AutofillProfileService, logger *zap.Logger) *AutofillProfileHandler {
	return &AutofillProfileHandler{service: service, logger: logger}
}

// Get godoc
// @Summary Get the autofill profile of an uploaded resume
// @Description Returns structured form-filling data AI-extracted from the resume PDF. First request on a paid plan extracts and caches; later requests are free cache hits.
// @Tags resumes
// @Security BearerAuth
// @Produce json
// @Param id path string true "Resume ID"
// @Success 200 {object} model.AutofillProfileDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 403 {object} httpPlatform.ErrorResponse "PAID_FEATURE or PLAN_LIMIT_REACHED"
// @Failure 404 {object} httpPlatform.ErrorResponse "Resume not found"
// @Failure 422 {object} httpPlatform.ErrorResponse "Resume file unreadable or missing"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /resumes/{id}/autofill-profile [get]
func (h *AutofillProfileHandler) Get(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	resumeID := c.Param("id")

	profile, err := h.service.GetProfile(c.Request.Context(), userID, resumeID)
	if err != nil {
		switch {
		case errors.Is(err, subModel.ErrPaidFeature):
			httpPlatform.RespondWithError(c, http.StatusForbidden, "PAID_FEATURE", "Autofill from uploaded resumes is available on paid plans.")
		case errors.Is(err, subModel.ErrLimitReached):
			httpPlatform.RespondWithError(c, http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the AI usage limit for your current plan.")
		case errors.Is(err, model.ErrResumeNotFound):
			httpPlatform.RespondWithError(c, http.StatusNotFound, string(model.CodeResumeNotFound), model.GetErrorMessage(err))
		case errors.Is(err, model.ErrResumeUnreadable), errors.Is(err, model.ErrResumeFileMissing):
			httpPlatform.RespondWithError(c, http.StatusUnprocessableEntity, string(model.GetErrorCode(err)), model.GetErrorMessage(err))
		default:
			// Downloads, AI and storage failures carry internals — log, don't leak.
			h.logger.Error("autofill profile extraction failed",
				zap.String("user_id", userID), zap.String("resume_id", resumeID), zap.Error(err))
			httpPlatform.RespondWithError(c, http.StatusInternalServerError, "AUTOFILL_PROFILE_FAILED", "Failed to prepare the autofill profile")
		}
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, profile)
}

func (h *AutofillProfileHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware, rateLimiter gin.HandlerFunc) {
	router.GET("/resumes/:id/autofill-profile", authMiddleware, rateLimiter, h.Get)
}
