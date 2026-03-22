package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/storage"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/matchscore/model"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMatchScoreCacheRepo implements ports.MatchScoreCacheRepository for testing.
type MockMatchScoreCacheRepo struct {
	GetFunc              func(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error)
	UpsertFunc           func(ctx context.Context, userID, jobID, resumeID string, result *model.MatchScoreResponse) error
	InvalidateByJobFunc  func(ctx context.Context, jobID string) error
	InvalidateByResumeFunc func(ctx context.Context, resumeID string) error
}

func (m *MockMatchScoreCacheRepo) Get(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, userID, jobID, resumeID)
	}
	return nil, nil
}

func (m *MockMatchScoreCacheRepo) Upsert(ctx context.Context, userID, jobID, resumeID string, result *model.MatchScoreResponse) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, userID, jobID, resumeID, result)
	}
	return nil
}

func (m *MockMatchScoreCacheRepo) InvalidateByJob(ctx context.Context, jobID string) error {
	if m.InvalidateByJobFunc != nil {
		return m.InvalidateByJobFunc(ctx, jobID)
	}
	return nil
}

func (m *MockMatchScoreCacheRepo) InvalidateByResume(ctx context.Context, resumeID string) error {
	if m.InvalidateByResumeFunc != nil {
		return m.InvalidateByResumeFunc(ctx, resumeID)
	}
	return nil
}

func TestCheckMatch_CacheHit(t *testing.T) {
	cached := &model.MatchScoreResponse{
		OverallScore:    85,
		Categories:      []model.MatchScoreCategory{{Name: "Skills", Score: 90, Details: "Good match"}},
		MissingKeywords: []string{"Docker"},
		Strengths:       []string{"Go"},
		Summary:         "Strong match",
	}

	cacheRepo := &MockMatchScoreCacheRepo{
		GetFunc: func(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error) {
			assert.Equal(t, "user-1", userID)
			assert.Equal(t, "job-1", jobID)
			assert.Equal(t, "resume-1", resumeID)
			return cached, nil
		},
	}

	// AI client, s3, jobRepo, resumeRepo are all nil — cache hit must return before touching them
	svc := NewMatchScoreService(nil, nil, nil, nil, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	result, err := svc.CheckMatch(context.Background(), "user-1", req)

	require.NoError(t, err)
	assert.Equal(t, 85, result.OverallScore)
	assert.Equal(t, "Strong match", result.Summary)
	assert.True(t, result.FromCache)
}

func TestCheckMatch_CacheReadError_FallsThrough(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{
		GetFunc: func(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error) {
			return nil, errors.New("connection refused")
		},
	}

	// limitChecker returns an error so we can verify the code fell through past the cache
	mockLimitChecker := &MockLimitChecker{
		CheckLimitFunc: func(ctx context.Context, userID, resource string) error {
			return errors.New("limit reached")
		},
	}

	svc := NewMatchScoreService(nil, nil, nil, nil, mockLimitChecker, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	// Should have fallen through cache and hit the limit checker
	assert.Error(t, err)
	assert.Equal(t, "limit reached", err.Error())
}

func TestCheckMatch_CacheMiss_FallsThrough(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{
		GetFunc: func(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error) {
			return nil, nil // cache miss
		},
	}

	mockLimitChecker := &MockLimitChecker{
		CheckLimitFunc: func(ctx context.Context, userID, resource string) error {
			return errors.New("limit reached")
		},
	}

	svc := NewMatchScoreService(nil, nil, nil, nil, mockLimitChecker, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	// Should have fallen through cache miss and hit the limit checker
	assert.Error(t, err)
	assert.Equal(t, "limit reached", err.Error())
}

func TestCheckMatch_NilCacheRepo_DoesNotPanic(t *testing.T) {
	mockLimitChecker := &MockLimitChecker{
		CheckLimitFunc: func(ctx context.Context, userID, resource string) error {
			return errors.New("limit reached")
		},
	}

	svc := NewMatchScoreService(nil, nil, nil, nil, mockLimitChecker, nil)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	// Should skip cache and hit limit checker
	assert.Error(t, err)
	assert.Equal(t, "limit reached", err.Error())
}

// MockLimitChecker implements LimitChecker for testing.
type MockLimitChecker struct {
	CheckLimitFunc   func(ctx context.Context, userID, resource string) error
	RecordAIUsageFunc func(ctx context.Context, userID string) error
}

func (m *MockLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, userID, resource)
	}
	return nil
}

func (m *MockLimitChecker) RecordAIUsage(ctx context.Context, userID string) error {
	if m.RecordAIUsageFunc != nil {
		return m.RecordAIUsageFunc(ctx, userID)
	}
	return nil
}

// --- isPrivateIP tests ---

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"loopback IPv4", "127.0.0.1", true},
		{"loopback range", "127.255.255.255", true},
		{"private 10.x", "10.0.0.1", true},
		{"private 10.255.x", "10.255.255.255", true},
		{"private 172.16.x", "172.16.0.1", true},
		{"private 172.31.x", "172.31.255.255", true},
		{"private 192.168.x", "192.168.1.1", true},
		{"link-local", "169.254.1.1", true},
		{"public IP", "8.8.8.8", false},
		{"public IP 2", "1.1.1.1", false},
		{"public 172.15", "172.15.255.255", false},
		{"public 172.32", "172.32.0.1", false},
		{"IPv6 loopback", "::1", true},
		{"IPv6 unique local", "fc00::1", true},
		{"IPv6 public", "2001:4860:4860::8888", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			require.NotNil(t, ip, "failed to parse IP: %s", tt.ip)
			result := isPrivateIP(ip)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- validateExternalURL tests ---

func TestValidateExternalURL(t *testing.T) {
	t.Run("rejects non-https scheme", func(t *testing.T) {
		err := validateExternalURL("http://example.com/file.pdf")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only https URLs are allowed")
	})

	t.Run("rejects empty hostname", func(t *testing.T) {
		err := validateExternalURL("https:///file.pdf")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("rejects localhost", func(t *testing.T) {
		err := validateExternalURL("https://localhost/file.pdf")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("rejects localhost case-insensitive", func(t *testing.T) {
		err := validateExternalURL("https://LOCALHOST/file.pdf")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not allowed")
	})

	t.Run("rejects invalid URL", func(t *testing.T) {
		err := validateExternalURL("://not-a-url")
		assert.Error(t, err)
	})

	t.Run("rejects ftp scheme", func(t *testing.T) {
		err := validateExternalURL("ftp://example.com/file.pdf")
		assert.Error(t, err)
	})

	t.Run("accepts valid https URL", func(t *testing.T) {
		// This test performs actual DNS lookup, so it may fail in offline environments.
		// We test with a well-known public domain.
		err := validateExternalURL("https://example.com/file.pdf")
		assert.NoError(t, err)
	})
}

// --- ssrfSafeClient tests ---

func TestSsrfSafeClient(t *testing.T) {
	client := ssrfSafeClient()
	assert.NotNil(t, client)
	assert.Equal(t, 30*time.Second, client.Timeout)
	assert.NotNil(t, client.CheckRedirect)
}

// --- CheckMatch with job/resume error paths ---

func TestCheckMatch_NoLimitChecker(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{
		GetFunc: func(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error) {
			return nil, nil // cache miss
		},
	}

	// No limit checker, jobRepo returns not found
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return nil, jobModel.ErrJobNotFound
		},
	}

	svc := NewMatchScoreService(nil, nil, mockJobRepo, nil, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	assert.Error(t, err)
	assert.Equal(t, jobModel.ErrJobNotFound, err)
}

func TestCheckMatch_JobFetchGenericError(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{}

	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return nil, errors.New("database unavailable")
		},
	}

	svc := NewMatchScoreService(nil, nil, mockJobRepo, nil, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get job")
}

func TestCheckMatch_JobDescriptionEmpty(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{}

	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test", Description: nil}, nil
		},
	}

	svc := NewMatchScoreService(nil, nil, mockJobRepo, nil, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	assert.Error(t, err)
	assert.Equal(t, model.ErrJobDescriptionEmpty, err)
}

func TestCheckMatch_JobDescriptionOnlyWhitespace(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{}

	desc := "   "
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test", Description: &desc}, nil
		},
	}

	svc := NewMatchScoreService(nil, nil, mockJobRepo, nil, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	assert.Error(t, err)
	assert.Equal(t, model.ErrJobDescriptionEmpty, err)
}

func TestCheckMatch_ResumeNotFound(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{}

	desc := "A real job description"
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test", Description: &desc}, nil
		},
	}

	mockResumeRepo := &MockResumeRepository{
		GetByIDFunc: func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return nil, resumeModel.ErrResumeNotFound
		},
	}

	svc := NewMatchScoreService(nil, nil, mockJobRepo, mockResumeRepo, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	assert.Error(t, err)
	assert.Equal(t, resumeModel.ErrResumeNotFound, err)
}

func TestCheckMatch_ResumeFetchGenericError(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{}

	desc := "A real job description"
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test", Description: &desc}, nil
		},
	}

	mockResumeRepo := &MockResumeRepository{
		GetByIDFunc: func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return nil, errors.New("db error")
		},
	}

	svc := NewMatchScoreService(nil, nil, mockJobRepo, mockResumeRepo, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get resume")
}

func TestCheckMatch_ResumeFileEmpty(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{}

	desc := "A real job description"
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test", Description: &desc}, nil
		},
	}

	mockResumeRepo := &MockResumeRepository{
		GetByIDFunc: func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{
				ID:          rid,
				StorageType: resumeModel.StorageTypeExternal,
				FileURL:     nil, // no file
				StorageKey:  nil,
			}, nil
		},
	}

	svc := NewMatchScoreService(nil, nil, mockJobRepo, mockResumeRepo, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	assert.Error(t, err)
	assert.Equal(t, model.ErrResumeFileEmpty, err)
}

func TestCheckMatch_ResumeEmptyURL(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{}

	desc := "A real job description"
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test", Description: &desc}, nil
		},
	}

	emptyURL := ""
	mockResumeRepo := &MockResumeRepository{
		GetByIDFunc: func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{
				ID:          rid,
				StorageType: resumeModel.StorageTypeExternal,
				FileURL:     &emptyURL,
			}, nil
		},
	}

	svc := NewMatchScoreService(nil, nil, mockJobRepo, mockResumeRepo, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)

	assert.Error(t, err)
	assert.Equal(t, model.ErrResumeFileEmpty, err)
}

// --- Mock repositories for matchscore ---

type MockJobRepository struct {
	GetByIDFunc func(ctx context.Context, uid, jid string) (*jobModel.Job, error)
}

func (m *MockJobRepository) Create(ctx context.Context, job *jobModel.Job) error { return nil }
func (m *MockJobRepository) GetByID(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, uid, jid)
	}
	return nil, nil
}
func (m *MockJobRepository) List(ctx context.Context, uid string, limit, offset int, status, sortBy, sortOrder string) ([]*jobModel.JobDTO, int, error) {
	return nil, 0, nil
}
func (m *MockJobRepository) Update(ctx context.Context, job *jobModel.Job) error { return nil }
func (m *MockJobRepository) Delete(ctx context.Context, uid, jid string) error   { return nil }
func (m *MockJobRepository) ToggleFavorite(ctx context.Context, uid, jid string) (bool, error) {
	return false, nil
}

type MockResumeRepository struct {
	GetByIDFunc func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error)
}

func (m *MockResumeRepository) Create(ctx context.Context, resume *resumeModel.Resume) error {
	return nil
}
func (m *MockResumeRepository) GetByID(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, uid, rid)
	}
	return nil, nil
}
func (m *MockResumeRepository) List(ctx context.Context, uid string, limit, offset int, sortBy, sortDir string) ([]*resumePorts.ResumeWithCount, int, error) {
	return nil, 0, nil
}
func (m *MockResumeRepository) Update(ctx context.Context, resume *resumeModel.Resume) error {
	return nil
}
func (m *MockResumeRepository) Delete(ctx context.Context, uid, rid string) error { return nil }

// ---------------------------------------------------------------------------
// downloadResumePDF tests
// ---------------------------------------------------------------------------

func TestDownloadResumePDF_S3Path_Success(t *testing.T) {
	pdfData := []byte("%PDF-1.4 test content")
	s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{
		"resumes/user-1/resume.pdf": pdfData,
	})
	defer cleanup()

	storageKey := "resumes/user-1/resume.pdf"
	svc := NewMatchScoreService(nil, s3Client, nil, nil, nil, nil)

	resume := &resumeModel.Resume{
		ID:          "resume-1",
		StorageType: resumeModel.StorageTypeS3,
		StorageKey:  &storageKey,
	}

	data, err := svc.downloadResumePDF(context.Background(), resume)
	require.NoError(t, err)
	assert.Equal(t, pdfData, data)
}

func TestDownloadResumePDF_S3Path_NotFound(t *testing.T) {
	s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{})
	defer cleanup()

	storageKey := "resumes/user-1/nonexistent.pdf"
	svc := NewMatchScoreService(nil, s3Client, nil, nil, nil, nil)

	resume := &resumeModel.Resume{
		ID:          "resume-1",
		StorageType: resumeModel.StorageTypeS3,
		StorageKey:  &storageKey,
	}

	_, err := svc.downloadResumePDF(context.Background(), resume)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download resume from S3")
}

func TestDownloadResumePDF_S3Path_NilClient(t *testing.T) {
	storageKey := "resumes/user-1/resume.pdf"
	svc := NewMatchScoreService(nil, nil, nil, nil, nil, nil)

	resume := &resumeModel.Resume{
		ID:          "resume-1",
		StorageType: resumeModel.StorageTypeS3,
		StorageKey:  &storageKey,
		FileURL:     nil,
	}

	// s3Client is nil, so S3 branch is skipped. No FileURL either, so ErrResumeFileEmpty.
	_, err := svc.downloadResumePDF(context.Background(), resume)
	assert.Equal(t, model.ErrResumeFileEmpty, err)
}

func TestDownloadResumePDF_ExternalURL_ValidationFails(t *testing.T) {
	svc := NewMatchScoreService(nil, nil, nil, nil, nil, nil)

	// Non-HTTPS URL triggers validation error
	httpURL := "http://example.com/resume.pdf"
	resume := &resumeModel.Resume{
		ID:          "resume-1",
		StorageType: resumeModel.StorageTypeExternal,
		FileURL:     &httpURL,
	}

	_, err := svc.downloadResumePDF(context.Background(), resume)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download resume from URL")
}

func TestDownloadResumePDF_ExternalURL_PrivateIP(t *testing.T) {
	svc := NewMatchScoreService(nil, nil, nil, nil, nil, nil)

	privateURL := "https://localhost/resume.pdf"
	resume := &resumeModel.Resume{
		ID:          "resume-1",
		StorageType: resumeModel.StorageTypeExternal,
		FileURL:     &privateURL,
	}

	_, err := svc.downloadResumePDF(context.Background(), resume)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download resume from URL")
}

func TestDownloadResumePDF_NoFileAtAll(t *testing.T) {
	svc := NewMatchScoreService(nil, nil, nil, nil, nil, nil)

	resume := &resumeModel.Resume{
		ID:          "resume-1",
		StorageType: resumeModel.StorageTypeExternal,
		FileURL:     nil,
		StorageKey:  nil,
	}

	_, err := svc.downloadResumePDF(context.Background(), resume)
	assert.Equal(t, model.ErrResumeFileEmpty, err)
}

func TestDownloadResumePDF_S3Path_NilStorageKey(t *testing.T) {
	svc := NewMatchScoreService(nil, nil, nil, nil, nil, nil)

	resume := &resumeModel.Resume{
		ID:          "resume-1",
		StorageType: resumeModel.StorageTypeS3,
		StorageKey:  nil,  // nil key, won't try S3
		FileURL:     nil,
	}

	_, err := svc.downloadResumePDF(context.Background(), resume)
	assert.Equal(t, model.ErrResumeFileEmpty, err)
}

// ---------------------------------------------------------------------------
// downloadFromURL tests
// ---------------------------------------------------------------------------

func TestDownloadFromURL_NonHTTPS(t *testing.T) {
	_, err := downloadFromURL(context.Background(), "http://example.com/file.pdf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "URL validation failed")
}

func TestDownloadFromURL_InvalidURL(t *testing.T) {
	_, err := downloadFromURL(context.Background(), "://not-a-url")
	assert.Error(t, err)
}

func TestDownloadFromURL_PrivateIPBlocked(t *testing.T) {
	_, err := downloadFromURL(context.Background(), "https://localhost/file.pdf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "URL validation failed")
}

func TestDownloadFromURL_HTTPSServerNon200(t *testing.T) {
	// Create an HTTPS test server that returns 404
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	// This will fail at URL validation because httptest.NewTLSServer
	// uses 127.0.0.1 which is a private IP - expected behavior
	_, err := downloadFromURL(context.Background(), srv.URL+"/file.pdf")
	assert.Error(t, err)
}

func TestDownloadFromURL_FileTooLarge(t *testing.T) {
	// This test verifies the validation path (SSRF check blocks 127.0.0.1)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Would send a large file, but SSRF blocks this first
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	_, err := downloadFromURL(context.Background(), srv.URL+"/large.pdf")
	assert.Error(t, err)
}

func TestDownloadFromURL_ValidURLButConnectionFails(t *testing.T) {
	// Use a valid HTTPS URL with non-private IP that won't actually serve content.
	// This exercises the HTTP request creation + client.Do path in downloadFromURL.
	// Using port 1 which is almost certainly not listening.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := downloadFromURL(ctx, "https://example.com:1/nonexistent.pdf")
	// Should fail during HTTP request (connection timeout/refused)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// ssrfSafeClient tests (additional)
// ---------------------------------------------------------------------------

func TestSsrfSafeClient_RedirectLimit(t *testing.T) {
	client := ssrfSafeClient()
	assert.NotNil(t, client)
	assert.Equal(t, 30*time.Second, client.Timeout)
	assert.NotNil(t, client.CheckRedirect)

	// Test that too many redirects returns error
	viaSlice := make([]*http.Request, 10)
	for i := range viaSlice {
		viaSlice[i], _ = http.NewRequest("GET", "https://example.com", nil)
	}
	req, _ := http.NewRequest("GET", "https://example.com/redirect", nil)
	err := client.CheckRedirect(req, viaSlice)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many redirects")
}

func TestSsrfSafeClient_RedirectToPrivateIP(t *testing.T) {
	client := ssrfSafeClient()

	viaSlice := make([]*http.Request, 1)
	viaSlice[0], _ = http.NewRequest("GET", "https://example.com", nil)
	req, _ := http.NewRequest("GET", "https://localhost/evil", nil)
	err := client.CheckRedirect(req, viaSlice)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestSsrfSafeClient_RedirectToHTTP(t *testing.T) {
	client := ssrfSafeClient()

	viaSlice := make([]*http.Request, 1)
	viaSlice[0], _ = http.NewRequest("GET", "https://example.com", nil)
	req, _ := http.NewRequest("GET", "http://example.com/page", nil)
	err := client.CheckRedirect(req, viaSlice)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only https URLs are allowed")
}

// ---------------------------------------------------------------------------
// CheckMatch full happy path via AI mock
// ---------------------------------------------------------------------------

func TestCheckMatch_FullHappyPath_S3(t *testing.T) {
	matchResult := ai.MatchResult{
		OverallScore:    85,
		Categories:      []ai.MatchCategory{{Name: "Skills", Score: 90, Details: "Great match"}},
		MissingKeywords: []string{"Docker"},
		Strengths:       []string{"Go", "Kubernetes"},
		Summary:         "Strong technical fit",
	}
	resultJSON, err := json.Marshal(matchResult)
	require.NoError(t, err)

	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: string(resultJSON)},
			},
		}, nil
	})

	// Create a mock S3 server with resume data
	pdfData := []byte("%PDF-1.4 fake resume content")
	s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{
		"resumes/user-1/resume.pdf": pdfData,
	})
	defer cleanup()

	desc := "We need a Go developer with Kubernetes experience"
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(_ context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Go Developer", Description: &desc}, nil
		},
	}

	storageKey := "resumes/user-1/resume.pdf"
	mockResumeRepo := &MockResumeRepository{
		GetByIDFunc: func(_ context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{
				ID:          rid,
				StorageType: resumeModel.StorageTypeS3,
				StorageKey:  &storageKey,
			}, nil
		},
	}

	var recordedUserID string
	lc := &MockLimitChecker{
		RecordAIUsageFunc: func(_ context.Context, userID string) error {
			recordedUserID = userID
			return nil
		},
	}

	cacheRepo := &MockMatchScoreCacheRepo{}
	svc := NewMatchScoreService(aiClient, s3Client, mockJobRepo, mockResumeRepo, lc, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	result, err := svc.CheckMatch(context.Background(), "user-1", req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 85, result.OverallScore)
	assert.Equal(t, "Strong technical fit", result.Summary)
	assert.Equal(t, "user-1", recordedUserID)
	assert.Len(t, result.Categories, 1)
	assert.Equal(t, "Skills", result.Categories[0].Name)
	assert.Equal(t, 90, result.Categories[0].Score)
	assert.Equal(t, []string{"Docker"}, result.MissingKeywords)
	assert.Equal(t, []string{"Go", "Kubernetes"}, result.Strengths)
	assert.False(t, result.FromCache)
}

func TestCheckMatch_AICallFails_S3(t *testing.T) {
	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return nil, errors.New("AI service unavailable")
	})

	pdfData := []byte("%PDF-1.4 fake resume content")
	s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{
		"resumes/user-1/resume.pdf": pdfData,
	})
	defer cleanup()

	desc := "A real job description"
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(_ context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test", Description: &desc}, nil
		},
	}

	storageKey := "resumes/user-1/resume.pdf"
	mockResumeRepo := &MockResumeRepository{
		GetByIDFunc: func(_ context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{
				ID:          rid,
				StorageType: resumeModel.StorageTypeS3,
				StorageKey:  &storageKey,
			}, nil
		},
	}

	svc := NewMatchScoreService(aiClient, s3Client, mockJobRepo, mockResumeRepo, nil, nil)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)
	assert.Error(t, err)
	assert.ErrorIs(t, err, model.ErrMatchFailed)
}

func TestCheckMatch_RecordAIUsageFailure_NonFatal(t *testing.T) {
	matchResult := ai.MatchResult{
		OverallScore: 75,
		Summary:      "Good match",
	}
	resultJSON, _ := json.Marshal(matchResult)

	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: string(resultJSON)},
			},
		}, nil
	})

	pdfData := []byte("%PDF-1.4 fake")
	s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{
		"resumes/user-1/resume.pdf": pdfData,
	})
	defer cleanup()

	cacheRepo := &MockMatchScoreCacheRepo{}
	desc := "A real job description"
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(_ context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test", Description: &desc}, nil
		},
	}

	storageKey := "resumes/user-1/resume.pdf"
	mockResumeRepo := &MockResumeRepository{
		GetByIDFunc: func(_ context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{
				ID:          rid,
				StorageType: resumeModel.StorageTypeS3,
				StorageKey:  &storageKey,
			}, nil
		},
	}

	lc := &MockLimitChecker{
		RecordAIUsageFunc: func(_ context.Context, _ string) error {
			return fmt.Errorf("redis down")
		},
	}

	svc := NewMatchScoreService(aiClient, s3Client, mockJobRepo, mockResumeRepo, lc, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	// RecordAIUsage failure should NOT cause the overall call to fail
	result, err := svc.CheckMatch(context.Background(), "user-1", req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 75, result.OverallScore)
}

func TestCheckMatch_CacheUpsertFailure_NonFatal(t *testing.T) {
	matchResult := ai.MatchResult{
		OverallScore: 80,
		Summary:      "Solid match",
	}
	resultJSON, _ := json.Marshal(matchResult)

	aiClient := ai.NewTestClient(func(_ context.Context, _ anthropic.MessageNewParams) (*anthropic.Message, error) {
		return &anthropic.Message{
			Content: []anthropic.ContentBlockUnion{
				{Type: "text", Text: string(resultJSON)},
			},
		}, nil
	})

	pdfData := []byte("%PDF-1.4 fake")
	s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{
		"resumes/user-1/resume.pdf": pdfData,
	})
	defer cleanup()

	cacheRepo := &MockMatchScoreCacheRepo{
		UpsertFunc: func(_ context.Context, _, _, _ string, _ *model.MatchScoreResponse) error {
			return fmt.Errorf("cache write failed")
		},
	}

	desc := "A real job description"
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(_ context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test", Description: &desc}, nil
		},
	}

	storageKey := "resumes/user-1/resume.pdf"
	mockResumeRepo := &MockResumeRepository{
		GetByIDFunc: func(_ context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{
				ID:          rid,
				StorageType: resumeModel.StorageTypeS3,
				StorageKey:  &storageKey,
			}, nil
		},
	}

	svc := NewMatchScoreService(aiClient, s3Client, mockJobRepo, mockResumeRepo, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	// Cache upsert failure should be non-fatal
	result, err := svc.CheckMatch(context.Background(), "user-1", req)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 80, result.OverallScore)
}

// ---------------------------------------------------------------------------
// validateExternalURL additional tests
// ---------------------------------------------------------------------------

func TestValidateExternalURL_DNSLookupFailure(t *testing.T) {
	err := validateExternalURL("https://this-domain-definitely-does-not-exist-abcxyz123.com/file.pdf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DNS lookup failed")
}

func TestValidateExternalURL_PrivateIPResolution(t *testing.T) {
	// 127.0.0.1.nip.io resolves to 127.0.0.1 which is private
	// This may not work in all environments, so we test with localhost instead
	err := validateExternalURL("https://localhost/file.pdf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

// ---------------------------------------------------------------------------
// isPrivateIP additional edge cases
// ---------------------------------------------------------------------------

func TestIsPrivateIP_ZeroIP(t *testing.T) {
	ip := net.ParseIP("0.0.0.0")
	result := isPrivateIP(ip)
	// 0.0.0.0 is not in the private CIDR ranges defined
	assert.False(t, result)
}

func TestIsPrivateIP_MulticastIP(t *testing.T) {
	ip := net.ParseIP("224.0.0.1")
	result := isPrivateIP(ip)
	// Multicast is not in the private CIDR list
	assert.False(t, result)
}

// ---------------------------------------------------------------------------
// CheckMatch with S3 storage type but nil S3 client (falls through to ext URL)
// ---------------------------------------------------------------------------

func TestCheckMatch_S3Resume_NilClient_FallsToURLCheck(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{}

	desc := "A real job description"
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(_ context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test", Description: &desc}, nil
		},
	}

	storageKey := "resumes/user-1/resume.pdf"
	mockResumeRepo := &MockResumeRepository{
		GetByIDFunc: func(_ context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{
				ID:          rid,
				StorageType: resumeModel.StorageTypeS3,
				StorageKey:  &storageKey,
				FileURL:     nil,
			}, nil
		},
	}

	// s3Client is nil, so S3 branch skipped, FileURL is nil -> ErrResumeFileEmpty
	svc := NewMatchScoreService(nil, nil, mockJobRepo, mockResumeRepo, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)
	assert.Equal(t, model.ErrResumeFileEmpty, err)
}

func TestCheckMatch_S3Resume_WithFallbackURL(t *testing.T) {
	cacheRepo := &MockMatchScoreCacheRepo{}

	desc := "A real job description"
	mockJobRepo := &MockJobRepository{
		GetByIDFunc: func(_ context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test", Description: &desc}, nil
		},
	}

	storageKey := "resumes/user-1/resume.pdf"
	fileURL := "http://example.com/resume.pdf" // http not https, will fail validation
	mockResumeRepo := &MockResumeRepository{
		GetByIDFunc: func(_ context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{
				ID:          rid,
				StorageType: resumeModel.StorageTypeS3,
				StorageKey:  &storageKey,
				FileURL:     &fileURL,
			}, nil
		},
	}

	// s3Client is nil, falls through to URL which fails SSRF check
	svc := NewMatchScoreService(nil, nil, mockJobRepo, mockResumeRepo, nil, cacheRepo)
	req := &model.MatchScoreRequest{JobID: "job-1", ResumeID: "resume-1"}

	_, err := svc.CheckMatch(context.Background(), "user-1", req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to download resume from URL")
}

// ---------------------------------------------------------------------------
// Additional SSRF protection tests
// ---------------------------------------------------------------------------

func TestValidateExternalURL_EmptyScheme(t *testing.T) {
	err := validateExternalURL("example.com/file.pdf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only https URLs are allowed")
}

func TestIsPrivateIP_AllPrivateRanges(t *testing.T) {
	privateIPs := []string{
		"10.0.0.0", "10.255.255.255",
		"172.16.0.0", "172.31.255.255",
		"192.168.0.0", "192.168.255.255",
		"127.0.0.0", "127.255.255.255",
		"169.254.0.0", "169.254.255.255",
	}
	for _, ipStr := range privateIPs {
		ip := net.ParseIP(ipStr)
		require.NotNil(t, ip, "failed to parse IP: %s", ipStr)
		assert.True(t, isPrivateIP(ip), "expected %s to be private", ipStr)
	}
}

func TestIsPrivateIP_PublicIPs(t *testing.T) {
	publicIPs := []string{
		"8.8.8.8", "1.1.1.1", "142.250.80.46",
		"172.32.0.0", "172.15.255.255",
		"11.0.0.0", "192.169.0.0",
	}
	for _, ipStr := range publicIPs {
		ip := net.ParseIP(ipStr)
		require.NotNil(t, ip, "failed to parse IP: %s", ipStr)
		assert.False(t, isPrivateIP(ip), "expected %s to be public", ipStr)
	}
}

