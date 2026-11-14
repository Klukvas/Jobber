package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// Every spelling uuid.Parse accepts must name the same credit operation, so a
// retry that reformats its Idempotency-Key cannot open a second charge.
func TestAIRequestScopeCanonicalisesIdempotencyKey(t *testing.T) {
	const key = "7e57c0de-0000-4000-8000-00000000c0de"
	forms := idstest.NonCanonicalForms(key)
	forms["canonical"] = key

	for name, form := range forms {
		t.Run(name, func(t *testing.T) {
			var seen string
			r := gin.New()
			r.Use(func(c *gin.Context) { c.Set("user_id", "user-123") })
			r.Use(AIRequestScope(nil))
			r.GET("/api/v1/match-score", func(c *gin.Context) {
				op, _, found := ai.CurrentCreditOperation(c.Request.Context())
				require.True(t, found)
				seen = op.Key
				c.Status(204)
			})
			request := httptest.NewRequest("GET", "/api/v1/match-score", nil)
			request.Header.Set("Idempotency-Key", form)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, request)

			require.Equal(t, 204, w.Code)
			require.Equal(t, key, seen)
		})
	}
}

func TestAIRequestScopeRejectsMalformedIdempotencyKey(t *testing.T) {
	r := gin.New()
	r.Use(AIRequestScope(nil))
	r.GET("/api/v1/match-score", func(c *gin.Context) { c.Status(204) })
	request := httptest.NewRequest("GET", "/api/v1/match-score", nil)
	request.Header.Set("Idempotency-Key", "not-a-uuid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)
	require.Equal(t, 400, w.Code)
}
