//go:build integration

package e2e

import (
	"net/url"
	"testing"

	"github.com/andreypavlenko/jobber/modules/oauth"
	"github.com/stretchr/testify/require"
)

func TestOAuthTokenOnlyRevocation(t *testing.T) {
	for _, kind := range []string{"access", "refresh"} {
		t.Run(kind, func(t *testing.T) {
			fixture := newOAuthFixture(t)
			client, code := fixture.consent(t)
			tokens := fixture.exchange(t, url.Values{"grant_type": {"authorization_code"}, "client_id": {client.ID}, "code": {code}, "redirect_uri": {oauthCallback}, "code_verifier": {oauthVerifier}})
			// Both credentials work before revocation; use the freshly rotated pair.
			tokens = fixture.exchange(t, url.Values{"grant_type": {"refresh_token"}, "client_id": {client.ID}, "refresh_token": {tokens.RefreshToken}})
			identity, err := fixture.service.Authenticate(t.Context(), tokens.AccessToken)
			require.NoError(t, err)
			token := tokens.AccessToken
			if kind == "refresh" {
				token = tokens.RefreshToken
			}
			// An explicitly wrong binding must not revoke somebody else's grant.
			wrong := fixture.request("POST", "/api/oauth/revoke", url.Values{"token": {token}, "client_id": {"other-client"}}.Encode(), "application/x-www-form-urlencoded", "")
			require.Equal(t, 200, wrong.Code)
			_, err = fixture.service.Authenticate(t.Context(), tokens.AccessToken)
			require.NoError(t, err)
			// No client_id, with a deliberately wrong hint: lookup must search both kinds.
			revoked := fixture.request("POST", "/api/oauth/revoke", url.Values{"token": {token}, "token_type_hint": {"unknown"}}.Encode(), "application/x-www-form-urlencoded", "")
			require.Equal(t, 200, revoked.Code)
			_, err = fixture.service.Authenticate(t.Context(), tokens.AccessToken)
			require.ErrorIs(t, err, oauth.ErrInvalidGrant)
			refresh := fixture.request("POST", "/api/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {client.ID}, "refresh_token": {tokens.RefreshToken}}.Encode(), "application/x-www-form-urlencoded", "")
			require.Equal(t, 400, refresh.Code, refresh.Body.String())
			require.Contains(t, refresh.Body.String(), "invalid_grant")
			assertOAuthGrantRevoked(t, fixture, identity.GrantID)
			require.Equal(t, 200, fixture.request("POST", "/api/oauth/revoke", url.Values{"token": {token}}.Encode(), "application/x-www-form-urlencoded", "").Code)
		})
	}
}
