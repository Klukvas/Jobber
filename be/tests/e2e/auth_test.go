//go:build integration

package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registerUser registers a user through the API (202 + verification email flow).
func registerUser(t *testing.T, email, password string) {
	t.Helper()
	resp := doRequest(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	assertStatus(t, resp, http.StatusAccepted)
	resp.Body.Close()
}

// verifyUserEmail marks a user's email as verified directly in the DB
// (the verification code is only delivered by email in production).
func verifyUserEmail(t *testing.T, email string) {
	t.Helper()
	tag, err := pool.Exec(context.Background(),
		"UPDATE users SET email_verified = true WHERE email = $1", email)
	require.NoError(t, err)
	require.EqualValues(t, 1, tag.RowsAffected())
}

// loginUser logs in and returns the response body (user + tokens).
func loginUser(t *testing.T, email, password string) map[string]interface{} {
	t.Helper()
	resp := doRequest(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, "")
	assertStatus(t, resp, http.StatusOK)
	return parseJSON[map[string]interface{}](t, resp)
}

func TestIntegrationRegister(t *testing.T) {
	cleanupAll(t)

	registerUser(t, "new@example.com", "securepass123")

	// The account exists but is not verified yet — login must be rejected.
	resp := doRequest(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"email":    "new@example.com",
		"password": "securepass123",
	}, "")
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assertErrorCode(t, resp, "EMAIL_NOT_VERIFIED")
}

func TestIntegrationRegisterDuplicateEmail(t *testing.T) {
	cleanupAll(t)

	registerUser(t, "dup@example.com", "securepass123")

	resp := doRequest(t, http.MethodPost, "/api/v1/auth/register", map[string]string{
		"email":    "dup@example.com",
		"password": "securepass123",
	}, "")
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

	registerUser(t, "login@example.com", "securepass123")
	verifyUserEmail(t, "login@example.com")

	body := loginUser(t, "login@example.com", "securepass123")

	user := body["user"].(map[string]interface{})
	tokens := body["tokens"].(map[string]interface{})
	assert.Equal(t, "login@example.com", user["email"])
	assert.NotEmpty(t, tokens["access_token"])
	assert.NotEmpty(t, tokens["refresh_token"])
}

func TestIntegrationLoginWrongPassword(t *testing.T) {
	cleanupAll(t)

	registerUser(t, "wrongpw@example.com", "securepass123")
	verifyUserEmail(t, "wrongpw@example.com")

	resp := doRequest(t, http.MethodPost, "/api/v1/auth/login", map[string]string{
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

	registerUser(t, "refresh@example.com", "securepass123")
	verifyUserEmail(t, "refresh@example.com")

	body := loginUser(t, "refresh@example.com", "securepass123")
	tokens := body["tokens"].(map[string]interface{})
	refreshToken := tokens["refresh_token"].(string)

	resp := doRequest(t, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, "")
	assertStatus(t, resp, http.StatusOK)

	newTokens := parseJSON[map[string]interface{}](t, resp)
	assert.NotEmpty(t, newTokens["access_token"])

	// Rotation: the old refresh token must be rejected on reuse.
	resp = doRequest(t, http.MethodPost, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
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

	registerUser(t, "logout@example.com", "securepass123")
	verifyUserEmail(t, "logout@example.com")

	body := loginUser(t, "logout@example.com", "securepass123")
	tokens := body["tokens"].(map[string]interface{})
	accessToken := "Bearer " + tokens["access_token"].(string)

	resp := doRequest(t, http.MethodPost, "/api/v1/auth/logout", nil, accessToken)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestIntegrationLogoutWithoutAuth(t *testing.T) {
	resp := doRequest(t, http.MethodPost, "/api/v1/auth/logout", nil, "")
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationProtectedEndpointWithoutAuth(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/api/v1/resume-builder", nil, "")
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()
}
