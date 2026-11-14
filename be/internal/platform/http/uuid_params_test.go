package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	paramJobID   = "5c000000-0000-4000-8000-000000000001"
	paramStageID = "5c000000-0000-4000-8000-000000000002"
)

// uuidParamsRouter records the params the handler sees after the middleware.
func uuidParamsRouter(seen *gin.Params) *gin.Engine {
	router := gin.New()
	router.Use(CanonicalUUIDParams())
	record := func(c *gin.Context) {
		*seen = c.Params
		c.Status(http.StatusNoContent)
	}
	router.PATCH("/jobs/:id/stages/:stageId", record)
	router.GET("/share/:token", record)
	router.POST("/users/:id/manual-access/:grant/revoke", record)
	return router
}

func serve(router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, nil))
	return w
}

func TestCanonicalUUIDParamsRewritesEverySpelling(t *testing.T) {
	jobForms, stageForms := idstest.NonCanonicalForms(paramJobID), idstest.NonCanonicalForms(paramStageID)
	jobForms["canonical"], stageForms["canonical"] = paramJobID, paramStageID

	for name := range jobForms {
		t.Run(name, func(t *testing.T) {
			var seen gin.Params
			path := "/jobs/" + url.PathEscape(jobForms[name]) + "/stages/" + url.PathEscape(stageForms[name])
			w := serve(uuidParamsRouter(&seen), http.MethodPatch, path)

			require.Equal(t, http.StatusNoContent, w.Code)
			assert.Equal(t, paramJobID, seen.ByName("id"))
			assert.Equal(t, paramStageID, seen.ByName("stageId"))
		})
	}
}

func TestCanonicalUUIDParamsRejectsMalformedIDs(t *testing.T) {
	tests := []struct {
		name, path, param string
	}{
		{"not a uuid", "/jobs/not-a-uuid/stages/" + paramStageID, "id"},
		{"second param", "/jobs/" + paramJobID + "/stages/123", "stageId"},
		{"admin grant", "/users/" + paramJobID + "/manual-access/x/revoke", "grant"},
		{"traversal", "/jobs/" + url.PathEscape("../etc") + "/stages/" + paramStageID, "id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var seen gin.Params
			method := http.MethodPatch
			if tt.param == "grant" {
				method = http.MethodPost
			}
			w := serve(uuidParamsRouter(&seen), method, tt.path)

			require.Equal(t, http.StatusBadRequest, w.Code)
			assert.Nil(t, seen, "handler must not run")
			var body ErrorResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, "VALIDATION_ERROR", body.ErrorCode)
			assert.Equal(t, "Invalid "+tt.param, body.ErrorMessage)
		})
	}
}

// Public share tokens are not UUIDs and must reach their handler untouched.
func TestCanonicalUUIDParamsLeavesOtherParamsAlone(t *testing.T) {
	var seen gin.Params
	w := serve(uuidParamsRouter(&seen), http.MethodGet, "/share/Abc_123-token")

	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "Abc_123-token", seen.ByName("token"))
}

func TestIsUUIDParam(t *testing.T) {
	for name, want := range map[string]bool{
		"id": true, "stageId": true, "templateId": true, "grant": true,
		"token": false, "any": false, "identity": false,
	} {
		assert.Equal(t, want, isUUIDParam(name), name)
	}
}

func TestUnclassifiedPathParams(t *testing.T) {
	router := gin.New()
	noop := func(*gin.Context) {}
	router.GET("/jobs/:id/stages/:stageId", noop)
	router.GET("/share/:token", noop)
	router.GET("/swagger/*any", noop)
	router.GET("/users/:id/manual-access/:grant", noop)
	assert.Empty(t, UnclassifiedPathParams(router.Routes()))

	router.DELETE("/jobs/:jobID", noop)
	router.GET("/files/:uuid/raw", noop)
	assert.ElementsMatch(t, []string{"DELETE /jobs/:jobID", "GET /files/:uuid/raw"}, UnclassifiedPathParams(router.Routes()))
}
