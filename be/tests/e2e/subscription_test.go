//go:build integration

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegrationFreePlanBuilderLimit(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "sub-free@test.com", "securepass123")
	seedSubscription(t, userID, "free")
	token := authToken(t, userID)

	// The two free builders — OK
	for i := 0; i < 2; i++ {
		resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
			"title": "Builder",
		}, token)
		assertStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}

	// Third builder — should be blocked
	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
		"title": "Builder 3",
	}, token)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assertErrorCode(t, resp, "PLAN_LIMIT_REACHED")
}

func TestIntegrationFreePlanCoverLetterLimit(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "sub-freecl@test.com", "securepass123")
	seedSubscription(t, userID, "free")
	token := authToken(t, userID)

	// Both config/plans.yaml and the hardcoded fallback (used here, without
	// plans.yaml) give Free one cover letter.
	resp := doRequest(t, http.MethodPost, "/api/v1/cover-letters", map[string]string{
		"title": "Free CL",
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	resp = doRequest(t, http.MethodPost, "/api/v1/cover-letters", map[string]string{
		"title": "CL Over Limit",
	}, token)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assertErrorCode(t, resp, "PLAN_LIMIT_REACHED")

	// The usage endpoint reports the same limit the create path enforced.
	resp = doRequest(t, http.MethodGet, "/api/v1/subscription", nil, token)
	assertStatus(t, resp, http.StatusOK)
	sub := parseJSON[struct {
		Limits struct {
			MaxCoverLetters int `json:"max_cover_letters"`
		} `json:"limits"`
	}](t, resp)
	assert.Equal(t, 1, sub.Limits.MaxCoverLetters)
}

func TestIntegrationProPlanMultipleBuilders(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "sub-pro@test.com")

	for i := 0; i < 5; i++ {
		resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
			"title": "Builder",
		}, token)
		assertStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}
}

func TestIntegrationProPlanCoverLetters(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "sub-procl@test.com")

	resp := doRequest(t, http.MethodPost, "/api/v1/cover-letters", map[string]string{
		"title": "Pro CL",
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()
}

func TestIntegrationLimitCheckAfterDelete(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "sub-delcheck@test.com", "securepass123")
	seedSubscription(t, userID, "free")
	token := authToken(t, userID)

	// Create two builders (uses up the free slots)
	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
		"title": "Builder 1",
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	body := parseJSON[map[string]interface{}](t, resp)
	rbID := body["id"].(string)

	resp = doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
		"title": "Builder 2",
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Verify limit reached
	resp = doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
		"title": "Builder 3",
	}, token)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// Delete the first builder
	resp = doRequest(t, http.MethodDelete, "/api/v1/resume-builder/"+rbID, nil, token)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Now creating another should work
	resp = doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
		"title": "Builder After Delete",
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()
}
