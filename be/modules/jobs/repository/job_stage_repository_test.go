package repository

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newJobStageRepo(t *testing.T) (*JobStageRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewJobStageRepository(mock), mock
}

var jobStageCols = []string{"id", "job_id", "stage_template_id", "status", "order", "started_at", "completed_at", "created_at"}

func TestJobStageRepository_Create(t *testing.T) {
	t.Run("creates stage successfully", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)
		stage := &model.JobStage{JobID: "job-1", StageTemplateID: "tpl-1", Status: "active", Order: 0, StartedAt: time.Now()}

		mock.ExpectExec("INSERT INTO job_stages").
			WithArgs(pgxmock.AnyArg(), stage.JobID, stage.StageTemplateID, stage.Status, stage.Order, stage.StartedAt, stage.CompletedAt, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), stage)

		require.NoError(t, err)
		assert.NotEmpty(t, stage.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)
		stage := &model.JobStage{JobID: "job-1", StageTemplateID: "tpl-1", Status: "active", StartedAt: time.Now()}

		mock.ExpectExec("INSERT INTO job_stages").
			WithArgs(pgxmock.AnyArg(), stage.JobID, stage.StageTemplateID, stage.Status, stage.Order, stage.StartedAt, stage.CompletedAt, pgxmock.AnyArg()).
			WillReturnError(errDB)

		err := repo.Create(context.Background(), stage)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobStageRepository_GetByID(t *testing.T) {
	t.Run("returns stage successfully", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)
		now := time.Now()

		mock.ExpectQuery("FROM job_stages WHERE id = ").
			WithArgs("stage-1", "job-1").
			WillReturnRows(pgxmock.NewRows(jobStageCols).AddRow(
				"stage-1", "job-1", "tpl-1", "active", 0, now, nil, now,
			))

		stage, err := repo.GetByID(context.Background(), "stage-1", "job-1")

		require.NoError(t, err)
		assert.Equal(t, "stage-1", stage.ID)
		assert.Equal(t, "active", stage.Status)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found on ErrNoRows", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)

		mock.ExpectQuery("FROM job_stages WHERE id = ").
			WithArgs("nope", "job-1").
			WillReturnError(pgx.ErrNoRows)

		stage, err := repo.GetByID(context.Background(), "nope", "job-1")

		assert.Nil(t, stage)
		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)

		mock.ExpectQuery("FROM job_stages WHERE id = ").
			WithArgs("stage-1", "job-1").
			WillReturnError(errDB)

		stage, err := repo.GetByID(context.Background(), "stage-1", "job-1")

		assert.Nil(t, stage)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobStageRepository_ListByJob(t *testing.T) {
	t.Run("returns stages ordered", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)
		now := time.Now()

		mock.ExpectQuery("FROM job_stages WHERE job_id = ").
			WithArgs("job-1").
			WillReturnRows(pgxmock.NewRows(jobStageCols).
				AddRow("stage-1", "job-1", "tpl-1", "completed", 0, now, &now, now).
				AddRow("stage-2", "job-1", "tpl-2", "active", 1, now, nil, now))

		stages, err := repo.ListByJob(context.Background(), "job-1")

		require.NoError(t, err)
		require.Len(t, stages, 2)
		assert.Equal(t, 1, stages[1].Order)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates query error", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)

		mock.ExpectQuery("FROM job_stages WHERE job_id = ").
			WithArgs("job-1").
			WillReturnError(errDB)

		stages, err := repo.ListByJob(context.Background(), "job-1")

		assert.Nil(t, stages)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates scan error", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)
		now := time.Now()

		mock.ExpectQuery("FROM job_stages WHERE job_id = ").
			WithArgs("job-1").
			WillReturnRows(pgxmock.NewRows(jobStageCols).
				AddRow("stage-1", "job-1", "tpl-1", "active", "bad-order", now, nil, now))

		stages, err := repo.ListByJob(context.Background(), "job-1")

		assert.Nil(t, stages)
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobStageRepository_Update(t *testing.T) {
	t.Run("updates stage successfully", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)
		now := time.Now()
		stage := &model.JobStage{ID: "stage-1", Status: "completed", CompletedAt: &now}

		mock.ExpectExec("UPDATE job_stages").
			WithArgs(stage.ID, stage.Status, stage.CompletedAt, stage.JobID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(context.Background(), stage)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)
		stage := &model.JobStage{ID: "nope", Status: "completed"}

		mock.ExpectExec("UPDATE job_stages").
			WithArgs(stage.ID, stage.Status, stage.CompletedAt, stage.JobID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(context.Background(), stage)

		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)
		stage := &model.JobStage{ID: "stage-1", Status: "completed"}

		mock.ExpectExec("UPDATE job_stages").
			WithArgs(stage.ID, stage.Status, stage.CompletedAt, stage.JobID).
			WillReturnError(errDB)

		err := repo.Update(context.Background(), stage)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobStageRepository_Delete(t *testing.T) {
	t.Run("deletes stage successfully", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)

		mock.ExpectExec("DELETE FROM job_stages").
			WithArgs("stage-1", "job-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), "stage-1", "job-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)

		mock.ExpectExec("DELETE FROM job_stages").
			WithArgs("nope", "job-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), "nope", "job-1")

		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newJobStageRepo(t)

		mock.ExpectExec("DELETE FROM job_stages").
			WithArgs("stage-1", "job-1").
			WillReturnError(errDB)

		err := repo.Delete(context.Background(), "stage-1", "job-1")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
