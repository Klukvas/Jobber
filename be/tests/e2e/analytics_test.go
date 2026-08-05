//go:build integration

package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type funnelResponse struct {
	Stages []struct {
		StageName      string  `json:"stage_name"`
		StageOrder     int     `json:"stage_order"`
		Count          int     `json:"count"`
		ConversionRate float64 `json:"conversion_rate"`
	} `json:"stages"`
}

type overviewResponse struct {
	TotalApplications  int `json:"total_applications"`
	ActiveApplications int `json:"active_applications"`
}

func createJob(t *testing.T, token, title, status string) {
	t.Helper()
	body := map[string]string{"title": title}
	if status != "" {
		body["status"] = status
	}
	resp := doRequest(t, http.MethodPost, "/api/v1/jobs", body, token)
	assertStatus(t, resp, http.StatusCreated)
}

func TestIntegrationAnalyticsFunnel(t *testing.T) {
	t.Run("shows Applied bucket for applications without stages or templates", func(t *testing.T) {
		userID := seedUser(t, "funnel-no-stages@example.com", "password123")
		seedSubscription(t, userID, "free")
		token := authToken(t, userID)

		for i := 1; i <= 2; i++ {
			createJob(t, token, fmt.Sprintf("Applied Job %d", i), "applied")
		}
		// Saved (wishlist) cards must NOT count as applications.
		createJob(t, token, "Saved Job", "")

		resp := doRequest(t, http.MethodGet, "/api/v1/analytics/funnel", nil, token)
		assertStatus(t, resp, http.StatusOK)

		funnel := parseJSON[funnelResponse](t, resp)
		require.Len(t, funnel.Stages, 1)
		assert.Equal(t, "Applied", funnel.Stages[0].StageName)
		assert.Equal(t, 2, funnel.Stages[0].Count)
	})

	t.Run("returns empty funnel for user with no applications", func(t *testing.T) {
		userID := seedUser(t, "funnel-empty@example.com", "password123")
		seedSubscription(t, userID, "free")
		token := authToken(t, userID)

		createJob(t, token, "Saved Only", "")

		resp := doRequest(t, http.MethodGet, "/api/v1/analytics/funnel", nil, token)
		assertStatus(t, resp, http.StatusOK)

		funnel := parseJSON[funnelResponse](t, resp)
		assert.Empty(t, funnel.Stages)
	})
}

func TestIntegrationAnalyticsOverview(t *testing.T) {
	userID := seedUser(t, "overview-basic@example.com", "password123")
	seedSubscription(t, userID, "free")
	token := authToken(t, userID)

	createJob(t, token, "Applied Job", "applied")
	createJob(t, token, "Saved Job", "")

	resp := doRequest(t, http.MethodGet, "/api/v1/analytics/overview", nil, token)
	assertStatus(t, resp, http.StatusOK)

	overview := parseJSON[overviewResponse](t, resp)
	assert.Equal(t, 1, overview.TotalApplications)
	assert.Equal(t, 1, overview.ActiveApplications)
}
