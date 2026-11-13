package apitokens

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/auth"
	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/gin-gonic/gin"
)

type stubAuthenticator struct {
	err   error
	calls int
}

func (a *stubAuthenticator) Authenticate(_ context.Context, _ string) (string, string, error) {
	a.calls++
	if a.err != nil {
		return "", "", a.err
	}
	return "owner", "token-id", nil
}

func TestPATAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pat := "jbr_pat_" + strings.Repeat("a", 43)
	tests := []struct {
		name, header  string
		err           error
		status, calls int
	}{
		{name: "missing", status: 401},
		{name: "JWT", header: "Bearer eyJhbGciOiJIUzI1NiJ9.payload.signature", status: 401},
		{name: "wrong scheme", header: "Basic " + pat, status: 401},
		{name: "rejected token", header: "Bearer " + pat, err: errors.New("invalid token"), status: 401, calls: 1},
		{name: "valid", header: "Bearer " + pat, status: 200, calls: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authenticator := &stubAuthenticator{err: test.err}
			router := gin.New()
			router.GET("/protected", PATAuthMiddleware(authenticator), func(c *gin.Context) {
				if c.GetString("user_id") != "owner" || c.GetString("auth_via") != "pat" || c.GetString("api_token_id") != "token-id" {
					t.Error("missing PAT identity in context")
				}
				c.Status(200)
			})
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", test.header)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status || authenticator.calls != test.calls {
				t.Fatalf("status=%d calls=%d, want status=%d calls=%d", response.Code, authenticator.calls, test.status, test.calls)
			}
			if test.status == 401 {
				checkUnauthorized(t, response)
			}
		})
	}
}

func checkUnauthorized(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	var envelope httpPlatform.ErrorResponse
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.ErrorCode != "UNAUTHORIZED" {
		t.Errorf("unexpected error code %q", envelope.ErrorCode)
	}
}

func TestPATRejectedByExistingJWTAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := auth.NewJWTManager("unit-test-access", "unit-test-refresh", time.Hour, time.Hour)
	jwtToken, err := manager.GenerateAccessToken("owner")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/api/v1/api-tokens", auth.AuthMiddleware(manager), func(c *gin.Context) { c.Status(http.StatusOK) })
	pat := "jbr_pat_" + strings.Repeat("a", 43)
	for _, withCookie := range []bool{false, true} {
		t.Run(map[bool]string{false: "PAT only", true: "PAT overrides valid JWT cookie"}[withCookie], func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/v1/api-tokens", nil)
			request.Header.Set("Authorization", "Bearer "+pat)
			if withCookie {
				request.AddCookie(&http.Cookie{Name: auth.AccessTokenCookie, Value: jwtToken})
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("PAT status=%d, want 401", response.Code)
			}
			checkUnauthorized(t, response)
		})
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/api-tokens", nil)
	request.Header.Set("Authorization", "Bearer "+jwtToken)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("JWT status=%d, want 200", response.Code)
	}
}
