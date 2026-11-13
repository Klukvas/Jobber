package mcp

import (
	"net/http"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	patModel "github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHTTPRateLimit(t *testing.T) {
	server := miniredis.NewMiniRedis()
	if err := server.Start(); err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("sandbox blocks miniredis TCP bind: %v", err)
		}
		require.NoError(t, err)
	}
	t.Cleanup(server.Close)
	redisClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })
	router := gin.New()
	tokens := &fakeTokens{info: &patModel.AuthenticationInfo{UserID: userA, TokenID: tokenA}}
	RegisterRoutes(router, NewServer(newFakeJobs(), nil, nil, nil), tokens, &fakePaid{}, redisClient, zap.NewNop(), "")
	server.Set("ratelimit:ai_import:user:"+userA, "999")
	for index := 1; index <= 61; index++ {
		response := rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, listToolsCall)
		expected := http.StatusOK
		if index == 61 {
			expected = http.StatusTooManyRequests
		}
		require.Equal(t, expected, response.Code, "request %d: %s", index, response.Body.String())
		require.Equal(t, int64(index), tokens.calls.Load())
	}
	value, err := server.Get("ratelimit:mcp:user:" + userA)
	require.NoError(t, err)
	require.Equal(t, "61", value)
	// A different user's requests have a separate MCP bucket.
	tokens.info = &patModel.AuthenticationInfo{UserID: userB, TokenID: tokenA}
	require.Equal(t, http.StatusOK, rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, listToolsCall).Code)
}

func TestHTTPIPRateLimitBeforeAuthentication(t *testing.T) {
	server := miniredis.NewMiniRedis()
	if err := server.Start(); err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("sandbox blocks miniredis TCP bind: %v", err)
		}
		require.NoError(t, err)
	}
	t.Cleanup(server.Close)
	redisClient := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, redisClient.Close()) })
	tokens := &fakeTokens{err: patModel.ErrInvalidToken}
	paid, jobs := &fakePaid{}, newFakeJobs()
	router := gin.New()
	RegisterRoutes(router, NewServer(jobs, nil, jobs, nil), tokens, paid, redisClient, zap.NewNop(), "")
	for index := 1; index <= mcpIPRequestsPerMinute+1; index++ {
		response := rpcHTTP(router, http.MethodPost, "Bearer "+validPAT, listCall)
		if index <= mcpIPRequestsPerMinute {
			require.Equal(t, http.StatusUnauthorized, response.Code, response.Body.String())
			require.Equal(t, int64(index), tokens.calls.Load())
		} else {
			require.Equal(t, http.StatusTooManyRequests, response.Code, response.Body.String())
			require.Equal(t, int64(mcpIPRequestsPerMinute), tokens.calls.Load(), "rejected before Authenticate")
		}
	}
	require.Zero(t, paid.calls.Load())
	require.Zero(t, jobs.calls.Load())
}
