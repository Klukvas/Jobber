package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	jwtAuth "github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var validPAT = "jbr_pat_" + strings.Repeat("A", 43)

type fakeTokens struct {
	info  *patModel.AuthenticationInfo
	err   error
	calls atomic.Int64
}

func (f *fakeTokens) AuthenticateWithInfo(_ context.Context, token string) (*patModel.AuthenticationInfo, error) {
	f.calls.Add(1)
	if token != validPAT {
		return nil, patModel.ErrInvalidToken
	}
	return f.info, f.err
}

type fakePaid struct {
	err    error
	calls  atomic.Int64
	userID string
}

func (f *fakePaid) RequirePaidPlan(_ context.Context, userID string) error {
	f.calls.Add(1)
	f.userID = userID
	return f.err
}

func testRouter(jobs *fakeJobs, tokens *fakeTokens, paid *fakePaid) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(httpPlatform.RequestIDMiddleware())
	server := NewServer(jobs, nil, jobs, nil)
	registerHTTP(router, server, tokens, paid, testHTTPConfig(), zap.NewNop())
	jwt := jwtAuth.NewJWTManager("test-access", "test-refresh", time.Hour, time.Hour)
	router.GET("/api/v1/jobs", jwtAuth.AuthMiddleware(jwt), func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func rpcHTTP(router http.Handler, method, token, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, "/api/mcp", strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", token)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set("MCP-Protocol-Version", "2025-06-18")
	request.Header.Set("X-Request-ID", "test-request")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

const listCall = `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_jobs","arguments":{}}}`
const listToolsCall = `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`
const initializeCall = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`

func TestHTTPAuthenticationAndPlan(t *testing.T) {
	tests := []struct {
		name, header     string
		info             *patModel.AuthenticationInfo
		authErr, paidErr error
		status           int
		lookups          int64
	}{
		{name: "missing", status: 401},
		{name: "wrong scheme", header: "Basic " + validPAT, status: 401},
		{name: "jwt", header: "Bearer eyJwt", status: 401},
		{name: "malformed", header: "Bearer jbr_pat_bad", status: 401, lookups: 1},
		{name: "revoked", header: "Bearer " + validPAT, authErr: patModel.ErrInvalidToken, status: 401, lookups: 1},
		{name: "expired", header: "Bearer " + validPAT, info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA, ExpiresAt: time.Now().Add(-time.Hour)}, status: 401, lookups: 1},
		{name: "free", header: "Bearer " + validPAT, info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA}, paidErr: subModel.ErrPaidFeature, status: 403, lookups: 1},
		{name: "paid lookup fails", header: "Bearer " + validPAT, info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA}, paidErr: errors.New("database details"), status: 500, lookups: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			jobs := newFakeJobs()
			tokens := &fakeTokens{info: test.info, err: test.authErr}
			paid := &fakePaid{err: test.paidErr}
			response := rpcHTTP(testRouter(jobs, tokens, paid), http.MethodPost, test.header, listCall)
			require.Equal(t, test.status, response.Code, response.Body.String())
			require.Equal(t, test.lookups, tokens.calls.Load())
			require.Zero(t, jobs.calls.Load())
			require.NotContains(t, response.Body.String(), "database details")
			var envelope httpPlatform.ErrorResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
			require.NotEmpty(t, envelope.ErrorCode)
			if test.status == 403 {
				require.Contains(t, envelope.ErrorMessage, "Upgrade")
			}
		})
	}
}

func TestHTTPPaidJSONWorkflow(t *testing.T) {
	for _, plan := range []string{"Pro", "Enterprise", "nonexpiring"} {
		t.Run(plan, func(t *testing.T) {
			expiry := time.Now().Add(time.Hour)
			if plan == "nonexpiring" {
				expiry = time.Time{}
			}
			tokens := &fakeTokens{info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA, ExpiresAt: expiry}}
			jobs, paid := newFakeJobs(), &fakePaid{}
			router := testRouter(jobs, tokens, paid)
			for index, body := range []string{initializeCall, listToolsCall, listCall} {
				response := rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, body)
				require.Equal(t, 200, response.Code, response.Body.String())
				require.Contains(t, response.Header().Get("Content-Type"), "application/json")
				require.Empty(t, response.Header().Get("Mcp-Session-Id"))
				require.Equal(t, int64(index+1), tokens.calls.Load(), "one Authenticate per HTTP request")
				require.Equal(t, int64(index+1), paid.calls.Load())
				require.Equal(t, userA, paid.userID)
				require.NotContains(t, response.Body.String(), `"error"`)
				switch index {
				case 0:
					require.Contains(t, response.Body.String(), "cursopus")
					require.Contains(t, response.Body.String(), "untrusted")
				case 1:
					require.Contains(t, response.Body.String(), "list_stages")
					require.Contains(t, response.Body.String(), "archive_job")
				case 2:
					require.Contains(t, response.Body.String(), "Engineer")
					require.NotContains(t, response.Body.String(), "private posting")
				}
			}
			require.Equal(t, userA, jobs.lastUser)
		})
	}
}

func TestHTTPBodyLimitAndMethods(t *testing.T) {
	tokens := &fakeTokens{info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA}}
	jobs := newFakeJobs()
	router := testRouter(jobs, tokens, &fakePaid{})
	response := rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, strings.Repeat(" ", maxBodyBytes+1))
	require.Equal(t, 413, response.Code, response.Body.String())
	require.Zero(t, jobs.calls.Load())
	request := httptest.NewRequest(http.MethodPost, "/api/mcp", strings.NewReader(strings.Repeat(" ", maxBodyBytes+1)))
	request.ContentLength = -1 // Chunked/unknown size must also be capped.
	request.Header.Set("Authorization", "Bearer "+validPAT)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, 413, response.Code)
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		require.Equal(t, 401, rpcHTTP(router, method, "", "").Code)
		response = rpcHTTP(router, method, "Bearer "+validPAT, "")
		require.Equal(t, http.StatusMethodNotAllowed, response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	request.Header.Set("Authorization", "Bearer "+validPAT)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, 401, response.Code)
}

func TestHTTPTokenInfoReachesSDK(t *testing.T) {
	tokens := &fakeTokens{info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA, ExpiresAt: time.Now().Add(time.Hour)}}
	server := NewServer(newFakeJobs(), nil, nil, nil)
	server.AddReceivingMiddleware(func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			if call, ok := req.(*sdk.CallToolRequest); ok {
				require.NotNil(t, call.Extra)
				require.Equal(t, userA, call.Extra.TokenInfo.UserID)
				require.Equal(t, tokenA, call.Extra.TokenInfo.Extra["api_token_id"])
				require.Equal(t, tokens.info.ExpiresAt, call.Extra.TokenInfo.Expiration)
				require.Equal(t, "test-request", ctx.Value(requestIDKey{}))
			}
			return next(ctx, method, req)
		}
	})
	router := gin.New()
	router.Use(httpPlatform.RequestIDMiddleware())
	registerHTTP(router, server, tokens, &fakePaid{}, testHTTPConfig(), zap.NewNop())
	response := rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, listCall)
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), "Engineer")
	require.Equal(t, int64(1), tokens.calls.Load())
}

func testHTTPConfig() httpConfig {
	pass := func(c *gin.Context) { c.Next() }
	return httpConfig{ipLimiter: pass, userLimiter: pass}
}
