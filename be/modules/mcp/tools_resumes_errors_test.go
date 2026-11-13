package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	rb "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	sub "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestResumeSnapshotFailurePreventsWrite(t *testing.T) {
	for _, section := range resumeSectionCases() {
		for _, op := range resumeOperations(section.section) {
			t.Run(section.section+"/"+op, func(t *testing.T) {
				store := newFakeResumeStore()
				store.snapshotErr = errors.New("SQL private resume content")
				client := resumeToolClient(t, store, userA)
				input := UpdateResumeSectionInput{ResumeID: resumeA, Section: section.section, Op: op, Data: section.data}
				if op == "update" {
					input.EntryID = entryA
				}
				requireResumeError(t, callTool(t, client, "update_resume_section", input), "Could not save an undo snapshot")
				require.Len(t, store.events, 2)
				require.Empty(t, store.snapshots)
			})
		}
	}
}

func TestResumeErrorsAreFriendlyAndPrivate(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{err: rb.ErrResumeBuilderNotFound, want: "not found"},
		{err: rb.ErrNotOwner, want: "not found"},
		{err: rb.ErrSectionEntryNotFound, want: "Section entry not found"},
		{err: rb.ErrUnsupportedSnapshotVersion, want: "unsupported version"},
		{err: rb.ErrInvalidSnapshotContent, want: "incomplete or invalid"},
		{err: sub.ErrLimitReached, want: "resume limit"},
		{err: sub.ErrPaidFeature, want: "Upgrade to Pro or Enterprise"},
		{err: errors.New("private resume content"), want: "Something went wrong"},
	} {
		t.Run(test.want, func(t *testing.T) {
			store := newFakeResumeStore()
			store.writeErr = fmt.Errorf("section write: %w", test.err)
			core, logs := observer.New(zap.ErrorLevel)
			jobs := newFakeJobs()
			server := NewServer(jobs, nil, jobs, zap.New(core), WithResumeTools(ResumeTools{
				Builder: store, Uploaded: &fakeUploadedResumes{}, Snapshots: &fakeResumeSnapshots{store: store},
			}))
			client := toolClientForServer(t, server, userA)
			requireResumeError(t, callTool(t, client, "update_resume_section", UpdateResumeSectionInput{
				ResumeID: resumeA, Section: "summary", Op: "upsert", Data: map[string]any{"content": "sensitive argument"},
			}), test.want)
			encoded, err := json.Marshal(logs.All())
			require.NoError(t, err)
			require.NotContains(t, string(encoded), "private resume content")
			require.NotContains(t, string(encoded), "sensitive argument")
			if test.want == "Something went wrong" {
				require.Len(t, logs.All(), 1)
			}
		})
	}
}

func TestResumeRestoreDomainErrors(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{err: rb.ErrUnsupportedSnapshotVersion, want: "unsupported version"},
		{err: rb.ErrInvalidSnapshotContent, want: "incomplete or invalid"},
		{err: sub.ErrLimitReached, want: "resume limit"},
		{err: errors.New("database private resume"), want: "Something went wrong"},
	} {
		t.Run(test.want, func(t *testing.T) {
			store := newFakeResumeStore()
			service := &fakeResumeSnapshots{store: store}
			require.NoError(t, service.SnapshotBeforeWrite(t.Context(), userA, resumeA, "mcp"))
			store.snapshotErr = test.err
			client := resumeToolClient(t, store, userA)
			for id := range store.snapshots {
				requireResumeError(t, callTool(t, client, "restore_resume_snapshot", RestoreResumeSnapshotInput{SnapshotID: id}), test.want)
			}
			require.Len(t, store.snapshots, 1)
		})
	}
}
