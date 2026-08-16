package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/auth/model"
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailVerificationRepository_Create(t *testing.T) {
	t.Run("inserts token and assigns generated id", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		token := &model.EmailVerificationToken{
			UserID:    "user-1",
			Code:      "654321",
			ExpiresAt: time.Now().Add(time.Hour),
			CreatedAt: time.Now(),
		}

		mock.ExpectExec("INSERT INTO email_verification_tokens").
			WithArgs(pgxmock.AnyArg(), token.UserID, token.Code, token.ExpiresAt, token.CreatedAt).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := NewEmailVerificationRepository(mock)
		err = repo.Create(context.Background(), token)

		require.NoError(t, err)
		assert.NotEmpty(t, token.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO email_verification_tokens").
			WithArgs(pgxmock.AnyArg(), "u", "c", pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errors.New("boom"))

		repo := NewEmailVerificationRepository(mock)
		err = repo.Create(context.Background(), &model.EmailVerificationToken{UserID: "u", Code: "c"})

		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEmailVerificationRepository_GetActiveForUser(t *testing.T) {
	t.Run("returns the active token", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "user_id", "code", "attempts", "expires_at", "used_at", "created_at",
		}).AddRow("tok-1", "user-1", "654321", 1, now.Add(time.Hour), (*time.Time)(nil), now)

		mock.ExpectQuery("SELECT id, user_id, code, attempts, expires_at, used_at, created_at").
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewEmailVerificationRepository(mock)
		token, err := repo.GetActiveForUser(context.Background(), "user-1")

		require.NoError(t, err)
		assert.Equal(t, "tok-1", token.ID)
		assert.Equal(t, "654321", token.Code)
		assert.Equal(t, 1, token.Attempts)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrInvalidVerificationToken", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, code, attempts, expires_at, used_at, created_at").
			WithArgs("user-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewEmailVerificationRepository(mock)
		token, err := repo.GetActiveForUser(context.Background(), "user-1")

		assert.Nil(t, token)
		assert.ErrorIs(t, err, userModel.ErrInvalidVerificationToken)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, code, attempts, expires_at, used_at, created_at").
			WithArgs("user-1").
			WillReturnError(errors.New("db down"))

		repo := NewEmailVerificationRepository(mock)
		token, err := repo.GetActiveForUser(context.Background(), "user-1")

		assert.Nil(t, token)
		require.Error(t, err)
		assert.NotErrorIs(t, err, userModel.ErrInvalidVerificationToken)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEmailVerificationRepository_IncrementAttempts(t *testing.T) {
	t.Run("returns new attempts count", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"attempts"}).AddRow(4)
		mock.ExpectQuery("UPDATE email_verification_tokens SET attempts = attempts").
			WithArgs("tok-1", 5).
			WillReturnRows(rows)

		repo := NewEmailVerificationRepository(mock)
		n, err := repo.IncrementAttempts(context.Background(), "tok-1", 5)

		require.NoError(t, err)
		assert.Equal(t, 4, n)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrTooManyAttempts", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("UPDATE email_verification_tokens SET attempts = attempts").
			WithArgs("tok-1", 5).
			WillReturnError(pgx.ErrNoRows)

		repo := NewEmailVerificationRepository(mock)
		n, err := repo.IncrementAttempts(context.Background(), "tok-1", 5)

		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, userModel.ErrTooManyAttempts)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("UPDATE email_verification_tokens SET attempts = attempts").
			WithArgs("tok-1", 5).
			WillReturnError(errors.New("boom"))

		repo := NewEmailVerificationRepository(mock)
		n, err := repo.IncrementAttempts(context.Background(), "tok-1", 5)

		assert.Equal(t, 0, n)
		require.Error(t, err)
		assert.NotErrorIs(t, err, userModel.ErrTooManyAttempts)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEmailVerificationRepository_MarkUsed(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE email_verification_tokens SET used_at").
		WithArgs("tok-1", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewEmailVerificationRepository(mock)
	require.NoError(t, repo.MarkUsed(context.Background(), "tok-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_DeleteForUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM email_verification_tokens WHERE user_id").
		WithArgs("user-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	repo := NewEmailVerificationRepository(mock)
	require.NoError(t, repo.DeleteForUser(context.Background(), "user-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestEmailVerificationRepository_DeleteExpired(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM email_verification_tokens WHERE expires_at").
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("DELETE", 3))

	repo := NewEmailVerificationRepository(mock)
	require.NoError(t, repo.DeleteExpired(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}
