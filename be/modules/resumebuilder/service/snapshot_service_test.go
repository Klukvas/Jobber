package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotBeforeWrite_Window(t *testing.T) {
	for _, tc := range []struct {
		name   string
		age    time.Duration
		source string
		count  int
	}{
		{"within window", SnapshotWindow - time.Second, "mcp", 1},
		{"at boundary", SnapshotWindow, "mcp", 2},
		{"after window", SnapshotWindow + time.Second, "mcp", 2},
		{"different source", time.Minute, "web", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := newSnapshotTestRepo(t)
			service := NewSnapshotService(repo)
			now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
			service.now = func() time.Time { return now }
			require.NoError(t, service.SnapshotBeforeWrite(t.Context(), "owner", "resume", "mcp"))
			original := snapshotTestClone(t, repo.current)
			repo.current.Summary.Content = "Edited"
			now = now.Add(tc.age)
			require.NoError(t, service.SnapshotBeforeWrite(t.Context(), "owner", "resume", tc.source))
			assert.Len(t, repo.snapshots, tc.count)
			saved, err := decodeResumeSnapshot(repo.snapshots[len(repo.snapshots)-1])
			require.NoError(t, err)
			assert.Equal(t, original, saved, "the first pre-edit copy must survive deduplication")
		})
	}
}

func TestSnapshotBeforeWrite_PrunesOldestPerResume(t *testing.T) {
	repo := newSnapshotTestRepo(t)
	service := NewSnapshotService(repo)
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	repo.snapshots = []*model.ResumeSnapshot{{ID: "unrelated", ResumeBuilderID: "other", CreatedAt: now}}
	for i := 0; i < SnapshotLimit+1; i++ {
		now = now.Add(SnapshotWindow + time.Second)
		repo.current.Title = fmt.Sprintf("Version %d", i)
		require.NoError(t, service.SnapshotBeforeWrite(t.Context(), "owner", "resume", "mcp"))
	}
	items, err := service.ListSnapshots(t.Context(), "owner", "resume")
	require.NoError(t, err)
	assert.Len(t, items, SnapshotLimit)
	assert.Len(t, repo.snapshots, SnapshotLimit+1, "other resumes are unaffected")
	oldest, err := repo.GetSnapshot(t.Context(), "owner", items[len(items)-1].ID)
	require.NoError(t, err)
	full, err := decodeResumeSnapshot(oldest)
	require.NoError(t, err)
	assert.Equal(t, "Version 1", full.Title)
	assert.True(t, items[0].CreatedAt.After(items[len(items)-1].CreatedAt))
	data, err := json.Marshal(items)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "content")
	assert.NotContains(t, string(data), "summary")
}

func TestSnapshotService_Ownership(t *testing.T) {
	for _, method := range []string{"snapshot", "list", "restore", "restore resume ownership"} {
		t.Run(method, func(t *testing.T) {
			repo := newSnapshotTestRepo(t)
			service := NewSnapshotService(repo)
			require.NoError(t, service.SnapshotBeforeWrite(t.Context(), "owner", "resume", "mcp"))
			before := snapshotTestClone(t, repo.current)
			snapshots := snapshotTestClone(t, repo.snapshots)
			var err error
			switch method {
			case "snapshot":
				err = service.SnapshotBeforeWrite(t.Context(), "stranger", "resume", "mcp")
			case "list":
				_, err = service.ListSnapshots(t.Context(), "stranger", "resume")
			case "restore":
				_, err = service.RestoreSnapshot(t.Context(), "stranger", repo.snapshots[0].ID)
			case "restore resume ownership":
				repo.snapshots[0].ResumeBuilderID = "other"
				_, err = service.RestoreSnapshot(t.Context(), "owner", repo.snapshots[0].ID)
				snapshots = repo.snapshots
			}
			require.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
			assert.Equal(t, before, repo.current)
			assert.Equal(t, snapshots, repo.snapshots)
		})
	}
}

func TestSnapshotBeforeWrite_ConcurrentDedupe(t *testing.T) {
	repo := newSnapshotTestRepo(t)
	service := NewSnapshotService(repo)
	var workers sync.WaitGroup
	results := make(chan error, 10)
	for i := 0; i < cap(results); i++ {
		workers.Go(func() { results <- service.SnapshotBeforeWrite(t.Context(), "owner", "resume", "mcp") })
	}
	workers.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	assert.Len(t, repo.snapshots, 1)
}

func TestRestoreSnapshot_RoundTripAndUnconditionalUndo(t *testing.T) {
	repo := newSnapshotTestRepo(t)
	service := NewSnapshotService(repo)
	original := snapshotTestClone(t, repo.current)
	require.NoError(t, service.SnapshotBeforeWrite(t.Context(), "owner", "resume", "mcp"))
	id := repo.snapshots[0].ID
	// Replace every editable field and section; also add entries absent from the saved copy.
	repo.current = &model.FullResumeDTO{ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: "resume", Title: "Changed"},
		Contact: &model.ContactDTO{FullName: "Changed"}, Summary: &model.SummaryDTO{Content: "Changed"},
		Skills:       []*model.SkillDTO{{ID: "added", Name: "Added"}},
		SectionOrder: []*model.SectionOrderDTO{{SectionKey: "summary", SortOrder: 9, IsVisible: true, Column: "main"}}}
	modified := snapshotTestClone(t, repo.current)
	result, err := service.RestoreSnapshot(t.Context(), "owner", id)
	require.NoError(t, err)
	assert.Equal(t, snapshotContent(t, original), snapshotContent(t, result))
	assert.Equal(t, result, repo.current)
	assert.Len(t, repo.snapshots, 2)
	undo := findSnapshotSource(t, repo.snapshots, "restore")
	saved, err := decodeResumeSnapshot(undo)
	require.NoError(t, err)
	assert.Equal(t, modified, saved)
	_, err = service.RestoreSnapshot(t.Context(), "owner", id)
	require.NoError(t, err)
	assert.Len(t, repo.snapshots, 3, "restores must bypass window dedupe")
	undone, err := service.RestoreSnapshot(t.Context(), "owner", undo.ID)
	require.NoError(t, err)
	assert.Equal(t, snapshotContent(t, modified), snapshotContent(t, undone))
	assert.Empty(t, undone.Experiences, "restore must also remove sections absent in a snapshot")
}

func TestRestoreSnapshot_PrunesAndRollsBackFailures(t *testing.T) {
	for _, phase := range []string{"success", "metadata", "section", "prune", "decode"} {
		t.Run(phase, func(t *testing.T) {
			repo := newSnapshotTestRepo(t)
			service := NewSnapshotService(repo)
			now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
			service.now = func() time.Time { return now }
			for i := 0; i < SnapshotLimit; i++ {
				require.NoError(t, service.SnapshotBeforeWrite(t.Context(), "owner", "resume", "mcp"))
				now = now.Add(SnapshotWindow + time.Second)
			}
			id := repo.snapshots[0].ID
			failure := errors.New("injected failure")
			switch phase {
			case "metadata":
				repo.writeErr = failure
			case "section":
				repo.CreateProjectFunc = func(context.Context, *model.Project) error { return failure }
			case "prune":
				repo.pruneErr = failure
			case "decode":
				repo.snapshots[0].Content = json.RawMessage(`{}`)
			}
			before := snapshotTestClone(t, repo.current)
			snapshots := append([]*model.ResumeSnapshot(nil), repo.snapshots...)
			result, err := service.RestoreSnapshot(t.Context(), "owner", id)
			assertSnapshotRestoreOutcome(t, phase, before, snapshots, result, err, repo)
		})
	}
}

func assertSnapshotRestoreOutcome(t *testing.T, phase string, before *model.FullResumeDTO, snapshots []*model.ResumeSnapshot, result *model.FullResumeDTO, err error, repo *snapshotTestRepo) {
	t.Helper()
	if phase == "success" {
		require.NoError(t, err)
		assert.Len(t, repo.snapshots, SnapshotLimit)
		assert.Equal(t, "restore", repo.snapshots[0].Source)
		return
	}
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, before, repo.current)
	assert.Equal(t, snapshots, repo.snapshots, "failed restore must roll back its undo copy and pruning")
}

func TestSnapshotBeforeWrite_InvalidSource(t *testing.T) {
	for _, source := range []string{"", " ", "123456789012345678901"} {
		t.Run(fmt.Sprintf("%q", source), func(t *testing.T) {
			repo := newSnapshotTestRepo(t)
			err := NewSnapshotService(repo).SnapshotBeforeWrite(t.Context(), "owner", "resume", source)
			assert.ErrorIs(t, err, model.ErrInvalidSnapshotSource)
			assert.Empty(t, repo.snapshots)
		})
	}
}

func snapshotContent(t *testing.T, full *model.FullResumeDTO) map[string]any {
	t.Helper()
	data, err := json.Marshal(full)
	require.NoError(t, err)
	var value map[string]any
	require.NoError(t, json.Unmarshal(data, &value))
	delete(value, "updated_at")
	for _, section := range []string{"experiences", "educations", "skills", "languages", "certifications", "projects", "volunteering", "custom_sections"} {
		if items, ok := value[section].([]any); ok {
			for _, item := range items {
				delete(item.(map[string]any), "id")
			}
		}
	}
	return value
}

func findSnapshotSource(t *testing.T, snapshots []*model.ResumeSnapshot, source string) *model.ResumeSnapshot {
	t.Helper()
	for _, snapshot := range snapshots {
		if snapshot.Source == source {
			return snapshot
		}
	}
	t.Fatalf("snapshot source %q not found", source)
	return nil
}
