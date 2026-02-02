package handler

import (
	"net/http"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	authModel "github.com/andreypavlenko/jobber/modules/auth/model"
	"github.com/andreypavlenko/jobber/modules/auth/service"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterResponse represents the registration response
type RegisterResponse struct {
	User   *userModel.UserDTO      `json:"user"`
	Tokens *authModel.AuthTokens   `json:"tokens"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	User   *userModel.UserDTO      `json:"user"`
	Tokens *authModel.AuthTokens   `json:"tokens"`
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body authModel.RegisterRequest true "Registration request"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 409 {object} httpPlatform.ErrorResponse "User already exists"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req authModel.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(userModel.CodeValidationError), "Invalid request payload")
		return
	}

	user, tokens, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		errorCode := userModel.GetErrorCode(err)
		errorMessage := userModel.GetErrorMessage(err)
		
		statusCode := http.StatusInternalServerError
		if errorCode == userModel.CodeUserAlreadyExists {
			statusCode = http.StatusConflict
		} else if errorCode == userModel.CodeInvalidEmail || errorCode == userModel.CodeInvalidPassword {
			statusCode = http.StatusBadRequest
		}
		
		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusCreated, RegisterResponse{
		User:   user,
		Tokens: tokens,
	})
}

// Login godoc
// @Summary User login
// @Description Authenticate user and receive JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body authModel.LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse "Invalid credentials"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req authModel.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(userModel.CodeValidationError), "Invalid request payload")
		return
	}

	user, tokens, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		errorCode := userModel.GetErrorCode(err)
		errorMessage := userModel.GetErrorMessage(err)
		
		statusCode := http.StatusUnauthorized
		if errorCode != userModel.CodeInvalidCredentials {
			statusCode = http.StatusInternalServerError
		}
		
		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, LoginResponse{
		User:   user,
		Tokens: tokens,
	})
}

// Refresh godoc
// @Summary Refresh access token
// @Description Get a new access token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body authModel.RefreshRequest true "Refresh token"
// @Success 200 {object} authModel.AuthTokens
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 401 {object} httpPlatform.ErrorResponse "Invalid or expired refresh token"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req authModel.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(userModel.CodeValidationError), "Invalid request payload")
		return
	}

	tokens, err := h.authService.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, string(userModel.CodeUnauthorized), "Invalid or expired refresh token")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, tokens)
}

// Logout godoc
// @Summary User logout
// @Description Revoke all refresh tokens for the authenticated user
// @Tags auth
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 401 {object} httpPlatform.ErrorResponse "Unauthorized"
// @Failure 500 {object} httpPlatform.ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := auth.GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, string(userModel.CodeUnauthorized), "Unauthorized")
		return
	}

	if err := h.authService.Logout(c.Request.Context(), userID); err != nil {
		httpPlatform.RespondWithError(c, http.StatusInternalServerError, string(userModel.CodeInternalError), "Failed to logout")
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// RegisterRoutes registers auth routes
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/logout", h.Logout)
	}
}
