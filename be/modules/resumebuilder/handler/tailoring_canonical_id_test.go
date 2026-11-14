package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ids/idstest"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

const (
	canonicalJobID      = "4a1b2c3d-0000-4000-8000-0000000000a1"
	canonicalProposalID = "4a1b2c3d-0000-4000-8000-0000000000b2"
	canonicalSourceID   = "4a1b2c3d-0000-4000-8000-0000000000c3"
	canonicalBuilderID  = "4a1b2c3d-0000-4000-8000-0000000000d4"
)

// recordingTailoringService remembers the ids its methods received.
type recordingTailoringService struct{ jobID, proposalID string }

func (s *recordingTailoringService) Generate(_ context.Context, _, jobID, _, _ string) (*model.TailoringProposalDTO, error) {
	s.jobID = jobID
	return &model.TailoringProposalDTO{ProposalID: canonicalProposalID, Changes: []model.TailoringChange{}}, nil
}
func (s *recordingTailoringService) Apply(_ context.Context, _, jobID, proposalID string, _ *model.ApplyTailoringRequest) (*model.ApplyTailoringResult, bool, error) {
	s.jobID, s.proposalID = jobID, proposalID
	return &model.ApplyTailoringResult{ResumeBuilder: &model.ResumeBuilderDTO{ID: "copy"}, JobID: jobID}, false, nil
}

type recordingPDFTailoringService struct{ jobID, sourceID string }

func (s *recordingPDFTailoringService) Import(_ context.Context, _, jobID string, _ *model.PDFImportRequest) (*model.PDFImportResult, error) {
	s.jobID = jobID
	return &model.PDFImportResult{}, nil
}
func (s *recordingPDFTailoringService) ConfirmRecognition(_ context.Context, _, jobID, _, _ string) error {
	s.jobID = jobID
	return nil
}
func (s *recordingPDFTailoringService) SourceReview(_ context.Context, _, jobID, sourceID string) (string, bool, error) {
	s.jobID, s.sourceID = jobID, sourceID
	return "", false, nil
}

func canonicalIDTestRouter(register func(*gin.RouterGroup, gin.HandlerFunc, gin.HandlerFunc)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	register(router.Group("/api/v1"), func(c *gin.Context) { c.Set("user_id", "user-123") }, func(c *gin.Context) { c.Next() })
	return router
}

func serveJSON(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

// uuid.Parse accepts spellings such as `urn:uuid:…` that Postgres rejects, so
// path ids must reach the service in canonical form or the request is a 500.
func TestTailoringHandlersForwardCanonicalPathIDs(t *testing.T) {
	jobForms := idstest.NonCanonicalForms(canonicalJobID)
	proposalForms := idstest.NonCanonicalForms(canonicalProposalID)
	sourceForms := idstest.NonCanonicalForms(canonicalSourceID)

	for name := range jobForms {
		job := url.PathEscape(jobForms[name])
		t.Run(name+"/generate", func(t *testing.T) {
			s := &recordingTailoringService{}
			router := canonicalIDTestRouter(NewTailoringHandler(s).RegisterRoutes)
			response := serveJSON(router, http.MethodPost, "/api/v1/jobs/"+job+"/resume-tailoring",
				`{"resume_builder_id":"`+canonicalBuilderID+`","locale":"en"}`)
			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, canonicalJobID, s.jobID)
		})
		t.Run(name+"/apply", func(t *testing.T) {
			s := &recordingTailoringService{}
			router := canonicalIDTestRouter(NewTailoringHandler(s).RegisterRoutes)
			response := serveJSON(router, http.MethodPost,
				"/api/v1/jobs/"+job+"/resume-tailoring/"+url.PathEscape(proposalForms[name])+"/apply",
				`{"title":"Job copy","accepted_change_ids":["change"]}`)
			assert.Equal(t, http.StatusCreated, response.Code)
			assert.Equal(t, canonicalJobID, s.jobID)
			assert.Equal(t, canonicalProposalID, s.proposalID)
		})
		t.Run(name+"/source review", func(t *testing.T) {
			s := &recordingPDFTailoringService{}
			router := canonicalIDTestRouter(NewPDFTailoringHandler(s).RegisterRoutes)
			response := serveJSON(router, http.MethodGet,
				"/api/v1/jobs/"+job+"/resume-tailoring/source/"+url.PathEscape(sourceForms[name]), "")
			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, canonicalJobID, s.jobID)
			assert.Equal(t, canonicalSourceID, s.sourceID)
		})
		t.Run(name+"/confirm recognition", func(t *testing.T) {
			s := &recordingPDFTailoringService{}
			router := canonicalIDTestRouter(NewPDFTailoringHandler(s).RegisterRoutes)
			response := serveJSON(router, http.MethodPost, "/api/v1/jobs/"+job+"/resume-tailoring/recognition",
				`{"import_id":"`+canonicalProposalID+`","resume_builder_id":"`+canonicalBuilderID+`"}`)
			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, canonicalJobID, s.jobID)
		})
	}
}
