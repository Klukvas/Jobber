//go:build integration

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests guard the regression from the single-pipeline finalize: migration
// 041 dropped jobs.status, but count queries in the subscriptions and companies
// modules still referenced it and 500'd once the column was gone. They run
// against the real migrated (v41) schema, so a future column change that breaks
// one of these counts fails here instead of in production.

// TestIntegrationJobCreateExercisesLimitCount proves the plan-limit count query
// (subscriptions CountUserJobs, run as Create's FIRST step) works against the
// finalized schema — this is exactly the query whose stale `status != 'archived'`
// made every job create return 500.
func TestIntegrationJobCreateExercisesLimitCount(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "counts-create@test.com", "securepass123")
	seedSubscription(t, userID, "free")
	token := authToken(t, userID)
	seedPipeline(t, userID, "Wishlist", "Applied")

	resp := doRequest(t, http.MethodPost, "/api/v1/jobs",
		map[string]interface{}{"title": "First job"}, token)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()
}

// TestIntegrationSubscriptionJobUsageExcludesArchived hits GET /subscription,
// which drives subscriptions GetAllCounts — the second query that referenced the
// dropped status column. It also pins the is_archived semantics: an archived card
// must not consume the job usage/limit.
func TestIntegrationSubscriptionJobUsageExcludesArchived(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "counts-usage@test.com", "securepass123")
	seedSubscription(t, userID, "free")
	token := authToken(t, userID)
	cols := seedPipeline(t, userID, "Wishlist", "Applied")

	archived := createTestJob(t, token, "Job A", cols["Applied"], "")
	createTestJob(t, token, "Job B", cols["Applied"], "")
	createTestJob(t, token, "Job C", cols["Wishlist"], "")
	archiveJob(t, token, archived["id"].(string))

	resp := doRequest(t, http.MethodGet, "/api/v1/subscription", nil, token)
	assertStatus(t, resp, http.StatusOK)
	sub := parseJSON[struct {
		Limits struct {
			MaxJobs int `json:"max_jobs"`
		} `json:"limits"`
		Usage struct {
			Jobs int `json:"jobs"`
		} `json:"usage"`
	}](t, resp)

	assert.Equal(t, 2, sub.Usage.Jobs, "archived card must not count toward job usage")
	assert.Equal(t, 25, sub.Limits.MaxJobs, "free plan job limit")
}

// TestIntegrationCompanyApplicationCounts hits GET /companies, which drives the
// per-company FILTER counts (applications_count / active_applications_count) —
// the last two queries that referenced jobs.status='applied'. applications_count
// is applied cards (applied_at set); active is applied AND not archived.
func TestIntegrationCompanyApplicationCounts(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "counts-company@test.com", "securepass123")
	seedSubscription(t, userID, "free")
	token := authToken(t, userID)
	cols := seedPipeline(t, userID, "Wishlist", "Applied")

	resp := doRequest(t, http.MethodPost, "/api/v1/companies",
		map[string]interface{}{"name": "Acme"}, token)
	assertStatus(t, resp, http.StatusCreated)
	company := parseJSON[map[string]interface{}](t, resp)
	companyID := company["id"].(string)

	// Wishlist card (order 0) has no applied_at → not an application yet.
	createTestJob(t, token, "Wishlisted", cols["Wishlist"], companyID)
	// Applied card (order > 0) stamps applied_at → an active application.
	createTestJob(t, token, "Applied", cols["Applied"], companyID)
	// Applied then archived → still an application, but not active.
	arch := createTestJob(t, token, "Archived", cols["Applied"], companyID)
	archiveJob(t, token, arch["id"].(string))

	resp = doRequest(t, http.MethodGet, "/api/v1/companies", nil, token)
	assertStatus(t, resp, http.StatusOK)
	companies := parseJSON[struct {
		Items []struct {
			ID                      string `json:"id"`
			ApplicationsCount       int    `json:"applications_count"`
			ActiveApplicationsCount int    `json:"active_applications_count"`
		} `json:"items"`
	}](t, resp)

	require.Len(t, companies.Items, 1)
	c := companies.Items[0]
	assert.Equal(t, companyID, c.ID)
	assert.Equal(t, 2, c.ApplicationsCount, "two cards have applied_at set")
	assert.Equal(t, 1, c.ActiveApplicationsCount, "only the non-archived applied card is active")
}
