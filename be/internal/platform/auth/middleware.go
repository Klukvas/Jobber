package auth

import (
	"strings"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates JWT access tokens.
// It checks the Authorization header first, then falls back to the httpOnly access_token cookie.
func AuthMiddleware(jwtManager *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 1. Try Authorization header (for API clients, mobile, etc.)
		if authHeader := c.GetHeader("Authorization"); authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// 2. Fall back to httpOnly cookie (for browser SPA)
		if tokenString == "" {
			if cookie, err := c.Cookie(AccessTokenCookie); err == nil && cookie != "" {
				tokenString = cookie
			}
		}

		if tokenString == "" {
			httpPlatform.RespondWithError(c, 401, "UNAUTHORIZED", "Authentication required")
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			httpPlatform.RespondWithError(c, 401, "UNAUTHORIZED", "Invalid or expired token")
			c.Abort()
			return
		}

		// Set user ID in context
		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}
	return userID.(string), true
}

// MustGetUserID extracts user ID from context and responds with 401 if not found.
// Returns the user ID and true if successful, or empty string and false if unauthorized.
func MustGetUserID(c *gin.Context) (string, bool) {
	userID, exists := GetUserID(c)
	if !exists {
		httpPlatform.RespondWithError(c, 401, "UNAUTHORIZED", "Unauthorized")
		c.Abort()
		return "", false
	}
	return userID, true
}
