package apitokens

import (
	"context"
	"net/http"
	"strings"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/apitokens/service"
	"github.com/gin-gonic/gin"
)

type Authenticator interface {
	Authenticate(ctx context.Context, plaintext string) (userID, tokenID string, err error)
}

// PATAuthMiddleware is reserved for the MCP surface, never the JWT API routes.
func PATAuthMiddleware(authenticator Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		scheme, token, found := strings.Cut(c.GetHeader("Authorization"), " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || !strings.HasPrefix(token, service.TokenPrefix) {
			rejectPAT(c)
			return
		}
		userID, tokenID, err := authenticator.Authenticate(c.Request.Context(), token)
		if err != nil || userID == "" || tokenID == "" {
			rejectPAT(c)
			return
		}
		c.Set("user_id", userID)
		c.Set("auth_via", "pat")
		c.Set("api_token_id", tokenID)
		c.Next()
	}
}

func rejectPAT(c *gin.Context) {
	httpPlatform.RespondWithError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
	c.Abort()
}
