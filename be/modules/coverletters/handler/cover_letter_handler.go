package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/andreypavlenko/jobber/modules/coverletters/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
)

// CoverLetterHandler handles cover letter HTTP requests.
type CoverLetterHandler struct {
	service *service.CoverLetterService
}

// NewCoverLetterHandler creates a new CoverLetterHandler.
func NewCoverLetterHandler(service *service.CoverLetterService) *CoverLetterHandler {
	return &CoverLetterHandler{service: service}
}

// RegisterRoutes registers cover letter routes.
func (h *CoverLetterHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := router.Group("/cover-letters")
	{
		group.POST("", authMiddleware, h.Create)
		group.GET("", authMiddleware, h.List)
		group.GET("/:id", authMiddleware, h.Get)
		group.PATCH("/:id", authMiddleware, h.Update)
		group.DELETE("/:id", authMiddleware, h.Delete)
		group.POST("/:id/duplicate", authMiddleware, h.Duplicate)
	}
}

// Create creates a new cover letter.
func (h *CoverLetterHandler) Create(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req model.CreateCoverLetterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	letter, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, letter)
}

// List returns all cover letters for the user.
func (h *CoverLetterHandler) List(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	letters, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to list cover letters")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, letters)
}

// Get returns a single cover letter.
func (h *CoverLetterHandler) Get(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	letter, err := h.service.Get(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, letter)
}

// Update updates a cover letter.
func (h *CoverLetterHandler) Update(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	var req model.UpdateCoverLetterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	letter, err := h.service.Update(c.Request.Context(), userID, c.Param("id"), &req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, letter)
}

// Delete deletes a cover letter.
func (h *CoverLetterHandler) Delete(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	if err := h.service.Delete(c.Request.Context(), userID, c.Param("id")); err != nil {
		h.handleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Duplicate creates a copy of a cover letter.
func (h *CoverLetterHandler) Duplicate(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}

	letter, err := h.service.Duplicate(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, letter)
}

func (h *CoverLetterHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, subModel.ErrLimitReached):
		httpPlatform.RespondWithError(c, http.StatusForbidden, "PLAN_LIMIT_REACHED", "You have reached the cover letter limit for your current plan.")
	case errors.Is(err, model.ErrNotAuthorized):
		httpPlatform.RespondWithError(c, http.StatusForbidden, "NOT_AUTHORIZED", "You don't have access to this cover letter")
	case errors.Is(err, model.ErrCoverLetterNotFound):
		httpPlatform.RespondWithError(c, http.StatusNotFound, "COVER_LETTER_NOT_FOUND", "Cover letter not found")
	case errors.Is(err, model.ErrInvalidFont),
		errors.Is(err, model.ErrInvalidColor),
		errors.Is(err, model.ErrInvalidFontSize),
		strings.Contains(err.Error(), "invalid font family"),
		strings.Contains(err.Error(), "invalid color format"):
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	default:
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}
}
