package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	subService "github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAIRequestScopeOnlyBoundsAIConsumers(t *testing.T) {
	for _, path := range []string{"/api/v1/match-score", "/api/v1/resume-builder/ai/improve-text", "/api/v1/resume-builder/:id/ats-check", "/api/v1/resumes/:id/autofill-profile", "/api/v1/jobs/:id/resume-tailoring", "/api/v1/resumes/:id/export"} {
		t.Run(path, func(t *testing.T) {
			r := gin.New()
			r.Use(AIRequestScope(nil))
			r.GET(path, func(c *gin.Context) {
				deadline, bounded := c.Request.Context().Deadline()
				if path == "/api/v1/resumes/:id/export" {
					require.False(t, bounded)
				} else {
					require.True(t, bounded)
					require.WithinDuration(t, time.Now().Add(90*time.Second), deadline, time.Second)
				}
				c.Status(204)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
			require.Equal(t, 204, w.Code)
		})
	}
}

func TestAIRequestScopeLeavesMCPRequestUnchanged(t *testing.T) {
	type requestMarker struct{}
	originalContext := context.WithValue(context.Background(), requestMarker{}, "mcp request")
	body := " \n" + `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"parse_job_page","arguments":{"page_text":"Posting data"}}}` + "\t\n"
	request := httptest.NewRequest("POST", "/api/mcp", strings.NewReader(body)).WithContext(originalContext)
	request.Header.Set("Idempotency-Key", "not-a-uuid")
	originalBody := request.Body
	subscriptions := subService.NewSubscriptionService(&MockSubscriptionRepository{}, nil, subService.BillingConfig{})
	router := gin.New()
	router.Use(AIRequestScope(subscriptions))
	called := false
	router.POST("/api/mcp", func(c *gin.Context) {
		called = true
		require.Same(t, originalContext, c.Request.Context())
		_, bounded := c.Request.Context().Deadline()
		require.False(t, bounded)
		operation, durable, found := ai.CurrentCreditOperation(c.Request.Context())
		require.False(t, found)
		require.False(t, durable)
		require.Zero(t, operation)
		require.True(t, originalBody == c.Request.Body, "middleware replaced the request body")
		actualBody, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Equal(t, []byte(body), actualBody)
		c.Status(204)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.True(t, called)
	require.Equal(t, 204, response.Code, response.Body.String())
	require.NoError(t, originalContext.Err())
}

type scopeFinisher struct {
	operation subModel.CreditOperation
	canceled  bool
	calls     int
}

func (f *scopeFinisher) FinishAIRequest(ctx context.Context) {
	f.operation, _, _ = ai.CurrentCreditOperation(ctx)
	f.canceled = ctx.Err() != nil
	f.calls++
}

func TestBeginAIRequestScopeIdentityDeadlineAndCleanup(t *testing.T) {
	finisher := &scopeFinisher{}
	owner := "before-auth"
	ctx, finish := BeginAIRequestScope(t.Context(), func() string { return owner }, "receipt", "tailor", "canonical-fingerprint", finisher)
	owner = "authenticated"
	operation, _, found := ai.CurrentCreditOperation(ctx)
	require.True(t, found)
	require.Equal(t, owner, operation.UserID)
	require.Equal(t, 7, operation.Credits)
	require.Equal(t, "receipt", operation.Key)
	require.Equal(t, "canonical-fingerprint", operation.Fingerprint)
	_, foreign := ai.CreditOperationFromContext(ctx, "other-user", "")
	require.False(t, foreign)
	deadline, bounded := ctx.Deadline()
	require.True(t, bounded)
	require.WithinDuration(t, time.Now().Add(90*time.Second), deadline, time.Second)
	finish()
	require.Equal(t, 1, finisher.calls)
	require.False(t, finisher.canceled)
	require.ErrorIs(t, ctx.Err(), context.Canceled)
	require.Equal(t, operation, finisher.operation)
}

func TestAIRequestScopeWebFingerprintAndOperationCosts(t *testing.T) {
	cases := []struct {
		path, kind string
		cost       int
	}{
		{"/api/v1/jobs/parse", "job_parse", 2},
		{"/api/v1/jobs/job/resume-tailoring", "tailor", 7},
		{"/api/v1/cover-letters/ai/generate", "cover_letter", 5},
		{"/api/v1/match-score", "match", 5},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			router := gin.New()
			router.Use(AIRequestScope(nil))
			body := `{"input":"posting"}`
			key := uuid.NewString()
			router.POST(tc.path, func(c *gin.Context) {
				c.Set("user_id", "authenticated")
				operation, _, found := ai.CurrentCreditOperation(c.Request.Context())
				require.True(t, found)
				require.Equal(t, "authenticated", operation.UserID)
				require.Equal(t, tc.kind, operation.Kind)
				require.Equal(t, tc.cost, operation.Credits)
				require.Equal(t, key, operation.Key)
				sum := sha256.Sum256([]byte("POST:" + tc.path + ":" + body))
				require.Equal(t, hex.EncodeToString(sum[:]), operation.Fingerprint)
				restored, err := io.ReadAll(c.Request.Body)
				require.NoError(t, err)
				require.Equal(t, body, string(restored))
				c.Status(204)
			})
			request := httptest.NewRequest("POST", tc.path, strings.NewReader(body))
			request.Header.Set("Idempotency-Key", key)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, 204, response.Code)
		})
	}
}
func TestAIRequestScopeRejectsInvalidKeyAndOversizedBody(t *testing.T) {
	for _, tc := range []struct {
		name, key, body string
		status          int
	}{
		{"invalid key", "bad", "{}", 400},
		{"oversize", "", strings.Repeat("x", 11_534_337), 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AIRequestScope(nil))
			called := false
			router.POST("/api/v1/jobs/parse", func(c *gin.Context) { called = true; c.Status(204) })
			request := httptest.NewRequest("POST", "/api/v1/jobs/parse", strings.NewReader(tc.body))
			request.Header.Set("Idempotency-Key", tc.key)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, tc.status, response.Code)
			require.False(t, called)
		})
	}
}
