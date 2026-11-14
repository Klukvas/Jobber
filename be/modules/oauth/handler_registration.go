package oauth

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// Registration has its own RFC 7591 error vocabulary, separate from token errors.
func registrationError(c *gin.Context, err error) {
	code, description, status := "server_error", "Unable to register client", 500
	switch {
	case errors.Is(err, ErrInvalidRedirectURI):
		code, description, status = "invalid_redirect_uri", "Provide 1–10 valid redirect URIs", 400
	case errors.Is(err, ErrInvalidRequest):
		code, description, status = "invalid_client_metadata", "Invalid public client metadata", 400
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(status, gin.H{"error": code, "error_description": description})
}
