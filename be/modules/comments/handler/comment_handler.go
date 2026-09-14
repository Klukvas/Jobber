package handler

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/andreypavlenko/jobber/modules/comments/service"
	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	service *service.CommentService
}

func NewCommentHandler(service *service.CommentService) *CommentHandler {
	return &CommentHandler{service: service}
}

// Create godoc
// @Summary Create a new comment
// @Description Create a comment for a job or a specific stage
// @Tags comments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body model.CreateCommentRequest true "Comment details"
// @Success 201 {object} model.CommentDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /comments [post]
func (h *CommentHandler) Create(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	var req model.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	comment, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := string(model.CodeInternalError)
		errorMessage := "Failed to create comment"

		switch {
		case errors.Is(err, model.ErrContentRequired):
			statusCode = http.StatusBadRequest
			errorCode = string(model.CodeContentRequired)
			errorMessage = "Content is required"
		case errors.Is(err, model.ErrJobNotFound):
			statusCode = http.StatusNotFound
			errorCode = string(model.CodeJobNotFound)
			errorMessage = "Job not found"
		}

		httpPlatform.RespondWithError(c, statusCode, errorCode, errorMessage)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusCreated, comment)
}

// ListByJob godoc
// @Summary List comments by job
// @Description Get all comments for a specific job
// @Tags comments
// @Security BearerAuth
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} []model.CommentDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /jobs/{id}/comments [get]
func (h *CommentHandler) ListByJob(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	jobID := c.Param("id")

	comments, err := h.service.ListByJob(c.Request.Context(), jobID, userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.CodeInternalError), "Failed to list comments")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, comments)
}

// Update godoc
// @Summary Edit a comment
// @Description Rewrite the body of a comment the authenticated user authored
// @Tags comments
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "Comment ID"
// @Param request body model.UpdateCommentRequest true "New comment body"
// @Success 200 {object} model.CommentDTO
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Comment not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /comments/{id} [patch]
func (h *CommentHandler) Update(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	commentID := c.Param("id")

	var req model.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// The same answer Create gives to the same mistake. Reporting a body
		// that is not JSON at all as CONTENT_REQUIRED pointed the client at a
		// field that was never the problem, and left the two write endpoints
		// of one resource disagreeing about what a malformed request is.
		// A body that *is* JSON but carries no content still reaches the
		// service, which answers with CONTENT_REQUIRED.
		httpPlatform.RespondWithError(c, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request payload")
		return
	}

	comment, err := h.service.Update(c.Request.Context(), userID, commentID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := string(model.CodeInternalError)
		errorMessage := "Failed to update comment"

		switch {
		case errors.Is(err, model.ErrContentRequired):
			statusCode = http.StatusBadRequest
			errorCode = string(model.CodeContentRequired)
			errorMessage = "Content is required"
		case errors.Is(err, model.ErrCommentNotFound):
			// Also the answer for another user's comment id: the update is
			// scoped to the author, so a foreign id matches nothing — and the
			// response must not confirm that the comment exists.
			statusCode = http.StatusNotFound
			errorCode = string(model.CodeCommentNotFound)
			errorMessage = "Comment not found"
		}

		httpPlatform.RespondWithError(c, statusCode, errorCode, errorMessage)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, comment)
}

// Delete godoc
// @Summary Delete a comment
// @Description Delete a specific comment by ID
// @Tags comments
// @Security BearerAuth
// @Produce json
// @Param id path string true "Comment ID"
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "Comment not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /comments/{id} [delete]
func (h *CommentHandler) Delete(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	commentID := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), userID, commentID); err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := string(model.CodeInternalError)
		errorMessage := "Failed to delete comment"

		if errors.Is(err, model.ErrCommentNotFound) {
			statusCode = http.StatusNotFound
			errorCode = string(model.CodeCommentNotFound)
			errorMessage = "Comment not found"
		}

		httpPlatform.RespondWithError(c, statusCode, errorCode, errorMessage)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Comment deleted successfully"})
}

func (h *CommentHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	comments := router.Group("/comments")
	comments.Use(authMiddleware)
	{
		comments.POST("", h.Create)
		comments.PATCH("/:id", h.Update)
		comments.DELETE("/:id", h.Delete)
	}

	// Comments for jobs (nested route; the /jobs group itself lives in the jobs handler)
	jobs := router.Group("/jobs")
	jobs.Use(authMiddleware)
	{
		jobs.GET("/:id/comments", h.ListByJob)
	}
}
