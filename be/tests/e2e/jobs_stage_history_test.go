//go:build integration

package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A card has at most one ACTIVE history row — that is the invariant every
// pipeline write maintains. It says nothing about 'pending' rows: those are
// rounds the user scheduled ahead of time ("Stage has not started yet" in the
// stage status dialog) and several may exist at once. Closing them along with
// the active one marked interviews the card had not even reached as completed,
// and dated them to whenever the next move happened.

type stageRow struct {
	ID          string  `json:"id"`
	TemplateID  string  `json:"stage_template_id"`
	Status      string  `json:"status"`
	CompletedAt *string `json:"completed_at"`
}

// addStage appends a history row for a column and returns it.
func addStage(t *testing.T, token, jobID, columnID string) stageRow {
	t.Helper()
	resp := doRequest(t, http.MethodPost, "/api/v1/jobs/"+jobID+"/stages",
		map[string]interface{}{"stage_template_id": columnID}, token)
	assertStatus(t, resp, http.StatusCreated)
	return parseJSON[stageRow](t, resp)
}

// setStageStatus patches one history row's status.
func setStageStatus(t *testing.T, token, jobID, stageID, status string) {
	t.Helper()
	resp := doRequest(t, http.MethodPatch, "/api/v1/jobs/"+jobID+"/stages/"+stageID,
		map[string]interface{}{"status": status}, token)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

// moveJob moves a card into a column.
func moveJob(t *testing.T, token, jobID, columnID string) {
	t.Helper()
	resp := doRequest(t, http.MethodPost, "/api/v1/jobs/"+jobID+"/move",
		map[string]interface{}{"stage_template_id": columnID}, token)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

// stageState reads one history row's status and completion straight from the
// database, by id — the rows a test tracks individually may share a column with
// another row, so the API listing cannot be indexed by column for them.
func stageState(t *testing.T, stageID string) (status string, completedAt *string) {
	t.Helper()
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT status, completed_at::text FROM job_stages WHERE id = $1`, stageID).
		Scan(&status, &completedAt))
	return status, completedAt
}

// countStagesWithStatus counts a card's history rows in one status.
func countStagesWithStatus(t *testing.T, jobID, status string) int {
	t.Helper()
	var count int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM job_stages WHERE job_id = $1 AND status = $2`, jobID, status).Scan(&count))
	return count
}

// firstStageOfColumn returns the card's history row for a column, from the API
// listing.
func firstStageOfColumn(t *testing.T, token, jobID, columnID string) stageRow {
	t.Helper()
	resp := doRequest(t, http.MethodGet, "/api/v1/jobs/"+jobID+"/stages", nil, token)
	assertStatus(t, resp, http.StatusOK)
	for _, row := range parseJSON[[]stageRow](t, resp) {
		if row.TemplateID == columnID {
			return row
		}
	}
	t.Fatalf("no history row for column %s", columnID)
	return stageRow{}
}

func TestIntegrationPlannedStagesSurvivePipelineWrites(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "planned-stages@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	job := createTestJob(t, token, "Engineer", cols["Applied"], "")
	jobID := job["id"].(string)
	appliedRow := firstStageOfColumn(t, token, jobID, cols["Applied"])

	// The user schedules the technical round before it happens: adding it makes
	// it the active row, then marking it pending turns it into a plan.
	plannedInterview := addStage(t, token, jobID, cols["Technical Interview"])
	setStageStatus(t, token, jobID, plannedInterview.ID, "pending")

	// Back to the column the card is really in. Re-activating a row closes
	// whatever else is open — a planned round is not "open", it is planned.
	setStageStatus(t, token, jobID, appliedRow.ID, "active")

	status, completedAt := stageState(t, plannedInterview.ID)
	assert.Equal(t, "pending", status,
		"re-activating another row must not complete a round the card has not reached")
	assert.Nil(t, completedAt)
	assert.Equal(t, 1, countActiveStages(t, jobID))

	// A move closes the active row and appends its own. The plan still stands.
	moveJob(t, token, jobID, cols["Screening"])

	status, completedAt = stageState(t, plannedInterview.ID)
	assert.Equal(t, "pending", status, "a move must not complete a scheduled round either")
	assert.Nil(t, completedAt)
	appliedStatus, _ := stageState(t, appliedRow.ID)
	assert.Equal(t, "completed", appliedStatus, "the column the card left is closed")
	assert.Equal(t, 1, countActiveStages(t, jobID))

	// Several rounds may be planned at once, and a later move leaves them all
	// alone.
	plannedOffer := addStage(t, token, jobID, cols["Offer"])
	setStageStatus(t, token, jobID, plannedOffer.ID, "pending")

	moveJob(t, token, jobID, cols["Rejected"])

	assert.Equal(t, 2, countStagesWithStatus(t, jobID, "pending"),
		"both scheduled rounds are still only planned")
	assert.Equal(t, 1, countActiveStages(t, jobID))

	// Skipped and cancelled are final for the same reason: the user set them
	// deliberately, and a move must not rewrite the timestamp.
	setStageStatus(t, token, jobID, plannedOffer.ID, "skipped")
	moveJob(t, token, jobID, cols["Wishlist"])

	status, _ = stageState(t, plannedOffer.ID)
	assert.Equal(t, "skipped", status)
	assert.Equal(t, 1, countActiveStages(t, jobID))
}
