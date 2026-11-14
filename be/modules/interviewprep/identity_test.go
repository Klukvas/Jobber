package interviewprep

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// uuid.Parse accepts spellings such as `urn:uuid:…` that Postgres rejects, so
// handlers must only ever see canonical path ids.
func TestIdentityCanonicalisesPathIDs(t *testing.T) {
	const (
		job     = "5b000000-0000-4000-8000-000000000001"
		session = "5b000000-0000-4000-8000-000000000002"
		answer  = "5b000000-0000-4000-8000-000000000003"
	)
	jobForms, sessionForms, answerForms := idstest.NonCanonicalForms(job), idstest.NonCanonicalForms(session), idstest.NonCanonicalForms(answer)

	for name := range jobForms {
		t.Run(name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			var seen pathIDs
			r := gin.New()
			r.POST("/jobs/:id/sessions/:sessionId/answers/:answerId/feedback", func(c *gin.Context) {
				c.Set("user_id", "owner")
				_, params, ok := identity(c)
				require.True(t, ok)
				seen = params
				c.Status(http.StatusNoContent)
			})
			path := "/jobs/" + url.PathEscape(jobForms[name]) + "/sessions/" + url.PathEscape(sessionForms[name]) +
				"/answers/" + url.PathEscape(answerForms[name]) + "/feedback"
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))

			require.Equal(t, http.StatusNoContent, w.Code)
			require.Equal(t, pathIDs{"id": job, "sessionId": session, "answerId": answer}, seen)
		})
	}
}

func TestIdentityRejectsNonUUIDPathParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/jobs/:id", func(c *gin.Context) {
		c.Set("user_id", "owner")
		if _, _, ok := identity(c); ok {
			c.Status(http.StatusNoContent)
		}
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/jobs/not-a-uuid", nil))
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// Feedback on a route that lacks :answerId must fail, never fall back to a
// differently charged summary.
func TestReviewRefusesMissingAnswerParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/jobs/:id/sessions/:sessionId/feedback", func(c *gin.Context) {
		c.Set("user_id", "owner")
		(&Handler{}).review(c, "answerId")
	})
	path := "/jobs/5b000000-0000-4000-8000-000000000001/sessions/5b000000-0000-4000-8000-000000000002/feedback"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
	require.Equal(t, http.StatusInternalServerError, w.Code)
}
