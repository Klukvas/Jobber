package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/assert"
)

// idRecorder collects every resume id the repository is asked about.
type idRecorder struct {
	mu  sync.Mutex
	ids []string
}

func (r *idRecorder) add(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ids = append(r.ids, id)
}

func (r *idRecorder) recordingRepo() *mockResumeBuilderRepository {
	return &mockResumeBuilderRepository{
		VerifyOwnershipFunc: func(_ context.Context, _, id string) error {
			r.add(id)
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, id string) (*model.FullResumeDTO, error) {
			r.add(id)
			return &model.FullResumeDTO{
				ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: id},
				Contact:          &model.ContactDTO{FullName: "Jane"},
				Experiences:      []*model.ExperienceDTO{{Position: "Dev", Company: "Co"}},
			}, nil
		},
	}
}

// uuid.Parse accepts spellings such as `urn:uuid:…` that Postgres rejects, so
// the resume id must reach the repository in canonical form or the call is a 500.
func TestAIHandler_ForwardsCanonicalResumeID(t *testing.T) {
	const userID = "user-123"
	const resumeID = "00000000-0000-4000-8000-0000000000e5"

	for name, form := range idstest.NonCanonicalForms(resumeID) {
		t.Run(name+"/suggest summary", func(t *testing.T) {
			recorder := &idRecorder{}
			aiClient := &mockResumeAIClient{
				SuggestSummaryFunc: func(_ context.Context, _, _, _ string) (string, error) { return "Summary", nil },
			}
			handler := newTestAIHandler(recorder.recordingRepo(), aiClient, &mockLimitChecker{})
			router := setupRouter()
			router.POST("/ai/suggest-summary", authMiddleware(userID), handler.SuggestSummary)

			body := `{"resume_id":` + strconv.Quote(form) + `}`
			req := httptest.NewRequest(http.MethodPost, "/ai/suggest-summary", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.NotEmpty(t, recorder.ids)
			for _, id := range recorder.ids {
				assert.Equal(t, resumeID, id)
			}
		})
		t.Run(name+"/ats check", func(t *testing.T) {
			recorder := &idRecorder{}
			aiClient := &mockResumeAIClient{
				AnalyzeATSFunc: func(_ context.Context, _, _ string) (*ai.ATSCheckResult, error) {
					return &ai.ATSCheckResult{Score: 80}, nil
				},
			}
			handler := newTestAIHandler(recorder.recordingRepo(), aiClient, &mockLimitChecker{})
			router := setupRouter()
			router.POST("/resume-builder/:id/ats-check", authMiddleware(userID), handler.ATSCheck)

			req := httptest.NewRequest(http.MethodPost, "/resume-builder/"+url.PathEscape(form)+"/ats-check", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.NotEmpty(t, recorder.ids)
			for _, id := range recorder.ids {
				assert.Equal(t, resumeID, id)
			}
		})
	}
}
