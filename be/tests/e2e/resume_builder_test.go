//go:build integration

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper: creates a user with pro subscription and returns (userID, token).
func setupProUser(t *testing.T, email string) (string, string) {
	t.Helper()
	userID := seedUser(t, email, "securepass123")
	seedSubscription(t, userID, "pro")
	return userID, authToken(t, userID)
}

// helper: creates a resume builder via API and returns its ID.
func createResumeBuilder(t *testing.T, token string, title string) string {
	t.Helper()
	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
		"title": title,
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	body := parseJSON[map[string]interface{}](t, resp)
	return body["id"].(string)
}

// --- CRUD Tests ---

func TestIntegrationCreateResumeBuilder(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-create@test.com")

	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder", map[string]string{
		"title": "My Resume",
	}, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "My Resume", body["title"])
	// Verify defaults
	assert.NotEmpty(t, body["font_family"])
	assert.NotEmpty(t, body["primary_color"])
}

func TestIntegrationListResumeBuilders(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-list@test.com")

	createResumeBuilder(t, token, "Resume 1")
	createResumeBuilder(t, token, "Resume 2")

	resp := doRequest(t, http.MethodGet, "/api/v1/resume-builder", nil, token)
	assertStatus(t, resp, http.StatusOK)

	body := parseJSON[[]interface{}](t, resp)
	assert.Len(t, body, 2)
}

func TestIntegrationGetResumeBuilder(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-get@test.com")

	rbID := createResumeBuilder(t, token, "Get Resume")

	resp := doRequest(t, http.MethodGet, "/api/v1/resume-builder/"+rbID, nil, token)
	assertStatus(t, resp, http.StatusOK)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, rbID, body["id"])
	assert.Equal(t, "Get Resume", body["title"])
	// Full resume should have section arrays
	assert.NotNil(t, body["experiences"])
	assert.NotNil(t, body["educations"])
	assert.NotNil(t, body["skills"])
}

func TestIntegrationUpdateResumeBuilder(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-update@test.com")

	rbID := createResumeBuilder(t, token, "Original")

	newTitle := "Updated Title"
	newFont := "Georgia"
	newColor := "#FF5733"
	spacing := 80
	resp := doRequest(t, http.MethodPatch, "/api/v1/resume-builder/"+rbID, map[string]interface{}{
		"title":         newTitle,
		"font_family":   newFont,
		"primary_color": newColor,
		"spacing":       spacing,
	}, token)
	assertStatus(t, resp, http.StatusOK)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, newTitle, body["title"])
	assert.Equal(t, newFont, body["font_family"])
	assert.Equal(t, newColor, body["primary_color"])
	assert.Equal(t, float64(spacing), body["spacing"])
}

func TestIntegrationUpdateInvalidFont(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-badfont@test.com")

	rbID := createResumeBuilder(t, token, "Font Test")

	resp := doRequest(t, http.MethodPatch, "/api/v1/resume-builder/"+rbID, map[string]interface{}{
		"font_family": "ComicSansXYZ_NOT_REAL",
	}, token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assertErrorCode(t, resp, "INVALID_FONT")
}

func TestIntegrationUpdateInvalidColor(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-badcolor@test.com")

	rbID := createResumeBuilder(t, token, "Color Test")

	resp := doRequest(t, http.MethodPatch, "/api/v1/resume-builder/"+rbID, map[string]interface{}{
		"primary_color": "not-a-color",
	}, token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assertErrorCode(t, resp, "INVALID_COLOR")
}

func TestIntegrationUpdateInvalidSpacing(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-badspace@test.com")

	rbID := createResumeBuilder(t, token, "Spacing Test")

	resp := doRequest(t, http.MethodPatch, "/api/v1/resume-builder/"+rbID, map[string]interface{}{
		"spacing": 200,
	}, token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assertErrorCode(t, resp, "INVALID_SPACING")
}

func TestIntegrationDeleteResumeBuilder(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-delete@test.com")

	rbID := createResumeBuilder(t, token, "Delete Me")

	resp := doRequest(t, http.MethodDelete, "/api/v1/resume-builder/"+rbID, nil, token)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Verify deleted
	resp = doRequest(t, http.MethodGet, "/api/v1/resume-builder/"+rbID, nil, token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationDuplicate(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-dup@test.com")

	rbID := createResumeBuilder(t, token, "Original Resume")

	// Add a contact to the original
	doRequest(t, http.MethodPut, "/api/v1/resume-builder/"+rbID+"/contact", map[string]string{
		"full_name": "John Doe",
		"email":     "john@example.com",
	}, token).Body.Close()

	// Duplicate
	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder/"+rbID+"/duplicate", nil, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.NotEqual(t, rbID, body["id"])
	assert.Contains(t, body["title"], "(Copy)")
}

// --- 1:1 Sections ---

func TestIntegrationUpsertContact(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-contact@test.com")

	rbID := createResumeBuilder(t, token, "Contact Test")

	resp := doRequest(t, http.MethodPut, "/api/v1/resume-builder/"+rbID+"/contact", map[string]string{
		"full_name": "Jane Smith",
		"email":     "jane@example.com",
		"phone":     "+1234567890",
		"location":  "New York, NY",
	}, token)
	assertStatus(t, resp, http.StatusOK)

	// Verify via GET
	resp = doRequest(t, http.MethodGet, "/api/v1/resume-builder/"+rbID, nil, token)
	assertStatus(t, resp, http.StatusOK)
	body := parseJSON[map[string]interface{}](t, resp)
	contact := body["contact"].(map[string]interface{})
	assert.Equal(t, "Jane Smith", contact["full_name"])
}

func TestIntegrationUpsertSummary(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-summary@test.com")

	rbID := createResumeBuilder(t, token, "Summary Test")

	resp := doRequest(t, http.MethodPut, "/api/v1/resume-builder/"+rbID+"/summary", map[string]string{
		"content": "Experienced software engineer with 10 years of expertise.",
	}, token)
	assertStatus(t, resp, http.StatusOK)

	resp = doRequest(t, http.MethodGet, "/api/v1/resume-builder/"+rbID, nil, token)
	assertStatus(t, resp, http.StatusOK)
	body := parseJSON[map[string]interface{}](t, resp)
	summary := body["summary"].(map[string]interface{})
	assert.Equal(t, "Experienced software engineer with 10 years of expertise.", summary["content"])
}

// --- 1:N Sections ---

func TestIntegrationCreateExperience(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-exp@test.com")

	rbID := createResumeBuilder(t, token, "Experience Test")

	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder/"+rbID+"/experiences", map[string]interface{}{
		"company":     "Acme Corp",
		"position":    "Senior Developer",
		"location":    "Remote",
		"start_date":  "2020-01",
		"end_date":    "2023-12",
		"is_current":  false,
		"description": "Built amazing stuff",
		"sort_order":  0,
	}, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.NotEmpty(t, body["id"])
	assert.Equal(t, "Acme Corp", body["company"])
}

func TestIntegrationUpdateExperience(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-updexp@test.com")

	rbID := createResumeBuilder(t, token, "Update Exp")

	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder/"+rbID+"/experiences", map[string]interface{}{
		"company":  "Old Corp",
		"position": "Junior Dev",
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	created := parseJSON[map[string]interface{}](t, resp)
	entryID := created["id"].(string)

	newCompany := "New Corp"
	resp = doRequest(t, http.MethodPatch, "/api/v1/resume-builder/"+rbID+"/experiences/"+entryID, map[string]interface{}{
		"company": newCompany,
	}, token)
	assertStatus(t, resp, http.StatusOK)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, newCompany, body["company"])
}

func TestIntegrationDeleteExperience(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-delexp@test.com")

	rbID := createResumeBuilder(t, token, "Delete Exp")

	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder/"+rbID+"/experiences", map[string]interface{}{
		"company":  "To Delete",
		"position": "Dev",
	}, token)
	assertStatus(t, resp, http.StatusCreated)
	created := parseJSON[map[string]interface{}](t, resp)
	entryID := created["id"].(string)

	resp = doRequest(t, http.MethodDelete, "/api/v1/resume-builder/"+rbID+"/experiences/"+entryID, nil, token)
	assertStatus(t, resp, http.StatusNoContent)
	resp.Body.Close()

	// Verify removed from GET
	resp = doRequest(t, http.MethodGet, "/api/v1/resume-builder/"+rbID, nil, token)
	assertStatus(t, resp, http.StatusOK)
	body := parseJSON[map[string]interface{}](t, resp)
	exps := body["experiences"].([]interface{})
	assert.Empty(t, exps)
}

func TestIntegrationCreateSkill(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-skill@test.com")

	rbID := createResumeBuilder(t, token, "Skill Test")

	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder/"+rbID+"/skills", map[string]interface{}{
		"name":  "Go",
		"level": "Expert",
	}, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, "Go", body["name"])
}

func TestIntegrationCreateEducation(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-edu@test.com")

	rbID := createResumeBuilder(t, token, "Education Test")

	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder/"+rbID+"/educations", map[string]interface{}{
		"institution":   "MIT",
		"degree":        "B.S.",
		"field_of_study": "Computer Science",
		"start_date":    "2016-09",
		"end_date":      "2020-06",
	}, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, "MIT", body["institution"])
}

func TestIntegrationCreateLanguage(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-lang@test.com")

	rbID := createResumeBuilder(t, token, "Language Test")

	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder/"+rbID+"/languages", map[string]interface{}{
		"name":        "English",
		"proficiency": "Native",
	}, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, "English", body["name"])
}

func TestIntegrationCreateCertification(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-cert@test.com")

	rbID := createResumeBuilder(t, token, "Cert Test")

	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder/"+rbID+"/certifications", map[string]interface{}{
		"name":   "AWS Solutions Architect",
		"issuer": "Amazon",
	}, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, "AWS Solutions Architect", body["name"])
}

func TestIntegrationCreateProject(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-proj@test.com")

	rbID := createResumeBuilder(t, token, "Project Test")

	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder/"+rbID+"/projects", map[string]interface{}{
		"name":        "Open Source Tool",
		"description": "A helpful CLI tool",
	}, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, "Open Source Tool", body["name"])
}

func TestIntegrationCreateVolunteering(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-vol@test.com")

	rbID := createResumeBuilder(t, token, "Volunteering Test")

	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder/"+rbID+"/volunteering", map[string]interface{}{
		"organization": "Red Cross",
		"role":         "Volunteer",
	}, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, "Red Cross", body["organization"])
}

func TestIntegrationCreateCustomSection(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-custom@test.com")

	rbID := createResumeBuilder(t, token, "Custom Section Test")

	resp := doRequest(t, http.MethodPost, "/api/v1/resume-builder/"+rbID+"/custom-sections", map[string]interface{}{
		"title":   "Awards",
		"content": "Employee of the Month - June 2023",
	}, token)
	assertStatus(t, resp, http.StatusCreated)

	body := parseJSON[map[string]interface{}](t, resp)
	assert.Equal(t, "Awards", body["title"])
}

// --- Section Order ---

func TestIntegrationUpdateSectionOrder(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-order@test.com")

	rbID := createResumeBuilder(t, token, "Order Test")

	resp := doRequest(t, http.MethodPut, "/api/v1/resume-builder/"+rbID+"/section-order", map[string]interface{}{
		"sections": []map[string]interface{}{
			{"section_key": "experience", "sort_order": 0, "is_visible": true},
			{"section_key": "education", "sort_order": 1, "is_visible": true},
			{"section_key": "skills", "sort_order": 2, "is_visible": false},
		},
	}, token)
	assertStatus(t, resp, http.StatusOK)

	// Verify via GET
	resp = doRequest(t, http.MethodGet, "/api/v1/resume-builder/"+rbID, nil, token)
	assertStatus(t, resp, http.StatusOK)
	body := parseJSON[map[string]interface{}](t, resp)
	sectionOrder := body["section_order"].([]interface{})
	require.GreaterOrEqual(t, len(sectionOrder), 3)
}

func TestIntegrationInvalidSectionKey(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-badkey@test.com")

	rbID := createResumeBuilder(t, token, "Bad Key Test")

	resp := doRequest(t, http.MethodPut, "/api/v1/resume-builder/"+rbID+"/section-order", map[string]interface{}{
		"sections": []map[string]interface{}{
			{"section_key": "invalid_section_xyz", "sort_order": 0, "is_visible": true},
		},
	}, token)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assertErrorCode(t, resp, "INVALID_SECTION_KEY")
}

// --- Ownership ---

func TestIntegrationOwnershipDenied(t *testing.T) {
	cleanupAll(t)
	_, tokenA := setupProUser(t, "rb-ownerA@test.com")
	_, tokenB := setupProUser(t, "rb-ownerB@test.com")

	rbID := createResumeBuilder(t, tokenA, "Owner A's Resume")

	// User B tries to GET
	resp := doRequest(t, http.MethodGet, "/api/v1/resume-builder/"+rbID, nil, tokenB)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// User B tries to PATCH
	resp = doRequest(t, http.MethodPatch, "/api/v1/resume-builder/"+rbID, map[string]interface{}{
		"title": "Hacked",
	}, tokenB)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()

	// User B tries to DELETE
	resp = doRequest(t, http.MethodDelete, "/api/v1/resume-builder/"+rbID, nil, tokenB)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
}

func TestIntegrationUpdateTemplateID(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-template@test.com")

	newTemplateIDs := []struct {
		name string
		uuid string
	}{
		{"bold", "00000000-0000-0000-0000-000000000009"},
		{"accent", "00000000-0000-0000-0000-00000000000a"},
		{"timeline", "00000000-0000-0000-0000-00000000000b"},
		{"vivid", "00000000-0000-0000-0000-00000000000c"},
	}

	for _, tc := range newTemplateIDs {
		t.Run(tc.name+" template succeeds", func(t *testing.T) {
			rbID := createResumeBuilder(t, token, "Template "+tc.name)

			resp := doRequest(t, http.MethodPatch, "/api/v1/resume-builder/"+rbID, map[string]interface{}{
				"template_id": tc.uuid,
			}, token)
			assertStatus(t, resp, http.StatusOK)

			body := parseJSON[map[string]interface{}](t, resp)
			assert.Equal(t, tc.uuid, body["template_id"])
		})
	}

	t.Run("invalid template ID returns error", func(t *testing.T) {
		rbID := createResumeBuilder(t, token, "Template Invalid")

		resp := doRequest(t, http.MethodPatch, "/api/v1/resume-builder/"+rbID, map[string]interface{}{
			"template_id": "00000000-0000-0000-0000-ffffffffffff",
		}, token)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assertErrorCode(t, resp, "INVALID_TEMPLATE")
	})
}

func TestIntegrationResumeBuilderNotFound(t *testing.T) {
	cleanupAll(t)
	_, token := setupProUser(t, "rb-notfound@test.com")

	resp := doRequest(t, http.MethodGet, "/api/v1/resume-builder/00000000-0000-0000-0000-000000000000", nil, token)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}
