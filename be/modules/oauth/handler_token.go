package oauth

import (
	"errors"
	"mime"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

func formValues(c *gin.Context) (url.Values, error) {
	content, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || content != "application/x-www-form-urlencoded" {
		return nil, ErrInvalidRequest
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	if err := c.Request.ParseForm(); err != nil {
		return nil, ErrInvalidRequest
	}
	values := c.Request.PostForm
	for _, entries := range values {
		if len(entries) != 1 {
			return nil, ErrInvalidRequest
		}
	}
	if values.Get("client_secret") != "" || c.GetHeader("Authorization") != "" {
		return nil, ErrInvalidClient
	}
	return values, nil
}

func (h *Handler) Token(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
	values, err := formValues(c)
	if err != nil {
		oauthError(c, err)
		return
	}
	credential := values.Get("code")
	if values.Get("grant_type") == "refresh_token" {
		credential = values.Get("refresh_token")
	}
	response, err := h.service.Token(c.Request.Context(), ExchangeRequest{
		Kind: values.Get("grant_type"), ClientID: values.Get("client_id"), Credential: credential,
		RedirectURI: values.Get("redirect_uri"), Verifier: values.Get("code_verifier"), Resource: values.Get("resource"), Scope: values.Get("scope"),
	})
	if err != nil {
		oauthError(c, err)
		return
	}
	c.JSON(200, response)
}

func (h *Handler) Revoke(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	values, err := formValues(c)
	if errors.Is(err, ErrInvalidClient) {
		oauthError(c, err)
		return
	}
	if err == nil {
		if revokeErr := h.service.Revoke(c.Request.Context(), values.Get("token"), values.Get("client_id")); revokeErr != nil {
			h.logger.Warn("OAuth revocation unavailable")
		}
	}
	// RFC 7009 deliberately hides whether the credential exists.
	c.Status(200)
}

func oauthError(c *gin.Context, err error) {
	code, status := "server_error", 500
	switch {
	case errors.Is(err, ErrInvalidRequest):
		code, status = "invalid_request", 400
	case errors.Is(err, ErrInvalidClient):
		code, status = "invalid_client", 400
	case errors.Is(err, ErrInvalidGrant):
		code, status = "invalid_grant", 400
	case errors.Is(err, ErrUnsupportedGrant):
		code, status = "unsupported_grant_type", 400
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(status, gin.H{"error": code})
}
