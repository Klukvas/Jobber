//go:build integration

package e2e

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Concurrency regression for `PATCH /stage-templates/{id}`.
//
// The endpoint used to read the column, mutate the copy in memory and write
// name, "order" and kind back as one row. A rename and a reclassification
// issued at the same time therefore lost one of the two: whichever statement
// committed second carried the other's pre-edit value back over the column, so
// either the user's new label or the column's new meaning silently disappeared
// — and the meaning is what every analytics rule and the applied_at rule read.
//
// These run against the real migrated schema, where the writes actually
// serialize. The HTTP plumbing (httpCall, runConcurrentCalls,
// requireAllStatus) is shared with jobs_concurrency_test.go, including its rule
// that nothing in a worker goroutine touches *testing.T.

// stageTemplateRow reads a column's stored label, position and meaning.
func stageTemplateRow(t *testing.T, templateID string) (name string, order int, kind string) {
	t.Helper()
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT name, "order", kind FROM stage_templates WHERE id = $1`, templateID).
		Scan(&name, &order, &kind))
	return name, order, kind
}

func TestIntegrationConcurrentStageTemplatePatchesKeepNameAndKind(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "stage-template-patch-kind@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	templateID := cols["Technical Interview"]
	_, seededOrder, _ := stageTemplateRow(t, templateID)

	// Which statement commits second differs from round to round, and a
	// whole-row write loses a field in either ordering. The kind alternates so
	// every round really changes it.
	kinds := []string{"response", "interview"}
	const rounds = 6
	for round := 0; round < rounds; round++ {
		renamed := fmt.Sprintf("Onsite loop %d", round)
		wantKind := kinds[round%len(kinds)]

		results := runConcurrentCalls(
			httpCall{
				method: http.MethodPatch,
				path:   "/api/v1/stage-templates/" + templateID,
				body:   map[string]interface{}{"name": renamed},
				token:  token,
			},
			httpCall{
				method: http.MethodPatch,
				path:   "/api/v1/stage-templates/" + templateID,
				body:   map[string]interface{}{"kind": wantKind},
				token:  token,
			},
		)
		requireAllStatus(t, http.StatusOK, results)

		name, order, kind := stageTemplateRow(t, templateID)
		assert.Equalf(t, renamed, name, "round %d: the rename must stick", round)
		assert.Equalf(t, wantKind, kind,
			"round %d: the reclassification must stick — a rename must not carry the old kind back", round)
		assert.Equalf(t, seededOrder, order, "round %d: neither request asked to move the column", round)
	}
}

func TestIntegrationConcurrentStageTemplatePatchesKeepNameAndOrder(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "stage-template-patch-order@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)

	templateID := cols["Screening"]
	_, _, seededKind := stageTemplateRow(t, templateID)

	const rounds = 6
	for round := 0; round < rounds; round++ {
		renamed := fmt.Sprintf("Recruiter call %d", round)
		wantOrder := 10 + round

		results := runConcurrentCalls(
			httpCall{
				method: http.MethodPatch,
				path:   "/api/v1/stage-templates/" + templateID,
				body:   map[string]interface{}{"name": renamed},
				token:  token,
			},
			httpCall{
				method: http.MethodPatch,
				path:   "/api/v1/stage-templates/" + templateID,
				body:   map[string]interface{}{"order": wantOrder},
				token:  token,
			},
		)
		requireAllStatus(t, http.StatusOK, results)

		name, order, kind := stageTemplateRow(t, templateID)
		assert.Equalf(t, renamed, name, "round %d: the rename must stick", round)
		assert.Equalf(t, wantOrder, order, "round %d: the reorder must stick", round)
		assert.Equalf(t, seededKind, kind, "round %d: neither request asked to reclassify the column", round)
	}
}

func TestIntegrationStageTemplatePatchFieldScoping(t *testing.T) {
	cleanupAll(t)
	userID := seedUser(t, "stage-template-patch-scope@test.com", "securepass123")
	seedSubscription(t, userID, "pro")
	token := authToken(t, userID)
	cols := seedPipelineWithKinds(t, userID, defaultKindPipeline()...)
	templateID := cols["Offer"]

	t.Run("an empty body leaves every field as it was", func(t *testing.T) {
		before, beforeOrder, beforeKind := stageTemplateRow(t, templateID)

		resp := doRequest(t, http.MethodPatch, "/api/v1/stage-templates/"+templateID,
			map[string]interface{}{}, token)
		assertStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		name, order, kind := stageTemplateRow(t, templateID)
		assert.Equal(t, before, name)
		assert.Equal(t, beforeOrder, order)
		assert.Equal(t, beforeKind, kind)
	})

	t.Run("a rename alone changes only the label", func(t *testing.T) {
		_, beforeOrder, beforeKind := stageTemplateRow(t, templateID)

		resp := doRequest(t, http.MethodPatch, "/api/v1/stage-templates/"+templateID,
			map[string]interface{}{"name": "  Offer received  "}, token)
		assertStatus(t, resp, http.StatusOK)
		resp.Body.Close()

		name, order, kind := stageTemplateRow(t, templateID)
		assert.Equal(t, "Offer received", name, "the label is trimmed and stored")
		assert.Equal(t, beforeOrder, order)
		assert.Equal(t, beforeKind, kind)
	})

	t.Run("a blank name is rejected and writes nothing", func(t *testing.T) {
		before, _, _ := stageTemplateRow(t, templateID)

		resp := doRequest(t, http.MethodPatch, "/api/v1/stage-templates/"+templateID,
			map[string]interface{}{"name": "   "}, token)
		assertStatus(t, resp, http.StatusBadRequest)
		assertErrorCode(t, resp, "STAGE_NAME_REQUIRED")

		name, _, _ := stageTemplateRow(t, templateID)
		assert.Equal(t, before, name)
	})

	t.Run("an unknown column is a 404", func(t *testing.T) {
		resp := doRequest(t, http.MethodPatch, "/api/v1/stage-templates/"+uuid.New().String(),
			map[string]interface{}{"name": "Nowhere"}, token)
		assertStatus(t, resp, http.StatusNotFound)
		assertErrorCode(t, resp, "STAGE_TEMPLATE_NOT_FOUND")
	})

	t.Run("another user's column is a 404 and stays untouched", func(t *testing.T) {
		otherID := seedUser(t, "stage-template-patch-other@test.com", "securepass123")
		seedSubscription(t, otherID, "pro")
		otherCols := seedPipelineWithKinds(t, otherID, pipelineColumn{Name: "Their Offer", Kind: "offer"})
		foreign := otherCols["Their Offer"]

		resp := doRequest(t, http.MethodPatch, "/api/v1/stage-templates/"+foreign,
			map[string]interface{}{"name": "Stolen", "kind": "rejected"}, token)
		assertStatus(t, resp, http.StatusNotFound)
		assertErrorCode(t, resp, "STAGE_TEMPLATE_NOT_FOUND")

		name, _, kind := stageTemplateRow(t, foreign)
		assert.Equal(t, "Their Offer", name)
		assert.Equal(t, "offer", kind)
	})
}
