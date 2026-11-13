package mcp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResumeSectionValidation(t *testing.T) {
	tests := []UpdateResumeSectionInput{
		{ResumeID: resumeA, Section: "summary", Op: "create", Data: map[string]any{"content": "x"}},
		{ResumeID: resumeA, Section: "contact", Op: "update", EntryID: entryA, Data: map[string]any{"full_name": "x"}},
		{ResumeID: resumeA, Section: "skill", Op: "upsert", Data: map[string]any{"name": "x"}},
		{ResumeID: resumeA, Section: "skill", Op: "delete", EntryID: entryA, Data: map[string]any{"name": "x"}},
		{ResumeID: resumeA, Section: "skills", Op: "create", Data: map[string]any{"name": "x"}},
		{ResumeID: resumeA, Section: "skill", Op: "update", Data: map[string]any{"name": "x"}},
		{ResumeID: resumeA, Section: "skill", Op: "update", EntryID: "invalid", Data: map[string]any{"name": "x"}},
		{ResumeID: "invalid", Section: "skill", Op: "create", Data: map[string]any{"name": "x"}},
		{ResumeID: resumeA, Section: "skill", Op: "create", EntryID: entryA, Data: map[string]any{"name": "x"}},
		{ResumeID: resumeA, Section: "summary", Op: "upsert", EntryID: entryA, Data: map[string]any{"content": "x"}},
	}
	for i, input := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			store := newFakeResumeStore()
			result := callTool(t, resumeToolClient(t, store, userA), "update_resume_section", input)
			require.True(t, result.IsError)
			require.Empty(t, store.events)
		})
	}
}

func TestResumeSectionDataValidation(t *testing.T) {
	for _, section := range resumeSectionCases() {
		for _, op := range resumeOperations(section.section) {
			for _, invalid := range []map[string]any{
				{}, {section.field: nil}, {section.field: 42}, {section.field: "bad\x00text"},
				{section.field: "bad\x1btext"}, {section.field: "bad\u007ftext"},
				{section.field: "valid", "sort_order": 2}, {section.field: "valid", "SORT_ORDER": 2},
				{section.field: "valid", "entry_id": entryB}, {section.field: "valid", "unknown": "value"},
				{section.field: "valid", "is_current": "yes"},
			} {
				t.Run(fmt.Sprintf("%s/%s/%v", section.section, op, invalid), func(t *testing.T) {
					store := newFakeResumeStore()
					input := UpdateResumeSectionInput{ResumeID: resumeA, Section: section.section, Op: op, Data: invalid}
					if op == "update" {
						input.EntryID = entryA
					}
					result := callTool(t, resumeToolClient(t, store, userA), "update_resume_section", input)
					require.True(t, result.IsError)
					require.Empty(t, store.events)
					require.Empty(t, store.snapshots)
				})
			}
		}
	}
}

func TestResumeLengthLimits(t *testing.T) {
	limits := map[string]map[string]int{
		"contact":    {"full_name": 255, "email": 255, "phone": 50, "location": 255, "website": 500, "linkedin": 500, "github": 500},
		"summary":    {"content": 5000},
		"experience": {"company": 255, "position": 255, "location": 255, "start_date": 20, "end_date": 20, "description": 10000},
		"education":  {"institution": 255, "degree": 255, "field_of_study": 255, "start_date": 20, "end_date": 20, "gpa": 20, "description": 10000},
		"skill":      {"name": 100, "level": 50}, "language": {"name": 100, "proficiency": 50},
		"certification": {"name": 255, "issuer": 255, "issue_date": 20, "expiry_date": 20, "url": 500},
		"project":       {"name": 255, "url": 500, "start_date": 20, "end_date": 20, "description": 10000},
		"volunteering":  {"organization": 255, "role": 255, "start_date": 20, "end_date": 20, "description": 10000},
		"custom":        {"title": 255, "content": 20000},
	}
	for section, fields := range limits {
		for _, op := range resumeOperations(section) {
			for field, max := range fields {
				t.Run(section+"/"+op+"/"+field, func(t *testing.T) {
					store := newFakeResumeStore()
					client := resumeToolClient(t, store, userA)
					input := UpdateResumeSectionInput{ResumeID: resumeA, Section: section, Op: op, Data: map[string]any{field: strings.Repeat("Ж", max+1)}}
					if op == "update" {
						input.EntryID = entryA
					}
					requireResumeError(t, callTool(t, client, "update_resume_section", input), "length limit")
					require.Empty(t, store.events)
					input.Data = map[string]any{field: strings.Repeat("Ж", max)}
					decodeOutput[UpdateResumeSectionOutput](t, callTool(t, client, "update_resume_section", input))
				})
			}
		}
	}
}

func TestResumeReadAndRestoreUUIDValidation(t *testing.T) {
	store := newFakeResumeStore()
	client := resumeToolClient(t, store, userA)
	for _, id := range []string{"bad", "00000000-0000-0000-0000-000000000000"} {
		for _, tool := range []string{"get_resume", "list_resume_snapshots"} {
			requireResumeError(t, callTool(t, client, tool, GetResumeInput{ResumeID: id}), "valid UUID")
		}
		requireResumeError(t, callTool(t, client, "restore_resume_snapshot", RestoreResumeSnapshotInput{SnapshotID: id}), "valid UUID")
	}
	require.Empty(t, store.events)
}
