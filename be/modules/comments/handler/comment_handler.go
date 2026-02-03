package handler

import (
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
// @Description Create a comment for an application or a specific stage
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
		
		if err == model.ErrContentRequired {
			statusCode = http.StatusBadRequest
			errorCode = string(model.CodeContentRequired)
			errorMessage = "Content is required"
		}
		
		httpPlatform.RespondWithError(c, statusCode, errorCode, errorMessage)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusCreated, comment)
}

// ListByApplication godoc
// @Summary List comments by application
// @Description Get all comments for a specific application
// @Tags comments
// @Security BearerAuth
// @Produce json
// @Param id path string true "Application ID"
// @Success 200 {object} []model.CommentDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /applications/{id}/comments [get]
func (h *CommentHandler) ListByApplication(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	appID := c.Param("id")

	comments, err := h.service.ListByApplication(c.Request.Context(), appID, userID)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(model.CodeInternalError), "Failed to list comments")
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, comments)
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
		
		if err == model.ErrCommentNotFound {
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
		comments.DELETE("/:id", h.Delete)
	}
	
	// Comments for applications (nested route)
	apps := router.Group("/applications")
	apps.Use(authMiddleware)
	{
		apps.GET("/:id/comments", h.ListByApplication)
	}
}
