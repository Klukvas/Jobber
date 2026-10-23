package contacts

import (
	"errors"
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func respond(c *gin.Context, status int, value any, err error) {
	switch {
	case err == nil:
		httpPlatform.RespondWithData(c, status, value)
	case errors.Is(err, ErrNotFound):
		httpPlatform.RespondWithError(c, 404, "CONTACT_NOT_FOUND", "Contact or job not found")
	case errors.Is(err, ErrInvalid):
		httpPlatform.RespondWithError(c, 400, "CONTACT_INVALID", "Invalid contact fields")
	default:
		httpPlatform.RespondWithError(c, 500, "INTERNAL_ERROR", "Unable to save or load contacts")
	}
}
func (h *Handler) list(c *gin.Context) {
	user, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	value, err := h.service.List(c.Request.Context(), user, c.Param("id"))
	respond(c, 200, value, err)
}
func (h *Handler) save(c *gin.Context) {
	user, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	var fields Fields
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<10)
	if err := c.ShouldBindJSON(&fields); err != nil {
		respond(c, 400, nil, ErrInvalid)
		return
	}
	value, err := h.service.Save(c.Request.Context(), user, c.Param("id"), c.Param("contactId"), fields)
	status := 200
	if c.Request.Method == http.MethodPost {
		status = 201
	}
	respond(c, status, value, err)
}
func (h *Handler) delete(c *gin.Context) {
	user, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}
	err := h.service.Delete(c.Request.Context(), user, c.Param("id"), c.Param("contactId"))
	if err != nil {
		respond(c, 200, nil, err)
		return
	}
	c.Status(http.StatusNoContent)
}
func (h *Handler) RegisterRoutes(router *gin.RouterGroup, middleware gin.HandlerFunc) {
	group := router.Group("/jobs/:id/contacts", middleware)
	group.GET("", h.list)
	group.POST("", h.save)
	group.PATCH("/:contactId", h.save)
	group.DELETE("/:contactId", h.delete)
}
