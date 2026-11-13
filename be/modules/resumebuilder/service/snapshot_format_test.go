package service

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Historical fixtures are literal storage documents, independent of today's DTO marshaler.
const snapshotV1Fixture = `{
 "format_version":1,"resume":{
 "id":"resume","title":"Historical","template_id":"00000000-0000-0000-0000-000000000001",
 "font_family":"Georgia","primary_color":"#2563eb","text_color":"#111827","spacing":100,
 "margin_top":0,"margin_bottom":40,"margin_left":40,"margin_right":40,
 "layout_mode":"single","sidebar_width":35,"font_size":12,"skill_display":"",
 "created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z",
 "contact":null,"summary":{"content":"Historical summary"},"experiences":null,"educations":null,
 "skills":[{"id":"skill","name":"Go","level":"expert","sort_order":0}],"languages":null,
 "certifications":null,"projects":null,"volunteering":null,"custom_sections":null,"section_order":null
 }}`

const snapshotV0Fixture = `{"id":"resume","title":"Historical","margin_top":0,
 "summary":{"content":"Historical summary"},
 "skills":[{"id":"skill","name":"Go","level":"expert","sort_order":0}]}`

func TestSnapshotFormat_V1LiteralRoundTrip(t *testing.T) {
	full, err := decodeResumeSnapshot(&model.ResumeSnapshot{ResumeBuilderID: "resume", Content: json.RawMessage(snapshotV1Fixture)})
	require.NoError(t, err)
	assert.Equal(t, DefaultFontSize, full.FontSize)
	assert.Equal(t, "Historical summary", full.Summary.Content)
	encoded, err := encodeResumeSnapshot(full)
	require.NoError(t, err)
	assert.JSONEq(t, snapshotV1Fixture, string(encoded))
}

func TestSnapshotFormat_V0DefaultsMissingFields(t *testing.T) {
	full, err := decodeResumeSnapshot(&model.ResumeSnapshot{ResumeBuilderID: "resume", Content: json.RawMessage(snapshotV0Fixture)})
	require.NoError(t, err)
	assert.Equal(t, DefaultFontSize, full.FontSize)
	assert.Equal(t, DefaultTemplateID, full.TemplateID)
	assert.Equal(t, DefaultFontFamily, full.FontFamily)
	assert.Equal(t, DefaultMargin, full.MarginBottom)
	assert.Zero(t, full.MarginTop, "an explicitly stored zero margin must survive migration")
	encoded, err := encodeResumeSnapshot(full)
	require.NoError(t, err)
	assert.JSONEq(t, snapshotV1Fixture, string(encoded))
}

func TestSnapshotFormat_RejectBeforeAnyWrite(t *testing.T) {
	cases := []invalidSnapshotCase{
		{"unknown version", `{"format_version":999,"resume":{"id":"resume"}}`, model.ErrUnsupportedSnapshotVersion},
		{"null version", `{"format_version":null,"resume":{"id":"resume"}}`, model.ErrUnsupportedSnapshotVersion},
		{"missing v1 field", strings.Replace(snapshotV1Fixture, `"font_size":12,`, "", 1), model.ErrInvalidSnapshotContent},
		{"null v1 field", strings.Replace(snapshotV1Fixture, `"font_size":12`, `"font_size":null`, 1), model.ErrInvalidSnapshotContent},
		{"null entry", strings.Replace(snapshotV1Fixture, `"experiences":null`, `"experiences":[null]`, 1), model.ErrInvalidSnapshotContent},
	}
	for _, tc := range append(cases, nestedInvalidSnapshotCases(t)...) {
		t.Run(tc.name, func(t *testing.T) {
			repo := newSnapshotTestRepo(t)
			before := snapshotTestClone(t, repo.current)
			repo.snapshots = []*model.ResumeSnapshot{{ID: "historic", UserID: "owner", ResumeBuilderID: "resume", Content: json.RawMessage(tc.content)}}
			history := snapshotTestClone(t, repo.snapshots)
			result, err := NewSnapshotService(repo).RestoreSnapshot(t.Context(), "owner", "historic")
			assert.ErrorIs(t, err, tc.want)
			assert.Nil(t, result)
			assert.Zero(t, repo.writes, "rollback alone would hide an attempted unsafe write")
			assert.Equal(t, before, repo.current)
			assert.Equal(t, history, repo.snapshots)
		})
	}
}

func TestSnapshotFormat_APISchemaChangesRequireVersionDecision(t *testing.T) {
	api := reflect.TypeFor[model.FullResumeDTO]()
	stored := reflect.TypeFor[snapshotResumeV1]()
	var apiSections, storedSections []string
	for _, target := range []struct {
		typeInfo reflect.Type
		names    *[]string
	}{{api, &apiSections}, {stored, &storedSections}} {
		for i := 0; i < target.typeInfo.NumField(); i++ {
			field := target.typeInfo.Field(i)
			if !field.Anonymous {
				*target.names = append(*target.names, field.Name)
			}
		}
	}
	assert.ElementsMatch(t, apiSections, storedSections, "new API sections require an explicit persisted-schema migration")
}
