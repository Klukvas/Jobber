package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/calendar/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenRepository_Upsert(t *testing.T) {
	t.Run("inserts with generated id when id is empty", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		token := &model.CalendarToken{
			UserID:     "user-1",
			TokenBlob:  "encrypted-blob",
			TokenNonce: "nonce",
		}

		// id + both timestamps are generated inside the repo; encrypted blob/nonce
		// are stored verbatim (AnyArg keeps the assertion resilient to ciphertext).
		mock.ExpectExec("INSERT INTO google_calendar_tokens").
			WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := NewTokenRepository(mock)
		require.NoError(t, repo.Upsert(context.Background(), token))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("uses provided id and stores blob/nonce verbatim", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		token := &model.CalendarToken{
			ID:         "tok-existing",
			UserID:     "user-1",
			TokenBlob:  "cipher",
			TokenNonce: "nonce-b64",
		}

		mock.ExpectExec("INSERT INTO google_calendar_tokens").
			WithArgs("tok-existing", "user-1", "cipher", "nonce-b64", pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := NewTokenRepository(mock)
		require.NoError(t, repo.Upsert(context.Background(), token))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO google_calendar_tokens").
			WithArgs(pgxmock.AnyArg(), "user-1", pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errors.New("boom"))

		repo := NewTokenRepository(mock)
		err = repo.Upsert(context.Background(), &model.CalendarToken{UserID: "user-1"})
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTokenRepository_GetByUserID(t *testing.T) {
	t.Run("returns the stored token round-trip", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "user_id", "token_blob", "token_nonce", "created_at", "updated_at",
		}).AddRow("tok-1", "user-1", "cipher", "nonce", now, now)

		mock.ExpectQuery("SELECT id, user_id, token_blob, token_nonce, created_at, updated_at").
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewTokenRepository(mock)
		token, err := repo.GetByUserID(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, "tok-1", token.ID)
		assert.Equal(t, "user-1", token.UserID)
		assert.Equal(t, "cipher", token.TokenBlob)
		assert.Equal(t, "nonce", token.TokenNonce)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrNotConnected", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, token_blob, token_nonce, created_at, updated_at").
			WithArgs("user-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewTokenRepository(mock)
		token, err := repo.GetByUserID(context.Background(), "user-1")
		assert.Nil(t, token)
		assert.ErrorIs(t, err, model.ErrNotConnected)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, token_blob, token_nonce, created_at, updated_at").
			WithArgs("user-1").
			WillReturnError(errors.New("boom"))

		repo := NewTokenRepository(mock)
		token, err := repo.GetByUserID(context.Background(), "user-1")
		assert.Nil(t, token)
		assert.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrNotConnected)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTokenRepository_Delete(t *testing.T) {
	t.Run("deletes token", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM google_calendar_tokens").
			WithArgs("user-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := NewTokenRepository(mock)
		require.NoError(t, repo.Delete(context.Background(), "user-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotConnected when no rows affected", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM google_calendar_tokens").
			WithArgs("user-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := NewTokenRepository(mock)
		err = repo.Delete(context.Background(), "user-1")
		assert.ErrorIs(t, err, model.ErrNotConnected)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM google_calendar_tokens").
			WithArgs("user-1").
			WillReturnError(errors.New("boom"))

		repo := NewTokenRepository(mock)
		err = repo.Delete(context.Background(), "user-1")
		assert.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrNotConnected)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
