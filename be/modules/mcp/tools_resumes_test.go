package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResumeToolsList(t *testing.T) {
	client := resumeToolClient(t, newFakeResumeStore(), userA)
	listed, err := client.ListTools(t.Context(), nil)
	require.NoError(t, err)
	require.Len(t, listed.Tools, 17)
	found := make([]string, 0, 5)
	for _, tool := range listed.Tools {
		if isAITool(tool.Name) || !strings.Contains(tool.Name, "resume") {
			continue
		}
		found = append(found, tool.Name)
		require.NotNil(t, tool.Annotations)
		require.False(t, *tool.Annotations.DestructiveHint)
		readOnly := tool.Name != "update_resume_section" && tool.Name != "restore_resume_snapshot"
		require.Equal(t, readOnly, tool.Annotations.ReadOnlyHint)
		require.Equal(t, readOnly, tool.Annotations.IdempotentHint)
		assertFieldDescriptions(t, tool.InputSchema)
		if !readOnly {
			require.Contains(t, tool.Description, "undo")
		}
		if tool.Name == "update_resume_section" {
			schema, err := resumeFakeConvert[map[string]any](tool.InputSchema)
			require.NoError(t, err)
			properties := (*schema)["properties"].(map[string]any)
			require.Len(t, properties["section"].(map[string]any)["enum"], 10)
			require.Len(t, properties["op"].(map[string]any)["enum"], 3)
		}
	}
	require.ElementsMatch(t, []string{"list_resumes", "get_resume", "update_resume_section", "list_resume_snapshots", "restore_resume_snapshot"}, found)
}

func TestResumeReadTools(t *testing.T) {
	store := newFakeResumeStore()
	client := resumeToolClient(t, store, userA)
	listed := decodeOutput[ListResumesOutput](t, callTool(t, client, "list_resumes", ListResumesInput{}))
	require.Len(t, listed.Resumes, 2)
	require.ElementsMatch(t, []string{"builder", "uploaded"}, []string{listed.Resumes[0].Kind, listed.Resumes[1].Kind})
	encoded, err := json.Marshal(listed)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "private.invalid")
	require.NotContains(t, string(encoded), "private-key")
	full := decodeOutput[GetResumeOutput](t, callTool(t, client, "get_resume", GetResumeInput{ResumeID: "urn:uuid:" + resumeA}))
	require.Equal(t, resumeA, full.Resume.ID)
	require.Equal(t, "Original", full.Resume.Summary.Content)
	require.Equal(t, entryA, full.Resume.Experiences[0].ID)
	require.Equal(t, entryA, full.Resume.CustomSections[0].ID)
	snapshots := decodeOutput[ListResumeSnapshotsOutput](t, callTool(t, client, "list_resume_snapshots", GetResumeInput{ResumeID: resumeA}))
	require.Empty(t, snapshots.Snapshots)
}

func TestResumeListAllUploadedPages(t *testing.T) {
	store := newFakeResumeStore()
	jobs := newFakeJobs()
	client := toolClientForServer(t, NewServer(jobs, nil, jobs, nil, WithResumeTools(ResumeTools{
		Builder: store, Uploaded: &fakeUploadedResumes{count: 201}, Snapshots: &fakeResumeSnapshots{store: store},
	})), userA)
	output := decodeOutput[ListResumesOutput](t, callTool(t, client, "list_resumes", ListResumesInput{}))
	require.Len(t, output.Resumes, 202)
}

type resumeSectionCase struct {
	section, field string
	data           map[string]any
}

func resumeSectionCases() []resumeSectionCase {
	return []resumeSectionCase{
		{section: "contact", field: "full_name", data: map[string]any{"full_name": "Ada", "email": "a@example.com", "phone": "+123", "location": "Kyiv", "website": "https://example.com", "linkedin": "link", "github": "git"}},
		{section: "summary", field: "content", data: map[string]any{"content": "Summary\nwith\ttabs"}},
		{section: "experience", field: "company", data: map[string]any{"company": "Acme", "position": "Engineer", "location": "Remote", "start_date": "2020", "end_date": "2025", "is_current": true, "description": "Built things"}},
		{section: "education", field: "institution", data: map[string]any{"institution": "University", "degree": "MS", "field_of_study": "CS", "start_date": "2018", "end_date": "2020", "is_current": false, "gpa": "4", "description": "Studied"}},
		{section: "skill", field: "name", data: map[string]any{"name": "Go", "level": "Expert"}},
		{section: "language", field: "name", data: map[string]any{"name": "English", "proficiency": "Fluent"}},
		{section: "certification", field: "name", data: map[string]any{"name": "Certified", "issuer": "Issuer", "issue_date": "2020", "expiry_date": "2026", "url": "https://example.com"}},
		{section: "project", field: "name", data: map[string]any{"name": "App", "url": "https://example.com", "start_date": "2020", "end_date": "2021", "description": "A project"}},
		{section: "volunteering", field: "organization", data: map[string]any{"organization": "Charity", "role": "Helper", "start_date": "2020", "end_date": "2021", "description": "Helped"}},
		{section: "custom", field: "title", data: map[string]any{"title": "Awards", "content": "First prize"}},
	}
}

func resumeOperations(section string) []string {
	if section == "contact" || section == "summary" {
		return []string{"upsert"}
	}
	return []string{"create", "update"}
}

func TestResumeSectionOperations(t *testing.T) {
	for _, section := range resumeSectionCases() {
		for _, op := range resumeOperations(section.section) {
			t.Run(section.section+"/"+op, func(t *testing.T) {
				store := newFakeResumeStore()
				client := resumeToolClient(t, store, userA)
				input := UpdateResumeSectionInput{ResumeID: "urn:uuid:" + resumeA, Section: section.section, Op: op, Data: section.data}
				if op == "update" {
					input.EntryID = "urn:uuid:" + entryA
				}
				before, err := json.Marshal(input)
				require.NoError(t, err)
				output := decodeOutput[UpdateResumeSectionOutput](t, callTool(t, client, "update_resume_section", input))
				for key, value := range section.data {
					require.Equal(t, value, output.Entry[key], key)
				}
				if op == "update" {
					require.Equal(t, entryA, output.Entry["id"])
				}
				if op == "create" {
					require.NotEmpty(t, output.Entry["id"])
				}
				require.Contains(t, output.SnapshotNote, "restore_resume_snapshot")
				require.Len(t, store.events, 3)
				require.Equal(t, "get:"+userA+":"+resumeA, store.events[0])
				require.Equal(t, "snapshot:mcp:"+userA+":"+resumeA, store.events[1])
				require.True(t, strings.HasPrefix(store.events[2], "write:"))
				after, err := json.Marshal(input)
				require.NoError(t, err)
				require.Equal(t, before, after)
			})
		}
	}
}

func TestResumeSectionPatchAndUpsert(t *testing.T) {
	client := resumeToolClient(t, newFakeResumeStore(), userA)
	patch := decodeOutput[UpdateResumeSectionOutput](t, callTool(t, client, "update_resume_section", UpdateResumeSectionInput{
		ResumeID: resumeA, Section: "experience", Op: "update", EntryID: entryA, Data: map[string]any{"position": "New", "is_current": false},
	}))
	require.Equal(t, "Original", patch.Entry["company"])
	require.Equal(t, "New", patch.Entry["position"])
	require.Equal(t, false, patch.Entry["is_current"])
	for _, data := range []map[string]any{{"full_name": "Ada", "phone": "123"}, {"full_name": "Grace"}} {
		decodeOutput[UpdateResumeSectionOutput](t, callTool(t, client, "update_resume_section", UpdateResumeSectionInput{
			ResumeID: resumeA, Section: "contact", Op: "upsert", Data: data,
		}))
	}
	full := decodeOutput[GetResumeOutput](t, callTool(t, client, "get_resume", GetResumeInput{ResumeID: resumeA}))
	require.Equal(t, "Grace", full.Resume.Contact.FullName)
	require.Empty(t, full.Resume.Contact.Phone)
}

func TestResumeRestoreRoundTrip(t *testing.T) {
	client := resumeToolClient(t, newFakeResumeStore(), userA)
	decodeOutput[UpdateResumeSectionOutput](t, callTool(t, client, "update_resume_section", UpdateResumeSectionInput{
		ResumeID: resumeA, Section: "summary", Op: "upsert", Data: map[string]any{"content": "Changed"},
	}))
	snapshots := decodeOutput[ListResumeSnapshotsOutput](t, callTool(t, client, "list_resume_snapshots", GetResumeInput{ResumeID: resumeA}))
	require.Len(t, snapshots.Snapshots, 1)
	restore := decodeOutput[RestoreResumeSnapshotOutput](t, callTool(t, client, "restore_resume_snapshot", RestoreResumeSnapshotInput{
		SnapshotID: "urn:uuid:" + snapshots.Snapshots[0].ID,
	}))
	require.Equal(t, resumeA, restore.Resume.ID)
	require.Equal(t, "builder", restore.Resume.Kind)
	require.Contains(t, restore.SnapshotNote, "undo this restore")
	full := decodeOutput[GetResumeOutput](t, callTool(t, client, "get_resume", GetResumeInput{ResumeID: resumeA}))
	require.Equal(t, "Original", full.Resume.Summary.Content)
	snapshots = decodeOutput[ListResumeSnapshotsOutput](t, callTool(t, client, "list_resume_snapshots", GetResumeInput{ResumeID: resumeA}))
	require.Len(t, snapshots.Snapshots, 2)
	for _, snapshot := range snapshots.Snapshots {
		if snapshot.Source == "restore" {
			decodeOutput[RestoreResumeSnapshotOutput](t, callTool(t, client, "restore_resume_snapshot", RestoreResumeSnapshotInput{SnapshotID: snapshot.ID}))
		}
	}
	full = decodeOutput[GetResumeOutput](t, callTool(t, client, "get_resume", GetResumeInput{ResumeID: resumeA}))
	require.Equal(t, "Changed", full.Resume.Summary.Content)
}

func TestResumeToolsRequireIdentity(t *testing.T) {
	client := resumeToolClient(t, newFakeResumeStore(), "")
	for _, test := range []struct {
		name  string
		input any
	}{
		{name: "list_resumes", input: ListResumesInput{}},
		{name: "get_resume", input: GetResumeInput{ResumeID: resumeA}},
		{name: "list_resume_snapshots", input: GetResumeInput{ResumeID: resumeA}},
		{name: "restore_resume_snapshot", input: RestoreResumeSnapshotInput{SnapshotID: entryA}},
		{name: "update_resume_section", input: UpdateResumeSectionInput{ResumeID: resumeA, Section: "summary", Op: "upsert", Data: map[string]any{"content": "New"}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			requireResumeError(t, callTool(t, client, test.name, test.input), "Authentication required")
		})
	}
}

var _ BuilderResumes = (*fakeResumeStore)(nil)
var _ ResumeSnapshots = (*fakeResumeSnapshots)(nil)
var _ UploadedResumes = (*fakeUploadedResumes)(nil)
