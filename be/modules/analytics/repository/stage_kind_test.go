package repository

import (
	"fmt"
	"strings"
	"testing"

	jobsModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The queries in this package spell the stage kinds out as SQL literals, while
// the jobs module owns the enum. These tests are the seam: they fail if the two
// ever drift apart, or if a name/order-based rule creeps back into the
// predicates that decide what counts as a response or an interview.

func TestResponseKindsMatchTheModel(t *testing.T) {
	var want []string
	for _, kind := range jobsModel.ResponseStageKinds {
		want = append(want, fmt.Sprintf("'%s'", kind))
	}
	require.NotEmpty(t, want)

	assert.Equal(t, "("+strings.Join(want, ", ")+")", responseKinds,
		"responseKinds must list exactly the kinds model.StageKind.IsResponse() accepts")

	// Spelled out, so a reader of this test knows what the SQL means.
	for _, kind := range []jobsModel.StageKind{
		jobsModel.StageKindResponse,
		jobsModel.StageKindInterview,
		jobsModel.StageKindOffer,
		jobsModel.StageKindRejected,
	} {
		assert.Contains(t, responseKinds, string(kind))
	}
	for _, kind := range []jobsModel.StageKind{
		jobsModel.StageKindSaved,
		jobsModel.StageKindApplied,
		jobsModel.StageKindOther,
	} {
		assert.NotContains(t, responseKinds, string(kind),
			"%s must not count as the employer responding", kind)
	}
}

func TestPredicatesReadKindsNotLabelsOrOrder(t *testing.T) {
	for name, predicate := range map[string]string{
		"responded": respondedPredicate,
		"interview": interviewPredicate,
	} {
		t.Run(name, func(t *testing.T) {
			assert.Contains(t, predicate, "st.kind")
			assert.NotContains(t, predicate, "st.name",
				"renaming a column must not change what it counts")
			assert.NotContains(t, predicate, `st."order"`,
				"reordering the board must not change what it counts")
			// Both branches: the card may sit in the column now, or have
			// recorded a history row for it.
			assert.Contains(t, predicate, "job_stages js")
			assert.Contains(t, predicate, "j.current_stage_template_id")
		})
	}

	assert.Contains(t, interviewPredicate, "st.kind = 'interview'")
}
