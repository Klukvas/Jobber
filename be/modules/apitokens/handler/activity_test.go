package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/apitokens/model"
	"github.com/andreypavlenko/jobber/modules/apitokens/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type handlerActivityRepo struct {
	userID string
	limit  int
	calls  int
	items  map[string][]*model.ActivityDTO
	err    error
}

func (r *handlerActivityRepo) ListActivity(_ context.Context, userID string, limit int) ([]*model.ActivityDTO, error) {
	r.userID, r.limit = userID, limit
	r.calls++
	items := append([]*model.ActivityDTO{}, r.items[userID]...)
	if len(items) > limit {
		items = items[:limit]
	}
	return items, r.err
}

func activityRouter(repo *handlerActivityRepo, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc := service.NewAPITokenService(&handlerRepo{}, handlerPaid{err: subModel.ErrPaidFeature}, nil, nil, service.WithActivityRepository(repo))
	NewAPITokenHandler(svc).RegisterRoutes(router.Group("/api/v1"), func(c *gin.Context) {
		if userID != "" {
			c.Set("user_id", userID)
		}
		c.Next()
	})
	return router
}

func TestActivityLimit(t *testing.T) {
	tests := []struct {
		name, query string
		limit       int
	}{
		{"default", "", 20}, {"minimum", "?limit=1", 1}, {"maximum", "?limit=100", 100},
		{"empty", "?limit=", 0}, {"zero", "?limit=0", 0}, {"negative", "?limit=-1", 0},
		{"above maximum", "?limit=101", 0}, {"fraction", "?limit=1.5", 0}, {"text", "?limit=no", 0},
		{"overflow", "?limit=99999999999999999999999999", 0}, {"duplicate", "?limit=1&limit=2", 0},
		{"bad escape", "?limit=%zz", 0}, {"semicolon", "?limit=1;limit=2", 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &handlerActivityRepo{}
			response := makeRequest(activityRouter(repo, "owner"), http.MethodGet, "/api/v1/api-tokens/activity"+test.query, "")
			if test.limit == 0 {
				require.Equal(t, 400, response.Code)
				require.Contains(t, response.Body.String(), `"error_code":"VALIDATION_ERROR"`)
				require.Zero(t, repo.calls)
			} else {
				require.Equal(t, 200, response.Code, "downgraded user can still read activity")
				require.Equal(t, test.limit, repo.limit)
				require.Equal(t, "owner", repo.userID)
				require.JSONEq(t, `[]`, response.Body.String())
			}
		})
	}
}

func TestActivityContractAndUserIsolation(t *testing.T) {
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC)
	code, tokenID, tokenName := "not_found", "token", "Laptop"
	repo := &handlerActivityRepo{items: map[string][]*model.ActivityDTO{
		"owner": {
			{ID: "new", Tool: "get_job", OK: false, ErrorCode: &code, DurationMS: 17, CreatedAt: now, APITokenID: &tokenID, TokenName: &tokenName},
			{ID: "old", Tool: "list_jobs", OK: true, DurationMS: 3, CreatedAt: now.Add(-time.Hour)},
		},
		"other": {{ID: "other-user-secret", Tool: "list_jobs", CreatedAt: now}},
	}}
	router := activityRouter(repo, "owner")
	response := makeRequest(router, http.MethodGet, "/api/v1/api-tokens/activity?user_id=other", "")
	require.Equal(t, 200, response.Code)
	require.JSONEq(t, `[
 {"id":"new","tool":"get_job","ok":false,"error_code":"not_found","duration_ms":17,"created_at":"2026-11-13T10:00:00Z","api_token_id":"token","token_name":"Laptop"},
 {"id":"old","tool":"list_jobs","ok":true,"error_code":null,"duration_ms":3,"created_at":"2026-11-13T09:00:00Z","api_token_id":null,"token_name":null}
]`, response.Body.String())
	var items []model.ActivityDTO
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &items))
	require.Len(t, items, 2)
	require.True(t, items[0].CreatedAt.After(items[1].CreatedAt))
	response = makeRequest(router, http.MethodGet, "/api/v1/api-tokens/activity?limit=1", "")
	require.Equal(t, 200, response.Code)
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &items))
	require.Len(t, items, 1)
	require.Equal(t, "new", items[0].ID)
	// The fixed GET route coexists with the existing DELETE wildcard.
	require.Equal(t, 204, makeRequest(router, http.MethodDelete, "/api/v1/api-tokens/54a9429e-8766-44fc-861a-70bc173993db", "").Code)
}

func TestActivityAuthenticationAndFailure(t *testing.T) {
	repo := &handlerActivityRepo{err: errors.New("private database details")}
	response := makeRequest(activityRouter(repo, ""), http.MethodGet, "/api/v1/api-tokens/activity", "")
	require.Equal(t, 401, response.Code)
	require.Zero(t, repo.calls)
	response = makeRequest(activityRouter(repo, "owner"), http.MethodGet, "/api/v1/api-tokens/activity", "")
	require.Equal(t, 500, response.Code)
	require.Contains(t, response.Body.String(), "INTERNAL_ERROR")
	require.NotContains(t, response.Body.String(), "private")
}
