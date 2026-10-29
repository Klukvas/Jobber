package admin

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
	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

const testUser = "00000000-0000-4000-8000-000000000001"
const testGrant = "00000000-0000-4000-8000-000000000002"

type identityStub struct {
	user *userModel.User
	err  error
}

func (s identityStub) GetByID(context.Context, string) (*userModel.User, error) { return s.user, s.err }

type storeStub struct {
	calls        int
	lastUser     string
	profile      ProfileUpdate
	grant        GrantRequest
	isTest       bool
	days         int
	includeTests bool
}

func (s *storeStub) List(_ context.Context, q string, page, limit int) (UserList, error) {
	s.calls++
	return UserList{Users: []User{}, Page: page, Limit: limit}, nil
}
func (s *storeStub) Detail(_ context.Context, id string) (Detail, error) {
	s.lastUser = id
	s.calls++
	return Detail{User: User{ID: testUser, Email: "test@example.com"}}, nil
}
func (s *storeStub) Update(_ context.Context, _ string, p ProfileUpdate) error {
	s.calls++
	s.profile = p
	return nil
}
func (s *storeStub) Grant(_ context.Context, _, _ string, p GrantRequest) error {
	s.calls++
	s.grant = p
	return nil
}
func (s *storeStub) Revoke(context.Context, string, string, string, string) error {
	s.calls++
	return nil
}
func (s *storeStub) Stats(_ context.Context, days int, includeTests bool) (Stats, error) {
	s.calls++
	s.days = days
	s.includeTests = includeTests
	return Stats{Days: days, IncludeTests: includeTests, Daily: []DailyRegistrations{}}, nil
}
func (s *storeStub) SetTest(_ context.Context, id string, isTest bool) error {
	s.calls++
	s.lastUser = id
	s.isTest = isTest
	return nil
}
func testRouter(identity identityStub, emails string, store *storeStub) (*gin.Engine, string) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	jwt := auth.NewJWTManager("fixture-access", "fixture-refresh", time.Hour, time.Hour)
	token, _ := jwt.GenerateAccessToken(testUser)
	NewHandler(store, identity, emails, "http://localhost:3000").RegisterRoutes(r.Group("/api/v1"), auth.AuthMiddleware(jwt))
	return r, token
}
func request(r http.Handler, token, method, path, body, contentType, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/admin"+path, strings.NewReader(body))
	if token != "" {
		req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookie, Value: token})
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
func TestAdminAuthorizationEveryEndpoint(t *testing.T) {
	routes := []struct{ method, path, body string }{{"GET", "/stats", ""}, {"PATCH", "/users/" + testUser + "/test-account", `{"is_test":true}`}, {"GET", "/access", ""}, {"GET", "/users", ""}, {"GET", "/users/" + testUser, ""}, {"PATCH", "/users/" + testUser, `{"name":"Safe","locale":"en"}`}, {"POST", "/users/" + testUser + "/manual-access", `{"id":"` + testGrant + `","days":30,"reason":"support"}`}, {"POST", "/users/" + testUser + "/manual-access/" + testGrant + "/revoke", `{"reason":"support"}`}}
	for _, test := range []struct {
		name, email, config string
		verified            bool
		err                 error
		id                  string
		want                int
	}{{"allowed normalized", " ADMIN@Example.COM ", " , admin@example.com, ", true, nil, testUser, 200}, {"empty config", "admin@example.com", "", true, nil, testUser, 403}, {"not allowlisted", "other@example.com", "admin@example.com", true, nil, testUser, 403}, {"unverified", "admin@example.com", "admin@example.com", false, nil, testUser, 403}, {"deleted", "admin@example.com", "admin@example.com", true, userModel.ErrUserNotFound, testUser, 403}, {"different database identity", "admin@example.com", "admin@example.com", true, nil, testGrant, 403}, {"database failure", "admin@example.com", "admin@example.com", true, errors.New("password=must-not-leak"), testUser, 503}} {
		t.Run(test.name, func(t *testing.T) {
			store := &storeStub{}
			r, token := testRouter(identityStub{&userModel.User{ID: test.id, Email: test.email, EmailVerified: test.verified}, test.err}, test.config, store)
			for _, route := range routes {
				w := request(r, token, route.method, route.path, route.body, "application/json", "http://localhost:3000")
				require.Equal(t, test.want, w.Code, "%s %s: %s", route.method, route.path, w.Body.String())
				require.NotContains(t, w.Body.String(), "must-not-leak")
				require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			}
			if test.want != 200 {
				require.Zero(t, store.calls)
			}
		})
	}
	r, _ := testRouter(identityStub{}, "admin@example.com", &storeStub{})
	for _, route := range routes {
		require.Equal(t, 401, request(r, "", route.method, route.path, route.body, "application/json", "http://localhost:3000").Code)
	}
}
func TestAdminValidationAndCSRF(t *testing.T) {
	for _, test := range []struct {
		method, path, body, content, origin string
		want                                int
	}{
		{"GET", "/users?page=0", "", "", "", 400}, {"GET", "/users?limit=101", "", "", "", 400}, {"GET", "/users?page=100001", "", "", "", 400}, {"GET", "/users?search=" + strings.Repeat("a", 201), "", "", "", 400}, {"GET", "/users/not-uuid", "", "", "", 400},
		{"PATCH", "/users/" + testUser, `{"name":"A","locale":"en","email":"hacker@example.com"}`, "application/json", "http://localhost:3000", 400},
		{"PATCH", "/users/" + testUser, `{"name":" ","locale":"en"}`, "application/json", "http://localhost:3000", 400},
		{"PATCH", "/users/" + testUser, `{"name":"A","locale":"xx"}`, "application/json", "http://localhost:3000", 400},
		{"PATCH", "/users/" + testUser, `{"name":"A","locale":"en"}`, "application/json", "https://evil.example", 403},
		{"PATCH", "/users/" + testUser, `{"name":"A","locale":"en"}`, "application/json", "", 403},
		{"PATCH", "/users/" + testUser, `name=A&locale=en`, "application/x-www-form-urlencoded", "http://localhost:3000", 415},
		{"POST", "/users/" + testUser + "/manual-access", `{"id":"` + testGrant + `","days":32,"reason":"support"}`, "application/json", "http://localhost:3000", 400},
		{"POST", "/users/" + testUser + "/manual-access", `{"id":"` + testGrant + `","days":0,"reason":"support"}`, "application/json", "http://localhost:3000", 400},
		{"POST", "/users/" + testUser + "/manual-access", `{"id":"` + testGrant + `","days":1,"reason":"x"}`, "application/json", "http://localhost:3000", 400},
		{"POST", "/users/" + testUser + "/manual-access/" + testGrant + "/revoke", `{"reason":"x"}`, "application/json", "http://localhost:3000", 400},
		{"PATCH", "/users/" + testUser, `{"name":"A","locale":"en"} {}`, "application/json", "http://localhost:3000", 400},
	} {
		t.Run(test.path+test.origin+test.body, func(t *testing.T) {
			s := &storeStub{}
			r, token := testRouter(identityStub{user: &userModel.User{ID: testUser, Email: "admin@example.com", EmailVerified: true}}, "admin@example.com", s)
			w := request(r, token, test.method, test.path, test.body, test.content, test.origin)
			require.Equal(t, test.want, w.Code, w.Body.String())
			require.Zero(t, s.calls)
		})
	}
}
func TestAdminBlankBearerCannotBypassCookieOrigin(t *testing.T) {
	s := &storeStub{}
	r, token := testRouter(identityStub{user: &userModel.User{ID: testUser, Email: "admin@example.com", EmailVerified: true}}, "admin@example.com", s)
	req := httptest.NewRequest("PATCH", "/api/v1/admin/users/"+testUser, strings.NewReader(`{"name":"A","locale":"en"}`))
	req.AddCookie(&http.Cookie{Name: auth.AccessTokenCookie, Value: token})
	req.Header.Set("Authorization", "Bearer ")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
	require.Zero(t, s.calls)
}
func TestAdminProfileWritesSafeColumns(t *testing.T) {
	s := &storeStub{}
	r, token := testRouter(identityStub{user: &userModel.User{ID: testUser, Email: "admin@example.com", EmailVerified: true}}, "admin@example.com", s)
	w := request(r, token, "PATCH", "/users/"+testUser, `{"name":"  Safe Name  ","locale":"uk"}`, "application/json", "http://localhost:3000")
	require.Equal(t, 200, w.Code)
	require.Equal(t, ProfileUpdate{Name: "Safe Name", Locale: "uk"}, s.profile)
	var data map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &data))
	require.Equal(t, map[string]any{"saved": true}, data)
}

func TestAdminCanonicalizesEquivalentUUIDRoutesAndGrantReceipts(t *testing.T) {
	for _, route := range []string{"aaaaaaaaaaaa4aaa8aaaaaaaaaaaaaaa", "AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA", "urn:uuid:aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"} {
		s := &storeStub{}
		r, token := testRouter(identityStub{user: &userModel.User{ID: testUser, Email: "admin@example.com", EmailVerified: true}}, "admin@example.com", s)
		w := request(r, token, "GET", "/users/"+route, "", "", "")
		require.Equal(t, 200, w.Code)
		require.Equal(t, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", s.lastUser)
		w = request(r, token, "POST", "/users/"+testUser+"/manual-access", `{"id":"AAAAAAAA-AAAA-4AAA-8AAA-AAAAAAAAAAAA","days":7,"reason":"Support trial"}`, "application/json", "http://localhost:3000")
		require.Equal(t, 200, w.Code)
		require.Equal(t, "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", s.grant.ID)
	}
}
