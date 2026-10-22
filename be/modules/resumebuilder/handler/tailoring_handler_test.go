package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tailoringHandlerTestService struct {
	err      error
	replayed bool
	calls    int
}

func (s *tailoringHandlerTestService) Generate(context.Context, string, string, string, string) (*model.TailoringProposalDTO, error) {
	s.calls++
	return &model.TailoringProposalDTO{ProposalID: uuid.NewString(), Changes: []model.TailoringChange{}}, s.err
}
func (s *tailoringHandlerTestService) Apply(context.Context, string, string, string, *model.ApplyTailoringRequest) (*model.ApplyTailoringResult, bool, error) {
	s.calls++
	return &model.ApplyTailoringResult{ResumeBuilder: &model.ResumeBuilderDTO{ID: "copy"}, JobID: "job"}, s.replayed, s.err
}
func tailoringTestRouter(s *tailoringHandlerTestService, authenticated bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewTailoringHandler(s)
	handler.RegisterRoutes(router.Group("/api/v1"), func(c *gin.Context) {
		if authenticated {
			c.Set("user_id", uuid.NewString())
		}
	}, func(c *gin.Context) { c.Next() })
	return router
}
func TestTailoringHandlerGenerate(t *testing.T) {
	job, source := uuid.NewString(), uuid.NewString()
	for _, test := range []struct {
		name, path, body string
		auth             bool
		status           int
		calls            int
	}{
		{name: "direct proposal JSON", path: job, body: `{"resume_builder_id":"` + source + `","locale":"en"}`, auth: true, status: 200, calls: 1},
		{name: "authentication", path: job, body: `{}`, status: 401},
		{name: "invalid job UUID", path: "invalid", body: `{}`, auth: true, status: 400},
		{name: "invalid source UUID", path: job, body: `{"resume_builder_id":"invalid"}`, auth: true, status: 400},
		{name: "invalid locale", path: job, body: `{"resume_builder_id":"` + source + `","locale":"xx"}`, auth: true, status: 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := &tailoringHandlerTestService{}
			router := tailoringTestRouter(s, test.auth)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/"+test.path+"/resume-tailoring", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(response, request)
			assert.Equal(t, test.status, response.Code)
			assert.Equal(t, test.calls, s.calls)
			if test.status == 200 {
				var body map[string]any
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
				assert.NotEmpty(t, body["proposal_id"])
				assert.NotContains(t, body, "data")
				assert.Equal(t, []any{}, body["changes"])
			}
		})
	}
}
func TestTailoringHandlerApplyAndErrors(t *testing.T) {
	for _, test := range []struct {
		name     string
		err      error
		replayed bool
		status   int
		code     string
	}{
		{name: "created", status: 201}, {name: "replay", replayed: true, status: 200},
		{name: "ownership", err: model.ErrTailoringNotFound, status: 404, code: "NOT_FOUND"},
		{name: "quota", err: subModel.ErrLimitReached, status: 403, code: "PLAN_LIMIT_REACHED"},
		{name: "source stale", err: model.ErrSourceChanged, status: 409, code: "SOURCE_CHANGED"},
		{name: "job stale", err: model.ErrJobChanged, status: 409, code: "JOB_CHANGED"},
		{name: "different replay", err: model.ErrProposalAlreadyApplied, status: 409, code: "PROPOSAL_ALREADY_APPLIED"},
		{name: "expired", err: model.ErrProposalExpired, status: 410, code: "PROPOSAL_EXPIRED"},
		{name: "no AI key", err: model.ErrAIUnavailable, status: 503, code: "AI_UNAVAILABLE"},
		{name: "store down", err: model.ErrProposalStoreUnavailable, status: 503, code: "PROPOSAL_STORE_UNAVAILABLE"},
		{name: "invalid model", err: model.ErrAIInvalidResponse, status: 502, code: "AI_INVALID_RESPONSE"},
		{name: "provider failed", err: model.ErrAIRequestFailed, status: 502, code: "AI_REQUEST_FAILED"},
		{name: "timeout", err: model.ErrAITimeout, status: 504, code: "AI_TIMEOUT"},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := &tailoringHandlerTestService{err: test.err, replayed: test.replayed}
			router := tailoringTestRouter(s, true)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/jobs/"+uuid.NewString()+"/resume-tailoring/"+uuid.NewString()+"/apply", strings.NewReader(`{"title":"Job copy","accepted_change_ids":["change"]}`))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(response, request)
			assert.Equal(t, test.status, response.Code)
			if test.code != "" {
				assert.Contains(t, response.Body.String(), `"error_code":"`+test.code+`"`)
			}
		})
	}
}
