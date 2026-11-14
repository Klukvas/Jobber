package oauth

import (
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/require"
)

func TestRepositoryExchangeAtomicity(t *testing.T) {
	for _, kind := range []string{"authorization_code", "refresh_token"} {
		for _, mode := range []string{"issue", "reuse", "expired", "revoked", "wrong client", "wrong audience", "wrong verifier", "wrong redirect"} {
			t.Run(kind+"/"+mode, func(t *testing.T) {
				mock, err := pgxmock.NewPool()
				require.NoError(t, err)
				defer mock.Close()
				request := ExchangeRequest{Kind: kind, ClientID: "client", Credential: "secret", RedirectURI: callback, Verifier: testVerifier}
				pair := TokenPair{AccessHash: "access", RefreshHash: "refresh", Resource: "https://cursopus.com/api/mcp", AccessExpiry: time.Now().Add(time.Hour), RefreshExpiry: time.Now().Add(24 * time.Hour)}
				record := credential{grantID: "grant", clientID: "client", resource: pair.Resource, expires: time.Now().Add(time.Hour), redirect: callback, challenge: testChallenge}
				now := time.Now()
				applyCredentialMode(&record, &request, mode, now)
				mock.ExpectBegin()
				mock.ExpectQuery(`SELECT g.id FROM oauth_grants`).WithArgs(hashSecret("secret")).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("grant"))
				expectCredentialRow(mock, kind, record)
				expectExchangeOutcome(mock, kind, mode, pair)
				err = NewRepository(mock).Exchange(t.Context(), request, pair)
				if mode == "issue" || kind == "refresh_token" && (mode == "wrong verifier" || mode == "wrong redirect") {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, ErrInvalidGrant)
				}
				require.NoError(t, mock.ExpectationsWereMet())
			})
		}
	}
}

func applyCredentialMode(record *credential, request *ExchangeRequest, mode string, now time.Time) {
	switch mode {
	case "reuse":
		record.used = &now
	case "expired":
		record.expires = now.Add(-time.Hour)
	case "revoked":
		record.revoked = &now
	case "wrong client":
		request.ClientID = "attacker"
	case "wrong audience":
		record.resource = "https://other.test/api/mcp"
	case "wrong verifier":
		request.Verifier = testVerifier + "a"
	case "wrong redirect":
		request.RedirectURI = "https://other.test/cb"
	}
}

func expectCredentialRow(mock pgxmock.PgxPoolIface, kind string, record credential) {
	if kind == "authorization_code" {
		mock.ExpectQuery(`SELECT g.client_id,g.revoked_at`).WithArgs(hashSecret("secret")).WillReturnRows(pgxmock.NewRows([]string{"client_id", "revoked_at", "resource", "expires_at", "used_at", "redirect_uri", "code_challenge"}).AddRow(record.clientID, record.revoked, record.resource, record.expires, record.used, record.redirect, record.challenge))
	} else {
		mock.ExpectQuery(`SELECT g.client_id,COALESCE`).WithArgs(hashSecret("secret")).WillReturnRows(pgxmock.NewRows([]string{"client_id", "revoked_at", "resource", "expires_at", "rotated_at"}).AddRow(record.clientID, record.revoked, record.resource, record.expires, record.used))
	}
}

func expectExchangeOutcome(mock pgxmock.PgxPoolIface, kind, mode string, pair TokenPair) {
	if mode == "reuse" {
		mock.ExpectExec(`UPDATE oauth_grants SET revoked_at`).WithArgs("grant").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`UPDATE oauth_tokens SET revoked_at`).WithArgs("grant").WillReturnResult(pgxmock.NewResult("UPDATE", 2))
		mock.ExpectCommit()
		return
	}
	success := mode == "issue" || kind == "refresh_token" && (mode == "wrong verifier" || mode == "wrong redirect")
	if !success {
		mock.ExpectRollback()
		return
	}
	if kind == "authorization_code" {
		mock.ExpectExec(`UPDATE oauth_authorization_codes SET used_at`).WithArgs(hashSecret("secret")).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	} else {
		mock.ExpectExec(`UPDATE oauth_tokens SET rotated_at`).WithArgs(hashSecret("secret")).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	}
	mock.ExpectExec(`INSERT INTO oauth_tokens`).WithArgs(pair.AccessHash, pair.RefreshHash, "grant", pair.Resource, pair.AccessExpiry, pair.RefreshExpiry).WillReturnResult(pgxmock.NewResult("INSERT", 2))
	mock.ExpectCommit()
}

func TestRevokeTokenRevokesGrant(t *testing.T) {
	for _, clientID := range []string{"", "client"} {
		t.Run("client="+clientID, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()
			mock.ExpectBegin()
			mock.ExpectQuery(`(?s)SELECT g.id FROM oauth_grants.*WHERE t.token_hash=\$1 AND \(\$2='' OR g.client_id=\$2\) FOR UPDATE OF g`).WithArgs("hash", clientID).WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow("grant"))
			mock.ExpectExec(`UPDATE oauth_grants SET revoked_at`).WithArgs("grant").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
			mock.ExpectExec(`UPDATE oauth_tokens SET revoked_at`).WithArgs("grant").WillReturnResult(pgxmock.NewResult("UPDATE", 2))
			mock.ExpectCommit()
			require.NoError(t, NewRepository(mock).RevokeToken(t.Context(), "hash", clientID))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
