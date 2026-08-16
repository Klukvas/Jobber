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

func TestPasswordResetRepository_Create(t *testing.T) {
	t.Run("inserts token and assigns generated id", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		token := &model.PasswordResetToken{
			UserID:    "user-1",
			Code:      "123456",
			ExpiresAt: time.Now().Add(time.Hour),
			CreatedAt: time.Now(),
		}

		mock.ExpectExec("INSERT INTO password_reset_tokens").
			WithArgs(pgxmock.AnyArg(), token.UserID, token.Code, token.ExpiresAt, token.CreatedAt).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := NewPasswordResetRepository(mock)
		err = repo.Create(context.Background(), token)

		require.NoError(t, err)
		assert.NotEmpty(t, token.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO password_reset_tokens").
			WithArgs(pgxmock.AnyArg(), "u", "c", pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errors.New("boom"))

		repo := NewPasswordResetRepository(mock)
		err = repo.Create(context.Background(), &model.PasswordResetToken{UserID: "u", Code: "c"})

		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPasswordResetRepository_GetActiveForUser(t *testing.T) {
	t.Run("returns the active token", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "user_id", "code", "attempts", "expires_at", "used_at", "created_at",
		}).AddRow("tok-1", "user-1", "123456", 2, now.Add(time.Hour), (*time.Time)(nil), now)

		mock.ExpectQuery("SELECT id, user_id, code, attempts, expires_at, used_at, created_at").
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewPasswordResetRepository(mock)
		token, err := repo.GetActiveForUser(context.Background(), "user-1")

		require.NoError(t, err)
		assert.Equal(t, "tok-1", token.ID)
		assert.Equal(t, "user-1", token.UserID)
		assert.Equal(t, "123456", token.Code)
		assert.Equal(t, 2, token.Attempts)
		assert.Nil(t, token.UsedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrInvalidResetToken", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, code, attempts, expires_at, used_at, created_at").
			WithArgs("user-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewPasswordResetRepository(mock)
		token, err := repo.GetActiveForUser(context.Background(), "user-1")

		assert.Nil(t, token)
		assert.ErrorIs(t, err, userModel.ErrInvalidResetToken)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, code, attempts, expires_at, used_at, created_at").
			WithArgs("user-1").
			WillReturnError(errors.New("db down"))

		repo := NewPasswordResetRepository(mock)
		token, err := repo.GetActiveForUser(context.Background(), "user-1")

		assert.Nil(t, token)
		require.Error(t, err)
		assert.NotErrorIs(t, err, userModel.ErrInvalidResetToken)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPasswordResetRepository_IncrementAttempts(t *testing.T) {
	t.Run("returns new attempts count", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"attempts"}).AddRow(3)
		mock.ExpectQuery("UPDATE password_reset_tokens SET attempts = attempts").
			WithArgs("tok-1", 5).
			WillReturnRows(rows)

		repo := NewPasswordResetRepository(mock)
		n, err := repo.IncrementAttempts(context.Background(), "tok-1", 5)

		require.NoError(t, err)
		assert.Equal(t, 3, n)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrTooManyAttempts", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("UPDATE password_reset_tokens SET attempts = attempts").
			WithArgs("tok-1", 5).
			WillReturnError(pgx.ErrNoRows)

		repo := NewPasswordResetRepository(mock)
		n, err := repo.IncrementAttempts(context.Background(), "tok-1", 5)

		assert.Equal(t, 0, n)
		assert.ErrorIs(t, err, userModel.ErrTooManyAttempts)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("UPDATE password_reset_tokens SET attempts = attempts").
			WithArgs("tok-1", 5).
			WillReturnError(errors.New("boom"))

		repo := NewPasswordResetRepository(mock)
		n, err := repo.IncrementAttempts(context.Background(), "tok-1", 5)

		assert.Equal(t, 0, n)
		require.Error(t, err)
		assert.NotErrorIs(t, err, userModel.ErrTooManyAttempts)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPasswordResetRepository_MarkUsed(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("UPDATE password_reset_tokens SET used_at").
		WithArgs("tok-1", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	repo := NewPasswordResetRepository(mock)
	require.NoError(t, repo.MarkUsed(context.Background(), "tok-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPasswordResetRepository_DeleteForUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM password_reset_tokens WHERE user_id").
		WithArgs("user-1").
		WillReturnResult(pgxmock.NewResult("DELETE", 2))

	repo := NewPasswordResetRepository(mock)
	require.NoError(t, repo.DeleteForUser(context.Background(), "user-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPasswordResetRepository_DeleteExpired(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM password_reset_tokens WHERE expires_at").
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("DELETE", 4))

	repo := NewPasswordResetRepository(mock)
	require.NoError(t, repo.DeleteExpired(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}
