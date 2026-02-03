package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/andreypavlenko/jobber/modules/resumes/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockResumeRepository implements ports.ResumeRepository
type MockResumeRepository struct {
	CreateFunc  func(ctx context.Context, resume *model.Resume) error
	GetByIDFunc func(ctx context.Context, userID, resumeID string) (*model.Resume, error)
	ListFunc    func(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error)
	UpdateFunc  func(ctx context.Context, resume *model.Resume) error
	DeleteFunc  func(ctx context.Context, userID, resumeID string) error
}

func (m *MockResumeRepository) Create(ctx context.Context, resume *model.Resume) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, resume)
	}
	return nil
}

func (m *MockResumeRepository) GetByID(ctx context.Context, userID, resumeID string) (*model.Resume, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, resumeID)
	}
	return nil, nil
}

func (m *MockResumeRepository) List(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, limit, offset, sortBy, sortDir)
	}
	return nil, 0, nil
}

func (m *MockResumeRepository) Update(ctx context.Context, resume *model.Resume) error {
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

func TestResumeHandler_Create(t *testing.T) {
	userID := "user-123"

	t.Run("creates resume successfully", func(t *testing.T) {
		mockRepo := &MockResumeRepository{
			CreateFunc: func(ctx context.Context, resume *model.Resume) error {
				resume.ID = "resume-1"
				resume.StorageType = model.StorageTypeExternal
				resume.IsActive = true
				resume.CreatedAt = time.Now()
				resume.UpdatedAt = time.Now()
				return nil
			},
		}

		svc := service.NewResumeService(mockRepo, nil)
		handler := NewResumeHandler(svc)

		router := setupTestRouter()
		router.POST("/resumes", mockAuthMiddleware(userID), handler.Create)

		body := `{"title":"Software Engineer Resume"}`
		req, _ := http.NewRequest(http.MethodPost, "/resumes", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response model.ResumeDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "Software Engineer Resume", response.Title)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockResumeRepository{}
		svc := service.NewResumeService(mockRepo, nil)
		handler := NewResumeHandler(svc)

		router := setupTestRouter()
		router.POST("/resumes", handler.Create) // No auth middleware

		body := `{"title":"Resume"}`
		req, _ := http.NewRequest(http.MethodPost, "/resumes", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid request", func(t *testing.T) {
		mockRepo := &MockResumeRepository{}
		svc := service.NewResumeService(mockRepo, nil)
		handler := NewResumeHandler(svc)

		router := setupTestRouter()
		router.POST("/resumes", mockAuthMiddleware(userID), handler.Create)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPost, "/resumes", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestResumeHandler_Get(t *testing.T) {
	userID := "user-123"
	resumeID := "resume-1"

	t.Run("returns resume successfully", func(t *testing.T) {
		expectedResume := &model.Resume{
			ID:          resumeID,
			UserID:      userID,
			Title:       "Software Engineer Resume",
			StorageType: model.StorageTypeExternal,
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return expectedResume, nil
			},
		}

		svc := service.NewResumeService(mockRepo, nil)
		handler := NewResumeHandler(svc)

		router := setupTestRouter()
		router.GET("/resumes/:id", mockAuthMiddleware(userID), handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/resumes/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.ResumeDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, expectedResume.Title, response.Title)
	})

	t.Run("returns 404 when resume not found", func(t *testing.T) {
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
		}

		svc := service.NewResumeService(mockRepo, nil)
		handler := NewResumeHandler(svc)

		router := setupTestRouter()
		router.GET("/resumes/:id", mockAuthMiddleware(userID), handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/resumes/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestResumeHandler_List(t *testing.T) {
	userID := "user-123"

	t.Run("returns resumes list", func(t *testing.T) {
		expectedResumes := []*ports.ResumeWithCount{
			{Resume: &model.Resume{ID: "resume-1", Title: "Resume A", StorageType: model.StorageTypeExternal}, ApplicationsCount: 5},
			{Resume: &model.Resume{ID: "resume-2", Title: "Resume B", StorageType: model.StorageTypeExternal}, ApplicationsCount: 3},
		}

		mockRepo := &MockResumeRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error) {
				return expectedResumes, 2, nil
			},
		}

		svc := service.NewResumeService(mockRepo, nil)
		handler := NewResumeHandler(svc)

		router := setupTestRouter()
		router.GET("/resumes", mockAuthMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/resumes?limit=20&offset=0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestResumeHandler_Update(t *testing.T) {
	userID := "user-123"
	resumeID := "resume-1"

	t.Run("updates resume successfully", func(t *testing.T) {
		existingResume := &model.Resume{
			ID:          resumeID,
			UserID:      userID,
			Title:       "Old Title",
			StorageType: model.StorageTypeExternal,
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume, nil
			},
			UpdateFunc: func(ctx context.Context, resume *model.Resume) error {
				return nil
			},
		}

		svc := service.NewResumeService(mockRepo, nil)
		handler := NewResumeHandler(svc)

		router := setupTestRouter()
		router.PATCH("/resumes/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"title":"New Title"}`
		req, _ := http.NewRequest(http.MethodPatch, "/resumes/"+resumeID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 when resume not found", func(t *testing.T) {
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
		}

		svc := service.NewResumeService(mockRepo, nil)
		handler := NewResumeHandler(svc)

		router := setupTestRouter()
		router.PATCH("/resumes/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"title":"New Title"}`
		req, _ := http.NewRequest(http.MethodPatch, "/resumes/nonexistent", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestResumeHandler_Delete(t *testing.T) {
	userID := "user-123"
	resumeID := "resume-1"

	t.Run("deletes resume successfully", func(t *testing.T) {
		existingResume := &model.Resume{
			ID:          resumeID,
			UserID:      userID,
			Title:       "Resume",
			StorageType: model.StorageTypeExternal,
		}

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume, nil
			},
			DeleteFunc: func(ctx context.Context, uid, rid string) error {
				return nil
			},
		}

		svc := service.NewResumeService(mockRepo, nil)
		handler := NewResumeHandler(svc)

		router := setupTestRouter()
		router.DELETE("/resumes/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/resumes/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 when resume not found", func(t *testing.T) {
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
		}

		svc := service.NewResumeService(mockRepo, nil)
		handler := NewResumeHandler(svc)

		router := setupTestRouter()
		router.DELETE("/resumes/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/resumes/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 400 when resume is in use", func(t *testing.T) {
		existingResume := &model.Resume{
			ID:          resumeID,
			UserID:      userID,
			Title:       "Resume",
			StorageType: model.StorageTypeExternal,
		}

		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
				return existingResume, nil
			},
			DeleteFunc: func(ctx context.Context, uid, rid string) error {
				return model.ErrResumeInUse
			},
		}

		svc := service.NewResumeService(mockRepo, nil)
		handler := NewResumeHandler(svc)

		router := setupTestRouter()
		router.DELETE("/resumes/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/resumes/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestResumeHandler_RegisterRoutes(t *testing.T) {
	mockRepo := &MockResumeRepository{
		CreateFunc: func(ctx context.Context, resume *model.Resume) error {
			resume.ID = "resume-1"
			resume.StorageType = model.StorageTypeExternal
			return nil
		},
		GetByIDFunc: func(ctx context.Context, uid, rid string) (*model.Resume, error) {
			return &model.Resume{ID: rid, Title: "Test", StorageType: model.StorageTypeExternal}, nil
		},
		ListFunc: func(ctx context.Context, uid string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error) {
			return []*ports.ResumeWithCount{}, 0, nil
		},
		DeleteFunc: func(ctx context.Context, uid, rid string) error {
			return nil
		},
	}

	svc := service.NewResumeService(mockRepo, nil)
	handler := NewResumeHandler(svc)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"))

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/resumes"},
		{http.MethodGet, "/api/v1/resumes"},
		{http.MethodGet, "/api/v1/resumes/test-id"},
		{http.MethodPatch, "/api/v1/resumes/test-id"},
		{http.MethodDelete, "/api/v1/resumes/test-id"},
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
