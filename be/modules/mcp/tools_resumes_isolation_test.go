package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	rb "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/require"
)

func TestResumeToolsCrossUserIsolation(t *testing.T) {
	store := newFakeResumeStore()
	snapshots := &fakeResumeSnapshots{store: store}
	require.NoError(t, snapshots.SnapshotBeforeWrite(t.Context(), userB, resumeB, "mcp"))
	foreignSnapshot := ""
	for id := range store.snapshots {
		foreignSnapshot = id
	}
	store.events = []string{}
	before, err := json.Marshal(store.resumes[resumeB])
	require.NoError(t, err)
	client := resumeToolClient(t, store, userA)
	listed := decodeOutput[ListResumesOutput](t, callTool(t, client, "list_resumes", ListResumesInput{}))
	for _, resume := range listed.Resumes {
		require.NotEqual(t, resumeB, resume.ID)
	}
	for _, name := range []string{"get_resume", "list_resume_snapshots"} {
		requireResumeError(t, callTool(t, client, name, GetResumeInput{ResumeID: resumeB}), "not found")
	}
	requireResumeError(t, callTool(t, client, "restore_resume_snapshot", RestoreResumeSnapshotInput{SnapshotID: foreignSnapshot}), "not found")
	for _, section := range resumeSectionCases() {
		for _, op := range resumeOperations(section.section) {
			input := UpdateResumeSectionInput{ResumeID: resumeB, Section: section.section, Op: op, Data: section.data}
			if op == "update" {
				input.EntryID = entryB
			}
			requireResumeError(t, callTool(t, client, "update_resume_section", input), "not found")
		}
	}
	after, err := json.Marshal(store.resumes[resumeB])
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Len(t, store.snapshots, 1)
	for _, event := range store.events {
		require.False(t, strings.HasPrefix(event, "snapshot:"), event)
		require.False(t, strings.HasPrefix(event, "write:"), event)
	}
}

func TestResumeForeignEntryRejectedBeforeSnapshot(t *testing.T) {
	for _, section := range resumeSectionCases()[2:] {
		t.Run(section.section, func(t *testing.T) {
			store := newFakeResumeStore()
			client := resumeToolClient(t, store, userA)
			requireResumeError(t, callTool(t, client, "update_resume_section", UpdateResumeSectionInput{
				ResumeID: resumeA, Section: section.section, Op: "update", EntryID: entryB, Data: section.data,
			}), "Section entry not found")
			require.Len(t, store.events, 1)
			require.Empty(t, store.snapshots)
		})
	}
}

func TestResumeSameUserForeignEntryRejectedBeforeSnapshot(t *testing.T) {
	store := newFakeResumeStore()
	store.owners[resumeB] = userA
	before, err := json.Marshal(store.resumes)
	require.NoError(t, err)
	client := resumeToolClient(t, store, userA)
	requireResumeError(t, callTool(t, client, "update_resume_section", UpdateResumeSectionInput{
		ResumeID: resumeA, Section: "experience", Op: "update", EntryID: entryB,
		Data: map[string]any{"position": "Changed"},
	}), "Section entry not found")
	require.Empty(t, store.snapshots)
	require.Equal(t, []string{"get:" + userA + ":" + resumeA}, store.events)
	after, err := json.Marshal(store.resumes)
	require.NoError(t, err)
	require.Equal(t, before, after)
}

func TestResumeUnknownResources(t *testing.T) {
	client := resumeToolClient(t, newFakeResumeStore(), userA)
	for _, name := range []string{"get_resume", "list_resume_snapshots"} {
		requireResumeError(t, callTool(t, client, name, GetResumeInput{ResumeID: entryB}), "not found")
	}
	requireResumeError(t, callTool(t, client, "restore_resume_snapshot", RestoreResumeSnapshotInput{SnapshotID: entryB}), "not found")
}

func TestResumeIdentityCannotComeFromArguments(t *testing.T) {
	store := newFakeResumeStore()
	client := resumeToolClient(t, store, userA)
	requireResumeError(t, callTool(t, client, "get_resume", map[string]any{"resume_id": resumeB, "user_id": userB}), "user_id")
	requireResumeError(t, callTool(t, client, "update_resume_section", map[string]any{
		"resume_id": resumeB, "user_id": userB, "section": "summary", "op": "upsert", "data": map[string]any{"content": "stolen"},
	}), "user_id")
	require.Empty(t, store.events)
}

func TestResumeOwnershipRecheckedAtSnapshot(t *testing.T) {
	store := newFakeResumeStore()
	store.snapshotErr = rb.ErrResumeBuilderNotFound
	client := resumeToolClient(t, store, userA)
	requireResumeError(t, callTool(t, client, "update_resume_section", UpdateResumeSectionInput{
		ResumeID: resumeA, Section: "summary", Op: "upsert", Data: map[string]any{"content": "Changed"},
	}), "not found")
	require.Len(t, store.events, 2)
	require.Empty(t, store.snapshots)
	require.Equal(t, "Original", store.resumes[resumeA]["summary"].(map[string]any)["content"])
}
