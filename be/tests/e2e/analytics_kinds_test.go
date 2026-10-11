//go:build integration

package e2e

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Analytics read what a column MEANS (stage_templates.kind), never its label or
// its position. These tests pin the three ways the old rules were wrong:
//
//   - "anything past the first column is a response" counted a card the user
//     had merely applied to, and any custom column they invented;
//   - LOWER(name) LIKE '%interview%' stopped counting a renamed interview
//     column, and counted a "No interview yet" column;
//   - reordering the board moved cards between metrics.

type kindOverviewResponse struct {
	TotalApplications      int     `json:"total_applications"`
	ActiveApplications     int     `json:"active_applications"`
	ClosedApplications     int     `json:"closed_applications"`
	RejectedApplications   int     `json:"rejected_applications"`
	ResponseRate           float64 `json:"response_rate"`
	AvgDaysToFirstResponse float64 `json:"avg_days_to_first_response"`
}

type resumeAnalyticsResponse struct {
	Resumes []struct {
		ResumeID          string  `json:"resume_id"`
		ResumeTitle       string  `json:"resume_title"`
		ApplicationsCount int     `json:"applications_count"`
		ResponsesCount    int     `json:"responses_count"`
		InterviewsCount   int     `json:"interviews_count"`
		ResponseRate      float64 `json:"response_rate"`
	} `json:"resumes"`
}

type sourceAnalyticsResponse struct {
	Sources []struct {
		SourceName        string  `json:"source_name"`
		ApplicationsCount int     `json:"applications_count"`
		ResponsesCount    int     `json:"responses_count"`
		ConversionRate    float64 `json:"conversion_rate"`
	} `json:"sources"`
}

func getOverview(t *testing.T, token string) kindOverviewResponse {
	t.Helper()
	resp := doRequest(t, http.MethodGet, "/api/v1/analytics/overview", nil, token)
	assertStatus(t, resp, http.StatusOK)
	return parseJSON[kindOverviewResponse](t, resp)
}

// seedResume inserts a resume directly and returns its id.
func seedResume(t *testing.T, userID, title string) string {
	t.Helper()
	id := uuid.New().String()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO resumes (id, user_id, title, file_url, storage_type, storage_key, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, NULL, 'external', NULL, true, NOW(), NOW())`,
		id, userID, title)
	require.NoError(t, err)
	return id
}

// renameColumn renames a pipeline column through the API.
func renameColumn(t *testing.T, token, columnID, name string) {
	t.Helper()
	resp := doRequest(t, http.MethodPatch, "/api/v1/stage-templates/"+columnID,
		map[string]interface{}{"name": name}, token)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}

func TestIntegrationAnalyticsAppliedIsNotAResponse(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "kinds-applied@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	// Two cards the user applied to, nothing back yet, plus a custom column the
	// user invented after the wishlist.
	createTestJob(t, token, "Applied A", cols["Applied"], "")
	createTestJob(t, token, "Applied B", cols["Applied"], "")
	custom := seedPipelineWithKinds(t, userID, pipelineColumn{Name: "Chased recruiter", Kind: "other"})
	// seedPipelineWithKinds restarts the ordering at 0, so put the custom column
	// at the end of the board instead.
	_, err := pool.Exec(context.Background(),
		`UPDATE stage_templates SET "order" = 99 WHERE id = $1`, custom["Chased recruiter"])
	require.NoError(t, err)
	createTestJob(t, token, "Chased", custom["Chased recruiter"], "")

	overview := getOverview(t, token)

	assert.Equal(t, 3, overview.ActiveApplications)
	assert.Equal(t, 0.0, overview.ResponseRate,
		"sending an application is not the employer responding, and a custom column means nothing on its own")
	assert.Equal(t, 0, overview.RejectedApplications)

	// Moving one card into the response column flips exactly that card.
	job := createTestJob(t, token, "Screening card", cols["Applied"], "")
	resp := doRequest(t, http.MethodPost, "/api/v1/jobs/"+job["id"].(string)+"/move",
		map[string]interface{}{"stage_template_id": cols["Screening"]}, token)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	overview = getOverview(t, token)
	assert.Equal(t, 4, overview.ActiveApplications)
	assert.Equal(t, 25.0, overview.ResponseRate, "one of four cards reached a response column")
}

func TestIntegrationAnalyticsRenamedInterviewStillCountsAsInterview(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "kinds-rename@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)
	resumeID := seedResume(t, userID, "Backend CV")

	// One card sitting in the interview column, attached to the resume.
	resp := doRequest(t, http.MethodPost, "/api/v1/jobs", map[string]interface{}{
		"title":             "Interviewing",
		"stage_template_id": cols["Technical Interview"],
		"resume_id":         resumeID,
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	resp.Body.Close()

	effectiveness := func() resumeAnalyticsResponse {
		resp := doRequest(t, http.MethodGet, "/api/v1/analytics/resumes", nil, token)
		assertStatus(t, resp, http.StatusOK)
		return parseJSON[resumeAnalyticsResponse](t, resp)
	}

	before := effectiveness()
	require.Len(t, before.Resumes, 1)
	assert.Equal(t, 1, before.Resumes[0].ApplicationsCount)
	assert.Equal(t, 1, before.Resumes[0].ResponsesCount)
	assert.Equal(t, 1, before.Resumes[0].InterviewsCount)

	// Rename the column to something with no "interview" in it, and drag it to
	// the front of the board.
	renameColumn(t, token, cols["Technical Interview"], "Onsite loop")
	resp = doRequest(t, http.MethodPost, "/api/v1/stage-templates/reorder", map[string]interface{}{
		"stage_ids": []string{
			cols["Technical Interview"], cols["Wishlist"], cols["Applied"],
			cols["Screening"], cols["Offer"], cols["Rejected"],
		},
	}, token)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()

	after := effectiveness()
	require.Len(t, after.Resumes, 1)
	assert.Equal(t, before.Resumes[0].ApplicationsCount, after.Resumes[0].ApplicationsCount)
	assert.Equal(t, before.Resumes[0].ResponsesCount, after.Resumes[0].ResponsesCount,
		"renaming and reordering a column must not change what it counts")
	assert.Equal(t, before.Resumes[0].InterviewsCount, after.Resumes[0].InterviewsCount,
		"a renamed interview column is still an interview column")
}

func TestIntegrationAnalyticsRejectedAndArchiveSemantics(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "kinds-rejected@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	// A rejection the user keeps on the board, and one they archived.
	createTestJob(t, token, "Rejected, visible", cols["Rejected"], "")
	archived := createTestJob(t, token, "Rejected, archived", cols["Rejected"], "")
	archiveJob(t, token, archived["id"].(string))
	createTestJob(t, token, "Still applied", cols["Applied"], "")

	overview := getOverview(t, token)

	assert.Equal(t, 3, overview.TotalApplications, "total spans archived cards too")
	assert.Equal(t, 2, overview.ActiveApplications)
	assert.Equal(t, 1, overview.ClosedApplications)
	assert.Equal(t, 1, overview.RejectedApplications,
		"the archived rejection is hidden from the metric, like everywhere else")
	assert.Equal(t, 50.0, overview.ResponseRate, "a rejection is a response; the applied card is not")
}

func TestIntegrationAnalyticsSourcesUseKinds(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "kinds-sources@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	withSource := func(title, columnID, source string) {
		resp := doRequest(t, http.MethodPost, "/api/v1/jobs", map[string]interface{}{
			"title":             title,
			"stage_template_id": columnID,
			"source":            source,
		}, token)
		assertStatus(t, resp, http.StatusCreated)
		resp.Body.Close()
	}

	withSource("Via LinkedIn, interviewing", cols["Technical Interview"], "LinkedIn")
	withSource("Via LinkedIn, applied", cols["Applied"], "LinkedIn")
	withSource("Via referral, applied", cols["Applied"], "Referral")

	resp := doRequest(t, http.MethodGet, "/api/v1/analytics/sources", nil, token)
	assertStatus(t, resp, http.StatusOK)
	sources := parseJSON[sourceAnalyticsResponse](t, resp)

	byName := map[string]int{}
	applications := map[string]int{}
	for _, s := range sources.Sources {
		byName[s.SourceName] = s.ResponsesCount
		applications[s.SourceName] = s.ApplicationsCount
	}

	assert.Equal(t, 2, applications["LinkedIn"])
	assert.Equal(t, 1, byName["LinkedIn"], "only the interviewing card is a response")
	assert.Equal(t, 1, applications["Referral"])
	assert.Equal(t, 0, byName["Referral"], "an applied-only card is not a response")
}
