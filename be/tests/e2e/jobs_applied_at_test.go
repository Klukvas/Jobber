//go:build integration

package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The create form's "I already applied" toggle sends applied_at and nothing
// else. The backend used to ignore the field entirely: the date was dropped and
// the card sat in the wishlist as if the user had only bookmarked it.

type createdJob struct {
	ID                     string     `json:"id"`
	CompanyID              *string    `json:"company_id"`
	AppliedAt              *time.Time `json:"applied_at"`
	CurrentStageTemplateID *string    `json:"current_stage_template_id"`
	CurrentStageName       *string    `json:"current_stage_name"`
}

// stageHistory returns the (column id, status, started_at) of every history row
// of a card, oldest first.
func stageHistory(t *testing.T, jobID string) []struct {
	TemplateID string
	Status     string
	StartedAt  time.Time
} {
	t.Helper()
	rows, err := pool.Query(context.Background(),
		`SELECT stage_template_id, status, started_at FROM job_stages WHERE job_id = $1 ORDER BY "order" ASC`, jobID)
	require.NoError(t, err)
	defer rows.Close()

	var history []struct {
		TemplateID string
		Status     string
		StartedAt  time.Time
	}
	for rows.Next() {
		var row struct {
			TemplateID string
			Status     string
			StartedAt  time.Time
		}
		require.NoError(t, rows.Scan(&row.TemplateID, &row.Status, &row.StartedAt))
		history = append(history, row)
	}
	require.NoError(t, rows.Err())
	return history
}

func TestIntegrationCreateWithAppliedAt(t *testing.T) {
	appliedOn := time.Date(2026, time.February, 11, 8, 0, 0, 0, time.UTC)

	t.Run("places the card in the applied column and keeps the date", func(t *testing.T) {
		cleanupAll(t)
		userID := seedUser(t, "applied-at-default@test.com", "securepass123")
		seedSubscription(t, userID, "pro")
		token := authToken(t, userID)
		cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs", map[string]interface{}{
			"title":      "Backend Engineer",
			"applied_at": appliedOn.Format(time.RFC3339),
		}, token)
		assertStatus(t, resp, http.StatusCreated)
		job := parseJSON[createdJob](t, resp)

		require.NotNil(t, job.CurrentStageTemplateID)
		assert.Equal(t, cols["Applied"], *job.CurrentStageTemplateID, "an applied card does not belong in the wishlist")
		require.NotNil(t, job.AppliedAt)
		assert.True(t, job.AppliedAt.Equal(appliedOn), "the date the user picked is stored as-is")

		// History records the column the card started in, dated from the
		// application — that is what time-to-response measures from.
		history := stageHistory(t, job.ID)
		require.Len(t, history, 1)
		assert.Equal(t, cols["Applied"], history[0].TemplateID)
		assert.Equal(t, "active", history[0].Status)
		assert.True(t, history[0].StartedAt.UTC().Equal(appliedOn))
	})

	t.Run("a pipeline with no applied column still records the date", func(t *testing.T) {
		cleanupAll(t)
		userID := seedUser(t, "applied-at-custom@test.com", "securepass123")
		seedSubscription(t, userID, "pro")
		token := authToken(t, userID)
		cols := seedPipelineWithKinds(t, userID,
			pipelineColumn{Name: "Leads", Kind: "saved"},
			pipelineColumn{Name: "Notes", Kind: "other"},
			pipelineColumn{Name: "Recruiter call", Kind: "response"},
		)

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs", map[string]interface{}{
			"title":      "Platform Engineer",
			"applied_at": appliedOn.Format(time.RFC3339),
		}, token)
		assertStatus(t, resp, http.StatusCreated)
		job := parseJSON[createdJob](t, resp)

		require.NotNil(t, job.CurrentStageTemplateID)
		assert.Equal(t, cols["Recruiter call"], *job.CurrentStageTemplateID,
			"the first post-application column is the deterministic fallback")
		require.NotNil(t, job.AppliedAt, "the toggle is never silently dropped")
		assert.True(t, job.AppliedAt.Equal(appliedOn))
	})

	t.Run("an explicit column wins over the applied date", func(t *testing.T) {
		cleanupAll(t)
		userID := seedUser(t, "applied-at-explicit@test.com", "securepass123")
		seedSubscription(t, userID, "pro")
		token := authToken(t, userID)
		cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs", map[string]interface{}{
			"title":             "Data Engineer",
			"stage_template_id": cols["Screening"],
			"applied_at":        appliedOn.Format(time.RFC3339),
		}, token)
		assertStatus(t, resp, http.StatusCreated)
		job := parseJSON[createdJob](t, resp)

		require.NotNil(t, job.CurrentStageTemplateID)
		assert.Equal(t, cols["Screening"], *job.CurrentStageTemplateID)
		require.NotNil(t, job.AppliedAt)
		assert.True(t, job.AppliedAt.Equal(appliedOn))
	})

	t.Run("a plain save stays in the wishlist with no history", func(t *testing.T) {
		cleanupAll(t)
		userID := seedUser(t, "applied-at-absent@test.com", "securepass123")
		seedSubscription(t, userID, "pro")
		token := authToken(t, userID)
		cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs",
			map[string]interface{}{"title": "Bookmarked role"}, token)
		assertStatus(t, resp, http.StatusCreated)
		job := parseJSON[createdJob](t, resp)

		require.NotNil(t, job.CurrentStageTemplateID)
		assert.Equal(t, cols["Wishlist"], *job.CurrentStageTemplateID)
		assert.Nil(t, job.AppliedAt)
		assert.Empty(t, stageHistory(t, job.ID))
	})

	t.Run("rejects a malformed applied_at", func(t *testing.T) {
		cleanupAll(t)
		userID := seedUser(t, "applied-at-bad@test.com", "securepass123")
		seedSubscription(t, userID, "pro")
		token := authToken(t, userID)
		seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs", map[string]interface{}{
			"title":      "Engineer",
			"applied_at": "yesterday",
		}, token)
		assertStatus(t, resp, http.StatusBadRequest)
		resp.Body.Close()
	})
}

// The applied date follows what a column MEANS, not where the user dragged it.
// Against a pipeline whose owner reordered the board, the old "anything past
// the first column counts as applied" rule was wrong in both directions: an
// Applied column pulled to the front stopped stamping the date, and a Wishlist
// column pushed into the middle started stamping it for a bookmark.
func TestIntegrationAppliedAtFollowsColumnKind(t *testing.T) {
	// Applied first, the wishlist parked in the middle.
	reordered := []pipelineColumn{
		{Name: "Applied", Kind: "applied"},
		{Name: "Screening", Kind: "response"},
		{Name: "Wishlist", Kind: "saved"},
		{Name: "Rejected", Kind: "rejected"},
	}

	setup := func(t *testing.T, email string) (string, map[string]string) {
		t.Helper()
		cleanupAll(t)
		userID := seedUser(t, email, "securepass123")
		seedSubscription(t, userID, "pro")
		token := authToken(t, userID)
		return token, seedPipelineWithKinds(t, userID, reordered...)
	}

	t.Run("creating in an applied column at order 0 stamps the date and records history", func(t *testing.T) {
		token, cols := setup(t, "kind-create-applied@test.com")

		job := createTestJob(t, token, "Engineer", cols["Applied"], "")
		jobID := job["id"].(string)

		_, appliedAt, _, _ := jobRow(t, jobID)
		require.NotNil(t, appliedAt, "the column means applied, wherever it sits")

		history := stageHistory(t, jobID)
		require.Len(t, history, 1, "the application is an event the funnel has to see")
		assert.Equal(t, cols["Applied"], history[0].TemplateID)
		assert.Equal(t, "active", history[0].Status)
	})

	t.Run("a bare save lands in the wishlist even when Applied is the first column", func(t *testing.T) {
		// The extension's bookmark: no column, no date. Placing it by position
		// dropped the card into Applied and stamped today's date on a role the
		// user had not applied to.
		token, cols := setup(t, "kind-create-bookmark@test.com")

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs",
			map[string]interface{}{"title": "Bookmarked role"}, token)
		assertStatus(t, resp, http.StatusCreated)
		job := parseJSON[createdJob](t, resp)

		require.NotNil(t, job.CurrentStageTemplateID)
		assert.Equal(t, cols["Wishlist"], *job.CurrentStageTemplateID,
			"a bookmark belongs in the wishlist wherever the user dragged it")
		assert.Nil(t, job.AppliedAt)

		_, appliedAt, _, _ := jobRow(t, job.ID)
		assert.Nil(t, appliedAt, "applied_at stays NULL in the row, not just in the response")
		assert.Empty(t, stageHistory(t, job.ID), "nothing has happened to a bookmarked card yet")
	})

	t.Run("a bare save into a board with no wishlist keeps the first column", func(t *testing.T) {
		cleanupAll(t)
		userID := seedUser(t, "kind-create-no-wishlist@test.com", "securepass123")
		seedSubscription(t, userID, "pro")
		token := authToken(t, userID)
		// No saved column anywhere: the first column is the only placement the
		// chain can offer, and it means the application is already out.
		cols := seedPipelineWithKinds(t, userID,
			pipelineColumn{Name: "Applied", Kind: "applied"},
			pipelineColumn{Name: "Screening", Kind: "response"},
		)

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs",
			map[string]interface{}{"title": "Engineer"}, token)
		assertStatus(t, resp, http.StatusCreated)
		job := parseJSON[createdJob](t, resp)

		require.NotNil(t, job.CurrentStageTemplateID)
		assert.Equal(t, cols["Applied"], *job.CurrentStageTemplateID)
		require.NotNil(t, job.AppliedAt,
			"the card visibly sits in Applied, so the date must not disagree with it")
		require.Len(t, stageHistory(t, job.ID), 1)
	})

	t.Run("creating in a wishlist column past the first one is still a bookmark", func(t *testing.T) {
		token, cols := setup(t, "kind-create-wishlist@test.com")

		job := createTestJob(t, token, "Bookmarked role", cols["Wishlist"], "")
		jobID := job["id"].(string)

		currentColumn, appliedAt, _, _ := jobRow(t, jobID)
		assert.Equal(t, cols["Wishlist"], currentColumn)
		assert.Nil(t, appliedAt, "position must not turn a bookmark into an application")
		assert.Empty(t, stageHistory(t, jobID))
	})

	t.Run("the already-applied toggle still places the card in the applied column", func(t *testing.T) {
		token, cols := setup(t, "kind-create-toggle@test.com")
		appliedOn := time.Date(2026, time.June, 1, 7, 0, 0, 0, time.UTC)

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs", map[string]interface{}{
			"title":      "Engineer",
			"applied_at": appliedOn.Format(time.RFC3339),
		}, token)
		assertStatus(t, resp, http.StatusCreated)
		job := parseJSON[createdJob](t, resp)

		require.NotNil(t, job.CurrentStageTemplateID)
		assert.Equal(t, cols["Applied"], *job.CurrentStageTemplateID,
			"placing a bookmark by kind must not change where an application goes")
		require.NotNil(t, job.AppliedAt)
		assert.True(t, job.AppliedAt.Equal(appliedOn))
		require.Len(t, stageHistory(t, job.ID), 1)
	})

	t.Run("moving into an applied column at order 0 stamps the date", func(t *testing.T) {
		token, cols := setup(t, "kind-move-applied@test.com")

		job := createTestJob(t, token, "Engineer", cols["Wishlist"], "")
		jobID := job["id"].(string)
		_, before, _, _ := jobRow(t, jobID)
		require.Nil(t, before)

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs/"+jobID+"/move",
			map[string]interface{}{"stage_template_id": cols["Applied"]}, token)
		assertStatus(t, resp, http.StatusOK)
		moved := parseJSON[createdJob](t, resp)

		require.NotNil(t, moved.AppliedAt, "the response reports the stamp it just wrote")
		_, appliedAt, _, _ := jobRow(t, jobID)
		assert.NotNil(t, appliedAt)
	})

	t.Run("moving into a wishlist column in the middle does not stamp the date", func(t *testing.T) {
		cleanupAll(t)
		userID := seedUser(t, "kind-move-wishlist@test.com", "securepass123")
		seedSubscription(t, userID, "pro")
		token := authToken(t, userID)
		// The card starts in an unclassified first column (no stamp there) and
		// moves to a wishlist column at order 2 — the move the positional rule
		// used to stamp.
		cols := seedPipelineWithKinds(t, userID,
			pipelineColumn{Name: "Ideas", Kind: "other"},
			pipelineColumn{Name: "Applied", Kind: "applied"},
			pipelineColumn{Name: "Wishlist", Kind: "saved"},
		)

		job := createTestJob(t, token, "Engineer", cols["Ideas"], "")
		jobID := job["id"].(string)
		_, before, _, _ := jobRow(t, jobID)
		require.Nil(t, before)

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs/"+jobID+"/move",
			map[string]interface{}{"stage_template_id": cols["Wishlist"]}, token)
		assertStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		currentColumn, appliedAt, _, _ := jobRow(t, jobID)
		assert.Equal(t, cols["Wishlist"], currentColumn)
		assert.Nil(t, appliedAt, "a bookmark column never invents an application date")
	})

	t.Run("moving back into the wishlist keeps the date the card already had", func(t *testing.T) {
		token, cols := setup(t, "kind-move-back@test.com")

		job := createTestJob(t, token, "Engineer", cols["Applied"], "")
		jobID := job["id"].(string)
		_, stamped, _, _ := jobRow(t, jobID)
		require.NotNil(t, stamped)

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs/"+jobID+"/move",
			map[string]interface{}{"stage_template_id": cols["Wishlist"]}, token)
		assertStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		currentColumn, after, _, _ := jobRow(t, jobID)
		assert.Equal(t, cols["Wishlist"], currentColumn)
		require.NotNil(t, after, "a move is never a reason to forget the card was applied to")
		assert.Equal(t, *stamped, *after)
	})

	t.Run("an all-rejected pipeline still records a supplied date", func(t *testing.T) {
		cleanupAll(t)
		userID := seedUser(t, "kind-only-rejected@test.com", "securepass123")
		seedSubscription(t, userID, "pro")
		token := authToken(t, userID)
		// A degenerate pipeline: the placement chain has nothing better to
		// offer, and the create must not fail over it.
		cols := seedPipelineWithKinds(t, userID, pipelineColumn{Name: "Rejected", Kind: "rejected"})

		resp := doRequest(t, http.MethodPost, "/api/v1/jobs", map[string]interface{}{
			"title":      "Engineer",
			"applied_at": time.Date(2026, time.May, 4, 6, 0, 0, 0, time.UTC).Format(time.RFC3339),
		}, token)
		assertStatus(t, resp, http.StatusCreated)
		created := parseJSON[createdJob](t, resp)

		require.NotNil(t, created.CurrentStageTemplateID)
		assert.Equal(t, cols["Rejected"], *created.CurrentStageTemplateID)
		require.NotNil(t, created.AppliedAt)
	})
}

// The detail form owns the whole field set, so it sends an explicit empty
// string to clear one. company_id is a UUID column: the old whole-row update
// passed "" straight through and the insert failed the cast, surfacing as a 500.
func TestIntegrationClearingOptionalFields(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "clear-fields@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	resp := doRequest(t, http.MethodPost, "/api/v1/companies",
		map[string]interface{}{"name": "Acme"}, token)
	assertStatus(t, resp, http.StatusCreated)
	company := parseJSON[map[string]interface{}](t, resp)

	job := createTestJob(t, token, "Engineer", cols["Applied"], company["id"].(string))
	jobID := job["id"].(string)

	t.Run("an empty company id clears the link", func(t *testing.T) {
		resp := doRequest(t, http.MethodPatch, "/api/v1/jobs/"+jobID,
			map[string]interface{}{"company_id": ""}, token)
		assertStatus(t, resp, http.StatusOK)
		updated := parseJSON[createdJob](t, resp)
		assert.Nil(t, updated.CompanyID)
	})

	t.Run("an omitted company id leaves the link alone", func(t *testing.T) {
		// Re-link, then PATCH something else entirely.
		resp := doRequest(t, http.MethodPatch, "/api/v1/jobs/"+jobID,
			map[string]interface{}{"company_id": company["id"].(string)}, token)
		assertStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		resp = doRequest(t, http.MethodPatch, "/api/v1/jobs/"+jobID,
			map[string]interface{}{"title": "Engineer II"}, token)
		assertStatus(t, resp, http.StatusOK)
		updated := parseJSON[createdJob](t, resp)
		require.NotNil(t, updated.CompanyID)
		assert.Equal(t, company["id"].(string), *updated.CompanyID)
	})

	t.Run("a rename leaves the pipeline position and applied_at alone", func(t *testing.T) {
		before, appliedBefore, _, _ := jobRow(t, jobID)
		require.NotNil(t, appliedBefore, "the card was created in the Applied column")

		resp := doRequest(t, http.MethodPatch, "/api/v1/jobs/"+jobID,
			map[string]interface{}{"title": "Engineer III"}, token)
		assertStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		after, appliedAfter, title, _ := jobRow(t, jobID)
		assert.Equal(t, before, after)
		assert.Equal(t, appliedBefore, appliedAfter)
		assert.Equal(t, "Engineer III", title)
	})
}
