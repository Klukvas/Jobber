//go:build integration

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationHealthCheck(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/health", nil, "")
	assertStatus(t, resp, http.StatusOK)

	var body struct {
		Status   string            `json:"status"`
		Version  string            `json:"version"`
		Services map[string]string `json:"services"`
	}
	body = parseJSON[struct {
		Status   string            `json:"status"`
		Version  string            `json:"version"`
		Services map[string]string `json:"services"`
	}](t, resp)

	assert.Equal(t, "healthy", body.Status)
	assert.Equal(t, "up", body.Services["postgres"])
	assert.Equal(t, "up", body.Services["redis"])
}

func TestIntegrationPing(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/ping", nil, "")
	assertStatus(t, resp, http.StatusOK)

	body := parseJSON[map[string]string](t, resp)
	require.Equal(t, "pong", body["message"])
}

func TestIntegrationSessionCheck(t *testing.T) {
	t.Run("unauthenticated returns 401", func(t *testing.T) {
		resp := doRequest(t, http.MethodGet, "/api/v1/session", nil, "")
		assertStatus(t, resp, http.StatusUnauthorized)
	})

	t.Run("authenticated returns 200", func(t *testing.T) {
		userID := seedUser(t, "session-check@example.com", "password123")
		token := authToken(t, userID)

		resp := doRequest(t, http.MethodGet, "/api/v1/session", nil, token)
		assertStatus(t, resp, http.StatusOK)

		body := parseJSON[map[string]string](t, resp)
		require.Equal(t, "ok", body["status"])
	})
}

func TestIntegrationNotFound(t *testing.T) {
	resp := doRequest(t, http.MethodGet, "/api/v1/nonexistent", nil, "")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}
