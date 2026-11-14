package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pathJobID      = "5e000000-0000-4000-8000-000000000001"
	pathTemplateID = "5e000000-0000-4000-8000-000000000002"
)

// jobRoutes mounts the real job routes behind the production path-id
// middleware and records the ids the repositories receive.
func jobRoutes(seen *[]string) *gin.Engine {
	jobRepo := &MockJobRepository{
		DeleteFunc: func(_ context.Context, _, jobID string) error {
			*seen = append(*seen, jobID)
			return nil
		},
	}
	templates := &ConfigStageTemplateRepo{
		DeleteFunc: func(_ context.Context, _, templateID string) error {
			*seen = append(*seen, templateID)
			return nil
		},
	}
	handler := NewJobHandler(newStageJobService(jobRepo, &MockJobStageRepository{}, templates))
	router := setupTestRouter()
	router.Use(httpPlatform.CanonicalUUIDParams())
	handler.RegisterRoutes(router.Group("/api/v1"), mockAuthMiddleware("user-123"))
	return router
}

func TestJobRoutesForwardCanonicalPathIDs(t *testing.T) {
	routes := []struct {
		name, prefix, canonical string
	}{
		{"job", "/api/v1/jobs/", pathJobID},
		{"stage template", "/api/v1/stage-templates/", pathTemplateID},
	}
	for _, route := range routes {
		for form, raw := range idstest.NonCanonicalForms(route.canonical) {
			t.Run(route.name+"/"+form, func(t *testing.T) {
				var seen []string
				w := httptest.NewRecorder()
				jobRoutes(&seen).ServeHTTP(w, httptest.NewRequest(http.MethodDelete, route.prefix+url.PathEscape(raw), nil))

				require.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, []string{route.canonical}, seen)
			})
		}
	}
}

// Malformed path ids used to reach Postgres and come back as a 500.
func TestJobRoutesRejectMalformedPathIDs(t *testing.T) {
	for _, path := range []string{
		"/api/v1/jobs/not-a-uuid",
		"/api/v1/stage-templates/stage-wishlist",
		"/api/v1/jobs/" + pathJobID + "/stages/1",
	} {
		t.Run(path, func(t *testing.T) {
			var seen []string
			w := httptest.NewRecorder()
			jobRoutes(&seen).ServeHTTP(w, httptest.NewRequest(http.MethodDelete, path, nil))

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Contains(t, w.Body.String(), `"error_code":"VALIDATION_ERROR"`)
			assert.Empty(t, seen, "repository must not be reached")
		})
	}
}
