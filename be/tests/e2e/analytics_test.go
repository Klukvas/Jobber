//go:build integration

package e2e

import (
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
	ClosedApplications int `json:"closed_applications"`
}

// The funnel is a positional funnel over the user's own ordered stage templates:
// it returns every column (LEFT JOIN), and a card "reaches" a column when it
// currently sits there or has a job_stages row for it. Archived cards are excluded.
func TestIntegrationAnalyticsFunnel(t *testing.T) {
	t.Run("returns every column in order with per-stage reached counts", func(t *testing.T) {
		cleanupAll(t)
		userID := seedUser(t, "funnel-stages@example.com", "password123")
		seedSubscription(t, userID, "free")
		token := authToken(t, userID)
		cols := seedPipeline(t, userID, "Wishlist", "Applied", "Offer")

		createTestJob(t, token, "Applied 1", cols["Applied"], "")
		createTestJob(t, token, "Applied 2", cols["Applied"], "")
		createTestJob(t, token, "Wishlisted", cols["Wishlist"], "")

		resp := doRequest(t, http.MethodGet, "/api/v1/analytics/funnel", nil, token)
		assertStatus(t, resp, http.StatusOK)
		funnel := parseJSON[funnelResponse](t, resp)

		require.Len(t, funnel.Stages, 3)
		assert.Equal(t, "Wishlist", funnel.Stages[0].StageName)
		assert.Equal(t, "Applied", funnel.Stages[1].StageName)
		assert.Equal(t, "Offer", funnel.Stages[2].StageName)
		assert.Equal(t, 2, funnel.Stages[1].Count, "two cards currently sit in Applied")
		assert.Equal(t, 0, funnel.Stages[2].Count, "no card reached Offer")
		assert.GreaterOrEqual(t, funnel.Stages[0].Count, 1, "the wishlist card reaches Wishlist")
	})

	t.Run("archived cards are excluded from the funnel", func(t *testing.T) {
		cleanupAll(t)
		userID := seedUser(t, "funnel-archived@example.com", "password123")
		seedSubscription(t, userID, "free")
		token := authToken(t, userID)
		cols := seedPipeline(t, userID, "Wishlist", "Applied")

		createTestJob(t, token, "Active", cols["Applied"], "")
		archived := createTestJob(t, token, "Archived", cols["Applied"], "")
		archiveJob(t, token, archived["id"].(string))

		resp := doRequest(t, http.MethodGet, "/api/v1/analytics/funnel", nil, token)
		assertStatus(t, resp, http.StatusOK)
		funnel := parseJSON[funnelResponse](t, resp)

		require.Len(t, funnel.Stages, 2)
		assert.Equal(t, 1, funnel.Stages[1].Count, "the archived card must not count toward Applied")
	})

	t.Run("empty funnel for a user with no pipeline", func(t *testing.T) {
		cleanupAll(t)
		userID := seedUser(t, "funnel-empty@example.com", "password123")
		seedSubscription(t, userID, "free")
		token := authToken(t, userID)

		resp := doRequest(t, http.MethodGet, "/api/v1/analytics/funnel", nil, token)
		assertStatus(t, resp, http.StatusOK)
		funnel := parseJSON[funnelResponse](t, resp)
		assert.Empty(t, funnel.Stages)
	})
}

// Overview counts non-archived cards (total/active) and archived cards (closed),
// with no reference to the dropped status column.
func TestIntegrationAnalyticsOverview(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "overview-stages@example.com", "password123")
	seedSubscription(t, userID, "free")
	token := authToken(t, userID)
	cols := seedPipeline(t, userID, "Wishlist", "Applied")

	createTestJob(t, token, "Applied Job", cols["Applied"], "")
	createTestJob(t, token, "Saved Job", cols["Wishlist"], "")
	archived := createTestJob(t, token, "Archived Job", cols["Applied"], "")
	archiveJob(t, token, archived["id"].(string))

	resp := doRequest(t, http.MethodGet, "/api/v1/analytics/overview", nil, token)
	assertStatus(t, resp, http.StatusOK)
	overview := parseJSON[overviewResponse](t, resp)

	assert.Equal(t, 2, overview.TotalApplications, "two non-archived cards")
	assert.Equal(t, 2, overview.ActiveApplications, "both non-archived cards sit in a column")
	assert.Equal(t, 1, overview.ClosedApplications, "one archived card")
}
