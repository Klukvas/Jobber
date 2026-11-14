package oauth

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestClientPersistence(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	repo := NewRepository(mock)
	client := Client{ID: "client", Name: "Assistant", RedirectURIs: []string{callback}, GrantTypes: []string{"authorization_code"}, ResponseTypes: []string{"code"}}
	mock.ExpectExec(`INSERT INTO oauth_clients`).WithArgs(client.ID, client.Name, client.RedirectURIs, client.GrantTypes, client.ResponseTypes, false, (*time.Time)(nil)).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	require.NoError(t, repo.SaveClient(t.Context(), client))
	mock.ExpectQuery(`SELECT client_id, client_name`).WithArgs("client").WillReturnRows(pgxmock.NewRows([]string{"client_id", "client_name", "redirect_uris", "grant_types", "response_types", "is_metadata_document", "fetched_at", "created_at"}).AddRow(client.ID, client.Name, client.RedirectURIs, client.GrantTypes, client.ResponseTypes, false, nil, int64(123)))
	loaded, err := repo.GetClient(t.Context(), "client")
	require.NoError(t, err)
	require.Equal(t, "none", loaded.AuthMethod)
	require.Equal(t, int64(123), loaded.IssuedAt)
	mock.ExpectQuery(`SELECT client_id, client_name`).WithArgs("missing").WillReturnError(pgx.ErrNoRows)
	_, err = repo.GetClient(t.Context(), "missing")
	require.ErrorIs(t, err, ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateCodeUsesExistingGrant(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	code := Code{Hash: "hash", GrantID: "new-grant", RedirectURI: callback, Challenge: testChallenge, Resource: "https://cursopus.com/api/mcp", ExpiresAt: time.Now().Add(time.Minute)}
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO oauth_grants`).WithArgs("new-grant", "owner", "client", callback).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("existing-grant"))
	mock.ExpectExec(`INSERT INTO oauth_authorization_codes`).WithArgs("hash", "existing-grant", callback, testChallenge, code.Resource, code.ExpiresAt).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectCommit()
	require.NoError(t, NewRepository(mock).CreateCode(t.Context(), "owner", "client", code))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAuthenticateAudienceAndExpiry(t *testing.T) {
	for _, valid := range []bool{true, false} {
		t.Run(map[bool]string{true: "valid", false: "invalid"}[valid], func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()
			// Guard against accidental removal of audience, lifetime, or revocation filters.
			query := mock.ExpectQuery(`(?s)t.resource=\$2.*t.expires_at>now\(\).*t.revoked_at IS NULL AND g.revoked_at IS NULL`).WithArgs("hash", "https://cursopus.com/api/mcp")
			if valid {
				query.WillReturnRows(pgxmock.NewRows([]string{"user_id", "id", "expires_at"}).AddRow("user", "grant", time.Now().Add(time.Hour)))
			} else {
				query.WillReturnError(pgx.ErrNoRows)
			}
			identity, err := NewRepository(mock).Authenticate(t.Context(), "hash", "https://cursopus.com/api/mcp")
			if valid {
				require.NoError(t, err)
				require.Equal(t, "grant", identity.GrantID)
			} else {
				require.ErrorIs(t, err, ErrInvalidGrant)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestListAndRevokeGrantsOwnership(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	repo := NewRepository(mock)
	mock.ExpectQuery(`(?s)FROM oauth_grants g JOIN oauth_clients.*WHERE g.user_id = \$1 AND g.revoked_at IS NULL`).WithArgs("owner").WillReturnRows(pgxmock.NewRows([]string{"id", "client_name", "client_id", "redirect_uri", "created_at", "last_used_at"}).AddRow("grant", "Assistant", "client", callback, time.Now(), nil)).RowsWillBeClosed()
	grants, err := repo.ListGrants(t.Context(), "owner")
	require.NoError(t, err)
	require.Len(t, grants, 1)
	require.Equal(t, "claude.ai", grants[0].RedirectHost)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM oauth_grants WHERE id=$1 AND user_id=$2 FOR UPDATE`)).WithArgs("other-grant", "owner").WillReturnError(pgx.ErrNoRows)
	mock.ExpectCommit()
	require.NoError(t, repo.RevokeGrant(t.Context(), "owner", "other-grant"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCleanupBoundsAndTransactionRollback(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()
	repo := NewRepository(mock)
	mock.ExpectExec(`(?s)DELETE FROM oauth_authorization_codes.*LIMIT 100 FOR UPDATE SKIP LOCKED`).WillReturnResult(pgxmock.NewResult("DELETE", 100))
	mock.ExpectExec(`(?s)DELETE FROM oauth_tokens.*LIMIT 100 FOR UPDATE SKIP LOCKED`).WillReturnResult(pgxmock.NewResult("DELETE", 100))
	mock.ExpectBegin()
	mock.ExpectQuery(`(?s)SELECT c.client_id FROM oauth_clients.*c.is_metadata_document.*LIMIT 100 FOR UPDATE OF c SKIP LOCKED`).WillReturnRows(pgxmock.NewRows([]string{"client_id"}))
	mock.ExpectCommit()
	require.NoError(t, repo.Cleanup(t.Context()))
	failure := errors.New("write failed")
	mock.ExpectBegin()
	mock.ExpectRollback()
	require.ErrorIs(t, repo.transaction(t.Context(), func(pgx.Tx) error { return failure }), failure)
	require.NoError(t, mock.ExpectationsWereMet())
}
