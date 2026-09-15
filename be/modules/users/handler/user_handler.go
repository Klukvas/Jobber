package handler

import (
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/andreypavlenko/jobber/modules/users/service"
	"github.com/gin-gonic/gin"
)

// UserHandler serves the caller's own account, read-only.
type UserHandler struct {
	service *service.UserService
}

// NewUserHandler creates a new user handler.
func NewUserHandler(service *service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// GetProfile godoc
// @Summary Get the signed-in user's profile
// @Description Returns the caller's own account details. The user is taken from the access token, so there is nothing to address but yourself.
// @Tags users
// @Security BearerAuth
// @Produce json
// @Success 200 {object} model.UserDTO
// @Failure 401 {object} httpPlatform.ErrorResponse
// @Failure 404 {object} httpPlatform.ErrorResponse "User not found"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := auth.MustGetUserID(c)
	if !ok {
		return
	}

	profile, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		respondWithUserError(c, err)
		return
	}
	httpPlatform.RespondWithData(c, http.StatusOK, profile)
}

// respondWithUserError maps a service error onto a status, keeping the
// internal detail out of the response body.
func respondWithUserError(c *gin.Context, err error) {
	code := model.GetErrorCode(err)
	status := http.StatusInternalServerError
	if code == model.CodeUserNotFound {
		status = http.StatusNotFound
	}
	httpPlatform.RespondWithError(c, status, string(code), model.GetErrorMessage(err))
}

// RegisterRoutes registers the profile route behind the auth middleware: there
// is no anonymous view of an account, and nothing here writes one.
func (h *UserHandler) RegisterRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	profile := router.Group("/profile")
	profile.Use(authMiddleware)
	{
		profile.GET("", h.GetProfile)
	}
}
