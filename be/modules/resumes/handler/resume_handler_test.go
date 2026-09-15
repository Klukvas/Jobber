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

	"github.com/andreypavlenko/jobber/internal/platform/storage"
	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/andreypavlenko/jobber/modules/resumes/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockResumeRepository implements ports.ResumeRepository
type MockResumeRepository struct {
	CreateFinalizedUploadFunc func(ctx context.Context, resume *model.Resume, maxResumes int) error
	CreateFunc                func(ctx context.Context, resume *model.Resume) error
	GetByIDFunc               func(ctx context.Context, userID, resumeID string) (*model.Resume, error)
	ListFunc                  func(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*ports.ResumeWithCount, int, error)
	UpdateFunc                func(ctx context.Context, resume *model.Resume) error
	DeleteFunc                func(ctx context.Context, userID, resumeID string) error
}

func (m *MockResumeRepository) CreateFinalizedUpload(ctx context.Context, resume *model.Resume, maxResumes int) error {
	if m.CreateFinalizedUploadFunc != nil {
		return m.CreateFinalizedUploadFunc(ctx, resume, maxResumes)
	}
	return m.Create(ctx, resume)
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

		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

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
		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

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
		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

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

		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

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

		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

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

		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

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

		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

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

		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

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

		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

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

		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

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

		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

		router := setupTestRouter()
		router.DELETE("/resumes/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/resumes/"+resumeID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// --- Mock limit checker ---

type MockLimitChecker struct {
	CheckLimitFunc    func(ctx context.Context, userID, resource string) error
	ResourceLimitFunc func(ctx context.Context, userID, resource string) (int, error)
}

func (m *MockLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, userID, resource)
	}
	return nil
}

func (m *MockLimitChecker) ResourceLimit(ctx context.Context, userID, resource string) (int, error) {
	if m.ResourceLimitFunc != nil {
		return m.ResourceLimitFunc(ctx, userID, resource)
	}
	return -1, nil
}

// --- Create: plan limit and service error ---

func TestResumeHandler_Create_PlanLimitReached(t *testing.T) {
	userID := "user-123"
	limiter := &MockLimitChecker{
		CheckLimitFunc: func(_ context.Context, _, _ string) error {
			return subModel.ErrLimitReached
		},
	}

	svc := service.NewResumeService(&MockResumeRepository{}, nil, limiter, nil)
	handler := NewResumeHandler(svc, zap.NewNop())

	router := setupTestRouter()
	router.POST("/resumes", mockAuthMiddleware(userID), handler.Create)

	body := `{"title":"Resume"}`
	req, _ := http.NewRequest(http.MethodPost, "/resumes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestResumeHandler_Create_ServiceError(t *testing.T) {
	userID := "user-123"
	mockRepo := &MockResumeRepository{
		CreateFunc: func(_ context.Context, _ *model.Resume) error {
			return errors.New("db error")
		},
	}

	svc := service.NewResumeService(mockRepo, nil, nil, nil)
	handler := NewResumeHandler(svc, zap.NewNop())

	router := setupTestRouter()
	router.POST("/resumes", mockAuthMiddleware(userID), handler.Create)

	body := `{"title":"Resume"}`
	req, _ := http.NewRequest(http.MethodPost, "/resumes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- Get: 401 ---

func TestResumeHandler_Get_Unauthorized(t *testing.T) {
	svc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
	handler := NewResumeHandler(svc, zap.NewNop())

	router := setupTestRouter()
	router.GET("/resumes/:id", handler.Get)

	req, _ := http.NewRequest(http.MethodGet, "/resumes/resume-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Get: internal error ---

func TestResumeHandler_Get_InternalError(t *testing.T) {
	userID := "user-123"
	mockRepo := &MockResumeRepository{
		GetByIDFunc: func(_ context.Context, _, _ string) (*model.Resume, error) {
			return nil, errors.New("db error")
		},
	}

	svc := service.NewResumeService(mockRepo, nil, nil, nil)
	handler := NewResumeHandler(svc, zap.NewNop())

	router := setupTestRouter()
	router.GET("/resumes/:id", mockAuthMiddleware(userID), handler.Get)

	req, _ := http.NewRequest(http.MethodGet, "/resumes/resume-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// --- List: 401, error, bad pagination ---

func TestResumeHandler_List_Unauthorized(t *testing.T) {
	svc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
	handler := NewResumeHandler(svc, zap.NewNop())

	router := setupTestRouter()
	router.GET("/resumes", handler.List)

	req, _ := http.NewRequest(http.MethodGet, "/resumes", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestResumeHandler_List_ServiceError(t *testing.T) {
	userID := "user-123"
	mockRepo := &MockResumeRepository{
		ListFunc: func(_ context.Context, _ string, _, _ int, _, _ string) ([]*ports.ResumeWithCount, int, error) {
			return nil, 0, errors.New("db error")
		},
	}

	svc := service.NewResumeService(mockRepo, nil, nil, nil)
	handler := NewResumeHandler(svc, zap.NewNop())

	router := setupTestRouter()
	router.GET("/resumes", mockAuthMiddleware(userID), handler.List)

	req, _ := http.NewRequest(http.MethodGet, "/resumes?limit=20&offset=0", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestResumeHandler_List_InvalidPagination(t *testing.T) {
	userID := "user-123"

	svc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
	handler := NewResumeHandler(svc, zap.NewNop())

	router := setupTestRouter()
	router.GET("/resumes", mockAuthMiddleware(userID), handler.List)

	req, _ := http.NewRequest(http.MethodGet, "/resumes?limit=abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Update: 401, bad JSON ---

func TestResumeHandler_Update_Unauthorized(t *testing.T) {
	svc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
	handler := NewResumeHandler(svc, zap.NewNop())

	router := setupTestRouter()
	router.PATCH("/resumes/:id", handler.Update)

	body := `{"title":"New"}`
	req, _ := http.NewRequest(http.MethodPatch, "/resumes/resume-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestResumeHandler_Update_InvalidJSON(t *testing.T) {
	userID := "user-123"

	svc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
	handler := NewResumeHandler(svc, zap.NewNop())

	router := setupTestRouter()
	router.PATCH("/resumes/:id", mockAuthMiddleware(userID), handler.Update)

	req, _ := http.NewRequest(http.MethodPatch, "/resumes/resume-1", bytes.NewBufferString("bad"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Delete: 401 ---

func TestResumeHandler_Delete_Unauthorized(t *testing.T) {
	svc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
	handler := NewResumeHandler(svc, zap.NewNop())

	router := setupTestRouter()
	router.DELETE("/resumes/:id", handler.Delete)

	req, _ := http.NewRequest(http.MethodDelete, "/resumes/resume-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- GenerateUploadURL tests ---

func TestResumeHandler_GenerateUploadURL(t *testing.T) {
	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		svc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

		router := setupTestRouter()
		router.POST("/resumes/upload-url", handler.GenerateUploadURL)

		body := `{"filename":"resume.pdf","content_type":"application/pdf"}`
		req, _ := http.NewRequest(http.MethodPost, "/resumes/upload-url", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid request", func(t *testing.T) {
		userID := "user-123"
		svc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

		router := setupTestRouter()
		router.POST("/resumes/upload-url", mockAuthMiddleware(userID), handler.GenerateUploadURL)

		req, _ := http.NewRequest(http.MethodPost, "/resumes/upload-url", bytes.NewBufferString("bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 403 when plan limit reached", func(t *testing.T) {
		userID := "user-123"
		limiter := &MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		svc := service.NewResumeService(&MockResumeRepository{}, nil, limiter, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

		router := setupTestRouter()
		router.POST("/resumes/upload-url", mockAuthMiddleware(userID), handler.GenerateUploadURL)

		body := `{"filename":"resume.pdf","content_type":"application/pdf"}`
		req, _ := http.NewRequest(http.MethodPost, "/resumes/upload-url", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("returns 500 when S3 client is nil", func(t *testing.T) {
		userID := "user-123"
		svc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

		router := setupTestRouter()
		router.POST("/resumes/upload-url", mockAuthMiddleware(userID), handler.GenerateUploadURL)

		body := `{"filename":"resume.pdf","content_type":"application/pdf"}`
		req, _ := http.NewRequest(http.MethodPost, "/resumes/upload-url", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// --- DownloadResume tests ---

func TestResumeHandler_DownloadResume(t *testing.T) {
	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		svc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

		router := setupTestRouter()
		router.GET("/resumes/:id/download", handler.DownloadResume)

		req, _ := http.NewRequest(http.MethodGet, "/resumes/resume-1/download", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 500 when S3 not configured", func(t *testing.T) {
		userID := "user-123"

		svc := service.NewResumeService(&MockResumeRepository{}, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

		router := setupTestRouter()
		router.GET("/resumes/:id/download", mockAuthMiddleware(userID), handler.DownloadResume)

		req, _ := http.NewRequest(http.MethodGet, "/resumes/nonexistent/download", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("returns 500 when S3 client is nil", func(t *testing.T) {
		userID := "user-123"
		mockRepo := &MockResumeRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Resume, error) {
				return &model.Resume{
					ID:          "resume-1",
					UserID:      userID,
					Title:       "Resume",
					StorageType: model.StorageTypeS3,
					FileURL:     strPtr("s3://bucket/file.pdf"),
				}, nil
			},
		}

		svc := service.NewResumeService(mockRepo, nil, nil, nil)
		handler := NewResumeHandler(svc, zap.NewNop())

		router := setupTestRouter()
		router.GET("/resumes/:id/download", mockAuthMiddleware(userID), handler.DownloadResume)

		req, _ := http.NewRequest(http.MethodGet, "/resumes/resume-1/download", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func strPtr(s string) *string { return &s }

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

	svc := service.NewResumeService(mockRepo, nil, nil, nil)
	handler := NewResumeHandler(svc, zap.NewNop())

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

// The finalize endpoint has to answer with the right *kind* of failure: a file
// the customer never uploaded is a 400 they can act on, while a storage read
// that fell over is a 500 and must not be dressed up as their mistake.
func TestResumeHandler_FinalizeUpload(t *testing.T) {
	const (
		userID   = "user-123"
		resumeID = "3f2a6c1e-0000-4000-8000-00000000abcd"
		key      = "users/user-123/resumes/3f2a6c1e-0000-4000-8000-00000000abcd.pdf"
	)

	newRouter := func(objects map[string][]byte, repo *MockResumeRepository, stopServer bool) (*gin.Engine, func()) {
		s3Client, cleanup := storage.NewTestS3Client(objects)
		if stopServer {
			cleanup()
			cleanup = func() {}
		}
		handler := NewResumeHandler(service.NewResumeService(repo, s3Client, nil, nil), zap.NewNop())
		router := setupTestRouter()
		router.POST("/resumes/:id/finalize", mockAuthMiddleware(userID), handler.FinalizeUpload)
		return router, cleanup
	}

	post := func(router *gin.Engine, id, body string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest(http.MethodPost, "/resumes/"+id+"/finalize", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	pdf := func() []byte {
		out := make([]byte, 2048)
		copy(out, "%PDF-1.7\n")
		return out
	}

	t.Run("returns 200 for a verified upload", func(t *testing.T) {
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
		}
		router, cleanup := newRouter(map[string][]byte{key: pdf()}, repo, false)
		defer cleanup()

		w := post(router, resumeID, `{"title":"Backend Engineer"}`)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	// Content-Length is -1 on any chunked request — a proxy that re-frames the
	// upload, a client that streams instead of buffering. Gating the decode on
	// "> 0" threw that body away: the title the customer typed was discarded in
	// silence and the resume kept its placeholder name.
	t.Run("reads the title from a request with no declared length", func(t *testing.T) {
		var created *model.Resume
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFunc: func(_ context.Context, r *model.Resume) error {
				created = r
				return nil
			},
		}
		router, cleanup := newRouter(map[string][]byte{key: pdf()}, repo, false)
		defer cleanup()

		req, _ := http.NewRequest(http.MethodPost, "/resumes/"+resumeID+"/finalize",
			bytes.NewBufferString(`{"title":"Backend Engineer"}`))
		req.Header.Set("Content-Type", "application/json")
		// What net/http reports for Transfer-Encoding: chunked.
		req.ContentLength = -1
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, created)
		assert.Equal(t, "Backend Engineer", created.Title)
	})

	// The body stays optional: an upload with no title keeps the placeholder.
	t.Run("accepts a finalize with no body at all", func(t *testing.T) {
		var created *model.Resume
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFunc: func(_ context.Context, r *model.Resume) error {
				created = r
				return nil
			},
		}
		router, cleanup := newRouter(map[string][]byte{key: pdf()}, repo, false)
		defer cleanup()

		w := post(router, resumeID, "")

		require.Equal(t, http.StatusOK, w.Code)
		require.NotNil(t, created)
		assert.NotEmpty(t, created.Title)
	})

	t.Run("still rejects a body that is not JSON", func(t *testing.T) {
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFunc: func(context.Context, *model.Resume) error {
				t.Fatal("a malformed request must not create a resume")
				return nil
			},
		}
		router, cleanup := newRouter(map[string][]byte{key: pdf()}, repo, false)
		defer cleanup()

		w := post(router, resumeID, `{"title":`)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 when the upload never landed", func(t *testing.T) {
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
		}
		router, cleanup := newRouter(map[string][]byte{}, repo, false)
		defer cleanup()

		w := post(router, resumeID, `{}`)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var body struct {
			ErrorCode string `json:"error_code"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, string(model.CodeResumeFileMissing), body.ErrorCode)
	})

	// An empty upload reaches storage as a real 416, not a 404. Before the range
	// classifier existed this surfaced as a 500; it is the customer's problem
	// and has to read as one.
	t.Run("returns 400 RESUME_FILE_MISSING for a present but empty object", func(t *testing.T) {
		created := false
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFunc: func(context.Context, *model.Resume) error {
				created = true
				return nil
			},
		}
		router, cleanup := newRouter(map[string][]byte{key: {}}, repo, false)
		defer cleanup()

		w := post(router, resumeID, `{}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)

		var body struct {
			ErrorCode    string `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, string(model.CodeResumeFileMissing), body.ErrorCode)
		assert.NotEmpty(t, body.ErrorMessage)
		assert.False(t, created, "an empty upload must not become a resume")
	})

	t.Run("an empty object belonging to another user is still unreachable", func(t *testing.T) {
		created := false
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFunc: func(context.Context, *model.Resume) error {
				created = true
				return nil
			},
		}
		// The empty object sits under a different user's prefix, so the key this
		// request derives names nothing at all.
		objects := map[string][]byte{"users/someone-else/resumes/" + resumeID + ".pdf": {}}
		router, cleanup := newRouter(objects, repo, false)
		defer cleanup()

		w := post(router, resumeID, `{}`)

		require.Equal(t, http.StatusBadRequest, w.Code)
		assert.False(t, created)
		assert.Contains(t, objects, "users/someone-else/resumes/"+resumeID+".pdf",
			"another user's object must be untouched")
	})

	t.Run("returns 500 when the store cannot be read", func(t *testing.T) {
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
		}
		router, cleanup := newRouter(map[string][]byte{}, repo, true)
		defer cleanup()

		w := post(router, resumeID, `{}`)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.NotContains(t, w.Body.String(), string(model.CodeResumeFileMissing))
	})

	// Presigning no longer reserves a slot, so finalization is where the plan
	// limit is actually enforced — and it has to answer with the same 403 the
	// upload-URL endpoint gives, not a 500. subModel.ErrLimitReached is a
	// subscriptions error, so the resume error mapping does not know it.
	t.Run("returns 403 PLAN_LIMIT_REACHED when the plan is full", func(t *testing.T) {
		created := false
		// The write is where the limit is enforced — it counts and inserts in
		// one transaction, which is the only place a *concurrent* finalize can
		// be caught. Here it reports the customer's allowance as already full.
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return nil, model.ErrResumeNotFound
			},
			CreateFinalizedUploadFunc: func(_ context.Context, _ *model.Resume, maxResumes int) error {
				assert.Equal(t, 0, maxResumes, "the plan's ceiling reaches the write")
				return model.ErrResumeLimitReached
			},
			CreateFunc: func(context.Context, *model.Resume) error {
				created = true
				return nil
			},
		}
		limiter := &MockLimitChecker{
			ResourceLimitFunc: func(_ context.Context, _, resource string) (int, error) {
				assert.Equal(t, "resumes", resource)
				return 0, nil
			},
		}

		s3Client, cleanup := storage.NewTestS3Client(map[string][]byte{key: pdf()})
		defer cleanup()
		handler := NewResumeHandler(service.NewResumeService(repo, s3Client, limiter, nil), zap.NewNop())
		router := setupTestRouter()
		router.POST("/resumes/:id/finalize", mockAuthMiddleware(userID), handler.FinalizeUpload)

		w := post(router, resumeID, `{}`)

		require.Equal(t, http.StatusForbidden, w.Code)
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)

		var body struct {
			ErrorCode    string `json:"error_code"`
			ErrorMessage string `json:"error_message"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, "PLAN_LIMIT_REACHED", body.ErrorCode)
		assert.NotEmpty(t, body.ErrorMessage)
		assert.False(t, created, "a refused upload must not become a resume")
	})

	t.Run("returns 200 again when the same upload is finalized twice", func(t *testing.T) {
		storageKey := key
		repo := &MockResumeRepository{
			GetByIDFunc: func(context.Context, string, string) (*model.Resume, error) {
				return &model.Resume{
					ID: resumeID, UserID: userID, Title: "Backend Engineer",
					StorageType: model.StorageTypeS3, StorageKey: &storageKey, IsActive: true,
				}, nil
			},
			CreateFunc: func(context.Context, *model.Resume) error {
				t.Fatal("a repeated finalize must not insert again")
				return nil
			},
		}
		objects := map[string][]byte{key: pdf()}
		router, cleanup := newRouter(objects, repo, false)
		defer cleanup()

		w := post(router, resumeID, `{}`)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "Backend Engineer")
		assert.Contains(t, objects, key, "the live object must survive")
	})
}
