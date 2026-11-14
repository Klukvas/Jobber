package oauth

import (
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// Remove the authorization query and form/JSON credentials from Sentry's
// request scope, including failed requests and panic events.
func redactRequest(c *gin.Context) {
	if hub := sentry.GetHubFromContext(c.Request.Context()); hub != nil {
		hub.Scope().SetRequest(nil)
		hub.Scope().SetRequestBody(nil)
	}
	c.Next()
}

func Privacy() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/oauth/") || strings.HasPrefix(c.Request.URL.Path, "/api/v1/oauth/") {
			redactRequest(c)
			return
		}
		c.Next()
	}
}
