//go:build integration

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	patRepo "github.com/andreypavlenko/jobber/modules/apitokens/repository"
	patService "github.com/andreypavlenko/jobber/modules/apitokens/service"
	mcpModule "github.com/andreypavlenko/jobber/modules/mcp"
	"github.com/andreypavlenko/jobber/modules/oauth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const oauthIssuer = "https://cursopus.com"
const oauthCallback = "https://claude.ai/api/mcp/auth_callback"
const oauthVerifier = "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
const oauthChallenge = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

type oauthPaid struct{}

func (oauthPaid) RequirePaidPlan(context.Context, string) error { return nil }

type oauthFixture struct {
	router         *gin.Engine
	service        *oauth.Service
	db             *pgxpool.Pool
	cookie, userID string
}

func newOAuthFixture(t *testing.T) *oauthFixture {
	t.Helper()
	dsn, _ := newThrowawayDatabase(t)
	migration := newMigrator(t, repoMigrationsDir(t), dsn)
	require.NoError(t, migration.Migrate(58))
	require.NoError(t, migration.Migrate(57))
	require.NoError(t, migration.Migrate(58))
	db := connectTo(t, dsn)
	owner := seedAuditUser(t, db)
	jwt := auth.NewJWTManager("oauth-e2e-access", "oauth-e2e-refresh", time.Hour, time.Hour)
	cookie, err := jwt.GenerateAccessToken(owner)
	require.NoError(t, err)
	service := oauth.NewService(oauth.NewRepository(db), oauthPaid{}, oauthIssuer, nil)
	router := gin.New()
	router.Use(oauth.PublicCORS(), oauth.Privacy(), httpPlatform.CORSMiddleware(oauthIssuer), httpPlatform.CanonicalUUIDParams())
	oauth.NewHandler(service).RegisterRoutes(router, auth.AuthMiddleware(jwt), rdb, zap.NewNop(), oauthIssuer)
	pat := patService.NewAPITokenService(patRepo.NewAPITokenRepository(db), oauthPaid{}, rdb, nil)
	mcpModule.RegisterRoutes(router, mcpModule.NewServer(nil, nil, nil, nil), mcpModule.NewTokenAuthenticator(pat, service), oauthPaid{}, rdb, zap.NewNop(), oauthIssuer, oauthIssuer)
	require.Empty(t, httpPlatform.UnclassifiedPathParams(router.Routes()))
	return &oauthFixture{router: router, service: service, db: db, cookie: cookie, userID: owner}
}

func (f *oauthFixture) request(method, path, body, media, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.RemoteAddr = "192.0.2.55:12345"
	request.Header.Set("Content-Type", media)
	if strings.HasPrefix(path, "/api/v1/") {
		request.AddCookie(&http.Cookie{Name: auth.AccessTokenCookie, Value: f.cookie})
		request.Header.Set("Origin", oauthIssuer)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if path == "/api/mcp" {
		request.Header.Set("Accept", "application/json, text/event-stream")
		request.Header.Set("MCP-Protocol-Version", "2025-06-18")
	}
	response := httptest.NewRecorder()
	f.router.ServeHTTP(response, request)
	return response
}

func (f *oauthFixture) consent(t *testing.T) (oauth.Client, string) {
	t.Helper()
	registered := f.request("POST", "/api/oauth/register", `{"client_name":"E2E Assistant","redirect_uris":["`+oauthCallback+`"],"token_endpoint_auth_method":"none"}`, "application/json", "")
	require.Equal(t, 201, registered.Code, registered.Body.String())
	var client oauth.Client
	require.NoError(t, json.Unmarshal(registered.Body.Bytes(), &client))
	query := url.Values{"response_type": {"code"}, "client_id": {client.ID}, "redirect_uri": {oauthCallback}, "code_challenge": {oauthChallenge}, "code_challenge_method": {"S256"}, "state": {"e2e-state"}, "scope": {"mcp"}, "resource": {oauthIssuer + "/api/mcp"}}
	preview := f.request("GET", "/api/v1/oauth/authorize/preview?"+query.Encode(), "", "", "")
	require.Equal(t, 200, preview.Code, preview.Body.String())
	require.Contains(t, preview.Body.String(), `"plan_allowed":true`)
	body, err := json.Marshal(oauth.AuthorizeParams{ResponseType: "code", ClientID: client.ID, RedirectURI: oauthCallback, Challenge: oauthChallenge, ChallengeMethod: "S256", State: "e2e-state", Scope: "mcp", Resource: oauthIssuer + "/api/mcp", Decision: "allow"})
	require.NoError(t, err)
	approved := f.request("POST", "/api/v1/oauth/authorize", string(body), "application/json", "")
	require.Equal(t, 200, approved.Code, approved.Body.String())
	var result struct {
		RedirectTo string `json:"redirect_to"`
	}
	require.NoError(t, json.Unmarshal(approved.Body.Bytes(), &result))
	target, err := url.Parse(result.RedirectTo)
	require.NoError(t, err)
	require.Equal(t, "e2e-state", target.Query().Get("state"))
	require.Equal(t, oauthIssuer, target.Query().Get("iss"))
	return client, target.Query().Get("code")
}

func (f *oauthFixture) exchange(t *testing.T, values url.Values) oauth.TokenResponse {
	t.Helper()
	response := f.request("POST", "/api/oauth/token", values.Encode(), "application/x-www-form-urlencoded", "")
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	var tokens oauth.TokenResponse
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &tokens))
	return tokens
}

func TestOAuthFullFlow(t *testing.T) {
	fixture := newOAuthFixture(t)
	client, code := fixture.consent(t)
	tokens := fixture.exchange(t, url.Values{"grant_type": {"authorization_code"}, "client_id": {client.ID}, "code": {code}, "redirect_uri": {oauthCallback}, "code_verifier": {oauthVerifier}, "resource": {oauthIssuer + "/api/mcp"}})
	initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"oauth-e2e","version":"1"}}}`
	require.Equal(t, 200, fixture.request("POST", "/api/mcp", initialize, "application/json", tokens.AccessToken).Code)
	identity, err := fixture.service.Authenticate(t.Context(), tokens.AccessToken)
	require.NoError(t, err)
	require.Equal(t, fixture.userID, identity.UserID)
	verifyOAuthActivity(t, fixture, identity.GrantID)
	// Real SQL audience filtering, including a valid token hash with a different audience.
	_, err = fixture.db.Exec(t.Context(), `UPDATE oauth_tokens SET resource='https://other.test/api/mcp' WHERE kind='access'`)
	require.NoError(t, err)
	_, err = fixture.service.Authenticate(t.Context(), tokens.AccessToken)
	require.ErrorIs(t, err, oauth.ErrInvalidGrant)
	refreshed := fixture.exchange(t, url.Values{"grant_type": {"refresh_token"}, "client_id": {client.ID}, "refresh_token": {tokens.RefreshToken}})
	require.NotEqual(t, tokens.RefreshToken, refreshed.RefreshToken)
	require.Equal(t, 200, fixture.request("POST", "/api/mcp", initialize, "application/json", refreshed.AccessToken).Code)
	reuse := fixture.request("POST", "/api/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {client.ID}, "refresh_token": {tokens.RefreshToken}}.Encode(), "application/x-www-form-urlencoded", "")
	require.Equal(t, 400, reuse.Code, reuse.Body.String())
	require.Contains(t, reuse.Body.String(), "invalid_grant")
	require.Equal(t, 401, fixture.request("POST", "/api/mcp", initialize, "application/json", refreshed.AccessToken).Code)
	assertOAuthGrantRevoked(t, fixture, identity.GrantID)
}

func assertOAuthGrantRevoked(t *testing.T, f *oauthFixture, grantID string) {
	t.Helper()
	var active int
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT count(*) FROM oauth_grants WHERE id=$1 AND revoked_at IS NULL`, grantID).Scan(&active))
	require.Zero(t, active)
	require.NoError(t, f.db.QueryRow(t.Context(), `SELECT count(*) FROM oauth_tokens WHERE grant_id=$1 AND revoked_at IS NULL`, grantID).Scan(&active))
	require.Zero(t, active)
}

func TestOAuthCodeReplayRevokesGrant(t *testing.T) {
	fixture := newOAuthFixture(t)
	client, code := fixture.consent(t)
	values := url.Values{"grant_type": {"authorization_code"}, "client_id": {client.ID}, "code": {code}, "redirect_uri": {oauthCallback}, "code_verifier": {oauthVerifier}}
	tokens := fixture.exchange(t, values)
	identity, err := fixture.service.Authenticate(t.Context(), tokens.AccessToken)
	require.NoError(t, err)
	replay := fixture.request("POST", "/api/oauth/token", values.Encode(), "application/x-www-form-urlencoded", "")
	require.Equal(t, 400, replay.Code)
	require.Contains(t, replay.Body.String(), "invalid_grant")
	assertOAuthGrantRevoked(t, fixture, identity.GrantID)
}

func verifyOAuthActivity(t *testing.T, fixture *oauthFixture, grantID string) {
	t.Helper()
	require.NoError(t, mcpModule.NewAuditRepository(fixture.db).Record(t.Context(), mcpModule.AuditEntry{ID: uuid.NewString(), UserID: fixture.userID, OAuthGrantID: &grantID, Tool: "list_jobs", OK: true}))
	rows, err := patRepo.NewAPITokenRepository(fixture.db).ListActivity(t.Context(), fixture.userID, 20)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Nil(t, rows[0].APITokenID)
	require.Equal(t, "E2E Assistant", *rows[0].TokenName)
}

func TestOAuthGrantManagement(t *testing.T) {
	fixture := newOAuthFixture(t)
	client, code := fixture.consent(t)
	tokens := fixture.exchange(t, url.Values{"grant_type": {"authorization_code"}, "client_id": {client.ID}, "code": {code}, "redirect_uri": {oauthCallback}, "code_verifier": {oauthVerifier}})
	identity, err := fixture.service.Authenticate(t.Context(), tokens.AccessToken)
	require.NoError(t, err)
	grants := fixture.request("GET", "/api/v1/oauth/grants", "", "", "")
	require.Equal(t, 200, grants.Code)
	require.Contains(t, grants.Body.String(), "E2E Assistant")
	require.Contains(t, grants.Body.String(), identity.GrantID)
	foreign := fixture.request("DELETE", "/api/v1/oauth/grants/"+uuid.NewString(), "", "", "")
	require.Equal(t, 204, foreign.Code)
	_, err = fixture.service.Authenticate(t.Context(), tokens.AccessToken)
	require.NoError(t, err)
	revoked := fixture.request("DELETE", "/api/v1/oauth/grants/"+identity.GrantID, "", "", "")
	require.Equal(t, 204, revoked.Code)
	assertOAuthGrantRevoked(t, fixture, identity.GrantID)
	grants = fixture.request("GET", "/api/v1/oauth/grants", "", "", "")
	require.Equal(t, "[]", grants.Body.String())
}
