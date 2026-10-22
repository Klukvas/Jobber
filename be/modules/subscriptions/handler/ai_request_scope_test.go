package handler

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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
