package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	clModel "github.com/andreypavlenko/jobber/modules/coverletters/model"
	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validAIResumeRepo() *mockAIResumeBuilderRepo {
	return &mockAIResumeBuilderRepo{GetFullResumeFunc: func(context.Context, string) (*rbModel.FullResumeDTO, error) {
		return &rbModel.FullResumeDTO{ResumeBuilderDTO: &rbModel.ResumeBuilderDTO{ID: "rb-1"}, Experiences: []*rbModel.ExperienceDTO{{Description: "Built APIs."}}}, nil
	}}
}

func TestAIHandler_GeneratePreflight(t *testing.T) {
	resumeID := "rb-1"
	for _, tc := range []struct {
		name         string
		job          string
		link         *string
		ownershipErr error
		providerErr  error
		wantStatus   int
		wantCode     string
		wantCalls    int
	}{
		{name: "empty job", wantStatus: 400, wantCode: "AI_JOB_DESCRIPTION_REQUIRED"},
		{name: "whitespace job", job: " \t\n\u00a0 ", wantStatus: 400, wantCode: "AI_JOB_DESCRIPTION_REQUIRED"},
		{name: "missing source", job: "Engineer", wantStatus: 400, wantCode: "AI_CANDIDATE_SOURCE_REQUIRED"},
		{name: "source not owned", job: "Engineer", link: &resumeID, ownershipErr: rbModel.ErrNotOwner, wantStatus: 400, wantCode: "AI_CANDIDATE_SOURCE_REQUIRED"},
		{name: "invalid AI response", job: "Engineer", link: &resumeID, providerErr: fmt.Errorf("generate: %w", ai.ErrInvalidCoverLetterResponse), wantStatus: 422, wantCode: "AI_INVALID_RESPONSE", wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checks, calls, usage := 0, 0, 0
			clRepo := &mockAICoverLetterRepo{GetByIDFunc: func(context.Context, string) (*clModel.CoverLetter, error) {
				return &clModel.CoverLetter{UserID: "user-1", ResumeBuilderID: tc.link}, nil
			}}
			rbRepo := validAIResumeRepo()
			rbRepo.VerifyOwnershipFunc = func(context.Context, string, string) error { return tc.ownershipErr }
			limiter := &mockAILimitChecker{
				CheckLimitFunc:    func(context.Context, string, string) error { checks++; return nil },
				RecordAIUsageFunc: func(context.Context, string) error { usage++; return nil },
			}
			client := &mockCoverLetterAIClient{GenerateCoverLetterFunc: func(context.Context, string, string, string, string, string) (*ai.CoverLetterContent, error) {
				calls++
				return nil, tc.providerErr
			}}
			handler := newTestCoverLetterAIHandler(clRepo, rbRepo, client, limiter)
			router := setupTestRouter()
			router.POST("/cover-letters/ai/generate", mockAuthMiddleware("user-1"), handler.Generate)
			body, err := json.Marshal(map[string]string{"cover_letter_id": "00000000-0000-0000-0000-000000000001", "job_description": tc.job})
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodPost, "/cover-letters/ai/generate", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			assert.Equal(t, tc.wantStatus, response.Code)
			var result aiErrorResponse
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
			assert.Equal(t, tc.wantCode, result.ErrorCode)
			assert.Equal(t, tc.wantCalls, calls)
			assert.Equal(t, tc.wantCalls, checks)
			assert.Zero(t, usage)
		})
	}
}
