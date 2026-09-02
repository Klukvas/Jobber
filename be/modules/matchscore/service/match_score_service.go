package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/netsafe"
	"github.com/andreypavlenko/jobber/internal/platform/storage"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	jobPorts "github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/andreypavlenko/jobber/modules/matchscore/model"
	matchPorts "github.com/andreypavlenko/jobber/modules/matchscore/ports"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
)

// isPrivateIP and validateExternalURL delegate to the shared netsafe package,
// kept as thin wrappers so existing call sites and tests stay unchanged.
func isPrivateIP(ip net.IP) bool { return netsafe.IsPrivateIP(ip) }

func validateExternalURL(rawURL string) error { return netsafe.ValidateExternalURL(rawURL) }

func ssrfSafeClient() *http.Client { return netsafe.SafeClient() }

// LimitChecker checks subscription limits before resource creation.
type LimitChecker interface {
	CheckLimit(ctx context.Context, userID, resource string) error
	RecordAIUsage(ctx context.Context, userID string) error
}

// MatchScoreService handles resume-job match scoring.
type MatchScoreService struct {
	aiClient     *ai.AnthropicClient
	s3Client     *storage.S3Client
	jobRepo      jobPorts.JobRepository
	resumeRepo   resumePorts.ResumeRepository
	limitChecker LimitChecker
	cacheRepo    matchPorts.MatchScoreCacheRepository
}

// NewMatchScoreService creates a new match score service.
func NewMatchScoreService(
	aiClient *ai.AnthropicClient,
	s3Client *storage.S3Client,
	jobRepo jobPorts.JobRepository,
	resumeRepo resumePorts.ResumeRepository,
	limitChecker LimitChecker,
	cacheRepo matchPorts.MatchScoreCacheRepository,
) *MatchScoreService {
	return &MatchScoreService{
		aiClient:     aiClient,
		s3Client:     s3Client,
		jobRepo:      jobRepo,
		resumeRepo:   resumeRepo,
		limitChecker: limitChecker,
		cacheRepo:    cacheRepo,
	}
}

// CheckMatch analyzes how well a resume matches a job posting.
func (s *MatchScoreService) CheckMatch(ctx context.Context, userID string, req *model.MatchScoreRequest) (*model.MatchScoreResponse, error) {
	// Check cache first — a hit skips AI call and quota entirely
	if s.cacheRepo != nil {
		cached, err := s.cacheRepo.Get(ctx, userID, req.JobID, req.ResumeID)
		if err != nil {
			log.Printf("[WARN] match score cache read failed for job=%s resume=%s: %v", req.JobID, req.ResumeID, err)
		} else if cached != nil {
			cached.FromCache = true
			return cached, nil
		}
	}

	// Check subscription limit for AI usage
	if s.limitChecker != nil {
		if err := s.limitChecker.CheckLimit(ctx, userID, "ai"); err != nil {
			return nil, err
		}
	}

	// Get job and validate description
	job, err := s.jobRepo.GetByID(ctx, userID, req.JobID)
	if err != nil {
		if errors.Is(err, jobModel.ErrJobNotFound) {
			return nil, jobModel.ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if job.Description == nil || strings.TrimSpace(*job.Description) == "" {
		return nil, model.ErrJobDescriptionEmpty
	}

	// Get resume
	resume, err := s.resumeRepo.GetByID(ctx, userID, req.ResumeID)
	if err != nil {
		if errors.Is(err, resumeModel.ErrResumeNotFound) {
			return nil, resumeModel.ErrResumeNotFound
		}
		return nil, fmt.Errorf("failed to get resume: %w", err)
	}

	// Download resume PDF
	pdfBytes, err := s.downloadResumePDF(ctx, resume)
	if err != nil {
		return nil, err
	}

	// Base64-encode the PDF
	pdfBase64 := base64.StdEncoding.EncodeToString(pdfBytes)

	// Call AI to analyze match
	result, err := s.aiClient.MatchResumeToJob(ctx, job.Title, *job.Description, pdfBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", model.ErrMatchFailed, err)
	}

	// Record AI usage
	if s.limitChecker != nil {
		if err := s.limitChecker.RecordAIUsage(ctx, userID); err != nil {
			log.Printf("[ERROR] failed to record AI usage for user=%s: %v", userID, err)
		}
	}

	// Map AI result to response
	categories := make([]model.MatchScoreCategory, len(result.Categories))
	for i, cat := range result.Categories {
		categories[i] = model.MatchScoreCategory{
			Name:    cat.Name,
			Score:   cat.Score,
			Details: cat.Details,
		}
	}

	resp := &model.MatchScoreResponse{
		OverallScore:    result.OverallScore,
		Categories:      categories,
		MissingKeywords: result.MissingKeywords,
		Strengths:       result.Strengths,
		Summary:         result.Summary,
	}

	// Store in cache (best-effort, don't fail the request)
	if s.cacheRepo != nil {
		if err := s.cacheRepo.Upsert(ctx, userID, req.JobID, req.ResumeID, resp); err != nil {
			log.Printf("[WARN] match score cache write failed for job=%s resume=%s: %v", req.JobID, req.ResumeID, err)
		}
	}

	return resp, nil
}

// downloadResumePDF retrieves the resume PDF bytes via the shared platform
// helper, mapping its missing-file sentinel to this module's error.
func (s *MatchScoreService) downloadResumePDF(ctx context.Context, resume *resumeModel.Resume) ([]byte, error) {
	data, err := storage.DownloadResumeFile(ctx, s.s3Client, string(resume.StorageType), resume.StorageKey, resume.FileURL)
	if errors.Is(err, storage.ErrResumeFileMissing) {
		return nil, model.ErrResumeFileEmpty
	}
	return data, err
}

// downloadFromURL delegates to the shared platform helper; kept as a thin
// wrapper so existing call sites and tests stay unchanged.
func downloadFromURL(ctx context.Context, rawURL string) ([]byte, error) {
	return storage.DownloadFileFromURL(ctx, rawURL)
}
