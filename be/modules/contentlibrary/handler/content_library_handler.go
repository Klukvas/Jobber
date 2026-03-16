package handler

import (
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	"github.com/andreypavlenko/jobber/modules/contentlibrary/model"
	"github.com/andreypavlenko/jobber/modules/contentlibrary/service"
	"github.com/gin-gonic/gin"
)

// ContentLibraryHandler handles content library HTTP requests.
type ContentLibraryHandler struct {
	service *service.ContentLibraryService
}

// NewContentLibraryHandler creates a new ContentLibraryHandler.
func NewContentLibraryHandler(service *service.ContentLibraryService) *ContentLibraryHandler {
	return &ContentLibraryHandler{service: service}
}

// RegisterRoutes registers content library routes.
func (h *ContentLibraryHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := router.Group("/content-library")
	{
		group.POST("", authMiddleware, h.Create)
		group.GET("", authMiddleware, h.List)
		group.PATCH("/:id", authMiddleware, h.Update)
		group.DELETE("/:id", authMiddleware, h.Delete)
	}
}

// Create creates a new content library entry.
func (h *ContentLibraryHandler) Create(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req model.CreateContentLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	entry, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create entry"})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

// List returns all content library entries for the user.
func (h *ContentLibraryHandler) List(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	entries, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list entries"})
		return
	}

	c.JSON(http.StatusOK, entries)
}

// Update updates a content library entry.
func (h *ContentLibraryHandler) Update(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")

	var req model.UpdateContentLibraryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	entry, err := h.service.Update(c.Request.Context(), userID, id, &req)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entry not found"})
		return
	}

	c.JSON(http.StatusOK, entry)
}

// Delete deletes a content library entry.
func (h *ContentLibraryHandler) Delete(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")

	if err := h.service.Delete(c.Request.Context(), userID, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entry not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
