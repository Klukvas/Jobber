//go:build integration

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegrationFreePlanLimitOneBuilder(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "sub-free@test.com", "securepass123")
	seedSubscription(t, userID, "free")
	token := authToken(t, userID)

	// First builder — OK
	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
		"title": "Builder 1",
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	// Second builder — should be blocked
	resp = doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
		"title": "Builder 2",
	}, token)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assertErrorCode(t, resp, "PLAN_LIMIT_REACHED")
}

func TestIntegrationFreePlanNoCoverLetters(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "sub-freecl@test.com", "securepass123")
	seedSubscription(t, userID, "free")
	token := authToken(t, userID)

	resp := doRequest(t, http.MethodPost, "/api/v1/cover-letters", map[string]string{
		"title": "CL Attempt",
	}, token)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	assertErrorCode(t, resp, "PLAN_LIMIT_REACHED")
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

	// Create first builder (uses up the 1 free slot)
	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
		"title": "Builder 1",
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	body := parseJSON[map[string]interface{}](t, resp)
	rbID := body["id"].(string)

	// Verify limit reached
	resp = doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
		"title": "Builder 2",
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
