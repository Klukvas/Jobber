package handler

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/reminders/model"
	"github.com/andreypavlenko/jobber/modules/reminders/service"
	"github.com/gin-gonic/gin"
)

type ReminderHandler struct {
	service *service.ReminderService
}

func NewReminderHandler(service *service.ReminderService) *ReminderHandler {
	return &ReminderHandler{service: service}
}

// Create godoc
// @Summary Create a reminder
// @Description Create a reminder for a job (optionally a specific stage)
// @Tags reminders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.CreateReminderRequest true "Reminder details"
// @Success 201 {object} model.ReminderDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Job not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /reminders [post]
func (h *ReminderHandler) Create(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	var req model.CreateReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(model.CodeValidationError), "Invalid request payload")
		return
	}

	reminder, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		h.respondError(c, err, "Failed to create reminder")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusCreated, reminder)
}

// List godoc
// @Summary List all reminders
// @Description Get all reminders for the authenticated user, ordered by remind_at
// @Tags reminders
// @Security BearerAuth
// @Produce json
// @Success 200 {object} []model.ReminderDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /reminders [get]
func (h *ReminderHandler) List(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}

	reminders, err := h.service.ListByUser(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.CodeInternalError), "Failed to list reminders")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, reminders)
}

// ListByJob godoc
// @Summary List reminders for a job
// @Description Get all reminders for a specific job
// @Tags reminders
// @Security BearerAuth
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} []model.ReminderDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id}/reminders [get]
func (h *ReminderHandler) ListByJob(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	jobID := c.Param("id")

	reminders, err := h.service.ListByJob(c.Request.Context(), userID, jobID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.CodeInternalError), "Failed to list reminders")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, reminders)
}

// Update godoc
// @Summary Update a reminder
// @Description Partially update a reminder (message, remind_at, and/or is_done)
// @Tags reminders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Reminder ID"
// @Param request body model.UpdateReminderRequest true "Fields to update"
// @Success 200 {object} model.ReminderDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Reminder not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /reminders/{id} [patch]
func (h *ReminderHandler) Update(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	reminderID := c.Param("id")

	var req model.UpdateReminderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(model.CodeValidationError), "Invalid request payload")
		return
	}

	reminder, err := h.service.Update(c.Request.Context(), userID, reminderID, &req)
	if err != nil {
		h.respondError(c, err, "Failed to update reminder")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, reminder)
}

// Delete godoc
// @Summary Delete a reminder
// @Description Delete a reminder by ID
// @Tags reminders
// @Security BearerAuth
// @Produce json
// @Param id path string true "Reminder ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Reminder not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /reminders/{id} [delete]
func (h *ReminderHandler) Delete(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	reminderID := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), userID, reminderID); err != nil {
		h.respondError(c, err, "Failed to delete reminder")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Reminder deleted successfully"})
}

// respondError maps domain errors to HTTP responses, defaulting to 500.
func (h *ReminderHandler) respondError(c *gin.Context, err error, fallbackMessage string) {
	statusCode := http.StatusInternalServerError
	errorCode := string(model.CodeInternalError)
	errorMessage := fallbackMessage

	switch {
	case errors.Is(err, model.ErrMessageRequired):
		statusCode = http.StatusBadRequest
		errorCode = string(model.CodeMessageRequired)
		errorMessage = "Message is required"
	case errors.Is(err, model.ErrJobNotFound):
		statusCode = http.StatusNotFound
		errorCode = string(model.CodeJobNotFound)
		errorMessage = "Job not found"
	case errors.Is(err, model.ErrReminderNotFound):
		statusCode = http.StatusNotFound
		errorCode = string(model.CodeReminderNotFound)
		errorMessage = "Reminder not found"
	}

	httpPlatform.RespondWithError(c, statusCode, errorCode, errorMessage)
}

func (h *ReminderHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	reminders := router.Group("/reminders")
	reminders.Use(authMiddleware)
	{
		reminders.POST("", h.Create)
		reminders.GET("", h.List)
		reminders.PATCH("/:id", h.Update)
		reminders.DELETE("/:id", h.Delete)
	}

	// Reminders nested under a job (the /jobs group itself lives in the jobs handler)
	jobs := router.Group("/jobs")
	jobs.Use(authMiddleware)
	{
		jobs.GET("/:id/reminders", h.ListByJob)
	}
}
