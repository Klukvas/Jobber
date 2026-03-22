package handler

import (
	"net/http"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	authModel "github.com/andreypavlenko/jobber/modules/auth/model"
	"github.com/andreypavlenko/jobber/modules/auth/service"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authService   *service.AuthService
	cookieCfg     auth.CookieConfig
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService, cookieCfg auth.CookieConfig, accessExpiry, refreshExpiry time.Duration) *AuthHandler {
	return &AuthHandler{
		authService:   authService,
		cookieCfg:     cookieCfg,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// LoginResponse represents the login response
type LoginResponse struct {
	User   *userModel.UserDTO    `json:"user"`
	Tokens *authModel.AuthTokens `json:"tokens"`
}

// Register godoc
// @Summary Register a new user
// @Description Create a new user account with email and password. A verification email will be sent.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body authModel.RegisterRequest true "Registration request"
// @Success 202 {object} service.RegisterResponse
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

	resp, err := h.authService.Register(c.Request.Context(), &req)
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

	httpPlatform.RespondWithData(c, http.StatusAccepted, resp)
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
// @Failure 403 {object} httpPlatform.ErrorResponse "Email not verified"
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
		if errorCode == userModel.CodeEmailNotVerified {
			statusCode = http.StatusForbidden
		} else if errorCode != userModel.CodeInvalidCredentials {
			statusCode = http.StatusInternalServerError
		}

		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	auth.SetTokenCookies(c, h.cookieCfg, tokens.AccessToken, h.accessExpiry, tokens.RefreshToken, h.refreshExpiry)

	httpPlatform.RespondWithData(c, http.StatusOK, LoginResponse{
		User:   user,
		Tokens: tokens,
	})
}

// VerifyEmail godoc
// @Summary Verify email address
// @Description Verify a user's email address using the 6-digit code from the verification email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body verifyEmailRequest true "Email and verification code"
// @Success 200 {object} map[string]string
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 429 {object} httpPlatform.ErrorResponse "Too many attempts"
// @Router /auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req verifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(userModel.CodeValidationError), "Invalid request payload")
		return
	}

	if err := h.authService.VerifyEmail(c.Request.Context(), req.Email, req.Code); err != nil {
		errorCode := userModel.GetErrorCode(err)
		errorMessage := userModel.GetErrorMessage(err)

		statusCode := http.StatusBadRequest
		if errorCode == userModel.CodeTooManyAttempts {
			statusCode = http.StatusTooManyRequests
		}

		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Email verified successfully"})
}

// ResendVerification godoc
// @Summary Resend verification email
// @Description Resend the verification email to the user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body resendVerificationRequest true "Email address"
// @Success 202 {object} map[string]string
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Router /auth/resend-verification [post]
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	var req resendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(userModel.CodeValidationError), "Invalid request payload")
		return
	}

	_ = h.authService.ResendVerification(c.Request.Context(), req.Email)

	httpPlatform.RespondWithData(c, http.StatusAccepted, gin.H{"message": "If your email is registered, you will receive a verification code"})
}

// ForgotPassword godoc
// @Summary Request password reset
// @Description Send a password reset code to the user's email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body forgotPasswordRequest true "Email address"
// @Success 202 {object} map[string]string
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Router /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req forgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(userModel.CodeValidationError), "Invalid request payload")
		return
	}

	_ = h.authService.ForgotPassword(c.Request.Context(), req.Email)

	httpPlatform.RespondWithData(c, http.StatusAccepted, gin.H{"message": "If your email is registered, you will receive a password reset code"})
}

// ResetPassword godoc
// @Summary Reset password
// @Description Reset the user's password using the 6-digit code from the reset email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body resetPasswordRequest true "Email, reset code, and new password"
// @Success 200 {object} map[string]string
// @Failure 400 {object} httpPlatform.ErrorResponse
// @Failure 429 {object} httpPlatform.ErrorResponse "Too many attempts"
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpPlatform.RespondWithError(c, http.StatusBadRequest, string(userModel.CodeValidationError), "Invalid request payload")
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req.Email, req.Code, req.Password); err != nil {
		errorCode := userModel.GetErrorCode(err)
		errorMessage := userModel.GetErrorMessage(err)

		statusCode := http.StatusBadRequest
		if errorCode == userModel.CodeInternalError {
			statusCode = http.StatusInternalServerError
		} else if errorCode == userModel.CodeTooManyAttempts {
			statusCode = http.StatusTooManyRequests
		}

		httpPlatform.RespondWithError(c, statusCode, string(errorCode), errorMessage)
		return
	}

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Password reset successfully"})
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
	// Read refresh token from cookie first, fall back to request body.
	refreshToken, _ := c.Cookie(auth.RefreshTokenCookie)
	if refreshToken == "" {
		var req authModel.RefreshRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			httpPlatform.RespondWithError(c, http.StatusBadRequest, string(userModel.CodeValidationError), "Invalid request payload")
			return
		}
		refreshToken = req.RefreshToken
	}

	tokens, err := h.authService.RefreshTokens(c.Request.Context(), refreshToken)
	if err != nil {
		httpPlatform.RespondWithError(c, http.StatusUnauthorized, string(userModel.CodeUnauthorized), "Invalid or expired refresh token")
		return
	}

	auth.SetTokenCookies(c, h.cookieCfg, tokens.AccessToken, h.accessExpiry, tokens.RefreshToken, h.refreshExpiry)

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

	auth.ClearTokenCookies(c, h.cookieCfg)

	httpPlatform.RespondWithData(c, http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// AuthRouteConfig holds middleware for auth route registration.
type AuthRouteConfig struct {
	AuthMiddleware    gin.HandlerFunc
	RateLimiter       gin.HandlerFunc
	EmailRateLimiter  gin.HandlerFunc // stricter limiter for email-sending endpoints
	CodeRateLimiter   gin.HandlerFunc // stricter limiter for code verification endpoints
}

// RegisterRoutes registers auth routes.
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup, cfg AuthRouteConfig) {
	authGroup := router.Group("/auth")

	withRL := func(handlers ...gin.HandlerFunc) []gin.HandlerFunc {
		if cfg.RateLimiter != nil {
			return append([]gin.HandlerFunc{cfg.RateLimiter}, handlers...)
		}
		return handlers
	}

	withEmailRL := func(handlers ...gin.HandlerFunc) []gin.HandlerFunc {
		var mw []gin.HandlerFunc
		if cfg.RateLimiter != nil {
			mw = append(mw, cfg.RateLimiter)
		}
		if cfg.EmailRateLimiter != nil {
			mw = append(mw, cfg.EmailRateLimiter)
		}
		return append(mw, handlers...)
	}

	withCodeRL := func(handlers ...gin.HandlerFunc) []gin.HandlerFunc {
		var mw []gin.HandlerFunc
		if cfg.RateLimiter != nil {
			mw = append(mw, cfg.RateLimiter)
		}
		if cfg.CodeRateLimiter != nil {
			mw = append(mw, cfg.CodeRateLimiter)
		}
		return append(mw, handlers...)
	}

	authGroup.POST("/register", withRL(h.Register)...)
	authGroup.POST("/login", withRL(h.Login)...)
	authGroup.POST("/refresh", withRL(h.Refresh)...)
	authGroup.POST("/verify-email", withCodeRL(h.VerifyEmail)...)
	authGroup.POST("/resend-verification", withEmailRL(h.ResendVerification)...)
	authGroup.POST("/forgot-password", withEmailRL(h.ForgotPassword)...)
	authGroup.POST("/reset-password", withCodeRL(h.ResetPassword)...)
	authGroup.POST("/logout", cfg.AuthMiddleware, h.Logout)
}

// Request DTOs

type verifyEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required,len=6"`
}

type resendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type forgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Email    string `json:"email"     binding:"required,email"`
	Code     string `json:"code"      binding:"required,len=6"`
	Password string `json:"password"  binding:"required,min=8,max=72"`
}
