package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/andreypavlenko/jobber/modules/oauth"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeOAuth struct{ err error }

func (f fakeOAuth) Authenticate(context.Context, string) (*oauth.Identity, error) {
	return &oauth.Identity{UserID: userA, GrantID: tokenA, ExpiresAt: time.Now().Add(time.Hour)}, f.err
}

func TestOAuthMCPAndChallenge(t *testing.T) {
	for _, mode := range []string{"valid OAuth", "invalid OAuth", "PAT", "missing"} {
		t.Run(mode, func(t *testing.T) {
			pat := &fakeTokens{info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA}}
			oauthToken := fakeOAuth{}
			header := "Bearer jbr_oat_" + strings.Repeat("A", 43)
			if mode == "invalid OAuth" {
				oauthToken.err = oauth.ErrInvalidGrant
			}
			if mode == "PAT" {
				header = "Bearer " + validPAT
			}
			if mode == "missing" {
				header = ""
			}
			config := testHTTPConfig()
			config.issuer = "https://cursopus.com"
			router := gin.New()
			jobs := newFakeJobs()
			registerHTTP(router, NewServer(jobs, nil, jobs, nil), NewTokenAuthenticator(pat, oauthToken), &fakePaid{}, config, zap.NewNop())
			response := rpcHTTP(router, http.MethodPost, header, initializeCall)
			if mode == "invalid OAuth" || mode == "missing" {
				require.Equal(t, 401, response.Code)
				expected := `Bearer resource_metadata="https://cursopus.com/.well-known/oauth-protected-resource/api/mcp", scope="mcp"`
				if mode == "invalid OAuth" {
					expected += `, error="invalid_token"`
				}
				require.Equal(t, expected, response.Header().Get("WWW-Authenticate"))
			} else {
				require.Equal(t, 200, response.Code, response.Body.String())
			}
			if mode == "PAT" {
				require.Equal(t, int64(1), pat.calls.Load())
			} else {
				require.Zero(t, pat.calls.Load())
			}
		})
	}
}

func TestOAuthTokenInfoHasOnlyGrant(t *testing.T) {
	tokens := NewTokenAuthenticator(nil, fakeOAuth{})
	info, err := tokens.AuthenticateWithInfo(t.Context(), "jbr_oat_token")
	require.NoError(t, err)
	require.Empty(t, info.TokenID)
	require.Equal(t, tokenA, info.OAuthGrantID)
	router := gin.New()
	router.GET("/", authenticatePAT(tokens), func(c *gin.Context) {
		verified, err := cachedVerifier(c.Request.Context(), "", c.Request)
		require.NoError(t, err)
		require.Equal(t, tokenA, verified.Extra["oauth_grant_id"])
		require.NotContains(t, verified.Extra, "api_token_id")
	})
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("Authorization", "Bearer jbr_oat_token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, 200, response.Code)
}

func TestOAuthAuditCarriesGrant(t *testing.T) {
	entries := make(chan AuditEntry, 1)
	server := auditProbeServer(t, auditRecorderFunc(func(_ context.Context, entry AuditEntry) error { entries <- entry; return nil }), zap.NewNop(), func(_ context.Context, _ string, input auditProbeInput) (auditProbeInput, error) { return input, nil })
	router := gin.New()
	registerHTTP(router, server, NewTokenAuthenticator(nil, fakeOAuth{}), &fakePaid{}, testHTTPConfig(), zap.NewNop())
	response := rpcHTTP(router, http.MethodPost, "Bearer jbr_oat_"+strings.Repeat("A", 43), auditProbeCall)
	require.Equal(t, 200, response.Code, response.Body.String())
	select {
	case entry := <-entries:
		require.Nil(t, entry.APITokenID)
		require.Equal(t, tokenA, *entry.OAuthGrantID)
	case <-time.After(time.Second):
		t.Fatal("OAuth call was not audited")
	}
}
