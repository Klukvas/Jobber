//go:build integration

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationRegister(t *testing.T) {
	cleanupAll(t)

	resp := doRequest(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "new@example.com",
		"password": "securepass123",
	}, "")
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	user := body["user"].(map[string]interface{})
	tokens := body["tokens"].(map[string]interface{})

	assert.Equal(t, "new@example.com", user["email"])
	assert.NotEmpty(t, user["id"])
	assert.NotEmpty(t, tokens["access_token"])
	assert.NotEmpty(t, tokens["refresh_token"])
}

func TestIntegrationRegisterDuplicateEmail(t *testing.T) {
	cleanupAll(t)

	payload := map[string]string{
		"email":    "dup@example.com",
		"password": "securepass123",
	}

	resp := doRequest(t, http.MethodPost, "/api/v1/auth/register", payload, "")
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = doRequest(t, http.MethodPost, "/api/v1/auth/register", payload, "")
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationRegisterInvalidPayload(t *testing.T) {
	resp := doRequest(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email": "not-an-email",
	}, "")
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationLogin(t *testing.T) {
	cleanupAll(t)

	// Register first
	resp := doRequest(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "login@example.com",
		"password": "securepass123",
	}, "")
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Login
	resp = doRequest(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "login@example.com",
		"password": "securepass123",
	}, "")
	assertStatus(t, resp, http.StatusOK)

	body := parseJSON[map[string]interface{}](t, resp)
	tokens := body["tokens"].(map[string]interface{})
	assert.NotEmpty(t, tokens["access_token"])
	assert.NotEmpty(t, tokens["refresh_token"])
}

func TestIntegrationLoginWrongPassword(t *testing.T) {
	cleanupAll(t)

	resp := doRequest(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "wrongpw@example.com",
		"password": "securepass123",
	}, "")
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = doRequest(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "wrongpw@example.com",
		"password": "wrongpassword",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationLoginNonexistentUser(t *testing.T) {
	cleanupAll(t)

	resp := doRequest(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "nobody@example.com",
		"password": "doesntmatter",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationRefreshToken(t *testing.T) {
	cleanupAll(t)

	// Register to get tokens
	resp := doRequest(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "refresh@example.com",
		"password": "securepass123",
	}, "")
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	tokens := body["tokens"].(map[string]interface{})
	refreshToken := tokens["refresh_token"].(string)

	// Refresh
	resp = doRequest(t, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, "")
	assertStatus(t, resp, http.StatusOK)

	newTokens := parseJSON[map[string]interface{}](t, resp)
	assert.NotEmpty(t, newTokens["access_token"])
}

func TestIntegrationRefreshInvalidToken(t *testing.T) {
	resp := doRequest(t, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": "invalid-token-string",
	}, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationLogout(t *testing.T) {
	cleanupAll(t)

	// Register to get tokens
	resp := doRequest(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "logout@example.com",
		"password": "securepass123",
	}, "")
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	tokens := body["tokens"].(map[string]interface{})
	accessToken := "Bearer " + tokens["access_token"].(string)

	// Logout with valid token
	resp = doRequest(t, http.MethodPost, "/api/v1/auth/logout", nil, accessToken)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestIntegrationLogoutWithoutAuth(t *testing.T) {
	// Logout without token should return 401
	resp := doRequest(t, http.MethodPost, "/api/v1/auth/logout", nil, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationProtectedEndpointWithoutAuth(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/api/v1/resume-builder", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}
