package oauth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestAuthorizationRateLimitsShareBothEndpoints(t *testing.T) {
	for _, identity := range []string{"user", "ip"} {
		t.Run(identity, func(t *testing.T) {
			redisServer := miniredis.RunT(t)
			redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
			t.Cleanup(func() { require.NoError(t, redisClient.Close()) })
			service, store := newTestService()
			router := gin.New()
			require.NoError(t, router.SetTrustedProxies(nil))
			auth := func(c *gin.Context) { c.Set("user_id", c.GetHeader("X-Test-User")); c.Next() }
			NewHandler(service).RegisterRoutes(router, auth, redisClient, nil, service.issuer)
			limit := authorizeUserRequestsPerMinute
			if identity == "ip" {
				limit = authorizeIPRequestsPerMinute
			}
			for index := 0; index <= limit+1; index++ {
				request := rateLimitRequest(t, identity, index)
				response := httptest.NewRecorder()
				router.ServeHTTP(response, request)
				if index >= limit {
					require.Equal(t, 429, response.Code, response.Body.String())
					require.JSONEq(t, `{"error_code":"RATE_LIMIT_EXCEEDED","error_message":"Too many requests, please try again later"}`, response.Body.String())
				} else {
					require.Equal(t, 400, response.Code, response.Body.String())
				}
			}
			require.Empty(t, store.code.Hash)
			redisServer.FastForward(time.Minute)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, rateLimitRequest(t, identity, 0))
			require.Equal(t, 400, response.Code)
		})
	}
}

func TestFreeCIMDAuthorizationErrorEnvelope(t *testing.T) {
	for _, test := range []struct{ cache, method string }{
		{"uncached", "GET"}, {"uncached", "POST"},
		{"expired", "GET"}, {"expired", "POST"},
	} {
		t.Run(test.cache+"/"+test.method, func(t *testing.T) {
			router, service, store, cookie := testHandler(t)
			service.paid = paidStub{subModel.ErrPaidFeature}
			fetcher := &fetchStub{}
			service.fetcher = fetcher
			params := validAuthorize()
			params.ClientID, params.Decision = "https://client.test/meta", "allow"
			if test.cache == "expired" {
				expired := time.Now().Add(-25 * time.Hour)
				store.client = &Client{ID: params.ClientID, IsMetadata: true, FetchedAt: &expired, RedirectURIs: []string{params.RedirectURI}}
			} else {
				store.client = nil
			}
			body, err := json.Marshal(params)
			require.NoError(t, err)
			path := "/api/v1/oauth/authorize"
			if test.method == "GET" {
				query := url.Values{"client_id": {params.ClientID}, "redirect_uri": {params.RedirectURI}, "response_type": {params.ResponseType},
					"code_challenge": {params.Challenge}, "code_challenge_method": {params.ChallengeMethod}, "scope": {params.Scope}, "resource": {params.Resource}, "state": {params.State}}
				path += "/preview?" + query.Encode()
			}
			response := oauthHTTP(router, test.method, path, string(body), "application/json", service.issuer, cookie)
			require.Equal(t, 403, response.Code)
			require.JSONEq(t, `{"error_code":"PAID_FEATURE","error_message":"Upgrade to Pro or Enterprise to connect your AI assistant."}`, response.Body.String())
			require.Empty(t, response.Header().Get("Location"))
			require.Zero(t, fetcher.calls)
			require.Zero(t, store.saved)
			require.Empty(t, store.code.Hash)
		})
	}
}

func rateLimitRequest(t *testing.T, identity string, index int) *http.Request {
	t.Helper()
	method, path := "GET", "/api/v1/oauth/authorize/preview"
	if index%2 == 1 {
		method, path = "POST", "/api/v1/oauth/authorize"
	}
	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("Origin", "https://cursopus.com")
	request.Header.Set("X-Test-User", "user")
	request.RemoteAddr = "192.0.2.1:1234"
	if identity == "user" {
		request.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", index+1)
	} else {
		request.Header.Set("X-Test-User", fmt.Sprintf("user-%d", index))
	}
	return request
}

func TestMetadataBusyResponse(t *testing.T) {
	router, service, store, cookie := testHandler(t)
	store.client = nil
	for range maxConcurrentMetadataFetches {
		service.metadataSlots <- struct{}{}
	}
	params := validAuthorize()
	params.ClientID, params.Decision = "https://client.test/meta", "allow"
	body, err := json.Marshal(params)
	require.NoError(t, err)
	response := oauthHTTP(router, "POST", "/api/v1/oauth/authorize", string(body), "application/json", service.issuer, cookie)
	require.Equal(t, 503, response.Code)
	require.Equal(t, "5", response.Header().Get("Retry-After"))
	require.JSONEq(t, `{"error_code":"OAUTH_METADATA_BUSY","error_message":"Client metadata service is busy. Please try again shortly."}`, response.Body.String())
	require.Empty(t, store.code.Hash)
}
