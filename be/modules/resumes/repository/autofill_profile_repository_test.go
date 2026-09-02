package repository

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutofillProfileRepository_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("returns stored profile", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		raw, err := json.Marshal(&ai.ParsedResume{FullName: "Jane Dev", Email: "jane@dev.io"})
		require.NoError(t, err)

		mock.ExpectQuery("SELECT profile FROM resume_autofill_profiles").
			WithArgs("resume-1", "user-1").
			WillReturnRows(pgxmock.NewRows([]string{"profile"}).AddRow(raw))

		repo := NewAutofillProfileRepository(mock)
		profile, err := repo.Get(ctx, "user-1", "resume-1")

		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, "Jane Dev", profile.FullName)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns nil, nil on cache miss", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT profile FROM resume_autofill_profiles").
			WithArgs("resume-1", "user-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewAutofillProfileRepository(mock)
		profile, err := repo.Get(ctx, "user-1", "resume-1")

		require.NoError(t, err)
		assert.Nil(t, profile)
	})

	t.Run("corrupt stored json self-heals: row dropped, reported as miss", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT profile FROM resume_autofill_profiles").
			WithArgs("resume-1", "user-1").
			WillReturnRows(pgxmock.NewRows([]string{"profile"}).AddRow([]byte("{broken")))
		mock.ExpectExec("DELETE FROM resume_autofill_profiles").
			WithArgs("resume-1", "user-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := NewAutofillProfileRepository(mock)
		profile, err := repo.Get(ctx, "user-1", "resume-1")

		require.NoError(t, err)
		assert.Nil(t, profile, "a dropped corrupt row must read as a cache miss")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("corrupt json with failed cleanup surfaces the error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT profile FROM resume_autofill_profiles").
			WithArgs("resume-1", "user-1").
			WillReturnRows(pgxmock.NewRows([]string{"profile"}).AddRow([]byte("{broken")))
		mock.ExpectExec("DELETE FROM resume_autofill_profiles").
			WithArgs("resume-1", "user-1").
			WillReturnError(pgx.ErrTxClosed)

		repo := NewAutofillProfileRepository(mock)
		_, err = repo.Get(ctx, "user-1", "resume-1")

		assert.Error(t, err)
	})
}

func TestAutofillProfileRepository_Upsert(t *testing.T) {
	t.Run("fresh insert reports inserted=true", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("INSERT INTO resume_autofill_profiles").
			WithArgs("resume-1", "user-1", pgxmock.AnyArg(), 1).
			WillReturnRows(pgxmock.NewRows([]string{"inserted"}).AddRow(true))

		repo := NewAutofillProfileRepository(mock)
		inserted, err := repo.Upsert(context.Background(), "user-1", "resume-1", &ai.ParsedResume{FullName: "Jane"}, 1)

		require.NoError(t, err)
		assert.True(t, inserted)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("conflict overwrite reports inserted=false", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("INSERT INTO resume_autofill_profiles").
			WithArgs("resume-1", "user-1", pgxmock.AnyArg(), 1).
			WillReturnRows(pgxmock.NewRows([]string{"inserted"}).AddRow(false))

		repo := NewAutofillProfileRepository(mock)
		inserted, err := repo.Upsert(context.Background(), "user-1", "resume-1", &ai.ParsedResume{FullName: "Jane"}, 1)

		require.NoError(t, err)
		assert.False(t, inserted)
	})
}

func TestAutofillProfileRepository_InvalidateByResume(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM resume_autofill_profiles").
		WithArgs("resume-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	repo := NewAutofillProfileRepository(mock)
	require.NoError(t, repo.InvalidateByResume(context.Background(), "resume-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}
