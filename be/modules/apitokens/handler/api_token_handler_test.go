package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/andreypavlenko/jobber/modules/apitokens/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type handlerRepo struct {
	token                *model.APIToken
	createCalls          int
	revokedID            string
	createErr, revokeErr error
}

func (r *handlerRepo) Create(_ context.Context, token *model.APIToken) (*model.APIToken, error) {
	r.createCalls++
	if r.createErr != nil {
		return nil, r.createErr
	}
	r.token = token
	return token, nil
}
func (r *handlerRepo) List(context.Context, string) ([]*model.APIToken, error) {
	if r.token == nil {
		return []*model.APIToken{}, nil
	}
	return []*model.APIToken{r.token}, nil
}
func (r *handlerRepo) Revoke(_ context.Context, _, tokenID string) error {
	r.revokedID = tokenID
	return r.revokeErr
}
func (r *handlerRepo) GetByHash(context.Context, string) (*model.APIToken, error) {
	return nil, model.ErrTokenNotFound
}
func (r *handlerRepo) TouchLastUsed(context.Context, string) error { return nil }

type handlerPaid struct{ err error }

func (p handlerPaid) RequirePaidPlan(context.Context, string) error { return p.err }

func handlerRouter(repo *handlerRepo, paidErr error) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc := service.NewAPITokenService(repo, handlerPaid{err: paidErr}, nil, nil)
	NewAPITokenHandler(svc).RegisterRoutes(router.Group("/api/v1"), func(c *gin.Context) { c.Set("user_id", "owner"); c.Next() })
	return router
}

func makeRequest(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestAPITokenHandler_CreateExpiry(t *testing.T) {
	tests := []struct {
		name, body string
		days       int
	}{
		{"omitted", `{"name":"laptop"}`, 90},
		{"null", `{"name":"laptop","expires_in_days":null}`, 0},
		{"30 days", `{"name":"laptop","expires_in_days":30}`, 30},
		{"365 days", `{"name":"laptop","expires_in_days":365}`, 365},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &handlerRepo{}
			response := makeRequest(handlerRouter(repo, nil), http.MethodPost, "/api/v1/api-tokens", test.body)
			checkStatus(t, response, 201)
			if response.Header().Get("Cache-Control") != "no-store" {
				t.Error("creation response must not be cached")
			}
			checkExpiry(t, repo.token.ExpiresAt, test.days)
			var created map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
				t.Fatal(err)
			}
			plaintext, ok := created["token"].(string)
			if !ok || !regexp.MustCompile(`^jbr_pat_[a-zA-Z0-9]{43}$`).MatchString(plaintext) {
				t.Fatal("missing plaintext token")
			}
			if strings.Contains(response.Body.String(), repo.token.TokenHash) {
				t.Error("creation leaked hash")
			}
			response = makeRequest(handlerRouter(repo, nil), http.MethodGet, "/api/v1/api-tokens", "")
			checkStatus(t, response, 200)
			if strings.Contains(response.Body.String(), `"token":`) || strings.Contains(response.Body.String(), repo.token.TokenHash) {
				t.Error("list leaked token or hash")
			}
		})
	}
}

func TestAPITokenHandler_CreateErrors(t *testing.T) {
	tests := []struct {
		name, body         string
		paidErr, createErr error
		status             int
		code               string
	}{
		{name: "free", body: `{"name":"ok"}`, paidErr: subModel.ErrPaidFeature, status: 403, code: "PAID_FEATURE"},
		{name: "cap", body: `{"name":"ok"}`, createErr: model.ErrTokenLimit, status: 403, code: "API_TOKEN_LIMIT_REACHED"},
		{name: "database", body: `{"name":"ok"}`, createErr: errors.New("private db details"), status: 500, code: "INTERNAL_ERROR"},
		{name: "blank", body: `{"name":" "}`, status: 400, code: "VALIDATION_ERROR"},
		{name: "invalid days", body: `{"name":"ok","expires_in_days":31}`, status: 400, code: "VALIDATION_ERROR"},
		{name: "fraction", body: `{"name":"ok","expires_in_days":30.1}`, status: 400, code: "VALIDATION_ERROR"},
		{name: "string", body: `{"name":"ok","expires_in_days":"90"}`, status: 400, code: "VALIDATION_ERROR"},
		{name: "malformed", body: `{`, status: 400, code: "VALIDATION_ERROR"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &handlerRepo{createErr: test.createErr}
			response := makeRequest(handlerRouter(repo, test.paidErr), http.MethodPost, "/api/v1/api-tokens", test.body)
			checkStatus(t, response, test.status)
			if !strings.Contains(response.Body.String(), `"error_code":"`+test.code+`"`) {
				t.Error("unexpected error envelope")
			}
			if strings.Contains(response.Body.String(), "private db details") {
				t.Error("leaked private details")
			}
			if repo.token != nil {
				t.Error("created token on rejected request")
			}
		})
	}
}

func TestAPITokenHandler_CreateControlCharacters(t *testing.T) {
	tests := []struct{ name, input string }{
		{"NUL", "laptop\x00"},
		{"newline", "lap\ntop"},
		{"tab", "lap\ttop"},
		{"escape", "laptop\x1b"},
		{"delete", "laptop\x7f"},
		{"C1 control", "laptop\u0085"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &handlerRepo{}
			body, err := json.Marshal(map[string]string{"name": test.input})
			if err != nil {
				t.Fatal(err)
			}
			response := makeRequest(handlerRouter(repo, nil), http.MethodPost, "/api/v1/api-tokens", string(body))
			checkStatus(t, response, http.StatusBadRequest)
			if !strings.Contains(response.Body.String(), `"error_code":"VALIDATION_ERROR"`) {
				t.Error("unexpected error envelope")
			}
			if repo.createCalls != 0 {
				t.Error("invalid name reached repository")
			}
		})
	}
}

func TestAPITokenHandler_Revoke(t *testing.T) {
	id := uuid.NewString()
	tests := []struct {
		name, id string
		err      error
		status   int
		wantID   string
	}{
		{"valid", id, nil, 204, id},
		{"URN", "urn:uuid:" + id, nil, 204, id},
		{"braces", "{" + id + "}", nil, 204, id},
		{"uppercase", strings.ToUpper(id), nil, 204, id},
		{"compact", strings.ReplaceAll(id, "-", ""), nil, 204, id},
		{"another user", id, model.ErrTokenNotFound, 404, id},
		{"invalid", "bad-id", nil, 400, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &handlerRepo{revokeErr: test.err}
			response := makeRequest(handlerRouter(repo, nil), http.MethodDelete, "/api/v1/api-tokens/"+test.id, "")
			checkStatus(t, response, test.status)
			if repo.revokedID != test.wantID {
				t.Errorf("repository id=%q want=%q", repo.revokedID, test.wantID)
			}
		})
	}
}

func TestAPITokenHandler_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAPITokenHandler(service.NewAPITokenService(&handlerRepo{}, handlerPaid{}, nil, nil))
	handler.RegisterRoutes(router.Group("/api/v1"), func(c *gin.Context) { c.Next() })
	for _, method := range []string{http.MethodPost, http.MethodGet, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			path := "/api/v1/api-tokens"
			if method == http.MethodDelete {
				path += "/" + uuid.NewString()
			}
			response := makeRequest(router, method, path, `{"name":"ok"}`)
			checkStatus(t, response, 401)
		})
	}
}

func checkStatus(t *testing.T, response *httptest.ResponseRecorder, want int) {
	t.Helper()
	if response.Code != want {
		t.Fatalf("status=%d want=%d", response.Code, want)
	}
}

func checkExpiry(t *testing.T, expiresAt *time.Time, days int) {
	t.Helper()
	if days == 0 {
		if expiresAt != nil {
			t.Error("unexpected expiration")
		}
		return
	}
	if expiresAt == nil {
		t.Fatal("missing expiration")
	}
	expected := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	if delta := expected.Sub(*expiresAt); delta < -time.Second || delta > time.Second {
		t.Error("incorrect expiration")
	}
}
