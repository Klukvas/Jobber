package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errDB = errors.New("boom: db failure")

func newJobRepo(t *testing.T) (*JobRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewJobRepository(mock), mock
}

func TestJobRepository_Create(t *testing.T) {
	t.Run("creates job successfully", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		stageID := "stage-wishlist"
		job := &model.Job{UserID: "user-123", Title: "Software Engineer", CurrentStageTemplateID: &stageID}

		mock.ExpectExec("INSERT INTO jobs").
			WithArgs(
				pgxmock.AnyArg(), job.UserID, job.CompanyID, job.Title, job.Source, job.URL, job.Notes, job.Description,
				job.IsArchived, job.AppliedAt, job.ResumeID, job.ResumeBuilderID,
				job.CurrentStageTemplateID, pgxmock.AnyArg(), pgxmock.AnyArg(),
			).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), job)

		require.NoError(t, err)
		assert.NotEmpty(t, job.ID)
		assert.False(t, job.IsArchived)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		job := &model.Job{UserID: "user-123", Title: "X"}

		mock.ExpectExec("INSERT INTO jobs").
			WithArgs(
				pgxmock.AnyArg(), job.UserID, job.CompanyID, job.Title, job.Source, job.URL, job.Notes, job.Description,
				job.IsArchived, job.AppliedAt, job.ResumeID, job.ResumeBuilderID,
				job.CurrentStageTemplateID, pgxmock.AnyArg(), pgxmock.AnyArg(),
			).
			WillReturnError(errDB)

		err := repo.Create(context.Background(), job)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobRepository_GetByID(t *testing.T) {
	cols := []string{
		"id", "user_id", "company_id", "title", "source", "url", "notes", "description",
		"is_favorite", "is_archived", "applied_at", "resume_id", "resume_builder_id",
		"current_stage_template_id", "current_stage_id", "created_at", "updated_at",
	}

	t.Run("returns job successfully", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		now := time.Now()
		stageID := "stage-wishlist"

		mock.ExpectQuery("SELECT id, user_id, company_id, title").
			WithArgs("job-1", "user-123").
			WillReturnRows(pgxmock.NewRows(cols).AddRow(
				"job-1", "user-123", nil, "Software Engineer", nil, nil, nil, nil,
				false, false, nil, nil, nil,
				&stageID, nil, now, now,
			))

		job, err := repo.GetByID(context.Background(), "user-123", "job-1")

		require.NoError(t, err)
		assert.Equal(t, "job-1", job.ID)
		assert.Equal(t, "Software Engineer", job.Title)
		require.NotNil(t, job.CurrentStageTemplateID)
		assert.Equal(t, stageID, *job.CurrentStageTemplateID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found on ErrNoRows", func(t *testing.T) {
		repo, mock := newJobRepo(t)

		mock.ExpectQuery("SELECT id, user_id, company_id, title").
			WithArgs("nonexistent", "user-123").
			WillReturnError(pgx.ErrNoRows)

		job, err := repo.GetByID(context.Background(), "user-123", "nonexistent")

		assert.Nil(t, job)
		assert.ErrorIs(t, err, model.ErrJobNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		repo, mock := newJobRepo(t)

		mock.ExpectQuery("SELECT id, user_id, company_id, title").
			WithArgs("job-1", "user-123").
			WillReturnError(errDB)

		job, err := repo.GetByID(context.Background(), "user-123", "job-1")

		assert.Nil(t, job)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobRepository_Update(t *testing.T) {
	t.Run("updates job successfully", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		job := &model.Job{ID: "job-1", UserID: "user-123", Title: "Updated Title", IsArchived: true}

		mock.ExpectExec("UPDATE jobs").
			WithArgs(
				job.ID, job.UserID, job.CompanyID, job.Title, job.Source, job.URL, job.Notes, job.Description,
				job.IsArchived, job.AppliedAt, job.ResumeID, job.ResumeBuilderID,
				job.CurrentStageTemplateID, pgxmock.AnyArg(),
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(context.Background(), job)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		job := &model.Job{ID: "nonexistent", UserID: "user-123", Title: "Test"}

		mock.ExpectExec("UPDATE jobs").
			WithArgs(
				job.ID, job.UserID, job.CompanyID, job.Title, job.Source, job.URL, job.Notes, job.Description,
				job.IsArchived, job.AppliedAt, job.ResumeID, job.ResumeBuilderID,
				job.CurrentStageTemplateID, pgxmock.AnyArg(),
			).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(context.Background(), job)

		assert.ErrorIs(t, err, model.ErrJobNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		job := &model.Job{ID: "job-1", UserID: "user-123", Title: "X"}

		mock.ExpectExec("UPDATE jobs").
			WithArgs(
				job.ID, job.UserID, job.CompanyID, job.Title, job.Source, job.URL, job.Notes, job.Description,
				job.IsArchived, job.AppliedAt, job.ResumeID, job.ResumeBuilderID,
				job.CurrentStageTemplateID, pgxmock.AnyArg(),
			).
			WillReturnError(errDB)

		err := repo.Update(context.Background(), job)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobRepository_ToggleFavorite(t *testing.T) {
	t.Run("toggles and returns new value", func(t *testing.T) {
		repo, mock := newJobRepo(t)

		mock.ExpectQuery("UPDATE jobs SET is_favorite").
			WithArgs("job-1", "user-123").
			WillReturnRows(pgxmock.NewRows([]string{"is_favorite"}).AddRow(true))

		fav, err := repo.ToggleFavorite(context.Background(), "user-123", "job-1")

		require.NoError(t, err)
		assert.True(t, fav)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found on ErrNoRows", func(t *testing.T) {
		repo, mock := newJobRepo(t)

		mock.ExpectQuery("UPDATE jobs SET is_favorite").
			WithArgs("nope", "user-123").
			WillReturnError(pgx.ErrNoRows)

		fav, err := repo.ToggleFavorite(context.Background(), "user-123", "nope")

		assert.False(t, fav)
		assert.ErrorIs(t, err, model.ErrJobNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		repo, mock := newJobRepo(t)

		mock.ExpectQuery("UPDATE jobs SET is_favorite").
			WithArgs("job-1", "user-123").
			WillReturnError(errDB)

		fav, err := repo.ToggleFavorite(context.Background(), "user-123", "job-1")

		assert.False(t, fav)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobRepository_Delete(t *testing.T) {
	t.Run("deletes job successfully", func(t *testing.T) {
		repo, mock := newJobRepo(t)

		mock.ExpectExec("DELETE FROM jobs").
			WithArgs("job-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), "user-123", "job-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newJobRepo(t)

		mock.ExpectExec("DELETE FROM jobs").
			WithArgs("nonexistent", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), "user-123", "nonexistent")

		assert.ErrorIs(t, err, model.ErrJobNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newJobRepo(t)

		mock.ExpectExec("DELETE FROM jobs").
			WithArgs("job-1", "user-123").
			WillReturnError(errDB)

		err := repo.Delete(context.Background(), "user-123", "job-1")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// listColumns mirrors the enriched List projection Scan order (22 columns).
var listColumns = []string{
	"id", "company_id", "title", "source", "url", "notes", "description",
	"is_favorite", "is_archived", "applied_at",
	"current_stage_template_id", "current_stage_id",
	"created_at", "updated_at",
	"last_activity_at",
	"company_name",
	"resume_id", "resume_title",
	"resume_builder_id", "resume_builder_title",
	"current_stage_name",
	"total_count",
}

func TestJobRepository_List(t *testing.T) {
	t.Run("returns jobs excluding archived by default", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		now := time.Now()
		companyName := "Acme"

		rows := pgxmock.NewRows(listColumns).
			AddRow("job-1", nil, "Software Engineer", nil, nil, nil, nil, false, false, nil, nil, nil, now, now, now, &companyName, nil, nil, nil, nil, nil, 2).
			AddRow("job-2", nil, "Product Manager", nil, nil, nil, nil, false, false, nil, nil, nil, now, now, now, nil, nil, nil, nil, nil, nil, 2)

		mock.ExpectQuery("AND j.is_archived = false").
			WithArgs("user-123", 20, 0).
			WillReturnRows(rows)

		jobs, total, err := repo.List(context.Background(), "user-123", &ports.ListOptions{Limit: 20, Offset: 0, Status: ""})

		require.NoError(t, err)
		require.Len(t, jobs, 2)
		assert.Equal(t, 2, total)
		assert.Equal(t, "Acme", *jobs[0].CompanyName)
		for _, j := range jobs {
			assert.False(t, j.IsArchived)
		}
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("archived filter returns only archived jobs", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		now := time.Now()

		rows := pgxmock.NewRows(listColumns).
			AddRow("job-3", nil, "Old Role", nil, nil, nil, nil, false, true, nil, nil, nil, now, now, now, nil, nil, nil, nil, nil, nil, 1)

		mock.ExpectQuery("AND j.is_archived = true").
			WithArgs("user-123", 20, 0).
			WillReturnRows(rows)

		jobs, total, err := repo.List(context.Background(), "user-123", &ports.ListOptions{Limit: 20, Offset: 0, Status: "archived"})

		require.NoError(t, err)
		require.Len(t, jobs, 1)
		assert.Equal(t, 1, total)
		assert.True(t, jobs[0].IsArchived)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("attaches uploaded resume nested dto", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		now := time.Now()
		resumeID, resumeTitle := "resume-1", "My CV"

		rows := pgxmock.NewRows(listColumns).
			AddRow("job-1", nil, "SWE", nil, nil, nil, nil, false, false, nil, nil, nil, now, now, now, nil, &resumeID, &resumeTitle, nil, nil, nil, 1)

		mock.ExpectQuery("FROM jobs j").
			WithArgs("user-123", 20, 0).
			WillReturnRows(rows)

		jobs, _, err := repo.List(context.Background(), "user-123", &ports.ListOptions{Limit: 20, Offset: 0})

		require.NoError(t, err)
		require.Len(t, jobs, 1)
		require.NotNil(t, jobs[0].Resume)
		assert.Equal(t, "resume-1", jobs[0].Resume.ID)
		assert.Equal(t, "My CV", jobs[0].Resume.Name)
		assert.Equal(t, "uploaded", jobs[0].Resume.Type)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("attaches builder resume nested dto", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		now := time.Now()
		builderID, builderTitle := "builder-1", "Generated"

		rows := pgxmock.NewRows(listColumns).
			AddRow("job-1", nil, "SWE", nil, nil, nil, nil, false, false, nil, nil, nil, now, now, now, nil, nil, nil, &builderID, &builderTitle, nil, 1)

		mock.ExpectQuery("FROM jobs j").
			WithArgs("user-123", 20, 0).
			WillReturnRows(rows)

		jobs, _, err := repo.List(context.Background(), "user-123", &ports.ListOptions{Limit: 20, Offset: 0})

		require.NoError(t, err)
		require.NotNil(t, jobs[0].Resume)
		assert.Equal(t, "builder-1", jobs[0].Resume.ID)
		assert.Equal(t, "builder", jobs[0].Resume.Type)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("applies search filter with extra arg", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		now := time.Now()

		rows := pgxmock.NewRows(listColumns).
			AddRow("job-1", nil, "Stripe Engineer", nil, nil, nil, nil, false, false, nil, nil, nil, now, now, now, nil, nil, nil, nil, nil, nil, 1)

		// search adds "%Stripe%" as $2, pushing limit/offset to $3/$4.
		mock.ExpectQuery("ILIKE").
			WithArgs("user-123", "%Stripe%", 20, 0).
			WillReturnRows(rows)

		jobs, total, err := repo.List(context.Background(), "user-123", &ports.ListOptions{Limit: 20, Offset: 0, Search: "Stripe"})

		require.NoError(t, err)
		require.Len(t, jobs, 1)
		assert.Equal(t, 1, total)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("applies sort by title", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		now := time.Now()

		rows := pgxmock.NewRows(listColumns).
			AddRow("job-1", nil, "A", nil, nil, nil, nil, false, false, nil, nil, nil, now, now, now, nil, nil, nil, nil, nil, nil, 1)

		mock.ExpectQuery("ORDER BY LOWER").
			WithArgs("user-123", 20, 0).
			WillReturnRows(rows)

		jobs, _, err := repo.List(context.Background(), "user-123", &ports.ListOptions{Limit: 20, Offset: 0, SortBy: "title", SortDir: "asc"})

		require.NoError(t, err)
		assert.Len(t, jobs, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates query error", func(t *testing.T) {
		repo, mock := newJobRepo(t)

		mock.ExpectQuery("FROM jobs j").
			WithArgs("user-123", 20, 0).
			WillReturnError(errDB)

		jobs, total, err := repo.List(context.Background(), "user-123", &ports.ListOptions{Limit: 20, Offset: 0})

		assert.Nil(t, jobs)
		assert.Zero(t, total)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates scan error", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		now := time.Now()

		rows := pgxmock.NewRows(listColumns).
			AddRow("job-1", nil, "A", nil, nil, nil, nil, false, false, nil, nil, nil, now, now, "not-a-time", nil, nil, nil, nil, nil, nil, 1)

		mock.ExpectQuery("FROM jobs j").
			WithArgs("user-123", 20, 0).
			WillReturnRows(rows)

		jobs, total, err := repo.List(context.Background(), "user-123", &ports.ListOptions{Limit: 20, Offset: 0})

		assert.Nil(t, jobs)
		assert.Zero(t, total)
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobRepository_GetLastActivityAt(t *testing.T) {
	t.Run("returns last activity timestamp", func(t *testing.T) {
		repo, mock := newJobRepo(t)
		now := time.Now()

		mock.ExpectQuery("GREATEST").
			WithArgs("job-1", "user-123").
			WillReturnRows(pgxmock.NewRows([]string{"last_activity_at"}).AddRow(now))

		got, err := repo.GetLastActivityAt(context.Background(), "user-123", "job-1")

		require.NoError(t, err)
		assert.WithinDuration(t, now, got, time.Second)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newJobRepo(t)

		mock.ExpectQuery("GREATEST").
			WithArgs("job-1", "user-123").
			WillReturnError(errDB)

		_, err := repo.GetLastActivityAt(context.Background(), "user-123", "job-1")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSafeString(t *testing.T) {
	assert.Equal(t, "", safeString(nil))
	s := "hi"
	assert.Equal(t, "hi", safeString(&s))
}
