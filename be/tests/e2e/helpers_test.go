//go:build integration

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// seedUser inserts a user directly into the DB and returns the user ID.
func seedUser(t *testing.T, email, password string) string {
	t.Helper()
	id := uuid.New().String()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (id, email, name, password_hash, locale, created_at, updated_at)
		 VALUES ($1, $2, '', $3, 'en', NOW(), NOW())`,
		id, email, string(hash),
	)
	require.NoError(t, err)
	return id
}

// seedSubscription inserts a subscription for a user.
func seedSubscription(t *testing.T, userID, plan string) {
	t.Helper()
	id := uuid.New().String()
	status := plan
	if plan == "pro" || plan == "enterprise" {
		status = "active"
	}
	_, err := pool.Exec(context.Background(),
		`INSERT INTO subscriptions (id, user_id, status, plan, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, NOW(), NOW())
		 ON CONFLICT (user_id) DO UPDATE SET plan = $4, status = $3, updated_at = NOW()`,
		id, userID, status, plan,
	)
	require.NoError(t, err)
}

// authToken generates a JWT access token for a user.
func authToken(t *testing.T, userID string) string {
	t.Helper()
	token, err := jwtManager.GenerateAccessToken(userID)
	require.NoError(t, err)
	return "Bearer " + token
}

// doRequest performs an HTTP request against the test server.
func doRequest(t *testing.T, method, path string, body interface{}, token string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, serverURL+path, bodyReader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// parseJSON decodes the response body into the given type.
func parseJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()
	defer resp.Body.Close()
	var result T
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	return result
}

// readBody reads and returns the response body as a string, then closes it.
func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(b)
}

// assertStatus asserts the HTTP status code and dumps the body on failure.
func assertStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("expected status %d, got %d; body: %s", expected, resp.StatusCode, string(body))
	}
}

// assertErrorCode parses an error response and checks the error_code field.
func assertErrorCode(t *testing.T, resp *http.Response, code string) {
	t.Helper()
	defer resp.Body.Close()
	var errResp struct {
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
	}
	err := json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)
	require.Equal(t, code, errResp.ErrorCode, "error_code mismatch; message: %s", errResp.ErrorMessage)
}

// truncateTables truncates the given tables with CASCADE.
func truncateTables(t *testing.T, tables ...string) {
	t.Helper()
	for _, table := range tables {
		_, err := pool.Exec(context.Background(), fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(t, err)
	}
}

// cleanupAll truncates all user-data tables.
func cleanupAll(t *testing.T) {
	t.Helper()
	truncateTables(t,
		"resume_custom_sections",
		"resume_volunteering",
		"resume_projects",
		"resume_certifications",
		"resume_languages",
		"resume_skills",
		"resume_educations",
		"resume_experiences",
		"resume_section_orders",
		"resume_summaries",
		"resume_contacts",
		"cover_letters",
		"content_library",
		"resume_builders",
		"ai_usage",
		"match_score_cache",
		"comments",
		"job_stages",
		"stage_templates",
		"resumes",
		"jobs",
		"companies",
		"subscriptions",
		"refresh_tokens",
		"users",
	)
}

// seedPipeline inserts an ordered set of pipeline columns (stage templates) for a
// user and returns a name->id map. Users created via seedUser have no columns
// (registration seeds them, but seedUser inserts the user directly), so any test
// that creates jobs must seed a pipeline first — otherwise POST /jobs fails with
// STAGE_TEMPLATE_NOT_FOUND.
func seedPipeline(t *testing.T, userID string, names ...string) map[string]string {
	t.Helper()
	ids := make(map[string]string, len(names))
	for i, name := range names {
		id := uuid.New().String()
		_, err := pool.Exec(context.Background(),
			`INSERT INTO stage_templates (id, user_id, name, "order", created_at, updated_at)
			 VALUES ($1, $2, $3, $4, NOW(), NOW())`,
			id, userID, name, i,
		)
		require.NoError(t, err)
		ids[name] = id
	}
	return ids
}

// createTestJob creates a job placed in stageTemplateID (empty = the user's first
// column) optionally linked to companyID, asserts 201, and returns the created
// job as a map.
func createTestJob(t *testing.T, token, title, stageTemplateID, companyID string) map[string]interface{} {
	t.Helper()
	body := map[string]interface{}{"title": title}
	if stageTemplateID != "" {
		body["stage_template_id"] = stageTemplateID
	}
	if companyID != "" {
		body["company_id"] = companyID
	}
	resp := doRequest(t, http.MethodPost, "/api/v1/jobs", body, token)
	assertStatus(t, resp, http.StatusCreated)
	return parseJSON[map[string]interface{}](t, resp)
}

// archiveJob flips is_archived on a job via PATCH.
func archiveJob(t *testing.T, token, jobID string) {
	t.Helper()
	resp := doRequest(t, http.MethodPatch, "/api/v1/jobs/"+jobID,
		map[string]interface{}{"is_archived": true}, token)
	assertStatus(t, resp, http.StatusOK)
	resp.Body.Close()
}
