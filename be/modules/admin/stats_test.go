package admin

import (
	"testing"

	userModel "github.com/andreypavlenko/jobber/modules/users/model"
	"github.com/stretchr/testify/require"
)

func TestAdminStatsAndTestAccountValidation(t *testing.T) {
	for _, tc := range []struct {
		method, path, body, content, origin string
		status                              int
	}{
		{"GET", "/stats?days=8", "", "", "", 400},
		{"GET", "/stats?days=07", "", "", "", 400},
		{"GET", "/stats?include_test=1", "", "", "", 400},
		{"PATCH", "/users/" + testUser + "/test-account", `{}`, "application/json", "http://localhost:3000", 400},
		{"PATCH", "/users/" + testUser + "/test-account", `null`, "application/json", "http://localhost:3000", 400},
		{"PATCH", "/users/" + testUser + "/test-account", `{"is_test":null}`, "application/json", "http://localhost:3000", 400},
		{"PATCH", "/users/" + testUser + "/test-account", `{"is_test":"false"}`, "application/json", "http://localhost:3000", 400},
		{"PATCH", "/users/" + testUser + "/test-account", `{"is_test":true,"email_verified":true}`, "application/json", "http://localhost:3000", 400},
		{"PATCH", "/users/" + testUser + "/test-account", `{"is_test":true}`, "application/json", "https://evil.example", 403},
		{"PATCH", "/users/" + testUser + "/test-account", `{"is_test":true}`, "application/json", "", 403},
		{"PATCH", "/users/" + testUser + "/test-account", `{"is_test":true}`, "text/plain", "http://localhost:3000", 415},
		{"PATCH", "/users/not-uuid/test-account", `{"is_test":true}`, "application/json", "http://localhost:3000", 400},
		{"PATCH", "/users/" + testUser, `{"name":"Safe","locale":"en","is_test":true}`, "application/json", "http://localhost:3000", 400},
	} {
		t.Run(tc.path+tc.body+tc.origin+tc.content, func(t *testing.T) {
			store := &storeStub{}
			router, token := testRouter(identityStub{user: &userModel.User{ID: testUser, Email: "admin@example.com", EmailVerified: true}}, "admin@example.com", store)
			require.Equal(t, tc.status, request(router, token, tc.method, tc.path, tc.body, tc.content, tc.origin).Code)
			require.Zero(t, store.calls)
		})
	}
}

func TestAdminStatsDefaultsAndExplicitFalse(t *testing.T) {
	store := &storeStub{}
	router, token := testRouter(identityStub{user: &userModel.User{ID: testUser, Email: "admin@example.com", EmailVerified: true}}, "admin@example.com", store)
	require.Equal(t, 200, request(router, token, "GET", "/stats", "", "", "").Code)
	require.Equal(t, 30, store.days)
	require.False(t, store.includeTests)
	require.Equal(t, 200, request(router, token, "GET", "/stats?days=7&include_test=true", "", "", "").Code)
	require.Equal(t, 7, store.days)
	require.True(t, store.includeTests)
	for _, value := range []string{"true", "false"} {
		require.Equal(t, 200, request(router, token, "PATCH", "/users/"+testUser+"/test-account", `{"is_test":`+value+`}`, "application/json", "http://localhost:3000").Code)
		require.Equal(t, value == "true", store.isTest)
	}
}
