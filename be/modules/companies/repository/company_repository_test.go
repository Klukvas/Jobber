package repository

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/companies/model"
	"github.com/andreypavlenko/jobber/modules/companies/ports"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompanyRepository_Create(t *testing.T) {
	t.Run("creates company successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		company := &model.Company{
			UserID: "user-123",
			Name:   "Test Company",
		}

		mock.ExpectExec("INSERT INTO companies").
			WithArgs(pgxmock.AnyArg(), company.UserID, company.Name, company.Location, company.Notes, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := &testCompanyRepo{mock: mock}
		err = repo.Create(context.Background(), company)

		require.NoError(t, err)
		assert.NotEmpty(t, company.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_GetByID(t *testing.T) {
	t.Run("returns company successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		userID := "user-123"
		companyID := "company-1"
		now := time.Now()

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "name", "location", "notes", "created_at", "updated_at",
		}).AddRow(
			companyID, userID, "Test Company", nil, nil, now, now,
		)

		mock.ExpectQuery("SELECT id, user_id, name, location, notes, created_at, updated_at").
			WithArgs(companyID, userID).
			WillReturnRows(rows)

		repo := &testCompanyRepo{mock: mock}
		company, err := repo.GetByID(context.Background(), userID, companyID)

		require.NoError(t, err)
		assert.Equal(t, companyID, company.ID)
		assert.Equal(t, "Test Company", company.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when company not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, name, location, notes, created_at, updated_at").
			WithArgs("nonexistent", "user-123").
			WillReturnError(pgx.ErrNoRows)

		repo := &testCompanyRepo{mock: mock}
		company, err := repo.GetByID(context.Background(), "user-123", "nonexistent")

		assert.Nil(t, company)
		assert.Equal(t, model.ErrCompanyNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_Update(t *testing.T) {
	t.Run("updates company successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		company := &model.Company{
			ID:     "company-1",
			UserID: "user-123",
			Name:   "Updated Company",
		}

		mock.ExpectExec("UPDATE companies").
			WithArgs(company.ID, company.UserID, company.Name, company.Location, company.Notes, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := &testCompanyRepo{mock: mock}
		err = repo.Update(context.Background(), company)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when company not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		company := &model.Company{
			ID:     "nonexistent",
			UserID: "user-123",
			Name:   "Test",
		}

		mock.ExpectExec("UPDATE companies").
			WithArgs(company.ID, company.UserID, company.Name, company.Location, company.Notes, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := &testCompanyRepo{mock: mock}
		err = repo.Update(context.Background(), company)

		assert.Equal(t, model.ErrCompanyNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_Delete(t *testing.T) {
	t.Run("deletes company successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM companies").
			WithArgs("company-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := &testCompanyRepo{mock: mock}
		err = repo.Delete(context.Background(), "user-123", "company-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when company not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM companies").
			WithArgs("nonexistent", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := &testCompanyRepo{mock: mock}
		err = repo.Delete(context.Background(), "user-123", "nonexistent")

		assert.Equal(t, model.ErrCompanyNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_List(t *testing.T) {
	t.Run("returns companies list", func(t *testing.T) {
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
			"id", "name", "location", "notes", "created_at", "updated_at",
			"applications_count", "active_applications_count", "last_activity_at", "max_stages",
		}).
			AddRow("company-1", "Company A", nil, nil, now, now, 5, 3, &now, 2).
			AddRow("company-2", "Company B", nil, nil, now, now, 3, 1, nil, 0)

		mock.ExpectQuery("WITH company_apps AS").
			WithArgs(userID, 20, 0).
			WillReturnRows(listRows)

		repo := &testCompanyRepo{mock: mock}
		opts := &ports.ListOptions{Limit: 20, Offset: 0}
		companies, total, err := repo.List(context.Background(), userID, opts)

		require.NoError(t, err)
		assert.Len(t, companies, 2)
		assert.Equal(t, 2, total)
		assert.Equal(t, "Company A", companies[0].Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_GetRelatedJobsAndApplicationsCount(t *testing.T) {
	t.Run("returns counts successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"jobs_count", "applications_count"}).AddRow(5, 10)

		mock.ExpectQuery("SELECT").
			WithArgs("company-1", "user-123").
			WillReturnRows(rows)

		repo := &testCompanyRepo{mock: mock}
		jobsCount, appsCount, err := repo.GetRelatedJobsAndApplicationsCount(context.Background(), "user-123", "company-1")

		require.NoError(t, err)
		assert.Equal(t, 5, jobsCount)
		assert.Equal(t, 10, appsCount)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// testCompanyRepo is a test wrapper that uses pgxmock
type testCompanyRepo struct {
	mock pgxmock.PgxPoolIface
}

func (r *testCompanyRepo) Create(ctx context.Context, company *model.Company) error {
	query := `
		INSERT INTO companies (id, user_id, name, location, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	company.ID = "test-company-id"
	now := time.Now().UTC()
	company.CreatedAt = now
	company.UpdatedAt = now

	_, err := r.mock.Exec(ctx, query,
		company.ID, company.UserID, company.Name, company.Location, company.Notes, company.CreatedAt, company.UpdatedAt,
	)
	return err
}

func (r *testCompanyRepo) GetByID(ctx context.Context, userID, companyID string) (*model.Company, error) {
	query := `
		SELECT id, user_id, name, location, notes, created_at, updated_at
		FROM companies
		WHERE id = $1 AND user_id = $2
	`
	company := &model.Company{}
	err := r.mock.QueryRow(ctx, query, companyID, userID).Scan(
		&company.ID, &company.UserID, &company.Name, &company.Location, &company.Notes, &company.CreatedAt, &company.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, model.ErrCompanyNotFound
		}
		return nil, err
	}
	return company, nil
}

func (r *testCompanyRepo) Update(ctx context.Context, company *model.Company) error {
	query := `
		UPDATE companies
		SET name = $3, location = $4, notes = $5, updated_at = $6
		WHERE id = $1 AND user_id = $2
	`
	company.UpdatedAt = time.Now().UTC()
	result, err := r.mock.Exec(ctx, query,
		company.ID, company.UserID, company.Name, company.Location, company.Notes, company.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrCompanyNotFound
	}
	return nil
}

func (r *testCompanyRepo) Delete(ctx context.Context, userID, companyID string) error {
	query := `DELETE FROM companies WHERE id = $1 AND user_id = $2`
	result, err := r.mock.Exec(ctx, query, companyID, userID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrCompanyNotFound
	}
	return nil
}

func (r *testCompanyRepo) List(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.CompanyDTO, int, error) {
	countQuery := `SELECT COUNT(*) FROM companies WHERE user_id = $1`
	var total int
	if err := r.mock.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		WITH company_apps AS (...)
		SELECT ...
		LIMIT $2 OFFSET $3
	`
	rows, err := r.mock.Query(ctx, query, userID, opts.Limit, opts.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var companies []*model.CompanyDTO
	for rows.Next() {
		dto := &model.CompanyDTO{}
		var maxStages int
		if err := rows.Scan(
			&dto.ID, &dto.Name, &dto.Location, &dto.Notes, &dto.CreatedAt, &dto.UpdatedAt,
			&dto.ApplicationsCount, &dto.ActiveApplicationsCount, &dto.LastActivityAt, &maxStages,
		); err != nil {
			return nil, 0, err
		}
		companies = append(companies, dto)
	}

	return companies, total, nil
}

func (r *testCompanyRepo) GetRelatedJobsAndApplicationsCount(ctx context.Context, userID, companyID string) (jobsCount, appsCount int, err error) {
	query := `SELECT ... FROM companies WHERE c.id = $1 AND c.user_id = $2`
	err = r.mock.QueryRow(ctx, query, companyID, userID).Scan(&jobsCount, &appsCount)
	return
}
