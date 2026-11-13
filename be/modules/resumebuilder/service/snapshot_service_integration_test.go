//go:build integration

package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Uses the same isolated, synthetic-only database guard as tailoring tests.
// The verification database must include migration 000056.
func TestSnapshotPostgresRoundTrip(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 1)
	service := NewSnapshotService(repository.NewSnapshotRepository(f.pool))
	require.NoError(t, service.SnapshotBeforeWrite(t.Context(), f.userID, f.sourceID, "mcp"))
	items, err := service.ListSnapshots(t.Context(), f.userID, f.sourceID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	modifySnapshotIntegrationResume(t, f)
	modified, err := f.repo.GetFullResume(t.Context(), f.userID, f.sourceID)
	require.NoError(t, err)
	result, err := service.RestoreSnapshot(t.Context(), f.userID, items[0].ID)
	require.NoError(t, err, "restore must use its existing transaction connection even with pool size one")
	assert.Equal(t, snapshotContent(t, f.source), snapshotContent(t, result))
	stored, err := f.repo.GetFullResume(t.Context(), f.userID, f.sourceID)
	require.NoError(t, err)
	assert.Equal(t, result, stored, "return the persisted restored DTO")
	items, err = service.ListSnapshots(t.Context(), f.userID, f.sourceID)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "restore", items[0].Source)
	undone, err := service.RestoreSnapshot(t.Context(), f.userID, items[0].ID)
	require.NoError(t, err)
	assert.Equal(t, snapshotContent(t, modified), snapshotContent(t, undone))
	assert.Nil(t, undone.Contact)
	assert.Nil(t, undone.Summary)
	assert.Empty(t, undone.SectionOrder)
}

func modifySnapshotIntegrationResume(t *testing.T, f *tailoringIntegrationFixture) {
	t.Helper()
	repo := repository.NewSnapshotRepository(f.pool)
	err := repo.RunSnapshotTransaction(t.Context(), func(tx ports.SnapshotTransaction) error {
		if err := tx.LockResume(t.Context(), f.userID, f.sourceID); err != nil {
			return err
		}
		metadata, err := tx.GetByID(t.Context(), f.userID, f.sourceID)
		if err != nil {
			return err
		}
		changed := *metadata
		changed.Title, changed.TemplateID, changed.FontFamily = "Changed", "00000000-0000-0000-0000-000000000002", "Arial"
		changed.PrimaryColor, changed.TextColor, changed.Spacing = "#abcdef", "#123456", 120
		changed.MarginTop, changed.MarginBottom, changed.MarginLeft, changed.MarginRight = 15, 25, 35, 45
		changed.LayoutMode, changed.SidebarWidth, changed.FontSize, changed.SkillDisplay = "double-left", 30, 14, "dots"
		if err := tx.Update(t.Context(), &changed); err != nil {
			return err
		}
		if err := tx.ClearResumeSections(t.Context(), f.sourceID); err != nil {
			return err
		}
		return tx.CreateSkill(t.Context(), &model.Skill{ResumeBuilderID: f.sourceID, Name: "Added after snapshot"})
	})
	require.NoError(t, err)
}

func TestSnapshotPostgresDedupePruneAndOwnership(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 1)
	service := NewSnapshotService(repository.NewSnapshotRepository(f.pool))
	now := time.Now().UTC().Truncate(time.Microsecond)
	service.now = func() time.Time { return now }
	for i := 0; i < SnapshotLimit+1; i++ {
		now = now.Add(SnapshotWindow + time.Second)
		require.NoError(t, service.SnapshotBeforeWrite(t.Context(), f.userID, f.sourceID, "mcp"))
		require.NoError(t, service.SnapshotBeforeWrite(t.Context(), f.userID, f.sourceID, "mcp"))
	}
	items, err := service.ListSnapshots(t.Context(), f.userID, f.sourceID)
	require.NoError(t, err)
	require.Len(t, items, SnapshotLimit)
	assert.WithinDuration(t, now, items[0].CreatedAt, 0)
	assert.WithinDuration(t, now.Add(-time.Duration(SnapshotLimit-1)*(SnapshotWindow+time.Second)), items[len(items)-1].CreatedAt, 0)
	stranger := uuid.NewString()
	err = service.SnapshotBeforeWrite(t.Context(), stranger, f.sourceID, "mcp")
	assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
	_, err = service.ListSnapshots(t.Context(), stranger, f.sourceID)
	assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
	_, err = service.RestoreSnapshot(t.Context(), stranger, items[0].ID)
	assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
	_, err = service.RestoreSnapshot(t.Context(), f.userID, items[len(items)-1].ID)
	require.NoError(t, err, "pruning may remove the selected oldest snapshot only after its content is loaded")
	items, err = service.ListSnapshots(t.Context(), f.userID, f.sourceID)
	require.NoError(t, err)
	assert.Len(t, items, SnapshotLimit)
}

func TestSnapshotPostgresRestoreRollback(t *testing.T) {
	f := newTailoringIntegrationFixture(t, 1)
	service := NewSnapshotService(repository.NewSnapshotRepository(f.pool))
	invalid := snapshotTestClone(t, f.source)
	invalid.TemplateID = uuid.NewString() // FK failure occurs after the undo copy is inserted.
	content, err := json.Marshal(invalid)
	require.NoError(t, err)
	id := uuid.NewString()
	_, err = f.pool.Exec(t.Context(), `INSERT INTO resume_builder_snapshots
		(id,resume_builder_id,user_id,source,content) VALUES($1,$2,$3,'mcp',$4)`, id, f.sourceID, f.userID, content)
	require.NoError(t, err)
	result, err := service.RestoreSnapshot(t.Context(), f.userID, id)
	require.Error(t, err)
	assert.Nil(t, result)
	current, err := f.repo.GetFullResume(t.Context(), f.userID, f.sourceID)
	require.NoError(t, err)
	assert.Equal(t, f.source, current)
	items, err := service.ListSnapshots(t.Context(), f.userID, f.sourceID)
	require.NoError(t, err)
	assert.Len(t, items, 1, "failed restore must roll back the undo copy")
}
