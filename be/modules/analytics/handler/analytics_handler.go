package handler

import (
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/analytics/service"
	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	service *service.AnalyticsService
}

func NewAnalyticsHandler(service *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{service: service}
}

// GetOverview godoc
// @Summary Get analytics overview
// @Description Get high-level application statistics for the authenticated user
// @Tags analytics
// @Security BearerAuth
// @Produce json
// @Success 200 {object} model.OverviewAnalytics
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /analytics/overview [get]
func (h *AnalyticsHandler) GetOverview(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}

	analytics, err := h.service.GetOverview(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "ANALYTICS_ERROR", "Failed to get overview analytics")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, analytics)
}

// GetFunnel godoc
// @Summary Get funnel analytics
// @Description Get stage-based funnel metrics for the authenticated user
// @Tags analytics
// @Security BearerAuth
// @Produce json
// @Success 200 {object} model.FunnelAnalytics
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /analytics/funnel [get]
func (h *AnalyticsHandler) GetFunnel(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}

	analytics, err := h.service.GetFunnel(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "ANALYTICS_ERROR", "Failed to get funnel analytics")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, analytics)
}

// GetStageTime godoc
// @Summary Get stage time analytics
// @Description Get timing metrics per stage for the authenticated user
// @Tags analytics
// @Security BearerAuth
// @Produce json
// @Success 200 {object} model.StageTimeAnalytics
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /analytics/stages [get]
func (h *AnalyticsHandler) GetStageTime(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}

	analytics, err := h.service.GetStageTime(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "ANALYTICS_ERROR", "Failed to get stage time analytics")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, analytics)
}

// GetResumeEffectiveness godoc
// @Summary Get resume effectiveness analytics
// @Description Get effectiveness metrics per resume for the authenticated user
// @Tags analytics
// @Security BearerAuth
// @Produce json
// @Success 200 {object} model.ResumeAnalytics
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /analytics/resumes [get]
func (h *AnalyticsHandler) GetResumeEffectiveness(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}

	analytics, err := h.service.GetResumeEffectiveness(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "ANALYTICS_ERROR", "Failed to get resume effectiveness analytics")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, analytics)
}

// GetSourceAnalytics godoc
// @Summary Get source analytics
// @Description Get metrics grouped by job source for the authenticated user
// @Tags analytics
// @Security BearerAuth
// @Produce json
// @Success 200 {object} model.SourceAnalytics
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /analytics/sources [get]
func (h *AnalyticsHandler) GetSourceAnalytics(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}

	analytics, err := h.service.GetSourceAnalytics(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "ANALYTICS_ERROR", "Failed to get source analytics")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, analytics)
}

// RegisterRoutes registers analytics routes
func (h *AnalyticsHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	analytics := router.Group("/analytics")
	analytics.Use(authMiddleware)
	{
		analytics.GET("/overview", h.GetOverview)
		analytics.GET("/funnel", h.GetFunnel)
		analytics.GET("/stages", h.GetStageTime)
		analytics.GET("/resumes", h.GetResumeEffectiveness)
		analytics.GET("/sources", h.GetSourceAnalytics)
	}
}
