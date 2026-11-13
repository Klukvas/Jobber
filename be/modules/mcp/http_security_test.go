package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/gin-gonic/gin"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHTTPOriginValidation(t *testing.T) {
	for _, test := range []struct {
		name, origin string
		allowed      bool
	}{
		{name: "missing", allowed: true}, {name: "allowed", origin: "https://app.example.com", allowed: true},
		{name: "second configured", origin: "https://admin.example.com", allowed: true},
		{name: "disallowed", origin: "https://attacker.example.com"}, {name: "null", origin: "null"},
		{name: "suffix attack", origin: "https://app.example.com.attacker.example.com"},
		{name: "path", origin: "https://app.example.com/untrusted"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tokens := &fakeTokens{info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA}}
			jobs := newFakeJobs()
			router := gin.New()
			config := testHTTPConfig()
			config.allowedOrigins = "https://app.example.com, https://admin.example.com"
			registerHTTP(router, NewServer(jobs, nil, jobs, nil), tokens, &fakePaid{}, config, zap.NewNop())
			request := httptest.NewRequest(http.MethodPost, "/api/mcp", strings.NewReader(listCall))
			request.Header.Set("Authorization", "Bearer "+validPAT)
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "application/json, text/event-stream")
			request.Header.Set("MCP-Protocol-Version", "2025-06-18")
			if test.origin != "" {
				request.Header.Set("Origin", test.origin)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if test.allowed {
				require.Equal(t, 200, response.Code, response.Body.String())
				require.Equal(t, int64(1), tokens.calls.Load())
			} else {
				require.Equal(t, 403, response.Code, response.Body.String())
				require.Zero(t, tokens.calls.Load())
				require.Zero(t, jobs.calls.Load())
			}
		})
	}
}

func TestHTTPBearerChallengeForAllMethods(t *testing.T) {
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodDelete} {
		for _, mode := range []string{"missing", "invalid", "expired in gin", "expired before SDK"} {
			t.Run(method+"/"+mode, func(t *testing.T) {
				info := &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA, ExpiresAt: time.Now().Add(time.Hour)}
				tokens := &fakeTokens{info: info}
				config := testHTTPConfig()
				header := "Bearer " + validPAT
				switch mode {
				case "missing":
					header = ""
				case "invalid":
					tokens.err = patModel.ErrInvalidToken
				case "expired in gin":
					info.ExpiresAt = time.Now().Add(-time.Hour)
				case "expired before SDK":
					config.userLimiter = expireVerifiedPAT
				}
				router := gin.New()
				jobs := newFakeJobs()
				registerHTTP(router, NewServer(jobs, nil, jobs, nil), tokens, &fakePaid{}, config, zap.NewNop())
				response := rpcHTTP(router, method, header, listCall)
				require.Equal(t, 401, response.Code, response.Body.String())
				require.Equal(t, "Bearer", response.Header().Get("WWW-Authenticate"))
				require.Zero(t, jobs.calls.Load())
			})
		}
	}
}

func expireVerifiedPAT(c *gin.Context) {
	info := c.Request.Context().Value(verifiedPATKey{}).(*auth.TokenInfo)
	expired := *info
	expired.Expiration = time.Now().Add(-time.Hour)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), verifiedPATKey{}, &expired))
	c.Next()
}

func TestCachedVerifierFailsClosed(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/mcp", nil)
	request.Header.Set("Authorization", "Bearer "+validPAT)
	info, err := cachedVerifier(request.Context(), validPAT, request)
	require.ErrorIs(t, err, auth.ErrInvalidToken)
	require.Nil(t, info)
}

func TestOriginWildcardMatchesCORSConfiguration(t *testing.T) {
	require.True(t, originAllowed("https://native-browser.example", nil, true))
	require.False(t, originAllowed("https://native-browser.example?", nil, true))
	require.False(t, originAllowed("https://native-browser.example#", nil, true))
	require.False(t, originAllowed("https://attacker.example", []string{" * "}, false))
	require.False(t, originAllowed("https://attacker.example", []string{"*", "https://app.example"}, false))
	require.True(t, originAllowed("https://app.example", []string{"*", "https://app.example"}, false))
}
