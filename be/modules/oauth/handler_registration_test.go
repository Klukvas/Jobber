package oauth

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRegistrationErrorVocabulary(t *testing.T) {
	for _, test := range []struct{ name, body, code string }{
		{"missing redirects", `{}`, "invalid_redirect_uri"},
		{"invalid redirect", `{"redirect_uris":["http://evil.test/cb"]}`, "invalid_redirect_uri"},
		{"too many redirects", `{"redirect_uris":[` + strings.Repeat(`"`+callback+`",`, 10) + `"` + callback + `"]}`, "invalid_redirect_uri"},
		{"confidential client", `{"redirect_uris":["` + callback + `"],"token_endpoint_auth_method":"client_secret_basic"}`, "invalid_client_metadata"},
		{"unsupported grant", `{"redirect_uris":["` + callback + `"],"grant_types":["password"]}`, "invalid_client_metadata"},
		{"unsupported response", `{"redirect_uris":["` + callback + `"],"response_types":["token"]}`, "invalid_client_metadata"},
		{"long Unicode name", `{"redirect_uris":["` + callback + `"],"client_name":"` + strings.Repeat("😀", 101) + `"}`, "invalid_client_metadata"},
		{"wrong field type", `{"redirect_uris":42}`, "invalid_client_metadata"},
		{"malformed JSON", `{`, "invalid_client_metadata"},
	} {
		t.Run(test.name, func(t *testing.T) {
			router, _, store, _ := testHandler(t)
			response := oauthHTTP(router, "POST", "/api/oauth/register", test.body, "application/json", "", "")
			require.Equal(t, 400, response.Code, response.Body.String())
			var body map[string]string
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
			require.Equal(t, test.code, body["error"])
			require.NotEmpty(t, body["error_description"])
			require.Zero(t, store.saved)
		})
	}
}

func TestRegistrationUnicodeNameBoundary(t *testing.T) {
	router, _, _, _ := testHandler(t)
	name := strings.Repeat("😀", 100)
	response := oauthHTTP(router, "POST", "/api/oauth/register", `{"redirect_uris":["`+callback+`"],"client_name":"`+name+`"}`, "application/json", "", "")
	require.Equal(t, 201, response.Code, response.Body.String())
	var client Client
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &client))
	require.Equal(t, name, client.Name)
}

func TestRegistrationRejectLogsReason(t *testing.T) {
	for _, test := range []struct{ name, body, reason string }{
		{"bad redirect", `{"client_name":"Probe","redirect_uris":["` + callback + `","http://evil.test/cb"]}`, "redirect_uris[1]"},
		{"bad grant", `{"redirect_uris":["` + callback + `"],"grant_types":["password"]}`, "grant_types"},
		{"confidential", `{"redirect_uris":["` + callback + `"],"token_endpoint_auth_method":"client_secret_basic"}`, `token_endpoint_auth_method "client_secret_basic"`},
		{"wrong type", `{"redirect_uris":42}`, "cannot unmarshal number"},
	} {
		t.Run(test.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.WarnLevel)
			service, _ := newTestService()
			handler := &Handler{service: service, logger: zap.New(core)}
			router := gin.New()
			router.POST("/api/oauth/register", handler.Register)

			response := oauthHTTP(router, "POST", "/api/oauth/register", test.body, "application/json", "", "")

			require.Equal(t, 400, response.Code)
			require.NotContains(t, response.Body.String(), test.reason)
			entries := logs.FilterMessage("OAuth client registration rejected").All()
			require.Len(t, entries, 1)
			require.Contains(t, entries[0].ContextMap()["reason"], test.reason)
		})
	}
}

func TestRegistrationRejectLogCapsFields(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	service, _ := newTestService()
	handler := &Handler{service: service, logger: zap.New(core)}
	router := gin.New()
	router.POST("/api/oauth/register", handler.Register)
	redirects := `"` + strings.Repeat("x", 500) + `"` + strings.Repeat(`,"http://evil.test/cb"`, 11)

	oauthHTTP(router, "POST", "/api/oauth/register", `{"redirect_uris":[`+redirects+`]}`, "application/json", "", "")

	entry := logs.All()[0].ContextMap()
	logged := entry["redirect_uris"].([]any)
	require.Len(t, logged, 10)
	require.LessOrEqual(t, len([]rune(logged[0].(string))), 201)
}
