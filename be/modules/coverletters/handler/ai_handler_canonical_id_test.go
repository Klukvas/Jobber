package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	clModel "github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/stretchr/testify/assert"
)

// uuid.Parse accepts spellings such as `urn:uuid:…` that Postgres rejects, so
// the id must reach the repository in canonical form or generation is a 500.
func TestAIHandler_Generate_CanonicalisesCoverLetterID(t *testing.T) {
	const userID = "user-123"
	const coverLetterID = "00000000-0000-4000-8000-0000000000c1"

	for name, form := range idstest.NonCanonicalForms(coverLetterID) {
		t.Run(name, func(t *testing.T) {
			var seen []string
			clRepo := &mockAICoverLetterRepo{
				GetByIDFunc: func(_ context.Context, id string) (*clModel.CoverLetter, error) {
					seen = append(seen, id)
					return &clModel.CoverLetter{ID: id, UserID: userID}, nil
				},
			}
			aiClient := &mockCoverLetterAIClient{
				GenerateCoverLetterFunc: func(_ context.Context, _, _, _, _, _ string) (*ai.CoverLetterContent, error) {
					return &ai.CoverLetterContent{Greeting: "Hi", Paragraphs: []string{"Body"}, Closing: "Bye"}, nil
				},
			}
			handler := newTestCoverLetterAIHandler(clRepo, &mockAIResumeBuilderRepo{}, aiClient, &mockAILimitChecker{})
			router := setupTestRouter()
			router.POST("/cover-letters/ai/generate", mockAuthMiddleware(userID), handler.Generate)

			body := `{"cover_letter_id":` + strconv.Quote(form) + `,"job_description":"Engineer"}`
			req := httptest.NewRequest(http.MethodPost, "/cover-letters/ai/generate", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.NotEmpty(t, seen)
			for _, id := range seen {
				assert.Equal(t, coverLetterID, id)
			}
		})
	}
}
