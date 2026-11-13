package mcp

import (
	"net/http"
	"net/url"
	"strings"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/gin-gonic/gin"
)

func requireOrigin(allowedOrigins string) gin.HandlerFunc {
	allowed := strings.Split(allowedOrigins, ",")
	return func(c *gin.Context) {
		origins := c.Request.Header.Values("Origin")
		if len(origins) == 0 {
			c.Next()
			return
		}
		if len(origins) == 1 && originAllowed(origins[0], allowed, allowedOrigins == "*") {
			c.Next()
			return
		}
		httpPlatform.RespondWithError(c, http.StatusForbidden, "ORIGIN_NOT_ALLOWED", "This origin is not allowed to access AI assistant tools.")
		c.Abort()
	}
}

func originAllowed(origin string, allowed []string, allowAny bool) bool {
	parsed, err := url.Parse(origin)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil || origin != parsed.Scheme+"://"+parsed.Host {
		return false
	}
	// Match CORS's exact '*' configuration, which production validation rejects.
	if allowAny {
		return true
	}
	for _, candidate := range allowed {
		if strings.EqualFold(strings.TrimSpace(candidate), origin) {
			return true
		}
	}
	return false
}

type bearerChallengeWriter struct{ gin.ResponseWriter }

func (w bearerChallengeWriter) WriteHeader(status int) {
	if status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	w.ResponseWriter.WriteHeader(status)
}

// The SDK can reject a token that expires after gin authentication. Intercept
// its 401 too, before response headers are written.
func bearerChallenge() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer = bearerChallengeWriter{ResponseWriter: c.Writer}
		c.Next()
	}
}
