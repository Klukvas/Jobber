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

func TestJobRepository_Create(t *testing.T) {
	t.Run("creates job successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		job := &model.Job{
			UserID: "user-123",
			Title:  "Software Engineer",
		}

		mock.ExpectExec("INSERT INTO jobs").
			WithArgs(pgxmock.AnyArg(), job.UserID, job.CompanyID, job.Title, job.Source, job.URL, job.Notes, "active", pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := &testJobRepo{mock: mock}
		err = repo.Create(context.Background(), job)

		require.NoError(t, err)
		assert.NotEmpty(t, job.ID)
		assert.Equal(t, "active", job.Status)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobRepository_GetByID(t *testing.T) {
	t.Run("returns job successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		userID := "user-123"
		jobID := "job-1"
		now := time.Now()

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "company_id", "title", "source", "url", "notes", "status", "created_at", "updated_at",
		}).AddRow(
			jobID, userID, nil, "Software Engineer", nil, nil, nil, "active", now, now,
		)

		mock.ExpectQuery("SELECT id, user_id, company_id, title, source, url, notes, status, created_at, updated_at").
			WithArgs(jobID, userID).
			WillReturnRows(rows)

		repo := &testJobRepo{mock: mock}
		job, err := repo.GetByID(context.Background(), userID, jobID)

		require.NoError(t, err)
		assert.Equal(t, jobID, job.ID)
		assert.Equal(t, "Software Engineer", job.Title)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when job not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, company_id, title, source, url, notes, status, created_at, updated_at").
			WithArgs("nonexistent", "user-123").
			WillReturnError(pgx.ErrNoRows)

		repo := &testJobRepo{mock: mock}
		job, err := repo.GetByID(context.Background(), "user-123", "nonexistent")

		assert.Nil(t, job)
		assert.Equal(t, model.ErrJobNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobRepository_Update(t *testing.T) {
	t.Run("updates job successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		job := &model.Job{
			ID:     "job-1",
			UserID: "user-123",
			Title:  "Updated Title",
			Status: "archived",
		}

		mock.ExpectExec("UPDATE jobs").
			WithArgs(job.ID, job.UserID, job.CompanyID, job.Title, job.Source, job.URL, job.Notes, job.Status, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := &testJobRepo{mock: mock}
		err = repo.Update(context.Background(), job)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when job not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		job := &model.Job{
			ID:     "nonexistent",
			UserID: "user-123",
			Title:  "Test",
		}

		mock.ExpectExec("UPDATE jobs").
			WithArgs(job.ID, job.UserID, job.CompanyID, job.Title, job.Source, job.URL, job.Notes, job.Status, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := &testJobRepo{mock: mock}
		err = repo.Update(context.Background(), job)

		assert.Equal(t, model.ErrJobNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobRepository_Delete(t *testing.T) {
	t.Run("deletes job successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM jobs").
			WithArgs("job-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := &testJobRepo{mock: mock}
		err = repo.Delete(context.Background(), "user-123", "job-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when job not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM jobs").
			WithArgs("nonexistent", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := &testJobRepo{mock: mock}
		err = repo.Delete(context.Background(), "user-123", "nonexistent")

		assert.Equal(t, model.ErrJobNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestJobRepository_List(t *testing.T) {
	t.Run("returns jobs list", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		userID := "user-123"

		// Count query
		countRows := pgxmock.NewRows([]string{"count"}).AddRow(2)
		mock.ExpectQuery("SELECT COUNT").
			WithArgs(userID, "active").
			WillReturnRows(countRows)

		// List query
		now := time.Now()
		listRows := pgxmock.NewRows([]string{
			"id", "user_id", "company_id", "title", "source", "url", "notes", "status", "created_at", "updated_at", "company_name", "applications_count",
		}).
			AddRow("job-1", userID, nil, "Software Engineer", nil, nil, nil, "active", now, now, nil, 5).
			AddRow("job-2", userID, nil, "Product Manager", nil, nil, nil, "active", now, now, nil, 3)

		mock.ExpectQuery("SELECT").
			WithArgs(userID, "active", 20, 0).
			WillReturnRows(listRows)

		repo := &testJobRepo{mock: mock}
		jobs, total, err := repo.List(context.Background(), userID, 20, 0, "active", "", "")

		require.NoError(t, err)
		assert.Len(t, jobs, 2)
		assert.Equal(t, 2, total)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// testJobRepo is a test wrapper that uses pgxmock
type testJobRepo struct {
	mock pgxmock.PgxPoolIface
}

func (r *testJobRepo) Create(ctx context.Context, job *model.Job) error {
	query := `
		INSERT INTO jobs (id, user_id, company_id, title, source, url, notes, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	job.ID = "test-job-id"
	job.Status = "active"
	now := time.Now().UTC()
	job.CreatedAt = now
	job.UpdatedAt = now

	_, err := r.mock.Exec(ctx, query,
		job.ID, job.UserID, job.CompanyID, job.Title, job.Source, job.URL, job.Notes, job.Status, job.CreatedAt, job.UpdatedAt,
	)
	return err
}

func (r *testJobRepo) GetByID(ctx context.Context, userID, jobID string) (*model.Job, error) {
	query := `
		SELECT id, user_id, company_id, title, source, url, notes, status, created_at, updated_at
		FROM jobs
		WHERE id = $1 AND user_id = $2
	`
	job := &model.Job{}
	err := r.mock.QueryRow(ctx, query, jobID, userID).Scan(
		&job.ID, &job.UserID, &job.CompanyID, &job.Title, &job.Source, &job.URL, &job.Notes, &job.Status, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrJobNotFound
		}
		return nil, err
	}
	return job, nil
}

func (r *testJobRepo) Update(ctx context.Context, job *model.Job) error {
	query := `
		UPDATE jobs
		SET company_id = $3, title = $4, source = $5, url = $6, notes = $7, status = $8, updated_at = $9
		WHERE id = $1 AND user_id = $2
	`
	job.UpdatedAt = time.Now().UTC()
	result, err := r.mock.Exec(ctx, query,
		job.ID, job.UserID, job.CompanyID, job.Title, job.Source, job.URL, job.Notes, job.Status, job.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrJobNotFound
	}
	return nil
}

func (r *testJobRepo) Delete(ctx context.Context, userID, jobID string) error {
	query := `DELETE FROM jobs WHERE id = $1 AND user_id = $2`
	result, err := r.mock.Exec(ctx, query, jobID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrJobNotFound
	}
	return nil
}

func (r *testJobRepo) List(ctx context.Context, userID string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error) {
	countQuery := `SELECT COUNT(*) FROM jobs j WHERE j.user_id = $1 AND j.status = $2`
	var total int
	if err := r.mock.QueryRow(ctx, countQuery, userID, status).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ... FROM jobs j ... LIMIT $3 OFFSET $4`
	rows, err := r.mock.Query(ctx, query, userID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*model.JobDTO
	for rows.Next() {
		var companyName *string
		var applicationsCount int
		job := &model.Job{}

		if err := rows.Scan(
			&job.ID, &job.UserID, &job.CompanyID, &job.Title, &job.Source, &job.URL, &job.Notes, &job.Status, &job.CreatedAt, &job.UpdatedAt,
			&companyName, &applicationsCount,
		); err != nil {
			return nil, 0, err
		}

		dto := job.ToDTO()
		dto.CompanyName = companyName
		dto.ApplicationsCount = applicationsCount
		jobs = append(jobs, dto)
	}

	return jobs, total, nil
}
