package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/auth/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenRepository_Create(t *testing.T) {
	t.Run("creates refresh token and assigns generated id", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		token := &model.RefreshToken{
			UserID:    "user-123",
			TokenHash: "hash123",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}

		mock.ExpectExec("INSERT INTO refresh_tokens").
			WithArgs(pgxmock.AnyArg(), token.UserID, token.TokenHash, token.ExpiresAt, token.CreatedAt).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := NewRefreshTokenRepository(mock)
		err = repo.Create(context.Background(), token)

		require.NoError(t, err)
		assert.NotEmpty(t, token.ID, "Create must assign a generated id")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO refresh_tokens").
			WithArgs(pgxmock.AnyArg(), "user-123", "hash", pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errors.New("boom"))

		repo := NewRefreshTokenRepository(mock)
		err = repo.Create(context.Background(), &model.RefreshToken{
			UserID:    "user-123",
			TokenHash: "hash",
			ExpiresAt: time.Now(),
			CreatedAt: time.Now(),
		})

		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRefreshTokenRepository_GetByTokenHash(t *testing.T) {
	t.Run("returns token successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		expected := &model.RefreshToken{
			ID:        "token-1",
			UserID:    "user-123",
			TokenHash: "hash123",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "token_hash", "expires_at", "created_at", "revoked_at",
		}).AddRow(expected.ID, expected.UserID, expected.TokenHash, expected.ExpiresAt, expected.CreatedAt, nil)

		mock.ExpectQuery("SELECT id, user_id, token_hash, expires_at, created_at, revoked_at").
			WithArgs(expected.TokenHash).
			WillReturnRows(rows)

		repo := NewRefreshTokenRepository(mock)
		token, err := repo.GetByTokenHash(context.Background(), expected.TokenHash)

		require.NoError(t, err)
		assert.Equal(t, expected.ID, token.ID)
		assert.Equal(t, expected.UserID, token.UserID)
		assert.Equal(t, expected.TokenHash, token.TokenHash)
		assert.Nil(t, token.RevokedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not-found error when no rows", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, token_hash, expires_at, created_at, revoked_at").
			WithArgs("nope").
			WillReturnError(pgx.ErrNoRows)

		repo := NewRefreshTokenRepository(mock)
		token, err := repo.GetByTokenHash(context.Background(), "nope")

		assert.Nil(t, token)
		require.Error(t, err)
		assert.Equal(t, "token not found", err.Error())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, token_hash, expires_at, created_at, revoked_at").
			WithArgs("hash").
			WillReturnError(errors.New("db down"))

		repo := NewRefreshTokenRepository(mock)
		token, err := repo.GetByTokenHash(context.Background(), "hash")

		assert.Nil(t, token)
		require.Error(t, err)
		assert.Equal(t, "db down", err.Error())
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRefreshTokenRepository_Revoke(t *testing.T) {
	t.Run("revokes token successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE refresh_tokens").
			WithArgs("hash123", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := NewRefreshTokenRepository(mock)
		err = repo.Revoke(context.Background(), "hash123")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE refresh_tokens").
			WithArgs("hash123", pgxmock.AnyArg()).
			WillReturnError(errors.New("boom"))

		repo := NewRefreshTokenRepository(mock)
		err = repo.Revoke(context.Background(), "hash123")

		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRefreshTokenRepository_RevokeIfValid(t *testing.T) {
	t.Run("returns true when a row was revoked", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE refresh_tokens").
			WithArgs("hash123", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := NewRefreshTokenRepository(mock)
		ok, err := repo.RevokeIfValid(context.Background(), "hash123")

		require.NoError(t, err)
		assert.True(t, ok)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns false when already revoked or expired", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE refresh_tokens").
			WithArgs("hash123", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := NewRefreshTokenRepository(mock)
		ok, err := repo.RevokeIfValid(context.Background(), "hash123")

		require.NoError(t, err)
		assert.False(t, ok)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE refresh_tokens").
			WithArgs("hash123", pgxmock.AnyArg()).
			WillReturnError(errors.New("boom"))

		repo := NewRefreshTokenRepository(mock)
		ok, err := repo.RevokeIfValid(context.Background(), "hash123")

		assert.Error(t, err)
		assert.False(t, ok)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRefreshTokenRepository_RevokeAllForUser(t *testing.T) {
	t.Run("revokes all tokens for user successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE refresh_tokens").
			WithArgs("user-123", pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 3))

		repo := NewRefreshTokenRepository(mock)
		err = repo.RevokeAllForUser(context.Background(), "user-123")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRefreshTokenRepository_DeleteExpired(t *testing.T) {
	t.Run("deletes expired tokens successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM refresh_tokens").
			WithArgs(pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("DELETE", 5))

		repo := NewRefreshTokenRepository(mock)
		err = repo.DeleteExpired(context.Background())

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRefreshToken_IsValid(t *testing.T) {
	t.Run("returns true for valid token", func(t *testing.T) {
		token := &model.RefreshToken{ExpiresAt: time.Now().Add(24 * time.Hour), RevokedAt: nil}
		assert.True(t, token.IsValid())
	})

	t.Run("returns false for expired token", func(t *testing.T) {
		token := &model.RefreshToken{ExpiresAt: time.Now().Add(-24 * time.Hour), RevokedAt: nil}
		assert.False(t, token.IsValid())
	})

	t.Run("returns false for revoked token", func(t *testing.T) {
		revokedAt := time.Now()
		token := &model.RefreshToken{ExpiresAt: time.Now().Add(24 * time.Hour), RevokedAt: &revokedAt}
		assert.False(t, token.IsValid())
	})
}
