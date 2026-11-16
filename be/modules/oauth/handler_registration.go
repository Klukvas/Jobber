package oauth

import (
	"errors"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

// logRegistrationReject records why a client failed to register: the response
// carries only a generic description, so the log is the one place to see it.
// Client metadata is public, but every field is capped so a caller cannot
// flood the log.
func (h *Handler) logRegistrationReject(c *gin.Context, request Client, err error) {
	fields := []zap.Field{
		zap.String("reason", truncate(err.Error(), 300)),
		zap.String("user_agent", truncate(c.GetHeader("User-Agent"), 200)),
		zap.String("client_name", truncate(request.Name, 100)),
		zap.Strings("redirect_uris", truncateAll(request.RedirectURIs, 10, 200)),
		zap.Strings("grant_types", truncateAll(request.GrantTypes, 10, 100)),
		zap.Strings("response_types", truncateAll(request.ResponseTypes, 10, 100)),
		zap.String("token_endpoint_auth_method", truncate(request.AuthMethod, 100)),
	}
	if errors.Is(err, ErrInvalidRequest) {
		h.logger.Warn("OAuth client registration rejected", fields...)
		return
	}
	h.logger.Error("OAuth client registration failed", fields...)
}

func truncate(value string, maxRunes int) string {
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	return string([]rune(value)[:maxRunes]) + "…"
}

func truncateAll(values []string, maxItems, maxRunes int) []string {
	limited := make([]string, 0, min(len(values), maxItems))
	for _, value := range values[:min(len(values), maxItems)] {
		limited = append(limited, truncate(value, maxRunes))
	}
	return limited
}
