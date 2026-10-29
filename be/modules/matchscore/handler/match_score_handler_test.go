package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	jobPorts "github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/andreypavlenko/jobber/modules/matchscore/model"
	matchPorts "github.com/andreypavlenko/jobber/modules/matchscore/ports"
	matchService "github.com/andreypavlenko/jobber/modules/matchscore/service"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockJobRepository implements jobPorts.JobRepository
type MockJobRepository struct {
	CreateFunc            func(ctx context.Context, job *jobModel.Job) error
	GetByIDFunc           func(ctx context.Context, userID, jobID string) (*jobModel.Job, error)
	ListFunc              func(ctx context.Context, userID string, opts *jobPorts.ListOptions) ([]*jobModel.JobDTO, int, error)
	UpdateFunc            func(ctx context.Context, userID, jobID string, patch *jobModel.JobPatch) (*jobModel.Job, error)
	DeleteFunc            func(ctx context.Context, userID, jobID string) error
	ToggleFavoriteFunc    func(ctx context.Context, userID, jobID string) (bool, error)
	GetLastActivityAtFunc func(ctx context.Context, jobID string) (time.Time, error)
}

func (m *MockJobRepository) Create(ctx context.Context, job *jobModel.Job) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, job)
	}
	return nil
}

func (m *MockJobRepository) CreateWithInitialStage(ctx context.Context, job *jobModel.Job, stage *jobModel.JobStage) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, job)
	}
	return nil
}

func (m *MockJobRepository) GetByID(ctx context.Context, userID, jobID string) (*jobModel.Job, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, jobID)
	}
	return nil, nil
}

func (m *MockJobRepository) List(ctx context.Context, userID string, opts *jobPorts.ListOptions) ([]*jobModel.JobDTO, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, opts)
	}
	return nil, 0, nil
}

func (m *MockJobRepository) Update(ctx context.Context, userID, jobID string, patch *jobModel.JobPatch) (*jobModel.Job, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, userID, jobID, patch)
	}
	return nil, nil
}

func (m *MockJobRepository) Delete(ctx context.Context, userID, jobID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, jobID)
	}
	return nil
}

func (m *MockJobRepository) ToggleFavorite(ctx context.Context, userID, jobID string) (bool, error) {
	if m.ToggleFavoriteFunc != nil {
		return m.ToggleFavoriteFunc(ctx, userID, jobID)
	}
	return false, nil
}

func (m *MockJobRepository) GetLastActivityAt(ctx context.Context, userID, jobID string) (time.Time, error) {
	if m.GetLastActivityAtFunc != nil {
		return m.GetLastActivityAtFunc(ctx, jobID)
	}
	return time.Time{}, nil
}

// MockResumeRepository implements resumePorts.ResumeRepository
type MockResumeRepository struct {
	CreateFinalizedUploadFunc func(ctx context.Context, resume *resumeModel.Resume, maxResumes int) error
	CreateFunc                func(ctx context.Context, resume *resumeModel.Resume) error
	GetByIDFunc               func(ctx context.Context, userID, resumeID string) (*resumeModel.Resume, error)
	ListFunc                  func(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*resumePorts.ResumeWithCount, int, error)
	UpdateFunc                func(ctx context.Context, resume *resumeModel.Resume) error
	DeleteFunc                func(ctx context.Context, userID, resumeID string) error
}

func (m *MockResumeRepository) CreateFinalizedUpload(ctx context.Context, resume *resumeModel.Resume, maxResumes int) error {
	if m.CreateFinalizedUploadFunc != nil {
		return m.CreateFinalizedUploadFunc(ctx, resume, maxResumes)
	}
	return m.Create(ctx, resume)
}

func (m *MockResumeRepository) Create(ctx context.Context, resume *resumeModel.Resume) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, resume)
	}
	return nil
}

func (m *MockResumeRepository) GetByID(ctx context.Context, userID, resumeID string) (*resumeModel.Resume, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, resumeID)
	}
	return nil, nil
}

func (m *MockResumeRepository) List(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*resumePorts.ResumeWithCount, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, limit, offset, sortBy, sortDir)
	}
	return nil, 0, nil
}

func (m *MockResumeRepository) Update(ctx context.Context, resume *resumeModel.Resume) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, resume)
	}
	return nil
}

func (m *MockResumeRepository) Delete(ctx context.Context, userID, resumeID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, resumeID)
	}
	return nil
}

// MockLimitChecker implements matchService.LimitChecker
type MockLimitChecker struct {
	CheckLimitFunc    func(ctx context.Context, userID, resource string) error
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

// MockMatchScoreCacheRepository implements matchPorts.MatchScoreCacheRepository
type MockMatchScoreCacheRepository struct {
	GetFunc                func(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error)
	UpsertFunc             func(ctx context.Context, userID, jobID, resumeID string, result *model.MatchScoreResponse) error
	InvalidateByJobFunc    func(ctx context.Context, jobID string) error
	InvalidateByResumeFunc func(ctx context.Context, resumeID string) error
}

func (m *MockMatchScoreCacheRepository) Get(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, userID, jobID, resumeID)
	}
	return nil, nil
}

func (m *MockMatchScoreCacheRepository) Upsert(ctx context.Context, userID, jobID, resumeID string, result *model.MatchScoreResponse) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, userID, jobID, resumeID, result)
	}
	return nil
}

func (m *MockMatchScoreCacheRepository) InvalidateByJob(ctx context.Context, jobID string) error {
	if m.InvalidateByJobFunc != nil {
		return m.InvalidateByJobFunc(ctx, jobID)
	}
	return nil
}

func (m *MockMatchScoreCacheRepository) InvalidateByResume(ctx context.Context, resumeID string) error {
	if m.InvalidateByResumeFunc != nil {
		return m.InvalidateByResumeFunc(ctx, resumeID)
	}
	return nil
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func mockAuthMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func noopRateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

func newTestHandler(jobRepo *MockJobRepository, resumeRepo *MockResumeRepository, limitChecker matchService.LimitChecker, cacheRepo matchPorts.MatchScoreCacheRepository) *MatchScoreHandler {
	svc := matchService.NewMatchScoreService(nil, nil, jobRepo, resumeRepo, limitChecker, cacheRepo)
	return NewMatchScoreHandler(svc)
}

func TestMatchScoreHandler_CheckMatch(t *testing.T) {
	userID := "user-123"

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestHandler(&MockJobRepository{}, &MockResumeRepository{}, nil, nil)

		router := setupTestRouter()
		router.POST("/match-score", handler.CheckMatch)

		body := `{"job_id":"job-1","resume_id":"resume-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/match-score", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		handler := newTestHandler(&MockJobRepository{}, &MockResumeRepository{}, nil, nil)

		router := setupTestRouter()
		router.POST("/match-score", mockAuthMiddleware(userID), handler.CheckMatch)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPost, "/match-score", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, string(model.CodeValidationError), response["error_code"])
	})

	t.Run("returns 400 for missing job_id", func(t *testing.T) {
		handler := newTestHandler(&MockJobRepository{}, &MockResumeRepository{}, nil, nil)

		router := setupTestRouter()
		router.POST("/match-score", mockAuthMiddleware(userID), handler.CheckMatch)

		body := `{"resume_id":"resume-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/match-score", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for missing resume_id", func(t *testing.T) {
		handler := newTestHandler(&MockJobRepository{}, &MockResumeRepository{}, nil, nil)

		router := setupTestRouter()
		router.POST("/match-score", mockAuthMiddleware(userID), handler.CheckMatch)

		body := `{"job_id":"job-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/match-score", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 403 when plan limit reached", func(t *testing.T) {
		mockLimiter := &MockLimitChecker{
			CheckLimitFunc: func(ctx context.Context, uid, resource string) error {
				return subModel.ErrLimitReached
			},
		}
		handler := newTestHandler(&MockJobRepository{}, &MockResumeRepository{}, mockLimiter, nil)

		router := setupTestRouter()
		router.POST("/match-score", mockAuthMiddleware(userID), handler.CheckMatch)

		body := `{"job_id":"job-1","resume_id":"resume-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/match-score", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "PLAN_LIMIT_REACHED", response["error_code"])
	})

	t.Run("returns 404 when job not found", func(t *testing.T) {
		mockJobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jobID string) (*jobModel.Job, error) {
				return nil, jobModel.ErrJobNotFound
			},
		}
		handler := newTestHandler(mockJobRepo, &MockResumeRepository{}, nil, nil)

		router := setupTestRouter()
		router.POST("/match-score", mockAuthMiddleware(userID), handler.CheckMatch)

		body := `{"job_id":"nonexistent","resume_id":"resume-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/match-score", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, string(model.CodeJobNotFound), response["error_code"])
	})

	t.Run("returns 400 when job description is empty", func(t *testing.T) {
		emptyDesc := "   "
		mockJobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jobID string) (*jobModel.Job, error) {
				return &jobModel.Job{ID: jobID, UserID: uid, Title: "Test Job", Description: &emptyDesc}, nil
			},
		}
		handler := newTestHandler(mockJobRepo, &MockResumeRepository{}, nil, nil)

		router := setupTestRouter()
		router.POST("/match-score", mockAuthMiddleware(userID), handler.CheckMatch)

		body := `{"job_id":"job-1","resume_id":"resume-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/match-score", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, string(model.CodeJobDescriptionEmpty), response["error_code"])
	})

	t.Run("returns 400 when job description is nil", func(t *testing.T) {
		mockJobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jobID string) (*jobModel.Job, error) {
				return &jobModel.Job{ID: jobID, UserID: uid, Title: "Test Job", Description: nil}, nil
			},
		}
		handler := newTestHandler(mockJobRepo, &MockResumeRepository{}, nil, nil)

		router := setupTestRouter()
		router.POST("/match-score", mockAuthMiddleware(userID), handler.CheckMatch)

		body := `{"job_id":"job-1","resume_id":"resume-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/match-score", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 404 when resume not found", func(t *testing.T) {
		desc := "A valid job description"
		mockJobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jobID string) (*jobModel.Job, error) {
				return &jobModel.Job{ID: jobID, UserID: uid, Title: "Test Job", Description: &desc}, nil
			},
		}
		mockResumeRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, resumeID string) (*resumeModel.Resume, error) {
				return nil, resumeModel.ErrResumeNotFound
			},
		}
		handler := newTestHandler(mockJobRepo, mockResumeRepo, nil, nil)

		router := setupTestRouter()
		router.POST("/match-score", mockAuthMiddleware(userID), handler.CheckMatch)

		body := `{"job_id":"job-1","resume_id":"nonexistent"}`
		req, _ := http.NewRequest(http.MethodPost, "/match-score", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, string(model.CodeResumeNotFound), response["error_code"])
	})

	t.Run("returns 400 when resume has no file", func(t *testing.T) {
		desc := "A valid job description"
		mockJobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jobID string) (*jobModel.Job, error) {
				return &jobModel.Job{ID: jobID, UserID: uid, Title: "Test Job", Description: &desc}, nil
			},
		}
		mockResumeRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, resumeID string) (*resumeModel.Resume, error) {
				return &resumeModel.Resume{
					ID:          resumeID,
					UserID:      uid,
					Title:       "Resume",
					StorageType: resumeModel.StorageTypeExternal,
					FileURL:     nil,
				}, nil
			},
		}
		handler := newTestHandler(mockJobRepo, mockResumeRepo, nil, nil)

		router := setupTestRouter()
		router.POST("/match-score", mockAuthMiddleware(userID), handler.CheckMatch)

		body := `{"job_id":"job-1","resume_id":"resume-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/match-score", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns cached result when available", func(t *testing.T) {
		cachedResult := &model.MatchScoreResponse{
			OverallScore:    85,
			Summary:         "Good match",
			Categories:      []model.MatchScoreCategory{{Name: "Skills", Score: 80, Details: "Good skills match"}},
			MissingKeywords: []string{"kubernetes"},
			Strengths:       []string{"golang"},
		}
		mockCache := &MockMatchScoreCacheRepository{
			GetFunc: func(ctx context.Context, uid, jobID, resumeID string) (*model.MatchScoreResponse, error) {
				return cachedResult, nil
			},
		}
		handler := newTestHandler(&MockJobRepository{}, &MockResumeRepository{}, nil, mockCache)

		router := setupTestRouter()
		router.POST("/match-score", mockAuthMiddleware(userID), handler.CheckMatch)

		body := `{"job_id":"job-1","resume_id":"resume-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/match-score", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.MatchScoreResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, 85, response.OverallScore)
		assert.True(t, response.FromCache)
		assert.Equal(t, "Good match", response.Summary)
	})
}

func TestMatchScoreHandler_RegisterRoutes(t *testing.T) {
	handler := newTestHandler(&MockJobRepository{}, &MockResumeRepository{}, nil, nil)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"), noopRateLimiter())

	// Verify the route is in gin's route table
	registeredRoutes := router.Routes()
	found := false
	for _, r := range registeredRoutes {
		if r.Method == http.MethodPost && r.Path == "/api/v1/match-score" {
			found = true
			break
		}
	}
	assert.True(t, found, "POST /api/v1/match-score should be registered")
}

func (m *MockResumeRepository) CountApplications(ctx context.Context, userID, resumeID string) (int, error) {
	return 0, nil
}

func TestMatchScoreHandler_CacheLookup(t *testing.T) {
	const jobID = "11111111-1111-4111-8111-111111111111"
	const resumeID = "22222222-2222-4222-8222-222222222222"
	for _, tt := range []struct {
		name      string
		user      string
		query     string
		jobErr    error
		resumeErr error
		cacheErr  error
		hit       bool
		status    int
		code      string
		lookups   int
	}{
		{name: "hit without AI or credits", user: "owner", hit: true, status: 200, lookups: 1},
		{name: "exact miss never generates", user: "owner", status: 404, code: "MATCH_SCORE_NOT_READY", lookups: 1},
		{name: "cache error is not a miss", user: "owner", cacheErr: errors.New("db unavailable"), status: 500, code: "INTERNAL_ERROR", lookups: 1},
		{name: "other user's job", user: "other", jobErr: jobModel.ErrJobNotFound, hit: true, status: 404, code: "JOB_NOT_FOUND"},
		{name: "other user's resume", user: "other", resumeErr: resumeModel.ErrResumeNotFound, hit: true, status: 404, code: "RESUME_NOT_FOUND"},
		{name: "job read error", user: "owner", jobErr: errors.New("db error"), status: 500, code: "INTERNAL_ERROR"},
		{name: "resume read error", user: "owner", resumeErr: errors.New("db error"), status: 500, code: "INTERNAL_ERROR"},
		{name: "invalid UUID", user: "owner", query: "job_id=invalid&resume_id=" + resumeID, status: 400, code: "VALIDATION_ERROR"},
		{name: "missing resume", user: "owner", query: "job_id=" + jobID, status: 400, code: "VALIDATION_ERROR"},
		{name: "unauthenticated", status: 401, code: "UNAUTHORIZED"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			lookups, creditChecks, writes := 0, 0, 0
			jobs := &MockJobRepository{GetByIDFunc: func(ctx context.Context, user, id string) (*jobModel.Job, error) {
				require.Equal(t, tt.user, user)
				require.Equal(t, jobID, id)
				return &jobModel.Job{ID: id, UserID: user}, tt.jobErr
			}}
			resumes := &MockResumeRepository{GetByIDFunc: func(ctx context.Context, user, id string) (*resumeModel.Resume, error) {
				require.Equal(t, tt.user, user)
				require.Equal(t, resumeID, id)
				return &resumeModel.Resume{ID: id, UserID: user}, tt.resumeErr
			}}
			cache := &MockMatchScoreCacheRepository{
				GetFunc: func(ctx context.Context, user, job, resume string) (*model.MatchScoreResponse, error) {
					lookups++
					require.Equal(t, tt.user, user)
					require.Equal(t, jobID, job)
					require.Equal(t, resumeID, resume)
					if tt.hit {
						return &model.MatchScoreResponse{OverallScore: 85, Summary: "Saved result"}, nil
					}
					return nil, tt.cacheErr
				},
				UpsertFunc: func(context.Context, string, string, string, *model.MatchScoreResponse) error { writes++; return nil },
			}
			limits := &MockLimitChecker{CheckLimitFunc: func(context.Context, string, string) error { creditChecks++; return subModel.ErrLimitReached }}
			h := newTestHandler(jobs, resumes, limits, cache)
			router := setupTestRouter()
			auth := func(c *gin.Context) {
				if tt.user != "" {
					c.Set("user_id", tt.user)
				}
				c.Next()
			}
			h.RegisterCacheRoutes(router.Group("/api/v1"), auth)
			query := tt.query
			if query == "" {
				query = "job_id=" + jobID + "&resume_id=" + resumeID
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/match-score?"+query, nil))
			require.Equal(t, tt.status, w.Code, w.Body.String())
			assert.Equal(t, "private, no-store", w.Header().Get("Cache-Control"))
			assert.Equal(t, tt.lookups, lookups)
			assert.Zero(t, creditChecks)
			assert.Zero(t, writes)
			if tt.code != "" {
				assert.Contains(t, w.Body.String(), `"error_code":"`+tt.code+`"`)
			} else {
				assert.Contains(t, w.Body.String(), `"from_cache":true`)
			}
		})
	}
}
