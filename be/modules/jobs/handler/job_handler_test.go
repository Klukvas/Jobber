package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/andreypavlenko/jobber/modules/jobs/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCompanyRepository implements companyPorts.CompanyRepository for handler tests
type MockCompanyRepository struct{}

func (m *MockCompanyRepository) Create(ctx context.Context, company *companyModel.Company) error {
	return nil
}
func (m *MockCompanyRepository) GetByID(ctx context.Context, userID, companyID string) (*companyModel.Company, error) {
	return &companyModel.Company{ID: companyID, UserID: userID}, nil
}
func (m *MockCompanyRepository) GetByIDEnriched(ctx context.Context, userID, companyID string) (*companyModel.CompanyDTO, error) {
	return nil, nil
}
func (m *MockCompanyRepository) List(ctx context.Context, userID string, opts *companyPorts.ListOptions) ([]*companyModel.CompanyDTO, int, error) {
	return nil, 0, nil
}
func (m *MockCompanyRepository) Update(ctx context.Context, company *companyModel.Company) error {
	return nil
}
func (m *MockCompanyRepository) Delete(ctx context.Context, userID, companyID string) error {
	return nil
}
func (m *MockCompanyRepository) GetRelatedJobsAndApplicationsCount(ctx context.Context, userID, companyID string) (int, int, error) {
	return 0, 0, nil
}
func (m *MockCompanyRepository) ToggleFavorite(ctx context.Context, userID, companyID string) (bool, error) {
	return false, nil
}

var defaultMockCompanyRepo = &MockCompanyRepository{}

// MockCommentRepository implements commentPorts.CommentRepository for handler tests
type MockCommentRepository struct{}

func (m *MockCommentRepository) Create(ctx context.Context, comment *commentModel.Comment) error {
	return nil
}
func (m *MockCommentRepository) ListByJob(ctx context.Context, jobID, userID string) ([]*commentModel.Comment, error) {
	return nil, nil
}
func (m *MockCommentRepository) Delete(ctx context.Context, userID, commentID string) error {
	return nil
}

var defaultMockCommentRepo = &MockCommentRepository{}

// MockStageTemplateRepository implements ports.StageTemplateRepository for
// handler tests. Its List returns a single first pipeline column so that
// Create can place a new card in the default column.
type MockStageTemplateRepository struct{}

func (m *MockStageTemplateRepository) Create(ctx context.Context, template *model.StageTemplate) error {
	return nil
}
func (m *MockStageTemplateRepository) GetByID(ctx context.Context, userID, templateID string) (*model.StageTemplate, error) {
	return &model.StageTemplate{ID: templateID, UserID: userID, Name: "Wishlist", Order: 0}, nil
}
func (m *MockStageTemplateRepository) List(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error) {
	return []*model.StageTemplate{{ID: "stage-wishlist", UserID: userID, Name: "Wishlist", Order: 0}}, 1, nil
}
func (m *MockStageTemplateRepository) Update(ctx context.Context, template *model.StageTemplate) error {
	return nil
}
func (m *MockStageTemplateRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	return nil
}
func (m *MockStageTemplateRepository) Delete(ctx context.Context, userID, templateID string) error {
	return nil
}

var defaultMockStageTemplateRepo = &MockStageTemplateRepository{}

// MockJobRepository implements ports.JobRepository
type MockJobRepository struct {
	CreateFunc            func(ctx context.Context, job *model.Job) error
	GetByIDFunc           func(ctx context.Context, userID, jobID string) (*model.Job, error)
	ListFunc              func(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.JobDTO, int, error)
	UpdateFunc            func(ctx context.Context, job *model.Job) error
	DeleteFunc            func(ctx context.Context, userID, jobID string) error
	ToggleFavoriteFunc    func(ctx context.Context, userID, jobID string) (bool, error)
	GetLastActivityAtFunc func(ctx context.Context, jobID string) (time.Time, error)
}

func (m *MockJobRepository) Create(ctx context.Context, job *model.Job) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, job)
	}
	return nil
}

func (m *MockJobRepository) GetByID(ctx context.Context, userID, jobID string) (*model.Job, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, jobID)
	}
	return nil, nil
}

func (m *MockJobRepository) List(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, opts)
	}
	return nil, 0, nil
}

func (m *MockJobRepository) Update(ctx context.Context, job *model.Job) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, job)
	}
	return nil
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

func newTestJobService(jobRepo *MockJobRepository) *service.JobService {
	return service.NewJobService(nil, jobRepo, nil, defaultMockStageTemplateRepo, defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, nil, nil)
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

func TestJobHandler_Create(t *testing.T) {
	userID := "user-123"

	t.Run("creates job successfully", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, job *model.Job) error {
				job.ID = "job-1"
				job.CreatedAt = time.Now()
				job.UpdatedAt = time.Now()
				return nil
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs", mockAuthMiddleware(userID), handler.Create)

		body := `{"title":"Software Engineer"}`
		req, _ := http.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response model.JobDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Software Engineer", response.Title)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs", handler.Create) // No auth middleware

		body := `{"title":"Software Engineer"}`
		req, _ := http.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid request", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs", mockAuthMiddleware(userID), handler.Create)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for empty title", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs", mockAuthMiddleware(userID), handler.Create)

		body := `{"title":"   "}`
		req, _ := http.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestJobHandler_Get(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	t.Run("returns job successfully", func(t *testing.T) {
		expectedJob := &model.Job{
			ID:        jobID,
			UserID:    userID,
			Title:     "Software Engineer",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return expectedJob, nil
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.GET("/jobs/:id", mockAuthMiddleware(userID), handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/jobs/"+jobID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.JobDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, expectedJob.Title, response.Title)
	})

	t.Run("returns 404 when job not found", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.GET("/jobs/:id", mockAuthMiddleware(userID), handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/jobs/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestJobHandler_List(t *testing.T) {
	userID := "user-123"

	t.Run("returns jobs list", func(t *testing.T) {
		expectedJobs := []*model.JobDTO{
			{ID: "job-1", Title: "Software Engineer"},
			{ID: "job-2", Title: "Product Manager"},
		}

		mockRepo := &MockJobRepository{
			ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
				return expectedJobs, 2, nil
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.GET("/jobs", mockAuthMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/jobs?limit=20&offset=0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("parses sort parameter correctly", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
				assert.Equal(t, "created_at", opts.SortBy)
				assert.Equal(t, "desc", opts.SortDir)
				return []*model.JobDTO{}, 0, nil
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.GET("/jobs", mockAuthMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/jobs?sort=created_at:desc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("accepts the archived filter values", func(t *testing.T) {
		// "" and the legacy "active" mean exclude-archived; "archived" and "all"
		// are the new explicit values. All must be accepted by the handler and
		// passed through to the service unchanged.
		for _, rawStatus := range []string{"", "active", "archived", "all"} {
			rawStatus := rawStatus
			var seen string
			mockRepo := &MockJobRepository{
				ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
					seen = opts.Status
					return []*model.JobDTO{}, 0, nil
				},
			}
			svc := newTestJobService(mockRepo)
			handler := NewJobHandler(svc)

			router := setupTestRouter()
			router.GET("/jobs", mockAuthMiddleware(userID), handler.List)

			url := "/jobs"
			if rawStatus != "" {
				url += "?status=" + rawStatus
			}
			req, _ := http.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "status=%q", rawStatus)
			assert.Equal(t, rawStatus, seen, "handler must forward status=%q unchanged", rawStatus)
		}
	})

	t.Run("rejects an unknown status filter", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.GET("/jobs", mockAuthMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/jobs?status=bogus", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestJobHandler_Update(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	t.Run("updates job successfully", func(t *testing.T) {
		existingJob := &model.Job{
			ID:        jobID,
			UserID:    userID,
			Title:     "Old Title",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob, nil
			},
			UpdateFunc: func(ctx context.Context, job *model.Job) error {
				return nil
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.PATCH("/jobs/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"title":"New Title"}`
		req, _ := http.NewRequest(http.MethodPatch, "/jobs/"+jobID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("archives a job", func(t *testing.T) {
		existingJob := &model.Job{ID: jobID, UserID: userID, Title: "Job Title"}
		var saved *model.Job
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob, nil
			},
			UpdateFunc: func(ctx context.Context, job *model.Job) error {
				saved = job
				return nil
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.PATCH("/jobs/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"is_archived":true}`
		req, _ := http.NewRequest(http.MethodPatch, "/jobs/"+jobID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, saved)
		assert.True(t, saved.IsArchived)
	})

	t.Run("returns 404 when job not found", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.PATCH("/jobs/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"title":"New Title"}`
		req, _ := http.NewRequest(http.MethodPatch, "/jobs/nonexistent", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestJobHandler_Move(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"
	stageID := "stage-applied"

	t.Run("no-op move returns the enriched job", func(t *testing.T) {
		existingJob := &model.Job{
			ID:                     jobID,
			UserID:                 userID,
			Title:                  "Engineer",
			CurrentStageTemplateID: &stageID,
		}
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob, nil
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/:id/move", mockAuthMiddleware(userID), handler.Move)

		body := `{"stage_template_id":"` + stageID + `"}`
		req, _ := http.NewRequest(http.MethodPost, "/jobs/"+jobID+"/move", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 400 for a missing stage template id", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/:id/move", mockAuthMiddleware(userID), handler.Move)

		body := `{}`
		req, _ := http.NewRequest(http.MethodPost, "/jobs/"+jobID+"/move", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// binding:"required" on StageTemplateID rejects the empty payload.
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestJobHandler_ReorderStageTemplates(t *testing.T) {
	userID := "user-123"

	t.Run("reorders successfully", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.POST("/stage-templates/reorder", mockAuthMiddleware(userID), handler.ReorderStageTemplates)

		body := `{"stage_ids":["a","b","c"]}`
		req, _ := http.NewRequest(http.MethodPost, "/stage-templates/reorder", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 400 for an empty stage id list", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.POST("/stage-templates/reorder", mockAuthMiddleware(userID), handler.ReorderStageTemplates)

		body := `{"stage_ids":[]}`
		req, _ := http.NewRequest(http.MethodPost, "/stage-templates/reorder", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestJobHandler_Delete(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	t.Run("deletes job successfully", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			DeleteFunc: func(ctx context.Context, uid, jid string) error {
				return nil
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.DELETE("/jobs/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/jobs/"+jobID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 when job not found", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			DeleteFunc: func(ctx context.Context, uid, jid string) error {
				return model.ErrJobNotFound
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.DELETE("/jobs/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/jobs/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestJobHandler_ToggleFavorite(t *testing.T) {
	userID := "user-123"
	jobID := "job-456"

	t.Run("toggles favorite successfully", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ToggleFavoriteFunc: func(ctx context.Context, uid, jid string) (bool, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, jobID, jid)
				return true, nil
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/:id/favorite", mockAuthMiddleware(userID), handler.ToggleFavorite)

		req, _ := http.NewRequest(http.MethodPost, "/jobs/"+jobID+"/favorite", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"is_favorite":true`)
	})

	t.Run("returns 404 when job not found", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ToggleFavoriteFunc: func(ctx context.Context, uid, jid string) (bool, error) {
				return false, model.ErrJobNotFound
			},
		}

		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/:id/favorite", mockAuthMiddleware(userID), handler.ToggleFavorite)

		req, _ := http.NewRequest(http.MethodPost, "/jobs/nonexistent/favorite", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 401 without auth", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := newTestJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.POST("/jobs/:id/favorite", handler.ToggleFavorite)

		req, _ := http.NewRequest(http.MethodPost, "/jobs/"+jobID+"/favorite", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestSplitSort(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"created_at:desc", []string{"created_at", "desc"}},
		{"title:asc", []string{"title", "asc"}},
		{"company_name:desc", []string{"company_name", "desc"}},
		{"noseparator", []string{"noseparator"}},
		{"", []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitSort(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJobHandler_RegisterRoutes(t *testing.T) {
	mockRepo := &MockJobRepository{
		CreateFunc: func(ctx context.Context, job *model.Job) error {
			job.ID = "job-1"
			return nil
		},
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
			return &model.Job{ID: jid, Title: "Test"}, nil
		},
		ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
			return []*model.JobDTO{}, 0, nil
		},
		DeleteFunc: func(ctx context.Context, uid, jid string) error {
			return nil
		},
	}

	svc := newTestJobService(mockRepo)
	handler := NewJobHandler(svc)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"))

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/jobs"},
		{http.MethodGet, "/api/v1/jobs"},
		{http.MethodGet, "/api/v1/jobs/test-id"},
		{http.MethodPatch, "/api/v1/jobs/test-id"},
		{http.MethodDelete, "/api/v1/jobs/test-id"},
		{http.MethodPost, "/api/v1/jobs/test-id/favorite"},
		{http.MethodPost, "/api/v1/jobs/test-id/move"},
		{http.MethodPost, "/api/v1/stage-templates/reorder"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			var body *bytes.Buffer
			if route.method == http.MethodPost || route.method == http.MethodPatch {
				body = bytes.NewBufferString(`{"title":"Test"}`)
			} else {
				body = bytes.NewBuffer(nil)
			}
			req, _ := http.NewRequest(route.method, route.path, body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.NotEqual(t, http.StatusNotFound, w.Code, "Route %s %s should be registered", route.method, route.path)
		})
	}
}
