package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errDB = errors.New("boom: db failure")

func newResumeRepo(t *testing.T) (*ResumeRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewResumeRepository(mock), mock
}

func TestResumeRepository_Create(t *testing.T) {
	t.Run("creates resume successfully and generates id", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := &model.Resume{
			UserID:      "user-123",
			Title:       "SWE Resume",
			StorageType: model.StorageTypeExternal,
			IsActive:    true,
		}

		mock.ExpectExec("INSERT INTO resumes").
			WithArgs(pgxmock.AnyArg(), resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), resume)

		require.NoError(t, err)
		assert.NotEmpty(t, resume.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("keeps preset id (S3 upload flow)", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := &model.Resume{
			ID:          "preset-id",
			UserID:      "user-123",
			Title:       "SWE Resume",
			StorageType: model.StorageTypeS3,
		}

		mock.ExpectExec("INSERT INTO resumes").
			WithArgs("preset-id", resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), resume)

		require.NoError(t, err)
		assert.Equal(t, "preset-id", resume.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := &model.Resume{UserID: "user-123", Title: "X", StorageType: model.StorageTypeExternal}

		mock.ExpectExec("INSERT INTO resumes").
			WithArgs(pgxmock.AnyArg(), resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errDB)

		err := repo.Create(context.Background(), resume)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeRepository_GetByID(t *testing.T) {
	cols := []string{"id", "user_id", "title", "file_url", "storage_type", "storage_key", "is_active", "created_at", "updated_at"}

	t.Run("returns resume successfully", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		now := time.Now()

		mock.ExpectQuery("SELECT id, user_id, title, file_url, storage_type, storage_key, is_active").
			WithArgs("resume-1", "user-123").
			WillReturnRows(pgxmock.NewRows(cols).AddRow(
				"resume-1", "user-123", "My Resume", nil, "external", nil, true, now, now,
			))

		resume, err := repo.GetByID(context.Background(), "user-123", "resume-1")

		require.NoError(t, err)
		assert.Equal(t, "resume-1", resume.ID)
		assert.Equal(t, "My Resume", resume.Title)
		assert.Equal(t, model.StorageTypeExternal, resume.StorageType)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found on ErrNoRows", func(t *testing.T) {
		repo, mock := newResumeRepo(t)

		mock.ExpectQuery("SELECT id, user_id, title, file_url, storage_type, storage_key, is_active").
			WithArgs("nonexistent", "user-123").
			WillReturnError(pgx.ErrNoRows)

		resume, err := repo.GetByID(context.Background(), "user-123", "nonexistent")

		assert.Nil(t, resume)
		assert.ErrorIs(t, err, model.ErrResumeNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		repo, mock := newResumeRepo(t)

		mock.ExpectQuery("SELECT id, user_id, title, file_url, storage_type, storage_key, is_active").
			WithArgs("resume-1", "user-123").
			WillReturnError(errDB)

		resume, err := repo.GetByID(context.Background(), "user-123", "resume-1")

		assert.Nil(t, resume)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeRepository_List(t *testing.T) {
	listCols := []string{
		"id", "user_id", "title", "file_url", "storage_type", "storage_key", "is_active", "created_at", "updated_at", "applications_count",
	}

	t.Run("returns resumes with count", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		now := time.Now()

		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

		mock.ExpectQuery("FROM resumes r").
			WithArgs("user-123", 20, 0).
			WillReturnRows(pgxmock.NewRows(listCols).
				AddRow("resume-1", "user-123", "Resume A", nil, "external", nil, true, now, now, 5).
				AddRow("resume-2", "user-123", "Resume B", nil, "s3", nil, false, now, now, 3))

		resumes, total, err := repo.List(context.Background(), "user-123", 20, 0, "created_at", "desc")

		require.NoError(t, err)
		require.Len(t, resumes, 2)
		assert.Equal(t, 2, total)
		assert.Equal(t, "Resume A", resumes[0].Resume.Title)
		assert.Equal(t, 5, resumes[0].ApplicationsCount)
		assert.Equal(t, model.StorageTypeS3, resumes[1].Resume.StorageType)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("applies title asc order", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		now := time.Now()

		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectQuery("ORDER BY r.title ASC").
			WithArgs("user-123", 10, 0).
			WillReturnRows(pgxmock.NewRows(listCols).
				AddRow("resume-1", "user-123", "A", nil, "external", nil, true, now, now, 0))

		resumes, total, err := repo.List(context.Background(), "user-123", 10, 0, "title", "asc")

		require.NoError(t, err)
		assert.Len(t, resumes, 1)
		assert.Equal(t, 1, total)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates count query error", func(t *testing.T) {
		repo, mock := newResumeRepo(t)

		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnError(errDB)

		resumes, total, err := repo.List(context.Background(), "user-123", 20, 0, "", "")

		assert.Nil(t, resumes)
		assert.Zero(t, total)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates list query error", func(t *testing.T) {
		repo, mock := newResumeRepo(t)

		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("FROM resumes r").
			WithArgs("user-123", 20, 0).
			WillReturnError(errDB)

		resumes, total, err := repo.List(context.Background(), "user-123", 20, 0, "", "")

		assert.Nil(t, resumes)
		assert.Zero(t, total)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates scan error", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		now := time.Now()

		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("FROM resumes r").
			WithArgs("user-123", 20, 0).
			WillReturnRows(pgxmock.NewRows(listCols).
				AddRow("resume-1", "user-123", "A", nil, "external", nil, true, now, now, "bad"))

		resumes, total, err := repo.List(context.Background(), "user-123", 20, 0, "", "")

		assert.Nil(t, resumes)
		assert.Zero(t, total)
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeRepository_Update(t *testing.T) {
	t.Run("updates resume successfully", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := &model.Resume{
			ID:          "resume-1",
			UserID:      "user-123",
			Title:       "Updated",
			StorageType: model.StorageTypeExternal,
		}

		mock.ExpectExec("UPDATE resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(context.Background(), resume)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := &model.Resume{ID: "nonexistent", UserID: "user-123", Title: "T", StorageType: model.StorageTypeExternal}

		mock.ExpectExec("UPDATE resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(context.Background(), resume)

		assert.ErrorIs(t, err, model.ErrResumeNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := &model.Resume{ID: "resume-1", UserID: "user-123", Title: "T", StorageType: model.StorageTypeExternal}

		mock.ExpectExec("UPDATE resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg()).
			WillReturnError(errDB)

		err := repo.Update(context.Background(), resume)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeRepository_Delete(t *testing.T) {
	t.Run("deletes resume successfully", func(t *testing.T) {
		repo, mock := newResumeRepo(t)

		mock.ExpectExec("DELETE FROM resumes").
			WithArgs("resume-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), "user-123", "resume-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newResumeRepo(t)

		mock.ExpectExec("DELETE FROM resumes").
			WithArgs("nonexistent", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), "user-123", "nonexistent")

		assert.ErrorIs(t, err, model.ErrResumeNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps foreign key violation to ErrResumeInUse", func(t *testing.T) {
		repo, mock := newResumeRepo(t)

		mock.ExpectExec("DELETE FROM resumes").
			WithArgs("resume-1", "user-123").
			WillReturnError(&pgconn.PgError{Code: "23503"})

		err := repo.Delete(context.Background(), "user-123", "resume-1")

		assert.ErrorIs(t, err, model.ErrResumeInUse)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates other db error", func(t *testing.T) {
		repo, mock := newResumeRepo(t)

		mock.ExpectExec("DELETE FROM resumes").
			WithArgs("resume-1", "user-123").
			WillReturnError(errDB)

		err := repo.Delete(context.Background(), "user-123", "resume-1")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
