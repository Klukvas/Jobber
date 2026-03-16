//go:build integration

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// helper: creates a content library item via API and returns its ID.
func createContentLibraryItem(t *testing.T, token string, title, content string) string {
	t.Helper()
	resp := doRequest(t, http.MethodPost, "/api/v1/content-library", map[string]string{
		"title":   title,
		"content": content,
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	body := parseJSON[map[string]interface{}](t, resp)
	return body["id"].(string)
}

func TestIntegrationCreateContentLibraryItem(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cl-create@test.com")

	resp := doRequest(t, http.MethodPost, "/api/v1/content-library", map[string]string{
		"title":    "Achievement",
		"content":  "Increased revenue by 30%",
		"category": "achievements",
	}, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "Achievement", body["title"])
	assert.Equal(t, "Increased revenue by 30%", body["content"])
}

func TestIntegrationListContentLibrary(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cl-list@test.com")

	createContentLibraryItem(t, token, "Item 1", "Content 1")
	createContentLibraryItem(t, token, "Item 2", "Content 2")

	resp := doRequest(t, http.MethodGet, "/api/v1/content-library", nil, token)
	assertStatus(t, resp, http.StatusOK)

	body := parseJSON[[]interface{}](t, resp)
	assert.Len(t, body, 2)
}

func TestIntegrationUpdateContentLibraryItem(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cl-update@test.com")

	itemID := createContentLibraryItem(t, token, "Original", "Original Content")

	newTitle := "Updated"
	resp := doRequest(t, http.MethodPatch, "/api/v1/content-library/"+itemID, map[string]interface{}{
		"title": newTitle,
	}, token)
	assertStatus(t, resp, http.StatusOK)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, newTitle, body["title"])
}

func TestIntegrationDeleteContentLibraryItem(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cl-delete@test.com")

	itemID := createContentLibraryItem(t, token, "Delete Me", "Content")

	resp := doRequest(t, http.MethodDelete, "/api/v1/content-library/"+itemID, nil, token)
	assertStatus(t, resp, http.StatusOK) // content library returns 200 with message
	resp.Body.Close()
}

func TestIntegrationContentLibraryOwnership(t *testing.T) {
	cleanupAll(t)
	_, tokenA := setupProUser(t, "cl-ownerA@test.com")
	_, tokenB := setupProUser(t, "cl-ownerB@test.com")

	itemID := createContentLibraryItem(t, tokenA, "Owner A's Item", "Content")

	// User B tries to update — should fail
	resp := doRequest(t, http.MethodPatch, "/api/v1/content-library/"+itemID, map[string]interface{}{
		"title": "Hacked",
	}, tokenB)
	// Content library handler returns 404 for ownership failures
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	// User B tries to delete
	resp = doRequest(t, http.MethodDelete, "/api/v1/content-library/"+itemID, nil, tokenB)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationContentLibraryNotFound(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cl-notfound@test.com")

	resp := doRequest(t, http.MethodPatch, "/api/v1/content-library/00000000-0000-0000-0000-000000000000", map[string]interface{}{
		"title": "Nope",
	}, token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}
