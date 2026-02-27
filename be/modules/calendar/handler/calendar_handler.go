package handler

import (
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/calendar/model"
	"github.com/andreypavlenko/jobber/modules/calendar/service"
	"github.com/gin-gonic/gin"
)

// CalendarHandler handles calendar HTTP requests
type CalendarHandler struct {
	service *service.CalendarService
}

// NewCalendarHandler creates a new calendar handler
func NewCalendarHandler(service *service.CalendarService) *CalendarHandler {
	return &CalendarHandler{service: service}
}

// GetAuthURL returns the Google OAuth URL
func (h *CalendarHandler) GetAuthURL(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	url, err := h.service.GetAuthURL(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to generate auth URL")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, model.OAuthURLResponse{URL: url})
}

// HandleCallback processes the OAuth callback from Google
func (h *CalendarHandler) HandleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		c.Redirect(http.StatusTemporaryRedirect, h.service.FrontendURL()+"/settings?calendar=error")
		return
	}

	redirectURL, err := h.service.HandleCallback(c.Request.Context(), code, state)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, h.service.FrontendURL()+"/settings?calendar=error")
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// GetStatus checks if the user's calendar is connected
func (h *CalendarHandler) GetStatus(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	status, err := h.service.GetStatus(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get status")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, status)
}

// Disconnect removes the user's calendar connection
func (h *CalendarHandler) Disconnect(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	if err := h.service.Disconnect(c.Request.Context(), userID); err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)

		statusCode := http.StatusInternalServerError
		if errorCode == model.CodeNotConnected {
			statusCode = http.StatusNotFound
		}

		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Calendar disconnected"})
}

// CreateEvent creates a calendar event for a stage
func (h *CalendarHandler) CreateEvent(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req model.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	event, err := h.service.CreateEvent(c.Request.Context(), userID, &req)
	if err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)

		statusCode := http.StatusInternalServerError
		switch errorCode {
		case model.CodeNotConnected, model.CodeTokenExpired:
			statusCode = http.StatusUnprocessableEntity
		case model.CodeStageNotFound:
			statusCode = http.StatusNotFound
		case model.CodeInvalidTimeRange:
			statusCode = http.StatusBadRequest
		case model.CodeEventAlreadyExists:
			statusCode = http.StatusConflict
		}

		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, event)
}

// DeleteEvent deletes a calendar event for a stage
func (h *CalendarHandler) DeleteEvent(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	stageID := c.Param("stageId")
	if stageID == "" {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Stage ID is required")
		return
	}

	if err := h.service.DeleteEvent(c.Request.Context(), userID, stageID); err != nil {
		errorCode := model.GetErrorCode(err)
		errorMessage := model.GetErrorMessage(err)

		statusCode := http.StatusInternalServerError
		switch errorCode {
		case model.CodeNotConnected, model.CodeTokenExpired:
			statusCode = http.StatusUnprocessableEntity
		case model.CodeStageNotFound, model.CodeEventNotFound:
			statusCode = http.StatusNotFound
		}

		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Calendar event deleted"})
}

// RegisterRoutes registers calendar routes
func (h *CalendarHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	cal := router.Group("/calendar")
	{
		cal.GET("/auth", authMiddleware, h.GetAuthURL)
		cal.GET("/callback", h.HandleCallback) // No JWT auth — uses OAuth state
		cal.GET("/status", authMiddleware, h.GetStatus)
		cal.DELETE("", authMiddleware, h.Disconnect)
		cal.POST("/events", authMiddleware, h.CreateEvent)
		cal.DELETE("/events/:stageId", authMiddleware, h.DeleteEvent)
	}
}
