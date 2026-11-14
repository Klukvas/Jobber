package oauth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ProtectedResource(c *gin.Context) {
	c.JSON(200, gin.H{"resource": h.service.Resource(), "authorization_servers": []string{h.service.issuer}, "scopes_supported": []string{"mcp"}, "bearer_methods_supported": []string{"header"}, "resource_name": "Cursopus"})
}

func (h *Handler) Metadata(c *gin.Context) {
	issuer := h.service.issuer
	c.JSON(200, gin.H{
		"issuer": issuer, "authorization_endpoint": issuer + "/oauth/authorize",
		"token_endpoint": issuer + "/api/oauth/token", "registration_endpoint": issuer + "/api/oauth/register", "revocation_endpoint": issuer + "/api/oauth/revoke",
		"response_types_supported": []string{"code"}, "grant_types_supported": []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported": []string{"S256"}, "token_endpoint_auth_methods_supported": []string{"none"}, "scopes_supported": []string{"mcp"},
		"revocation_endpoint_auth_methods_supported": []string{"none"},
		"client_id_metadata_document_supported":      true, "authorization_response_iss_parameter_supported": true,
	})
}

// PublicCORS must run before the application's credentialed CORS middleware so
// arbitrary browser connectors receive both normal responses and preflights.
func PublicCORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.URL.Path {
		case "/api/oauth/token", "/api/oauth/register", "/api/oauth/revoke", "/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/api/mcp", "/.well-known/oauth-authorization-server":
			c.Set("public_cors", true)
			c.Header("Access-Control-Allow-Origin", "*")
			c.Writer.Header().Del("Access-Control-Allow-Credentials")
			c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if c.Request.Method == http.MethodOptions {
				c.AbortWithStatus(204)
				return
			}
		}
		c.Next()
	}
}
