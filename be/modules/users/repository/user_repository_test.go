package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errDB = errors.New("boom: db failure")

func newUserRepo(t *testing.T) (*UserRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewUserRepository(mock), mock
}

func TestUserRepository_Create(t *testing.T) {
	t.Run("creates user successfully", func(t *testing.T) {
		repo, mock := newUserRepo(t)
		user := &model.User{
			Email:         "test@example.com",
			Name:          "Test User",
			PasswordHash:  "hashed-password",
			Locale:        "en",
			EmailVerified: false,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		mock.ExpectExec("INSERT INTO users").
			WithArgs(pgxmock.AnyArg(), user.Email, user.Name, user.PasswordHash, user.Locale, user.EmailVerified, user.CreatedAt, user.UpdatedAt).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), user)

		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps unique violation to ErrUserAlreadyExists", func(t *testing.T) {
		repo, mock := newUserRepo(t)
		user := &model.User{Email: "dup@example.com", Name: "Dup", Locale: "en"}

		mock.ExpectExec("INSERT INTO users").
			WithArgs(pgxmock.AnyArg(), user.Email, user.Name, user.PasswordHash, user.Locale, user.EmailVerified, user.CreatedAt, user.UpdatedAt).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		err := repo.Create(context.Background(), user)

		assert.ErrorIs(t, err, model.ErrUserAlreadyExists)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates other db error", func(t *testing.T) {
		repo, mock := newUserRepo(t)
		user := &model.User{Email: "x@example.com", Name: "X", Locale: "en"}

		mock.ExpectExec("INSERT INTO users").
			WithArgs(pgxmock.AnyArg(), user.Email, user.Name, user.PasswordHash, user.Locale, user.EmailVerified, user.CreatedAt, user.UpdatedAt).
			WillReturnError(errDB)

		err := repo.Create(context.Background(), user)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	cols := []string{"id", "email", "name", "password_hash", "locale", "email_verified", "created_at", "updated_at"}

	t.Run("returns user successfully", func(t *testing.T) {
		repo, mock := newUserRepo(t)
		now := time.Now()

		mock.ExpectQuery("WHERE id = ").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows(cols).AddRow(
				"user-123", "test@example.com", "Test User", "hashed-password", "en", true, now, now,
			))

		user, err := repo.GetByID(context.Background(), "user-123")

		require.NoError(t, err)
		assert.Equal(t, "user-123", user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.True(t, user.EmailVerified)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found on ErrNoRows", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectQuery("WHERE id = ").
			WithArgs("nonexistent").
			WillReturnError(pgx.ErrNoRows)

		user, err := repo.GetByID(context.Background(), "nonexistent")

		assert.Nil(t, user)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectQuery("WHERE id = ").
			WithArgs("user-123").
			WillReturnError(errDB)

		user, err := repo.GetByID(context.Background(), "user-123")

		assert.Nil(t, user)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	cols := []string{"id", "email", "name", "password_hash", "locale", "email_verified", "created_at", "updated_at"}

	t.Run("returns user successfully", func(t *testing.T) {
		repo, mock := newUserRepo(t)
		now := time.Now()

		mock.ExpectQuery("WHERE email = ").
			WithArgs("test@example.com").
			WillReturnRows(pgxmock.NewRows(cols).AddRow(
				"user-123", "test@example.com", "Test User", "hashed-password", "en", false, now, now,
			))

		user, err := repo.GetByEmail(context.Background(), "test@example.com")

		require.NoError(t, err)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "user-123", user.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found on ErrNoRows", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectQuery("WHERE email = ").
			WithArgs("nope@example.com").
			WillReturnError(pgx.ErrNoRows)

		user, err := repo.GetByEmail(context.Background(), "nope@example.com")

		assert.Nil(t, user)
		assert.ErrorIs(t, err, model.ErrUserNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectQuery("WHERE email = ").
			WithArgs("test@example.com").
			WillReturnError(errDB)

		user, err := repo.GetByEmail(context.Background(), "test@example.com")

		assert.Nil(t, user)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Update(t *testing.T) {
	t.Run("updates user successfully", func(t *testing.T) {
		repo, mock := newUserRepo(t)
		user := &model.User{ID: "user-123", Name: "Updated", Locale: "ua"}

		mock.ExpectExec("UPDATE users").
			WithArgs(user.ID, user.Name, user.Locale).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(context.Background(), user)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newUserRepo(t)
		user := &model.User{ID: "nonexistent", Name: "Updated", Locale: "ua"}

		mock.ExpectExec("UPDATE users").
			WithArgs(user.ID, user.Name, user.Locale).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(context.Background(), user)

		assert.ErrorIs(t, err, model.ErrUserNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newUserRepo(t)
		user := &model.User{ID: "user-123", Name: "Updated", Locale: "ua"}

		mock.ExpectExec("UPDATE users").
			WithArgs(user.ID, user.Name, user.Locale).
			WillReturnError(errDB)

		err := repo.Update(context.Background(), user)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Delete(t *testing.T) {
	t.Run("deletes user successfully", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectExec("DELETE FROM users").
			WithArgs("user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), "user-123")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectExec("DELETE FROM users").
			WithArgs("nonexistent").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), "nonexistent")

		assert.ErrorIs(t, err, model.ErrUserNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectExec("DELETE FROM users").
			WithArgs("user-123").
			WillReturnError(errDB)

		err := repo.Delete(context.Background(), "user-123")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_SetEmailVerified(t *testing.T) {
	t.Run("marks email verified", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectExec("UPDATE users SET email_verified").
			WithArgs("user-123").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.SetEmailVerified(context.Background(), "user-123")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectExec("UPDATE users SET email_verified").
			WithArgs("nonexistent").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.SetEmailVerified(context.Background(), "nonexistent")

		assert.ErrorIs(t, err, model.ErrUserNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectExec("UPDATE users SET email_verified").
			WithArgs("user-123").
			WillReturnError(errDB)

		err := repo.SetEmailVerified(context.Background(), "user-123")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_UpdatePasswordHash(t *testing.T) {
	t.Run("updates password hash", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectExec("UPDATE users SET password_hash").
			WithArgs("user-123", "new-hash").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.UpdatePasswordHash(context.Background(), "user-123", "new-hash")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectExec("UPDATE users SET password_hash").
			WithArgs("nonexistent", "new-hash").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.UpdatePasswordHash(context.Background(), "nonexistent", "new-hash")

		assert.ErrorIs(t, err, model.ErrUserNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newUserRepo(t)

		mock.ExpectExec("UPDATE users SET password_hash").
			WithArgs("user-123", "new-hash").
			WillReturnError(errDB)

		err := repo.UpdatePasswordHash(context.Background(), "user-123", "new-hash")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUser_ToDTO(t *testing.T) {
	now := time.Now()
	user := &model.User{
		ID:           "user-123",
		Email:        "test@example.com",
		Name:         "Test User",
		PasswordHash: "secret-hash",
		Locale:       "en",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	dto := user.ToDTO()

	assert.Equal(t, user.ID, dto.ID)
	assert.Equal(t, user.Email, dto.Email)
	assert.Equal(t, user.Name, dto.Name)
	assert.Equal(t, user.Locale, dto.Locale)
	assert.Equal(t, user.CreatedAt, dto.CreatedAt)
}
