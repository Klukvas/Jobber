package repository

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotRepository_Transaction(t *testing.T) {
	for _, phase := range []string{"success", "begin", "callback", "commit"} {
		t.Run(phase, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()
			failure := errors.New("transaction failure")
			begin := mock.ExpectBegin()
			if phase == "begin" {
				begin.WillReturnError(failure)
			} else if phase == "callback" {
				mock.ExpectRollback()
			} else {
				commit := mock.ExpectCommit()
				if phase == "commit" {
					commit.WillReturnError(failure)
					mock.ExpectRollback()
				}
			}
			called := false
			repo := &SnapshotRepository{newMockRepo(mock)}
			err = repo.RunSnapshotTransaction(t.Context(), func(tx ports.SnapshotTransaction) error {
				called = true
				if phase == "callback" {
					return failure
				}
				return nil
			})
			if phase == "success" {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, failure)
			}
			assert.Equal(t, phase != "begin", called)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSnapshotRepository_ListSnapshots(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	now := time.Now().UTC()
	returnedTime := now.In(time.FixedZone("EET", 2*60*60))
	mock.ExpectQuery("FROM resume_builders WHERE id").WithArgs("resume", "owner").
		WillReturnRows(pgxmock.NewRows(resumeBuilderColumns()).AddRow("resume", "owner", "Title", "template", "Arial", "#112233", "#223344", 100, 40, 40, 40, 40, "single", 35, 12, "dots", now, now))
	mock.ExpectQuery(`SELECT s.id, s.source, s.created_at FROM resume_builder_snapshots s JOIN resume_builders r ON r.id = s.resume_builder_id WHERE s.resume_builder_id = \$1 AND s.user_id = \$2 AND r.user_id = \$2 ORDER BY s.created_at DESC, s.id DESC`).
		WithArgs("resume", "owner").WillReturnRows(pgxmock.NewRows([]string{"id", "source", "created_at"}).
		AddRow("latest", "restore", returnedTime).AddRow("older", "mcp", returnedTime.Add(-time.Hour)))
	items, err := (&SnapshotRepository{newMockRepo(mock)}).ListSnapshots(t.Context(), "owner", "resume")
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "latest", items[0].ID)
	assert.WithinDuration(t, now, items[0].CreatedAt, 0)
	assert.WithinDuration(t, now.Add(-time.Hour), items[1].CreatedAt, 0)
	data, err := json.Marshal(items)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "content")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSnapshotRepository_ListSnapshotsNotOwned(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	mock.ExpectQuery("FROM resume_builders WHERE id").WithArgs("resume", "stranger").WillReturnError(pgx.ErrNoRows)
	items, err := (&SnapshotRepository{newMockRepo(mock)}).ListSnapshots(t.Context(), "stranger", "resume")
	assert.Nil(t, items)
	assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSnapshotTransaction_GetSnapshot(t *testing.T) {
	for _, found := range []bool{true, false} {
		t.Run(map[bool]string{true: "owned", false: "not owned"}[found], func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()
			query := mock.ExpectQuery(`FROM resume_builder_snapshots s JOIN resume_builders r ON r.id = s.resume_builder_id WHERE s.id = \$1 AND s.user_id = \$2 AND r.user_id = \$2`).WithArgs("snapshot", "owner")
			if found {
				query.WillReturnRows(pgxmock.NewRows([]string{"id", "resume_builder_id", "user_id", "source", "content", "created_at"}).
					AddRow("snapshot", "resume", "owner", "mcp", []byte(`{"id":"resume"}`), time.Now()))
			} else {
				query.WillReturnError(pgx.ErrNoRows)
			}
			snapshot, err := (&snapshotTransaction{newMockRepo(mock)}).GetSnapshot(t.Context(), "owner", "snapshot")
			if found {
				require.NoError(t, err)
				assert.Equal(t, "resume", snapshot.ResumeBuilderID)
			} else {
				assert.Nil(t, snapshot)
				assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSnapshotTransaction_WindowInsertPrune(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	now := time.Now().UTC()
	since := now.Add(-30 * time.Minute)
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM resume_builder_snapshots WHERE resume_builder_id = \$1 AND source = \$2 AND created_at > \$3\)`).
		WithArgs("resume", "mcp", since).WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(false))
	snapshot := &model.ResumeSnapshot{ID: "snapshot", ResumeBuilderID: "resume", UserID: "owner", Source: "mcp", Content: json.RawMessage(`{}`), CreatedAt: now}
	mock.ExpectExec("INSERT INTO resume_builder_snapshots").WithArgs("snapshot", "resume", "owner", "mcp", snapshot.Content, now).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(`DELETE FROM resume_builder_snapshots WHERE id IN \( SELECT id FROM resume_builder_snapshots WHERE resume_builder_id = \$1 ORDER BY created_at DESC, id DESC OFFSET \$2\)`).
		WithArgs("resume", 20).WillReturnResult(pgxmock.NewResult("DELETE", 1))
	repo := &snapshotTransaction{newMockRepo(mock)}
	exists, err := repo.HasRecentSnapshot(t.Context(), "resume", "mcp", since)
	require.NoError(t, err)
	assert.False(t, exists)
	before := *snapshot
	require.NoError(t, repo.InsertSnapshot(t.Context(), snapshot))
	assert.Equal(t, before, *snapshot)
	require.NoError(t, repo.PruneSnapshots(t.Context(), "resume", 20))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSnapshotTransaction_LockResume(t *testing.T) {
	for _, owned := range []bool{true, false} {
		t.Run(map[bool]string{true: "owned", false: "not owned"}[owned], func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()
			query := mock.ExpectQuery(`SELECT id FROM resume_builders WHERE id = \$1 AND user_id = \$2 FOR UPDATE`).WithArgs("resume", "owner")
			if owned {
				query.WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("resume"))
				expectSnapshotSectionLocks(mock)
			} else {
				query.WillReturnError(pgx.ErrNoRows)
			}
			err = (&snapshotTransaction{newMockRepo(mock)}).LockResume(t.Context(), "owner", "resume")
			if owned {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func expectSnapshotSectionLocks(mock pgxmock.PgxPoolIface) {
	for _, table := range []string{"resume_contacts", "resume_summaries", "resume_experiences", "resume_educations", "resume_skills", "resume_languages", "resume_certifications", "resume_projects", "resume_volunteering", "resume_custom_sections", "resume_section_orders"} {
		mock.ExpectQuery(`SELECT id FROM ` + table + ` WHERE resume_builder_id = \$1 ORDER BY id FOR UPDATE`).
			WithArgs("resume").WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("entry"))
	}
}

func TestSnapshotTransaction_ClearResumeSections(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	for _, table := range []string{"resume_contacts", "resume_summaries", "resume_experiences", "resume_educations", "resume_skills", "resume_languages", "resume_certifications", "resume_projects", "resume_volunteering", "resume_custom_sections", "resume_section_orders"} {
		mock.ExpectExec(`DELETE FROM ` + table + ` WHERE resume_builder_id = \$1`).WithArgs("resume").WillReturnResult(pgxmock.NewResult("DELETE", 2))
	}
	require.NoError(t, (&snapshotTransaction{newMockRepo(mock)}).ClearResumeSections(context.Background(), "resume"))
	require.NoError(t, mock.ExpectationsWereMet())
}
