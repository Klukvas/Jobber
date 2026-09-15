package repository

import (
	"context"
	"errors"
	"regexp"
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

	// The upload-finalization path retries with a fixed id, so the caller has to
	// be able to tell "this id is already stored" from a real write failure.
	t.Run("reports a duplicate id as ErrResumeAlreadyExists", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := &model.Resume{
			ID:          "3f2a6c1e-0000-4000-8000-00000000abcd",
			UserID:      "user-123",
			Title:       "SWE Resume",
			StorageType: model.StorageTypeS3,
		}

		mock.ExpectExec("INSERT INTO resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		err := repo.Create(context.Background(), resume)

		assert.ErrorIs(t, err, model.ErrResumeAlreadyExists)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("passes any other write failure through untouched", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := &model.Resume{
			ID:          "3f2a6c1e-0000-4000-8000-00000000abcd",
			UserID:      "user-123",
			Title:       "SWE Resume",
			StorageType: model.StorageTypeS3,
		}

		mock.ExpectExec("INSERT INTO resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(&pgconn.PgError{Code: "40001"})

		err := repo.Create(context.Background(), resume)

		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrResumeAlreadyExists)
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

/*
The plan's resume allowance, enforced where it can actually hold.

`CheckLimit` and the INSERT used to be separate statements on separate
connections with a storage round-trip between them: two finalizations racing for
a free plan's last slot both read "two of three used" and both wrote. Counting
and writing in one transaction, behind a per-user advisory lock, is what makes
the second one see the first.
*/
func TestResumeRepository_CreateFinalizedUpload(t *testing.T) {
	newUpload := func() *model.Resume {
		key := "users/user-123/resumes/r1.pdf"
		return &model.Resume{
			ID:          "3f2a6c1e-0000-4000-8000-00000000abcd",
			UserID:      "user-123",
			Title:       "Backend Engineer",
			StorageType: model.StorageTypeS3,
			StorageKey:  &key,
			IsActive:    true,
		}
	}

	t.Run("locks, counts and inserts in one transaction", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := newUpload()

		mock.ExpectBegin()
		// The lock comes first: counting before it would read a total another
		// transaction is still about to add to.
		mock.ExpectExec("pg_advisory_xact_lock").
			WithArgs(resume.UserID).
			WillReturnResult(pgxmock.NewResult("SELECT", 1))
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(resume.UserID).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))
		expectNothingToClaim(mock, resume)
		mock.ExpectExec("INSERT INTO resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		err := repo.CreateFinalizedUpload(context.Background(), resume, 3)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("refuses the write when the allowance is already full", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := newUpload()

		mock.ExpectBegin()
		mock.ExpectExec("pg_advisory_xact_lock").
			WithArgs(resume.UserID).
			WillReturnResult(pgxmock.NewResult("SELECT", 1))
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(resume.UserID).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(3))
		// No INSERT: the transaction is rolled back instead.
		mock.ExpectRollback()

		err := repo.CreateFinalizedUpload(context.Background(), resume, 3)

		assert.ErrorIs(t, err, model.ErrResumeLimitReached)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// Placeholder rows the presign-era upload flow left behind are not resumes
	// and must not fill a slot — the condition that excludes them is shared
	// with the subscription service's own count so the two cannot disagree.
	t.Run("counts only the rows that consume a slot", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := newUpload()

		mock.ExpectBegin()
		mock.ExpectExec("pg_advisory_xact_lock").WithArgs(resume.UserID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		mock.ExpectQuery(regexp.QuoteMeta(model.CountableResumeCondition)).
			WithArgs(resume.UserID).
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		expectNothingToClaim(mock, resume)
		mock.ExpectExec("INSERT INTO resumes").WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		require.NoError(t, repo.CreateFinalizedUpload(context.Background(), resume, 3))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("skips the count entirely on an unlimited plan", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := newUpload()

		mock.ExpectBegin()
		mock.ExpectExec("pg_advisory_xact_lock").WithArgs(resume.UserID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		expectNothingToClaim(mock, resume)
		mock.ExpectExec("INSERT INTO resumes").WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectCommit()

		require.NoError(t, repo.CreateFinalizedUpload(context.Background(), resume, -1))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// A repeated finalization of the same upload has to stay recognisable, so
	// the caller can answer it with the row that already exists.
	t.Run("reports a duplicate id as ErrResumeAlreadyExists", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := newUpload()

		mock.ExpectBegin()
		mock.ExpectExec("pg_advisory_xact_lock").WithArgs(resume.UserID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		mock.ExpectQuery("SELECT COUNT").WithArgs(resume.UserID).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		expectNothingToClaim(mock, resume)
		mock.ExpectExec("INSERT INTO resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(&pgconn.PgError{Code: "23505"})
		mock.ExpectRollback()

		err := repo.CreateFinalizedUpload(context.Background(), resume, 3)

		assert.ErrorIs(t, err, model.ErrResumeAlreadyExists)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("writes nothing when the count cannot be read", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := newUpload()

		mock.ExpectBegin()
		mock.ExpectExec("pg_advisory_xact_lock").WithArgs(resume.UserID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		mock.ExpectQuery("SELECT COUNT").WithArgs(resume.UserID).WillReturnError(errDB)
		mock.ExpectRollback()

		err := repo.CreateFinalizedUpload(context.Background(), resume, 3)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("reports a failure to open the transaction", func(t *testing.T) {
		repo, mock := newResumeRepo(t)

		mock.ExpectBegin().WillReturnError(errDB)

		err := repo.CreateFinalizedUpload(context.Background(), newUpload(), 3)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	/*
		A row written by the presign-era upload flow — `storage_type = 's3'`,
		inactive, no file_url, never updated — still sits under the id the
		client is finalizing, and the object it names really was uploaded.

		The INSERT could only collide with it, and the caller's re-lookup saw a
		placeholder again and answered 404. Every retry did the same, forever,
		with a perfectly good PDF sitting in the bucket. The row is this
		upload's own, so it is claimed in place instead.
	*/
	t.Run("claims a presign-era placeholder instead of colliding with it", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := newUpload()
		created := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)

		mock.ExpectBegin()
		mock.ExpectExec("pg_advisory_xact_lock").WithArgs(resume.UserID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		mock.ExpectQuery("SELECT COUNT").WithArgs(resume.UserID).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("UPDATE resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg()).
			WillReturnRows(pgxmock.NewRows([]string{"created_at", "updated_at"}).AddRow(created, created.Add(time.Hour)))
		// No INSERT at all: there is nothing left to insert.
		mock.ExpectCommit()

		err := repo.CreateFinalizedUpload(context.Background(), resume, 3)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
		assert.Equal(t, created, resume.CreatedAt,
			"the row keeps the moment it was first written")
		assert.True(t, resume.UpdatedAt.After(resume.CreatedAt),
			"a claimed row must no longer look like an untouched placeholder")
	})

	// The claim is scoped to the caller and to rows that are still
	// placeholders, so a different owner's row — or a real resume under the
	// same id — matches nothing and the INSERT reports the collision as
	// before. The caller answers that with a 404, never with someone's resume.
	t.Run("does not claim a row that is not this upload's to claim", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := newUpload()

		mock.ExpectBegin()
		mock.ExpectExec("pg_advisory_xact_lock").WithArgs(resume.UserID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		mock.ExpectQuery("SELECT COUNT").WithArgs(resume.UserID).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		expectNothingToClaim(mock, resume)
		mock.ExpectExec("INSERT INTO resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(&pgconn.PgError{Code: "23505"})
		mock.ExpectRollback()

		err := repo.CreateFinalizedUpload(context.Background(), resume, 3)

		assert.ErrorIs(t, err, model.ErrResumeAlreadyExists)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// A placeholder does not consume a slot, so claiming one adds a resume the
	// plan has to have room for. The limit is decided before anything is
	// written, inside the same locked transaction as the count.
	t.Run("refuses to claim a placeholder when the plan is already full", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := newUpload()

		mock.ExpectBegin()
		mock.ExpectExec("pg_advisory_xact_lock").WithArgs(resume.UserID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		mock.ExpectQuery("SELECT COUNT").WithArgs(resume.UserID).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(3))
		// Neither the claim nor the insert is attempted.
		mock.ExpectRollback()

		err := repo.CreateFinalizedUpload(context.Background(), resume, 3)

		assert.ErrorIs(t, err, model.ErrResumeLimitReached)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	// An UPDATE that fails for a reason of its own is a write failure, not
	// "nothing to claim": falling through to the INSERT would turn it into a
	// duplicate-key error and mislead the caller into a 404.
	t.Run("reports a failed claim rather than falling through to the insert", func(t *testing.T) {
		repo, mock := newResumeRepo(t)
		resume := newUpload()

		mock.ExpectBegin()
		mock.ExpectExec("pg_advisory_xact_lock").WithArgs(resume.UserID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		mock.ExpectQuery("SELECT COUNT").WithArgs(resume.UserID).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("UPDATE resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg()).
			WillReturnError(errDB)
		mock.ExpectRollback()

		err := repo.CreateFinalizedUpload(context.Background(), resume, 3)

		assert.ErrorIs(t, err, errDB)
		assert.NotErrorIs(t, err, model.ErrResumeAlreadyExists)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// expectNothingToClaim sets up the claim attempt finding no placeholder, which
// is the ordinary case: nothing has written one since the presign era.
func expectNothingToClaim(mock pgxmock.PgxPoolIface, resume *model.Resume) {
	mock.ExpectQuery("UPDATE resumes").
		WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.StorageType, resume.StorageKey, resume.IsActive, pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)
}
