package repository

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/auth/model"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenRepository_Create(t *testing.T) {
	t.Run("creates refresh token successfully", func(t *testing.T) {
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

		// Create a test wrapper
		repo := &testRefreshTokenRepo{mock: mock}
		err = repo.Create(context.Background(), token)

		require.NoError(t, err)
		assert.NotEmpty(t, token.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRefreshTokenRepository_GetByTokenHash(t *testing.T) {
	t.Run("returns token successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		tokenHash := "hash123"
		expectedToken := &model.RefreshToken{
			ID:        "token-1",
			UserID:    "user-123",
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(24 * time.Hour),
			CreatedAt: time.Now(),
		}

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "token_hash", "expires_at", "created_at", "revoked_at",
		}).AddRow(
			expectedToken.ID,
			expectedToken.UserID,
			expectedToken.TokenHash,
			expectedToken.ExpiresAt,
			expectedToken.CreatedAt,
			nil,
		)

		mock.ExpectQuery("SELECT id, user_id, token_hash, expires_at, created_at, revoked_at").
			WithArgs(tokenHash).
			WillReturnRows(rows)

		repo := &testRefreshTokenRepo{mock: mock}
		token, err := repo.GetByTokenHash(context.Background(), tokenHash)

		require.NoError(t, err)
		assert.Equal(t, expectedToken.ID, token.ID)
		assert.Equal(t, expectedToken.UserID, token.UserID)
		assert.Equal(t, expectedToken.TokenHash, token.TokenHash)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when token not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		tokenHash := "nonexistent-hash"

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "token_hash", "expires_at", "created_at", "revoked_at",
		})

		mock.ExpectQuery("SELECT id, user_id, token_hash, expires_at, created_at, revoked_at").
			WithArgs(tokenHash).
			WillReturnRows(rows)

		repo := &testRefreshTokenRepo{mock: mock}
		token, err := repo.GetByTokenHash(context.Background(), tokenHash)

		assert.Error(t, err)
		assert.Nil(t, token)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRefreshTokenRepository_Revoke(t *testing.T) {
	t.Run("revokes token successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		tokenHash := "hash123"

		mock.ExpectExec("UPDATE refresh_tokens").
			WithArgs(tokenHash, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := &testRefreshTokenRepo{mock: mock}
		err = repo.Revoke(context.Background(), tokenHash)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRefreshTokenRepository_RevokeAllForUser(t *testing.T) {
	t.Run("revokes all tokens for user successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		userID := "user-123"

		mock.ExpectExec("UPDATE refresh_tokens").
			WithArgs(userID, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 3))

		repo := &testRefreshTokenRepo{mock: mock}
		err = repo.RevokeAllForUser(context.Background(), userID)

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

		repo := &testRefreshTokenRepo{mock: mock}
		err = repo.DeleteExpired(context.Background())

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRefreshToken_IsValid(t *testing.T) {
	t.Run("returns true for valid token", func(t *testing.T) {
		token := &model.RefreshToken{
			ExpiresAt: time.Now().Add(24 * time.Hour),
			RevokedAt: nil,
		}
		assert.True(t, token.IsValid())
	})

	t.Run("returns false for expired token", func(t *testing.T) {
		token := &model.RefreshToken{
			ExpiresAt: time.Now().Add(-24 * time.Hour),
			RevokedAt: nil,
		}
		assert.False(t, token.IsValid())
	})

	t.Run("returns false for revoked token", func(t *testing.T) {
		revokedAt := time.Now()
		token := &model.RefreshToken{
			ExpiresAt: time.Now().Add(24 * time.Hour),
			RevokedAt: &revokedAt,
		}
		assert.False(t, token.IsValid())
	})
}

// testRefreshTokenRepo is a test wrapper that uses pgxmock
type testRefreshTokenRepo struct {
	mock pgxmock.PgxPoolIface
}

func (r *testRefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	token.ID = "test-token-id"
	_, err := r.mock.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.CreatedAt,
	)
	return err
}

func (r *testRefreshTokenRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`
	token := &model.RefreshToken{}
	err := r.mock.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.RevokedAt,
	)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (r *testRefreshTokenRepo) Revoke(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE token_hash = $1 AND revoked_at IS NULL
	`
	_, err := r.mock.Exec(ctx, query, tokenHash, time.Now().UTC())
	return err
}

func (r *testRefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := r.mock.Exec(ctx, query, userID, time.Now().UTC())
	return err
}

func (r *testRefreshTokenRepo) DeleteExpired(ctx context.Context) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE expires_at < $1
	`
	_, err := r.mock.Exec(ctx, query, time.Now().UTC())
	return err
}
