package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMock(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, mock.ExpectationsWereMet()); mock.Close() })
	return mock
}

func expectOwnerLock(mock pgxmock.PgxPoolIface, userID string, count int) {
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT id FROM users WHERE id = \$1 FOR UPDATE`).WithArgs(userID).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(userID))
	mock.ExpectQuery(`SELECT count\(\*\) FROM api_tokens WHERE user_id = \$1 AND revoked_at IS NULL AND \(expires_at IS NULL OR expires_at > now\(\)\)`).
		WithArgs(userID).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(count))
}

func TestAPITokenRepository_Create(t *testing.T) {
	mock := newMock(t)
	token := &model.APIToken{ID: "token", UserID: "owner", Name: "name", TokenPrefix: "jbr_pat_abcd", TokenHash: "hash", Scopes: []string{"mcp"}}
	expectOwnerLock(mock, "owner", 9)
	now := time.Now()
	mock.ExpectQuery(`INSERT INTO api_tokens`).WithArgs(token.ID, token.UserID, token.Name, token.TokenPrefix, token.TokenHash, token.Scopes, token.ExpiresAt).
		WillReturnRows(pgxmock.NewRows([]string{"created_at"}).AddRow(now))
	mock.ExpectCommit()
	got, err := NewAPITokenRepository(mock).Create(t.Context(), token)
	require.NoError(t, err)
	assert.Equal(t, now, got.CreatedAt)
	assert.True(t, token.CreatedAt.IsZero(), "must not mutate caller's token")
}

func TestAPITokenRepository_CreateCap(t *testing.T) {
	mock := newMock(t)
	expectOwnerLock(mock, "owner", 10)
	mock.ExpectRollback()
	got, err := NewAPITokenRepository(mock).Create(t.Context(), &model.APIToken{UserID: "owner"})
	require.ErrorIs(t, err, model.ErrTokenLimit)
	assert.Nil(t, got)
}

func TestAPITokenRepository_CreateRollback(t *testing.T) {
	mock := newMock(t)
	token := &model.APIToken{UserID: "owner"}
	expectOwnerLock(mock, "owner", 0)
	failure := errors.New("insert failed")
	mock.ExpectQuery(`INSERT INTO api_tokens`).
		WithArgs(token.ID, token.UserID, token.Name, token.TokenPrefix, token.TokenHash, token.Scopes, token.ExpiresAt).
		WillReturnError(failure)
	mock.ExpectRollback()
	got, err := NewAPITokenRepository(mock).Create(t.Context(), token)
	require.ErrorIs(t, err, failure)
	assert.Nil(t, got)
}

func tokenRows() *pgxmock.Rows {
	return pgxmock.NewRows([]string{"id", "user_id", "name", "token_prefix", "token_hash", "scopes", "last_used_at", "expires_at", "revoked_at", "created_at"})
}

func TestAPITokenRepository_GetByHash(t *testing.T) {
	mock := newMock(t)
	now := time.Now()
	mock.ExpectQuery(`FROM api_tokens WHERE token_hash = \$1`).WithArgs("sha256hash").
		WillReturnRows(tokenRows().AddRow("token", "owner", "laptop", "jbr_pat_abcd", "sha256hash", []string{"mcp"}, nil, nil, nil, now))
	got, err := NewAPITokenRepository(mock).GetByHash(t.Context(), "sha256hash")
	require.NoError(t, err)
	assert.Equal(t, "owner", got.UserID)
	assert.Equal(t, "token", got.ID)
}

func TestAPITokenRepository_GetByHashNotFound(t *testing.T) {
	mock := newMock(t)
	mock.ExpectQuery(`FROM api_tokens WHERE token_hash = \$1`).WithArgs("unknown").WillReturnError(pgx.ErrNoRows)
	_, err := NewAPITokenRepository(mock).GetByHash(t.Context(), "unknown")
	require.ErrorIs(t, err, model.ErrTokenNotFound)
}

func TestAPITokenRepository_List(t *testing.T) {
	mock := newMock(t)
	now := time.Now()
	mock.ExpectQuery(`FROM api_tokens WHERE user_id = \$1 ORDER BY created_at DESC, id DESC`).WithArgs("owner").
		WillReturnRows(tokenRows().AddRow("token", "owner", "laptop", "prefix", "hash", []string{"mcp"}, nil, nil, nil, now)).RowsWillBeClosed()
	got, err := NewAPITokenRepository(mock).List(t.Context(), "owner")
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "token", got[0].ID)
}

func TestAPITokenRepository_ListIterationFailure(t *testing.T) {
	mock := newMock(t)
	failure := errors.New("read failed")
	mock.ExpectQuery(`FROM api_tokens WHERE user_id`).WithArgs("owner").
		WillReturnRows(tokenRows().AddRow("token", "owner", "name", "prefix", "hash", []string{"mcp"}, nil, nil, nil, time.Now()).RowError(0, failure)).RowsWillBeClosed()
	_, err := NewAPITokenRepository(mock).List(t.Context(), "owner")
	require.ErrorIs(t, err, failure)
}

func TestAPITokenRepository_Revoke(t *testing.T) {
	tests := []struct {
		name     string
		affected int64
		want     error
	}{
		{"owned", 1, nil}, {"another user or missing", 0, model.ErrTokenNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mock := newMock(t)
			mock.ExpectExec(`UPDATE api_tokens SET revoked_at = COALESCE\(revoked_at, now\(\)\) WHERE id = \$1 AND user_id = \$2`).
				WithArgs("token", "owner").WillReturnResult(pgxmock.NewResult("UPDATE", test.affected))
			err := NewAPITokenRepository(mock).Revoke(t.Context(), "owner", "token")
			require.ErrorIs(t, err, test.want)
		})
	}
}

func TestAPITokenRepository_TouchLastUsed(t *testing.T) {
	mock := newMock(t)
	mock.ExpectExec(`UPDATE api_tokens SET last_used_at = now\(\) WHERE id = \$1 AND \(last_used_at IS NULL OR last_used_at <= now\(\) - interval '1 minute'\)`).
		WithArgs("token").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	require.NoError(t, NewAPITokenRepository(mock).TouchLastUsed(t.Context(), "token"))
}
