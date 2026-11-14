//go:build integration

package e2e

import (
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/oauth"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOAuthCIMDClientCleanup(t *testing.T) {
	fixture := newOAuthFixture(t)
	repo := oauth.NewRepository(fixture.db)
	old, fresh := time.Now().Add(-25*time.Hour), time.Now()
	for _, test := range []struct {
		name     string
		metadata bool
		fetched  *time.Time
		grant    string
		retained bool
	}{
		{"expired unused", true, &old, "", false},
		{"missing fetch date", true, nil, "", false},
		{"fresh unused", true, &fresh, "", true},
		{"DCR", false, nil, "", true},
		{"expired active grant", true, &old, "active", true},
		{"expired revoked grant", true, &old, "revoked", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			id := "https://client.test/" + uuid.NewString()
			require.NoError(t, repo.SaveClient(t.Context(), oauth.Client{ID: id, Name: test.name, RedirectURIs: []string{oauthCallback}, GrantTypes: []string{"authorization_code", "refresh_token"}, ResponseTypes: []string{"code"}, IsMetadata: test.metadata, FetchedAt: test.fetched}))
			grantID := uuid.NewString()
			if test.grant != "" {
				_, err := fixture.db.Exec(t.Context(), `INSERT INTO oauth_grants (id,user_id,client_id,scope,redirect_uri,revoked_at)
 VALUES ($1,$2,$3,'mcp',$4,CASE WHEN $5 THEN now() ELSE NULL END)`, grantID, fixture.userID, id, oauthCallback, test.grant == "revoked")
				require.NoError(t, err)
			}
			require.NoError(t, repo.Cleanup(t.Context()))
			client, err := repo.GetClient(t.Context(), id)
			if test.retained {
				require.NoError(t, err)
				require.Equal(t, id, client.ID)
			} else {
				require.ErrorIs(t, err, oauth.ErrNotFound)
			}
			if test.grant != "" {
				var count int
				require.NoError(t, fixture.db.QueryRow(t.Context(), `SELECT count(*) FROM oauth_grants WHERE id=$1`, grantID).Scan(&count))
				if test.retained {
					require.Equal(t, 1, count)
				} else {
					require.Zero(t, count)
				}
			}
		})
	}
}
