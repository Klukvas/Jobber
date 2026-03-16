package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/storage"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	jobPorts "github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/andreypavlenko/jobber/modules/matchscore/model"
	matchPorts "github.com/andreypavlenko/jobber/modules/matchscore/ports"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
)

const maxResumeSize = 20 * 1024 * 1024 // 20 MB

// privateCIDRs contains IP ranges that must not be accessed via downloadFromURL.
var privateCIDRs []*net.IPNet

func init() {
	for _, cidr := range []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("invalid CIDR in SSRF protection list: %s: %v", cidr, err))
		}
		privateCIDRs = append(privateCIDRs, block)
	}
}

// isPrivateIP checks whether an IP falls within a private/reserved range.
func isPrivateIP(ip net.IP) bool {
	for _, cidr := range privateCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// validateExternalURL ensures the URL is safe to fetch (SSRF protection).
func validateExternalURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if parsed.Scheme != "https" {
		return fmt.Errorf("only https URLs are allowed, got %q", parsed.Scheme)
	}

	hostname := parsed.Hostname()
	if hostname == "" || strings.EqualFold(hostname, "localhost") {
		return fmt.Errorf("hostname %q is not allowed", hostname)
	}

	ips, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("DNS lookup failed for %q: %w", hostname, err)
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if isPrivateIP(ip) {
			return fmt.Errorf("hostname %q resolves to private IP %s", hostname, ipStr)
		}
	}

	return nil
}

// ssrfSafeClient returns an HTTP client that re-validates each redirect target.
func ssrfSafeClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return validateExternalURL(req.URL.String())
		},
	}
}

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

// downloadResumePDF retrieves the resume PDF bytes from S3 or external URL.
func (s *MatchScoreService) downloadResumePDF(ctx context.Context, resume *resumeModel.Resume) ([]byte, error) {
	// Try S3 first
	if resume.StorageType == resumeModel.StorageTypeS3 && resume.StorageKey != nil && s.s3Client != nil {
		data, err := s.s3Client.GetObject(ctx, *resume.StorageKey)
		if err != nil {
			return nil, fmt.Errorf("failed to download resume from S3: %w", err)
		}
		return data, nil
	}

	// Try external URL
	if resume.FileURL != nil && *resume.FileURL != "" {
		data, err := downloadFromURL(ctx, *resume.FileURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download resume from URL: %w", err)
		}
		return data, nil
	}

	return nil, model.ErrResumeFileEmpty
}

// downloadFromURL fetches a file from an external URL with size limit, timeout, and SSRF protection.
func downloadFromURL(ctx context.Context, rawURL string) ([]byte, error) {
	if err := validateExternalURL(rawURL); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := ssrfSafeClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Limit read to maxResumeSize
	limitedReader := io.LimitReader(resp.Body, maxResumeSize+1)
	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}

	if len(data) > maxResumeSize {
		return nil, fmt.Errorf("resume file too large (max %d bytes)", maxResumeSize)
	}

	return data, nil
}
