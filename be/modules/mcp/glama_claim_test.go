package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGlamaClaimIsPublic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	jobs := newFakeJobs()
	registerHTTP(router, NewServer(jobs, nil, jobs, nil), &fakeTokens{}, &fakePaid{}, testHTTPConfig(), zap.NewNop())

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, glamaClaimPath, nil))

	require.Equal(t, http.StatusOK, recorder.Code, "the directory fetches it without credentials")
	var body map[string]string
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, map[string]string{"$schema": glamaClaimSchema, "claim": glamaClaim}, body)
}

func TestGlamaClaimDoesNotOpenTheEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	jobs := newFakeJobs()
	registerHTTP(router, NewServer(jobs, nil, jobs, nil), &fakeTokens{}, &fakePaid{}, testHTTPConfig(), zap.NewNop())

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/mcp", nil))

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}
