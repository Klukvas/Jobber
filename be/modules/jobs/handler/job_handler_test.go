package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockJobRepository implements ports.JobRepository
type MockJobRepository struct {
	CreateFunc  func(ctx context.Context, job *model.Job) error
	GetByIDFunc func(ctx context.Context, userID, jobID string) (*model.Job, error)
	ListFunc    func(ctx context.Context, userID string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error)
	UpdateFunc  func(ctx context.Context, job *model.Job) error
	DeleteFunc  func(ctx context.Context, userID, jobID string) error
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

func (m *MockJobRepository) List(ctx context.Context, userID string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, limit, offset, status, sortBy, sortOrder)
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
				job.Status = "active"
				job.CreatedAt = time.Now()
				job.UpdatedAt = time.Now()
				return nil
			},
		}

		svc := service.NewJobService(mockRepo)
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
		svc := service.NewJobService(mockRepo)
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
		svc := service.NewJobService(mockRepo)
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
		svc := service.NewJobService(mockRepo)
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
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return expectedJob, nil
			},
		}

		svc := service.NewJobService(mockRepo)
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

		svc := service.NewJobService(mockRepo)
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
			ListFunc: func(ctx context.Context, uid string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error) {
				return expectedJobs, 2, nil
			},
		}

		svc := service.NewJobService(mockRepo)
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
			ListFunc: func(ctx context.Context, uid string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error) {
				assert.Equal(t, "created_at", sortBy)
				assert.Equal(t, "desc", sortOrder)
				return []*model.JobDTO{}, 0, nil
			},
		}

		svc := service.NewJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.GET("/jobs", mockAuthMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/jobs?sort=created_at:desc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
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
			Status:    "active",
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

		svc := service.NewJobService(mockRepo)
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

	t.Run("returns 404 when job not found", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}

		svc := service.NewJobService(mockRepo)
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

	t.Run("returns 400 for invalid status", func(t *testing.T) {
		existingJob := &model.Job{
			ID:     jobID,
			UserID: userID,
			Title:  "Job Title",
			Status: "active",
		}

		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob, nil
			},
		}

		svc := service.NewJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.PATCH("/jobs/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"status":"invalid"}`
		req, _ := http.NewRequest(http.MethodPatch, "/jobs/"+jobID, bytes.NewBufferString(body))
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

		svc := service.NewJobService(mockRepo)
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

		svc := service.NewJobService(mockRepo)
		handler := NewJobHandler(svc)

		router := setupTestRouter()
		router.DELETE("/jobs/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/jobs/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
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
			return &model.Job{ID: jid, Title: "Test", Status: "active"}, nil
		},
		ListFunc: func(ctx context.Context, uid string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error) {
			return []*model.JobDTO{}, 0, nil
		},
		DeleteFunc: func(ctx context.Context, uid, jid string) error {
			return nil
		},
	}

	svc := service.NewJobService(mockRepo)
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
