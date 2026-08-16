package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/companies/model"
	"github.com/andreypavlenko/jobber/modules/companies/ports"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errDB = errors.New("boom: db failure")

func newCompanyRepo(t *testing.T) (*CompanyRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewCompanyRepository(mock), mock
}

func TestCompanyRepository_Create(t *testing.T) {
	t.Run("creates company successfully", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		company := &model.Company{UserID: "user-123", Name: "Test Company"}

		mock.ExpectExec("INSERT INTO companies").
			WithArgs(pgxmock.AnyArg(), company.UserID, company.Name, company.Location, company.Notes, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), company)

		require.NoError(t, err)
		assert.NotEmpty(t, company.ID)
		assert.False(t, company.CreatedAt.IsZero())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		company := &model.Company{UserID: "user-123", Name: "X"}
		mock.ExpectExec("INSERT INTO companies").
			WithArgs(pgxmock.AnyArg(), company.UserID, company.Name, company.Location, company.Notes, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errDB)

		err := repo.Create(context.Background(), company)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_GetByID(t *testing.T) {
	cols := []string{"id", "user_id", "name", "location", "notes", "is_favorite", "created_at", "updated_at"}

	t.Run("returns company successfully", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)
		now := time.Now()

		mock.ExpectQuery("SELECT id, user_id, name, location, notes, is_favorite").
			WithArgs("company-1", "user-123").
			WillReturnRows(pgxmock.NewRows(cols).AddRow("company-1", "user-123", "Test Company", nil, nil, true, now, now))

		company, err := repo.GetByID(context.Background(), "user-123", "company-1")

		require.NoError(t, err)
		assert.Equal(t, "company-1", company.ID)
		assert.Equal(t, "Test Company", company.Name)
		assert.True(t, company.IsFavorite)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found on ErrNoRows", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectQuery("SELECT id, user_id, name, location, notes, is_favorite").
			WithArgs("nonexistent", "user-123").
			WillReturnError(pgx.ErrNoRows)

		company, err := repo.GetByID(context.Background(), "user-123", "nonexistent")

		assert.Nil(t, company)
		assert.ErrorIs(t, err, model.ErrCompanyNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectQuery("SELECT id, user_id, name, location, notes, is_favorite").
			WithArgs("company-1", "user-123").
			WillReturnError(errDB)

		company, err := repo.GetByID(context.Background(), "user-123", "company-1")

		assert.Nil(t, company)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_GetByIDEnriched(t *testing.T) {
	cols := []string{
		"id", "name", "location", "notes", "is_favorite", "created_at", "updated_at",
		"applications_count", "active_applications_count", "last_activity_at", "max_stages",
	}

	t.Run("returns enriched dto with interviewing status", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)
		now := time.Now()

		mock.ExpectQuery("WITH stage_agg AS").
			WithArgs("company-1", "user-123").
			WillReturnRows(pgxmock.NewRows(cols).AddRow(
				"company-1", "Acme", nil, nil, false, now, now, 3, 2, &now, 2,
			))

		dto, err := repo.GetByIDEnriched(context.Background(), "user-123", "company-1")

		require.NoError(t, err)
		assert.Equal(t, "company-1", dto.ID)
		assert.Equal(t, 3, dto.ApplicationsCount)
		assert.Equal(t, 2, dto.ActiveApplicationsCount)
		assert.Equal(t, string(model.CompanyStatusInterviewing), dto.DerivedStatus)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("derives active status when single-stage active apps", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)
		now := time.Now()

		mock.ExpectQuery("WITH stage_agg AS").
			WithArgs("company-1", "user-123").
			WillReturnRows(pgxmock.NewRows(cols).AddRow(
				"company-1", "Acme", nil, nil, false, now, now, 2, 1, &now, 1,
			))

		dto, err := repo.GetByIDEnriched(context.Background(), "user-123", "company-1")

		require.NoError(t, err)
		assert.Equal(t, string(model.CompanyStatusActive), dto.DerivedStatus)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("derives idle status when no applications", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)
		now := time.Now()

		mock.ExpectQuery("WITH stage_agg AS").
			WithArgs("company-1", "user-123").
			WillReturnRows(pgxmock.NewRows(cols).AddRow(
				"company-1", "Acme", nil, nil, false, now, now, 0, 0, nil, 0,
			))

		dto, err := repo.GetByIDEnriched(context.Background(), "user-123", "company-1")

		require.NoError(t, err)
		assert.Equal(t, string(model.CompanyStatusIdle), dto.DerivedStatus)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found on ErrNoRows", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectQuery("WITH stage_agg AS").
			WithArgs("nope", "user-123").
			WillReturnError(pgx.ErrNoRows)

		dto, err := repo.GetByIDEnriched(context.Background(), "user-123", "nope")

		assert.Nil(t, dto)
		assert.ErrorIs(t, err, model.ErrCompanyNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectQuery("WITH stage_agg AS").
			WithArgs("company-1", "user-123").
			WillReturnError(errDB)

		dto, err := repo.GetByIDEnriched(context.Background(), "user-123", "company-1")

		assert.Nil(t, dto)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_Update(t *testing.T) {
	t.Run("updates company successfully", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)
		company := &model.Company{ID: "company-1", UserID: "user-123", Name: "Updated Company"}

		mock.ExpectExec("UPDATE companies").
			WithArgs(company.ID, company.UserID, company.Name, company.Location, company.Notes, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(context.Background(), company)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)
		company := &model.Company{ID: "nonexistent", UserID: "user-123", Name: "Test"}

		mock.ExpectExec("UPDATE companies").
			WithArgs(company.ID, company.UserID, company.Name, company.Location, company.Notes, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(context.Background(), company)

		assert.ErrorIs(t, err, model.ErrCompanyNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error on update", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)
		company := &model.Company{ID: "company-1", UserID: "user-123", Name: "X"}

		mock.ExpectExec("UPDATE companies").
			WithArgs(company.ID, company.UserID, company.Name, company.Location, company.Notes, pgxmock.AnyArg()).
			WillReturnError(errDB)

		err := repo.Update(context.Background(), company)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_ToggleFavorite(t *testing.T) {
	t.Run("toggles and returns new value", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectQuery("UPDATE companies SET is_favorite").
			WithArgs("company-1", "user-123").
			WillReturnRows(pgxmock.NewRows([]string{"is_favorite"}).AddRow(true))

		fav, err := repo.ToggleFavorite(context.Background(), "user-123", "company-1")

		require.NoError(t, err)
		assert.True(t, fav)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found on ErrNoRows", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectQuery("UPDATE companies SET is_favorite").
			WithArgs("nope", "user-123").
			WillReturnError(pgx.ErrNoRows)

		fav, err := repo.ToggleFavorite(context.Background(), "user-123", "nope")

		assert.False(t, fav)
		assert.ErrorIs(t, err, model.ErrCompanyNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectQuery("UPDATE companies SET is_favorite").
			WithArgs("company-1", "user-123").
			WillReturnError(errDB)

		fav, err := repo.ToggleFavorite(context.Background(), "user-123", "company-1")

		assert.False(t, fav)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_Delete(t *testing.T) {
	t.Run("deletes company successfully", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectExec("DELETE FROM companies").
			WithArgs("company-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), "user-123", "company-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectExec("DELETE FROM companies").
			WithArgs("nonexistent", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), "user-123", "nonexistent")

		assert.ErrorIs(t, err, model.ErrCompanyNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectExec("DELETE FROM companies").
			WithArgs("company-1", "user-123").
			WillReturnError(errDB)

		err := repo.Delete(context.Background(), "user-123", "company-1")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_List(t *testing.T) {
	cols := []string{
		"id", "name", "location", "notes", "is_favorite", "created_at", "updated_at",
		"applications_count", "active_applications_count", "last_activity_at", "max_stages", "total_count",
	}

	t.Run("returns companies with total and derived status", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)
		now := time.Now()

		rows := pgxmock.NewRows(cols).
			AddRow("company-1", "Company A", nil, nil, false, now, now, 5, 3, &now, 2, 2).
			AddRow("company-2", "Company B", nil, nil, true, now, now, 0, 0, nil, 0, 2)

		mock.ExpectQuery("WITH stage_agg AS").
			WithArgs("user-123", 20, 0).
			WillReturnRows(rows)

		companies, total, err := repo.List(context.Background(), "user-123", &ports.ListOptions{Limit: 20, Offset: 0})

		require.NoError(t, err)
		require.Len(t, companies, 2)
		assert.Equal(t, 2, total)
		assert.Equal(t, "Company A", companies[0].Name)
		assert.Equal(t, string(model.CompanyStatusInterviewing), companies[0].DerivedStatus)
		assert.Equal(t, string(model.CompanyStatusIdle), companies[1].DerivedStatus)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("applies sort options to order by", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)
		now := time.Now()

		mock.ExpectQuery("ORDER BY applications_count DESC").
			WithArgs("user-123", 10, 5).
			WillReturnRows(pgxmock.NewRows(cols).AddRow("c1", "A", nil, nil, false, now, now, 1, 1, &now, 1, 1))

		companies, total, err := repo.List(context.Background(), "user-123",
			&ports.ListOptions{Limit: 10, Offset: 5, SortBy: "applications_count", SortDir: "DESC"})

		require.NoError(t, err)
		assert.Len(t, companies, 1)
		assert.Equal(t, 1, total)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates query error", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectQuery("WITH stage_agg AS").
			WithArgs("user-123", 20, 0).
			WillReturnError(errDB)

		companies, total, err := repo.List(context.Background(), "user-123", &ports.ListOptions{Limit: 20, Offset: 0})

		assert.Nil(t, companies)
		assert.Zero(t, total)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates scan error", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)
		now := time.Now()

		// Wrong-typed max_stages (string) makes Scan fail.
		rows := pgxmock.NewRows(cols).
			AddRow("c1", "A", nil, nil, false, now, now, 1, 1, &now, "bad", 1)

		mock.ExpectQuery("WITH stage_agg AS").
			WithArgs("user-123", 20, 0).
			WillReturnRows(rows)

		companies, total, err := repo.List(context.Background(), "user-123", &ports.ListOptions{Limit: 20, Offset: 0})

		assert.Nil(t, companies)
		assert.Zero(t, total)
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCompanyRepository_GetRelatedJobsAndApplicationsCount(t *testing.T) {
	t.Run("returns counts successfully", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectQuery("FROM companies c").
			WithArgs("company-1", "user-123").
			WillReturnRows(pgxmock.NewRows([]string{"jobs_count", "applications_count"}).AddRow(5, 10))

		jobsCount, appsCount, err := repo.GetRelatedJobsAndApplicationsCount(context.Background(), "user-123", "company-1")

		require.NoError(t, err)
		assert.Equal(t, 5, jobsCount)
		assert.Equal(t, 10, appsCount)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newCompanyRepo(t)

		mock.ExpectQuery("FROM companies c").
			WithArgs("company-1", "user-123").
			WillReturnError(errDB)

		_, _, err := repo.GetRelatedJobsAndApplicationsCount(context.Background(), "user-123", "company-1")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
