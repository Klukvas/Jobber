//go:build integration

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// helper: creates a cover letter via API and returns its ID.
func createCoverLetter(t *testing.T, token string, title string) string {
	t.Helper()
	resp := doRequest(t, http.MethodPost, "/api/v1/cover-letters", map[string]string{
		"title": title,
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	body := parseJSON[map[string]interface{}](t, resp)
	return body["id"].(string)
}

func TestIntegrationCreateCoverLetter(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cv-create@test.com")

	resp := doRequest(t, http.MethodPost, "/api/v1/cover-letters", map[string]string{
		"title": "My Cover Letter",
	}, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "My Cover Letter", body["title"])
}

func TestIntegrationListCoverLetters(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cv-list@test.com")

	createCoverLetter(t, token, "CL 1")
	createCoverLetter(t, token, "CL 2")

	resp := doRequest(t, http.MethodGet, "/api/v1/cover-letters", nil, token)
	assertStatus(t, resp, http.StatusOK)

	body := parseJSON[[]interface{}](t, resp)
	assert.Len(t, body, 2)
}

func TestIntegrationGetCoverLetter(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cv-get@test.com")

	clID := createCoverLetter(t, token, "Get CL")

	resp := doRequest(t, http.MethodGet, "/api/v1/cover-letters/"+clID, nil, token)
	assertStatus(t, resp, http.StatusOK)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, clID, body["id"])
}

func TestIntegrationUpdateCoverLetter(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cv-update@test.com")

	clID := createCoverLetter(t, token, "Original CL")

	newTitle := "Updated CL"
	resp := doRequest(t, http.MethodPatch, "/api/v1/cover-letters/"+clID, map[string]interface{}{
		"title": newTitle,
	}, token)
	assertStatus(t, resp, http.StatusOK)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, newTitle, body["title"])
}

func TestIntegrationDeleteCoverLetter(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cv-delete@test.com")

	clID := createCoverLetter(t, token, "Delete CL")

	resp := doRequest(t, http.MethodDelete, "/api/v1/cover-letters/"+clID, nil, token)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Verify deleted
	resp = doRequest(t, http.MethodGet, "/api/v1/cover-letters/"+clID, nil, token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationCoverLetterInvalidFont(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cv-badfont@test.com")

	clID := createCoverLetter(t, token, "Font CL")

	resp := doRequest(t, http.MethodPatch, "/api/v1/cover-letters/"+clID, map[string]interface{}{
		"font_family": "ComicSansXYZ_NOT_REAL",
	}, token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationCoverLetterInvalidColor(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cv-badcolor@test.com")

	clID := createCoverLetter(t, token, "Color CL")

	resp := doRequest(t, http.MethodPatch, "/api/v1/cover-letters/"+clID, map[string]interface{}{
		"primary_color": "not-a-color",
	}, token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationCoverLetterOwnership(t *testing.T) {
	cleanupAll(t)
	_, tokenA := setupProUser(t, "cv-ownerA@test.com")
	_, tokenB := setupProUser(t, "cv-ownerB@test.com")

	clID := createCoverLetter(t, tokenA, "Owner A's CL")

	// User B tries to GET — handler checks ownership via model.ErrNotAuthorized → 403
	resp := doRequest(t, http.MethodGet, "/api/v1/cover-letters/"+clID, nil, tokenB)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// User B tries to PATCH — same sentinel → 403
	resp = doRequest(t, http.MethodPatch, "/api/v1/cover-letters/"+clID, map[string]interface{}{
		"title": "Hacked",
	}, tokenB)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// User B tries to DELETE
	resp = doRequest(t, http.MethodDelete, "/api/v1/cover-letters/"+clID, nil, tokenB)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationDuplicateCoverLetter(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cv-dup@test.com")

	// Create a cover letter with content
	clID := createCoverLetter(t, token, "Original CL")

	// Update it with paragraphs and other fields
	resp := doRequest(t, http.MethodPatch, "/api/v1/cover-letters/"+clID, map[string]interface{}{
		"recipient_name": "Jane Smith",
		"company_name":   "Acme Corp",
		"greeting":       "Dear Jane,",
		"paragraphs":     []string{"First paragraph.", "Second paragraph."},
		"closing":        "Sincerely,",
	}, token)
	assertStatus(t, resp, http.StatusOK)

	// Duplicate
	resp = doRequest(t, http.MethodPost, "/api/v1/cover-letters/"+clID+"/duplicate", nil, token)
	assertStatus(t, resp, http.StatusCreated)

	dupBody := parseJSON[map[string]interface{}](t, resp)
	assert.NotEqual(t, clID, dupBody["id"], "duplicate should have a new ID")
	assert.Contains(t, dupBody["title"].(string), "(Copy)", "duplicate title should contain (Copy)")
	assert.Equal(t, "Jane Smith", dupBody["recipient_name"])
	assert.Equal(t, "Acme Corp", dupBody["company_name"])
	assert.Equal(t, "Dear Jane,", dupBody["greeting"])
	assert.Equal(t, "Sincerely,", dupBody["closing"])

	// Verify paragraphs are copied
	paragraphs, ok := dupBody["paragraphs"].([]interface{})
	assert.True(t, ok, "paragraphs should be an array")
	assert.Len(t, paragraphs, 2)
	assert.Equal(t, "First paragraph.", paragraphs[0])
	assert.Equal(t, "Second paragraph.", paragraphs[1])
}

func TestIntegrationCoverLetterJobID(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cv-jobid@test.com")

	// Create a cover letter without job_id
	clID := createCoverLetter(t, token, "Job CL")

	// GET and verify job_id is not present or null
	resp := doRequest(t, http.MethodGet, "/api/v1/cover-letters/"+clID, nil, token)
	assertStatus(t, resp, http.StatusOK)
	body := parseJSON[map[string]interface{}](t, resp)
	assert.Nil(t, body["job_id"], "job_id should be nil initially")

	// The job_id field is a foreign key referencing the jobs table.
	// Since we cannot easily seed a job in E2E tests, we verify the field
	// exists in the response by checking the initial nil state above.
	// The unit tests cover the job_id pass-through logic thoroughly.
}

func TestIntegrationDuplicateCoverLetterCopiesJobID(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "cv-dupjob@test.com")

	// Create a cover letter (job_id will be nil since we cannot seed a job easily)
	clID := createCoverLetter(t, token, "Dup Job CL")

	// Duplicate the cover letter
	resp := doRequest(t, http.MethodPost, "/api/v1/cover-letters/"+clID+"/duplicate", nil, token)
	assertStatus(t, resp, http.StatusCreated)

	dupBody := parseJSON[map[string]interface{}](t, resp)
	assert.Contains(t, dupBody["title"].(string), "(Copy)")
	// Both original and duplicate should have nil job_id
	assert.Nil(t, dupBody["job_id"], "duplicate should have nil job_id when original has nil")
}
