package service

import (
	"encoding/json"
	"maps"
	"sort"
	"testing"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type invalidSnapshotCase struct {
	name, content string
	want          error
}

type nestedSnapshotFixture struct {
	section, object string
	collection      bool
}

// Each literal object spells out its historical fields independently of the validator.
// Blank text/dates/URLs, zero ordering, and false flags are legitimate stored values.
func nestedSnapshotFixtures() []nestedSnapshotFixture {
	return []nestedSnapshotFixture{
		{"contact", `{"full_name":"","email":"","phone":"","location":"","website":"","linkedin":"","github":""}`, false},
		{"summary", `{"content":""}`, false},
		{"experiences", `{"id":"entry","company":"","position":"","location":"","start_date":"","end_date":"","is_current":false,"description":"","sort_order":0}`, true},
		{"educations", `{"id":"entry","institution":"","degree":"","field_of_study":"","start_date":"","end_date":"","is_current":false,"gpa":"","description":"","sort_order":0}`, true},
		{"skills", `{"id":"entry","name":"","level":"","sort_order":0}`, true},
		{"languages", `{"id":"entry","name":"","proficiency":"","sort_order":0}`, true},
		{"certifications", `{"id":"entry","name":"","issuer":"","issue_date":"","expiry_date":"","url":"","sort_order":0}`, true},
		{"projects", `{"id":"entry","name":"","url":"","start_date":"","end_date":"","description":"","sort_order":0}`, true},
		{"volunteering", `{"id":"entry","organization":"","role":"","start_date":"","end_date":"","description":"","sort_order":0}`, true},
		{"custom_sections", `{"id":"entry","title":"","content":"","sort_order":0}`, true},
		{"section_order", `{"section_key":"skills","sort_order":0,"is_visible":false,"column":"main"}`, true},
	}
}

func nestedInvalidSnapshotCases(t *testing.T) []invalidSnapshotCase {
	t.Helper()
	var cases []invalidSnapshotCase
	for _, fixture := range nestedSnapshotFixtures() {
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(fixture.object), &fields))
		names := make([]string, 0, len(fields))
		for name := range fields {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			for _, variant := range []string{"missing", "null"} {
				altered := maps.Clone(fields)
				if variant == "missing" {
					delete(altered, name)
				} else {
					altered[name] = json.RawMessage("null")
				}
				data, err := json.Marshal(altered)
				require.NoError(t, err)
				if fixture.collection {
					data = []byte("[" + string(data) + "]")
				}
				cases = append(cases, invalidSnapshotCase{fixture.section + "/" + variant + "/" + name,
					snapshotWithSection(t, fixture.section, data), model.ErrInvalidSnapshotContent})
			}
		}
	}
	return cases
}

func snapshotWithSection(t *testing.T, section string, data json.RawMessage) string {
	t.Helper()
	var envelope, resume map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(snapshotV1Fixture), &envelope))
	require.NoError(t, json.Unmarshal(envelope["resume"], &resume))
	resume[section] = data
	encoded, err := json.Marshal(resume)
	require.NoError(t, err)
	envelope["resume"] = encoded
	encoded, err = json.Marshal(envelope)
	require.NoError(t, err)
	return string(encoded)
}

func TestSnapshotFormat_PreservesExplicitNestedValues(t *testing.T) {
	for _, fixture := range nestedSnapshotFixtures() {
		t.Run(fixture.section, func(t *testing.T) {
			data := fixture.object
			if fixture.collection {
				data = "[" + data + "]"
			}
			content := snapshotWithSection(t, fixture.section, json.RawMessage(data))
			full, err := decodeResumeSnapshot(&model.ResumeSnapshot{ResumeBuilderID: "resume", Content: json.RawMessage(content)})
			require.NoError(t, err)
			encoded, err := encodeResumeSnapshot(full)
			require.NoError(t, err)
			assert.JSONEq(t, content, string(encoded))
		})
	}
}

func TestSnapshotFormat_PreservesAbsentSectionsAndEmptyCollections(t *testing.T) {
	for _, fixture := range nestedSnapshotFixtures() {
		values := []string{"null"}
		if fixture.collection {
			values = append(values, "[]")
		}
		for _, value := range values {
			t.Run(fixture.section+"/"+value, func(t *testing.T) {
				content := snapshotWithSection(t, fixture.section, json.RawMessage(value))
				full, err := decodeResumeSnapshot(&model.ResumeSnapshot{ResumeBuilderID: "resume", Content: json.RawMessage(content)})
				require.NoError(t, err)
				encoded, err := encodeResumeSnapshot(full)
				require.NoError(t, err)
				assert.JSONEq(t, content, string(encoded))
			})
		}
	}
}
