package oauth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	webAuth "github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRegisteredRoutesHaveClassifiedParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _ := newTestService()
	router := gin.New()
	router.Use(Privacy(), PublicCORS(), httpPlatform.CORSMiddleware(service.issuer), httpPlatform.CanonicalUUIDParams())
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })
	NewHandler(service).RegisterRoutes(router, func(c *gin.Context) { c.Next() }, redisClient, nil, service.issuer)
	require.Empty(t, httpPlatform.UnclassifiedPathParams(router.Routes()))
	for _, endpoint := range []string{"register", "token", "revoke"} {
		response := oauthHTTP(router, "OPTIONS", "/api/oauth/"+endpoint, "", "", "https://client.test", "")
		require.Equal(t, 204, response.Code)
	}
}

func testHandler(t *testing.T) (*gin.Engine, *Service, *stubStore, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	service, store := newTestService()
	handler := NewHandler(service)
	router := gin.New()
	router.Use(PublicCORS(), httpPlatform.CORSMiddleware("https://cursopus.com"))
	router.GET("/.well-known/oauth-authorization-server", handler.Metadata)
	router.GET("/.well-known/oauth-protected-resource", handler.ProtectedResource)
	router.GET("/.well-known/oauth-protected-resource/api/mcp", handler.ProtectedResource)
	router.POST("/api/oauth/token", handler.Token)
	router.POST("/api/oauth/revoke", handler.Revoke)
	router.POST("/api/oauth/register", handler.Register)
	jwt := webAuth.NewJWTManager("test-oauth-access", "test-oauth-refresh", time.Hour, time.Hour)
	token, err := jwt.GenerateAccessToken("user")
	require.NoError(t, err)
	web := router.Group("/api/v1/oauth", webAuth.AuthMiddleware(jwt))
	web.GET("/authorize/preview", handler.Preview)
	web.POST("/authorize", trustedMutation("https://cursopus.com"), handler.Authorize)
	return router, service, store, token
}

func oauthHTTP(router http.Handler, method, path, body, media, origin, cookie string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", media)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: webAuth.AccessTokenCookie, Value: cookie})
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestMetadataAndPublicCORS(t *testing.T) {
	router, _, _, _ := testHandler(t)
	for _, path := range []string{"/.well-known/oauth-authorization-server", "/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/api/mcp"} {
		response := oauthHTTP(router, "GET", path, "", "", "https://unrelated.test", "")
		require.Equal(t, 200, response.Code)
		require.Equal(t, "*", response.Header().Get("Access-Control-Allow-Origin"))
		require.Empty(t, response.Header().Get("Access-Control-Allow-Credentials"))
		var metadata map[string]any
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &metadata))
		if strings.HasSuffix(path, "authorization-server") {
			require.Equal(t, "https://cursopus.com", metadata["issuer"])
			require.Equal(t, "https://cursopus.com/oauth/authorize", metadata["authorization_endpoint"])
			require.Equal(t, true, metadata["client_id_metadata_document_supported"])
			require.Equal(t, []any{"none"}, metadata["revocation_endpoint_auth_methods_supported"])
		} else {
			require.Equal(t, "https://cursopus.com/api/mcp", metadata["resource"])
		}
	}
	for _, path := range []string{"/api/oauth/token", "/api/oauth/register", "/api/oauth/revoke", "/.well-known/oauth-authorization-server", "/.well-known/oauth-protected-resource/api/mcp", "/.well-known/oauth-protected-resource"} {
		response := oauthHTTP(router, "OPTIONS", path, "", "", "https://cursopus.com", "")
		require.Equal(t, 204, response.Code)
		require.Equal(t, "*", response.Header().Get("Access-Control-Allow-Origin"))
		require.Empty(t, response.Header().Get("Access-Control-Allow-Credentials"))
	}
	require.Equal(t, 404, oauthHTTP(router, "GET", "/.well-known/openid-configuration", "", "", "", "").Code)
}

func TestAuthorizeCookieAndOrigin(t *testing.T) {
	router, _, store, cookie := testHandler(t)
	params := validAuthorize()
	params.Decision = "allow"
	body, err := json.Marshal(params)
	require.NoError(t, err)
	for _, test := range []struct {
		origin, cookie string
		status         int
	}{{"https://evil.test", cookie, 403}, {"", cookie, 403}, {"https://cursopus.com", "", 401}, {"https://cursopus.com", cookie, 200}} {
		response := oauthHTTP(router, "POST", "/api/v1/oauth/authorize", string(body), "application/json", test.origin, test.cookie)
		require.Equal(t, test.status, response.Code, response.Body.String())
		if test.status == 403 {
			require.JSONEq(t, `{"error_code":"ORIGIN_NOT_ALLOWED","error_message":"Trusted origin required"}`, response.Body.String())
		}
	}
	require.NotEmpty(t, store.code.Hash)
	response := oauthHTTP(router, "POST", "/api/v1/oauth/authorize", string(body), "text/plain", "https://cursopus.com", cookie)
	require.Equal(t, 400, response.Code)
}

func TestPreviewHTTPErrorSplit(t *testing.T) {
	router, _, _, cookie := testHandler(t)
	for _, redirect := range []string{"https://evil.test/cb", callback} {
		query := url.Values{"client_id": {"client"}, "redirect_uri": {redirect}, "response_type": {"token"}, "state": {"secret-state"}}
		response := oauthHTTP(router, "GET", "/api/v1/oauth/authorize/preview?"+query.Encode(), "", "", "", cookie)
		if redirect == callback {
			require.Equal(t, 200, response.Code)
			require.Contains(t, response.Body.String(), "redirect_to")
			require.Contains(t, response.Body.String(), "unsupported_response_type")
		} else {
			require.Equal(t, 400, response.Code)
			require.NotContains(t, response.Body.String(), "redirect_to")
		}
	}
}

func TestTokenHTTPValidation(t *testing.T) {
	router, _, _, _ := testHandler(t)
	for _, test := range []struct{ body, media, code string }{
		{"grant_type=password&client_id=client", "application/x-www-form-urlencoded", "unsupported_grant_type"},
		{"grant_type=refresh_token&client_id=client&refresh_token=invalid", "application/x-www-form-urlencoded", "invalid_grant"},
		{"grant_type=authorization_code&client_id=client&client_secret=secret", "application/x-www-form-urlencoded", "invalid_client"},
		{"grant_type=x&grant_type=y", "application/x-www-form-urlencoded", "invalid_request"},
		{"{}", "application/json", "invalid_request"},
	} {
		response := oauthHTTP(router, "POST", "/api/oauth/token", test.body, test.media, "https://client.test", "")
		require.Equal(t, 400, response.Code, response.Body.String())
		require.Contains(t, response.Body.String(), test.code)
		require.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	}
	for _, body := range []string{"token=unknown&client_id=client", "bad%", ""} {
		require.Equal(t, 200, oauthHTTP(router, "POST", "/api/oauth/revoke", body, "application/x-www-form-urlencoded", "", "").Code)
	}
}
