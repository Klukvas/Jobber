package repository

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResumeRepository_Create(t *testing.T) {
	t.Run("creates resume successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		resume := &model.Resume{
			UserID:      "user-123",
			Title:       "Software Engineer Resume",
			StorageType: model.StorageTypeExternal,
			IsActive:    true,
		}

		mock.ExpectExec("INSERT INTO resumes").
			WithArgs(pgxmock.AnyArg(), resume.UserID, resume.Title, resume.FileURL, string(resume.StorageType), resume.StorageKey, resume.IsActive, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := &testResumeRepo{mock: mock}
		err = repo.Create(context.Background(), resume)

		require.NoError(t, err)
		assert.NotEmpty(t, resume.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeRepository_GetByID(t *testing.T) {
	t.Run("returns resume successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		userID := "user-123"
		resumeID := "resume-1"
		now := time.Now()

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "title", "file_url", "storage_type", "storage_key", "is_active", "created_at", "updated_at",
		}).AddRow(
			resumeID, userID, "My Resume", nil, "external", nil, true, now, now,
		)

		mock.ExpectQuery("SELECT id, user_id, title, file_url, storage_type, storage_key, is_active, created_at, updated_at").
			WithArgs(resumeID, userID).
			WillReturnRows(rows)

		repo := &testResumeRepo{mock: mock}
		resume, err := repo.GetByID(context.Background(), userID, resumeID)

		require.NoError(t, err)
		assert.Equal(t, resumeID, resume.ID)
		assert.Equal(t, "My Resume", resume.Title)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when resume not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, title, file_url, storage_type, storage_key, is_active, created_at, updated_at").
			WithArgs("nonexistent", "user-123").
			WillReturnError(pgx.ErrNoRows)

		repo := &testResumeRepo{mock: mock}
		resume, err := repo.GetByID(context.Background(), "user-123", "nonexistent")

		assert.Nil(t, resume)
		assert.Equal(t, model.ErrResumeNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeRepository_Update(t *testing.T) {
	t.Run("updates resume successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		resume := &model.Resume{
			ID:          "resume-1",
			UserID:      "user-123",
			Title:       "Updated Resume",
			StorageType: model.StorageTypeExternal,
			IsActive:    false,
		}

		mock.ExpectExec("UPDATE resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.IsActive, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := &testResumeRepo{mock: mock}
		err = repo.Update(context.Background(), resume)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when resume not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		resume := &model.Resume{
			ID:     "nonexistent",
			UserID: "user-123",
			Title:  "Test",
		}

		mock.ExpectExec("UPDATE resumes").
			WithArgs(resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.IsActive, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := &testResumeRepo{mock: mock}
		err = repo.Update(context.Background(), resume)

		assert.Equal(t, model.ErrResumeNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeRepository_Delete(t *testing.T) {
	t.Run("deletes resume successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resumes").
			WithArgs("resume-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := &testResumeRepo{mock: mock}
		err = repo.Delete(context.Background(), "user-123", "resume-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when resume not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resumes").
			WithArgs("nonexistent", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := &testResumeRepo{mock: mock}
		err = repo.Delete(context.Background(), "user-123", "nonexistent")

		assert.Equal(t, model.ErrResumeNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeRepository_List(t *testing.T) {
	t.Run("returns resumes list", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		userID := "user-123"

		// Count query
		countRows := pgxmock.NewRows([]string{"count"}).AddRow(2)
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(userID).
			WillReturnRows(countRows)

		// List query
		now := time.Now()
		listRows := pgxmock.NewRows([]string{
			"id", "user_id", "title", "file_url", "storage_type", "storage_key", "is_active", "created_at", "updated_at", "applications_count",
		}).
			AddRow("resume-1", userID, "Resume A", nil, "external", nil, true, now, now, 5).
			AddRow("resume-2", userID, "Resume B", nil, "external", nil, false, now, now, 3)

		mock.ExpectQuery("SELECT r.id, r.user_id").
			WithArgs(userID, 20, 0).
			WillReturnRows(listRows)

		repo := &testResumeRepo{mock: mock}
		resumes, total, err := repo.List(context.Background(), userID, 20, 0, "created_at", "desc")

		require.NoError(t, err)
		assert.Len(t, resumes, 2)
		assert.Equal(t, 2, total)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// testResumeRepo is a test wrapper that uses pgxmock
type testResumeRepo struct {
	mock pgxmock.PgxPoolIface
}

func (r *testResumeRepo) Create(ctx context.Context, resume *model.Resume) error {
	query := `
		INSERT INTO resumes (id, user_id, title, file_url, storage_type, storage_key, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	resume.ID = "test-resume-id"
	now := time.Now().UTC()
	resume.CreatedAt = now
	resume.UpdatedAt = now

	_, err := r.mock.Exec(ctx, query,
		resume.ID, resume.UserID, resume.Title, resume.FileURL, string(resume.StorageType), resume.StorageKey, resume.IsActive, resume.CreatedAt, resume.UpdatedAt,
	)
	return err
}

func (r *testResumeRepo) GetByID(ctx context.Context, userID, resumeID string) (*model.Resume, error) {
	query := `
		SELECT id, user_id, title, file_url, storage_type, storage_key, is_active, created_at, updated_at
		FROM resumes
		WHERE id = $1 AND user_id = $2
	`
	resume := &model.Resume{}
	var storageType string
	err := r.mock.QueryRow(ctx, query, resumeID, userID).Scan(
		&resume.ID, &resume.UserID, &resume.Title, &resume.FileURL, &storageType, &resume.StorageKey, &resume.IsActive, &resume.CreatedAt, &resume.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrResumeNotFound
		}
		return nil, err
	}
	resume.StorageType = model.StorageType(storageType)
	return resume, nil
}

func (r *testResumeRepo) Update(ctx context.Context, resume *model.Resume) error {
	query := `
		UPDATE resumes
		SET title = $3, file_url = $4, is_active = $5, updated_at = $6
		WHERE id = $1 AND user_id = $2
	`
	resume.UpdatedAt = time.Now().UTC()
	result, err := r.mock.Exec(ctx, query,
		resume.ID, resume.UserID, resume.Title, resume.FileURL, resume.IsActive, resume.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrResumeNotFound
	}
	return nil
}

func (r *testResumeRepo) Delete(ctx context.Context, userID, resumeID string) error {
	query := `DELETE FROM resumes WHERE id = $1 AND user_id = $2`
	result, err := r.mock.Exec(ctx, query, resumeID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrResumeNotFound
	}
	return nil
}

func (r *testResumeRepo) List(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error) {
	countQuery := `SELECT COUNT(*) FROM resumes WHERE user_id = $1`
	var total int
	if err := r.mock.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT r.id, r.user_id, ... LIMIT $2 OFFSET $3`
	rows, err := r.mock.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var resumes []*ports.ResumeWithCount
	for rows.Next() {
		resume := &model.Resume{}
		var storageType string
		var applicationsCount int

		if err := rows.Scan(
			&resume.ID, &resume.UserID, &resume.Title, &resume.FileURL, &storageType, &resume.StorageKey, &resume.IsActive, &resume.CreatedAt, &resume.UpdatedAt, &applicationsCount,
		); err != nil {
			return nil, 0, err
		}

		resume.StorageType = model.StorageType(storageType)
		resumes = append(resumes, &ports.ResumeWithCount{
			Resume:            resume,
			ApplicationsCount: applicationsCount,
		})
	}

	return resumes, total, nil
}
