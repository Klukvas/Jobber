package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/stretchr/testify/assert"
)

// uuid.Parse accepts spellings such as `urn:uuid:…` that Postgres rejects, so
// the filter must reach the repository in canonical form or the list is a 500.
func TestJobHandler_List_CanonicalisesCompanyFilter(t *testing.T) {
	const companyID = "0149b69a-0000-4000-8000-000000000001"

	for name, form := range idstest.NonCanonicalForms(companyID) {
		t.Run(name, func(t *testing.T) {
			var seen string
			mockRepo := &MockJobRepository{
				ListFunc: func(_ context.Context, _ string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
					seen = opts.CompanyID
					return []*model.JobDTO{}, 0, nil
				},
			}
			router := setupTestRouter()
			router.GET("/jobs", mockAuthMiddleware("user-123"), NewJobHandler(newTestJobService(mockRepo)).List)

			req := httptest.NewRequest(http.MethodGet, "/jobs?company_id="+url.QueryEscape(form), nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, companyID, seen)
		})
	}
}
