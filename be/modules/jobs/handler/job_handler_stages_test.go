package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	httpPlatform "github.com/andreypavlenko/jobber/internal/platform/http"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertAnErr is a generic non-sentinel error used to exercise the default
// (INTERNAL_ERROR / 500) branch of the handler's error mapping.
var assertAnErr = errors.New("boom")

// ---- Configurable stage-template repo ----------------------------------------

// ConfigStageTemplateRepo is a fully configurable ports.StageTemplateRepository.
// Unset funcs fall back to sensible defaults so tests set only what they need.
type ConfigStageTemplateRepo struct {
	CreateFunc  func(ctx context.Context, template *model.StageTemplate) error
	GetByIDFunc func(ctx context.Context, userID, templateID string) (*model.StageTemplate, error)
	ListFunc    func(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error)
	UpdateFunc  func(ctx context.Context, template *model.StageTemplate) error
	ReorderFunc func(ctx context.Context, userID string, orderedIDs []string) error
	DeleteFunc  func(ctx context.Context, userID, templateID string) error
}

func (m *ConfigStageTemplateRepo) Create(ctx context.Context, t *model.StageTemplate) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, t)
	}
	return nil
}
func (m *ConfigStageTemplateRepo) GetByID(ctx context.Context, userID, templateID string) (*model.StageTemplate, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, templateID)
	}
	return &model.StageTemplate{ID: templateID, UserID: userID, Name: "Applied", Order: 1}, nil
}
func (m *ConfigStageTemplateRepo) List(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, limit, offset)
	}
	return []*model.StageTemplate{{ID: "stage-wishlist", UserID: userID, Name: "Wishlist", Order: 0}}, 1, nil
}
func (m *ConfigStageTemplateRepo) Update(ctx context.Context, t *model.StageTemplate) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, t)
	}
	return nil
}
func (m *ConfigStageTemplateRepo) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	if m.ReorderFunc != nil {
		return m.ReorderFunc(ctx, userID, orderedIDs)
	}
	return nil
}
func (m *ConfigStageTemplateRepo) Delete(ctx context.Context, userID, templateID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, templateID)
	}
	return nil
}

// ---- Job-stage repo ----------------------------------------------------------

type MockJobStageRepository struct {
	CreateFunc    func(ctx context.Context, stage *model.JobStage) error
	GetByIDFunc   func(ctx context.Context, stageID, jobID string) (*model.JobStage, error)
	ListByJobFunc func(ctx context.Context, jobID string) ([]*model.JobStage, error)
	UpdateFunc    func(ctx context.Context, stage *model.JobStage) error
	DeleteFunc    func(ctx context.Context, stageID string) error
}

func (m *MockJobStageRepository) Create(ctx context.Context, stage *model.JobStage) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, stage)
	}
	return nil
}
func (m *MockJobStageRepository) GetByID(ctx context.Context, stageID, jobID string) (*model.JobStage, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, stageID, jobID)
	}
	return &model.JobStage{ID: stageID, JobID: jobID}, nil
}
func (m *MockJobStageRepository) ListByJob(ctx context.Context, jobID string) ([]*model.JobStage, error) {
	if m.ListByJobFunc != nil {
		return m.ListByJobFunc(ctx, jobID)
	}
	return nil, nil
}
func (m *MockJobStageRepository) Update(ctx context.Context, stage *model.JobStage) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, stage)
	}
	return nil
}
func (m *MockJobStageRepository) Delete(ctx context.Context, stageID, jobID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, stageID)
	}
	return nil
}

// newStageJobService wires a JobService with configurable stage + template repos
// (and a nil pool — only the pool-free code paths are exercised here).
func newStageJobService(jobRepo *MockJobRepository, stageRepo *MockJobStageRepository, tmplRepo *ConfigStageTemplateRepo) *service.JobService {
	return service.NewJobService(nil, jobRepo, stageRepo, tmplRepo, defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, nil, nil)
}

func decodeErr(t *testing.T, w *httptest.ResponseRecorder) httpPlatform.ErrorResponse {
	t.Helper()
	var e httpPlatform.ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &e))
	return e
}

// ---- AddStage ----------------------------------------------------------------

func TestJobHandler_AddStage(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	t.Run("401 without auth", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.POST("/jobs/:id/stages", h.AddStage)

		req, _ := http.NewRequest(http.MethodPost, "/jobs/"+jobID+"/stages", bytes.NewBufferString(`{"stage_template_id":"s1"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("400 for malformed body", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.POST("/jobs/:id/stages", mockAuthMiddleware(userID), h.AddStage)

		req, _ := http.NewRequest(http.MethodPost, "/jobs/"+jobID+"/stages", bytes.NewBufferString(`not json`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "VALIDATION_ERROR", decodeErr(t, w).ErrorCode)
	})

	t.Run("400 for missing stage_template_id", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.POST("/jobs/:id/stages", mockAuthMiddleware(userID), h.AddStage)

		req, _ := http.NewRequest(http.MethodPost, "/jobs/"+jobID+"/stages", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("404 when job not found (service error mapped before pool use)", func(t *testing.T) {
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}
		svc := newStageJobService(jobRepo, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.POST("/jobs/:id/stages", mockAuthMiddleware(userID), h.AddStage)

		req, _ := http.NewRequest(http.MethodPost, "/jobs/missing/stages", bytes.NewBufferString(`{"stage_template_id":"s1"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, string(model.CodeJobNotFound), decodeErr(t, w).ErrorCode)
	})

	t.Run("404 when stage template not found", func(t *testing.T) {
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return &model.Job{ID: jid, UserID: uid}, nil
			},
		}
		tmpl := &ConfigStageTemplateRepo{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return nil, model.ErrStageTemplateNotFound
			},
		}
		svc := newStageJobService(jobRepo, &MockJobStageRepository{}, tmpl)
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.POST("/jobs/:id/stages", mockAuthMiddleware(userID), h.AddStage)

		req, _ := http.NewRequest(http.MethodPost, "/jobs/"+jobID+"/stages", bytes.NewBufferString(`{"stage_template_id":"s1"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, string(model.CodeStageTemplateNotFound), decodeErr(t, w).ErrorCode)
	})
}

// ---- ListStages --------------------------------------------------------------

func TestJobHandler_ListStages(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	t.Run("401 without auth", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.GET("/jobs/:id/stages", h.ListStages)
		req, _ := http.NewRequest(http.MethodGet, "/jobs/"+jobID+"/stages", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns enriched stage list", func(t *testing.T) {
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return &model.Job{ID: jid, UserID: uid}, nil
			},
		}
		stageRepo := &MockJobStageRepository{
			ListByJobFunc: func(ctx context.Context, jid string) ([]*model.JobStage, error) {
				return []*model.JobStage{
					{ID: "st-1", JobID: jid, StageTemplateID: "stage-wishlist", Status: "completed", Order: 0},
					{ID: "st-2", JobID: jid, StageTemplateID: "stage-applied", Status: "active", Order: 1},
				}, nil
			},
		}
		tmpl := &ConfigStageTemplateRepo{
			ListFunc: func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
				return []*model.StageTemplate{
					{ID: "stage-wishlist", UserID: uid, Name: "Wishlist", Order: 0},
					{ID: "stage-applied", UserID: uid, Name: "Applied", Order: 1},
				}, 2, nil
			},
		}
		svc := newStageJobService(jobRepo, stageRepo, tmpl)
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.GET("/jobs/:id/stages", mockAuthMiddleware(userID), h.ListStages)

		req, _ := http.NewRequest(http.MethodGet, "/jobs/"+jobID+"/stages", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var stages []model.JobStageDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &stages))
		require.Len(t, stages, 2)
		assert.Equal(t, "Wishlist", stages[0].StageName)
		assert.Equal(t, "Applied", stages[1].StageName)
		assert.Equal(t, "active", stages[1].Status)
	})

	t.Run("404 when job not found", func(t *testing.T) {
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}
		svc := newStageJobService(jobRepo, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.GET("/jobs/:id/stages", mockAuthMiddleware(userID), h.ListStages)

		req, _ := http.NewRequest(http.MethodGet, "/jobs/missing/stages", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// ---- UpdateStage -------------------------------------------------------------

func TestJobHandler_UpdateStage(t *testing.T) {
	userID := "user-123"

	t.Run("401 without auth", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.PATCH("/jobs/:id/stages/:stageId", h.UpdateStage)
		req, _ := http.NewRequest(http.MethodPatch, "/jobs/j1/stages/s1", bytes.NewBufferString(`{"status":"completed"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("400 for malformed body", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.PATCH("/jobs/:id/stages/:stageId", mockAuthMiddleware(userID), h.UpdateStage)
		req, _ := http.NewRequest(http.MethodPatch, "/jobs/j1/stages/s1", bytes.NewBufferString(`{"status":`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("400 for invalid status enum", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.PATCH("/jobs/:id/stages/:stageId", mockAuthMiddleware(userID), h.UpdateStage)
		req, _ := http.NewRequest(http.MethodPatch, "/jobs/j1/stages/s1", bytes.NewBufferString(`{"status":"bogus"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("404 when job not found (before pool use)", func(t *testing.T) {
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}
		svc := newStageJobService(jobRepo, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.PATCH("/jobs/:id/stages/:stageId", mockAuthMiddleware(userID), h.UpdateStage)
		req, _ := http.NewRequest(http.MethodPatch, "/jobs/j1/stages/s1", bytes.NewBufferString(`{"status":"completed"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// ---- DeleteStage -------------------------------------------------------------

func TestJobHandler_DeleteStage(t *testing.T) {
	userID := "user-123"

	t.Run("401 without auth", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.DELETE("/jobs/:id/stages/:stageId", h.DeleteStage)
		req, _ := http.NewRequest(http.MethodDelete, "/jobs/j1/stages/s1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("404 when job not found (before pool use)", func(t *testing.T) {
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}
		svc := newStageJobService(jobRepo, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.DELETE("/jobs/:id/stages/:stageId", mockAuthMiddleware(userID), h.DeleteStage)
		req, _ := http.NewRequest(http.MethodDelete, "/jobs/missing/stages/s1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("404 when stage not found", func(t *testing.T) {
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return &model.Job{ID: jid, UserID: uid}, nil
			},
		}
		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return nil, model.ErrJobStageNotFound
			},
		}
		svc := newStageJobService(jobRepo, stageRepo, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.DELETE("/jobs/:id/stages/:stageId", mockAuthMiddleware(userID), h.DeleteStage)
		req, _ := http.NewRequest(http.MethodDelete, "/jobs/j1/stages/missing", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, string(model.CodeJobStageNotFound), decodeErr(t, w).ErrorCode)
	})
}

// ---- CreateStageTemplate -----------------------------------------------------

func TestJobHandler_CreateStageTemplate(t *testing.T) {
	userID := "user-123"

	t.Run("creates template successfully", func(t *testing.T) {
		tmpl := &ConfigStageTemplateRepo{
			CreateFunc: func(ctx context.Context, tpl *model.StageTemplate) error {
				tpl.ID = "tmpl-1"
				return nil
			},
		}
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, tmpl)
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.POST("/stage-templates", mockAuthMiddleware(userID), h.CreateStageTemplate)

		req, _ := http.NewRequest(http.MethodPost, "/stage-templates", bytes.NewBufferString(`{"name":"Interview","order":2}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)
		var dto model.StageTemplateDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
		assert.Equal(t, "Interview", dto.Name)
		assert.Equal(t, 2, dto.Order)
	})

	t.Run("401 without auth", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.POST("/stage-templates", h.CreateStageTemplate)
		req, _ := http.NewRequest(http.MethodPost, "/stage-templates", bytes.NewBufferString(`{"name":"X"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("400 for empty name (binding required)", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.POST("/stage-templates", mockAuthMiddleware(userID), h.CreateStageTemplate)
		req, _ := http.NewRequest(http.MethodPost, "/stage-templates", bytes.NewBufferString(`{"name":""}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("500 when repo create fails", func(t *testing.T) {
		tmpl := &ConfigStageTemplateRepo{
			CreateFunc: func(ctx context.Context, tpl *model.StageTemplate) error {
				return assertAnErr
			},
		}
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, tmpl)
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.POST("/stage-templates", mockAuthMiddleware(userID), h.CreateStageTemplate)
		req, _ := http.NewRequest(http.MethodPost, "/stage-templates", bytes.NewBufferString(`{"name":"Interview"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, string(model.CodeInternalError), decodeErr(t, w).ErrorCode)
	})
}

// ---- ListStageTemplates ------------------------------------------------------

func TestJobHandler_ListStageTemplates(t *testing.T) {
	userID := "user-123"

	t.Run("returns paginated templates", func(t *testing.T) {
		tmpl := &ConfigStageTemplateRepo{
			ListFunc: func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
				assert.Equal(t, 50, limit)
				assert.Equal(t, 10, offset)
				return []*model.StageTemplate{
					{ID: "a", UserID: uid, Name: "Wishlist", Order: 0},
					{ID: "b", UserID: uid, Name: "Applied", Order: 1},
				}, 2, nil
			},
		}
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, tmpl)
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.GET("/stage-templates", mockAuthMiddleware(userID), h.ListStageTemplates)

		req, _ := http.NewRequest(http.MethodGet, "/stage-templates?limit=50&offset=10", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"total":2`)
		assert.Contains(t, w.Body.String(), "Wishlist")
	})

	t.Run("401 without auth", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.GET("/stage-templates", h.ListStageTemplates)
		req, _ := http.NewRequest(http.MethodGet, "/stage-templates", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("400 for invalid pagination", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.GET("/stage-templates", mockAuthMiddleware(userID), h.ListStageTemplates)
		req, _ := http.NewRequest(http.MethodGet, "/stage-templates?limit=abc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "INVALID_PAGINATION_PARAMS", decodeErr(t, w).ErrorCode)
	})

	t.Run("500 when repo list fails", func(t *testing.T) {
		tmpl := &ConfigStageTemplateRepo{
			ListFunc: func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
				return nil, 0, assertAnErr
			},
		}
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, tmpl)
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.GET("/stage-templates", mockAuthMiddleware(userID), h.ListStageTemplates)
		req, _ := http.NewRequest(http.MethodGet, "/stage-templates", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, "INTERNAL_ERROR", decodeErr(t, w).ErrorCode)
	})
}

// ---- UpdateStageTemplate -----------------------------------------------------

func TestJobHandler_UpdateStageTemplate(t *testing.T) {
	userID := "user-123"

	t.Run("updates template successfully", func(t *testing.T) {
		tmpl := &ConfigStageTemplateRepo{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return &model.StageTemplate{ID: tid, UserID: uid, Name: "Old", Order: 1}, nil
			},
		}
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, tmpl)
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.PATCH("/stage-templates/:templateId", mockAuthMiddleware(userID), h.UpdateStageTemplate)

		req, _ := http.NewRequest(http.MethodPatch, "/stage-templates/t1", bytes.NewBufferString(`{"name":"New Name","order":3}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var dto model.StageTemplateDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &dto))
		assert.Equal(t, "New Name", dto.Name)
		assert.Equal(t, 3, dto.Order)
	})

	t.Run("401 without auth", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.PATCH("/stage-templates/:templateId", h.UpdateStageTemplate)
		req, _ := http.NewRequest(http.MethodPatch, "/stage-templates/t1", bytes.NewBufferString(`{"name":"X"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("400 for malformed body", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.PATCH("/stage-templates/:templateId", mockAuthMiddleware(userID), h.UpdateStageTemplate)
		req, _ := http.NewRequest(http.MethodPatch, "/stage-templates/t1", bytes.NewBufferString(`{`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("404 when template not found", func(t *testing.T) {
		tmpl := &ConfigStageTemplateRepo{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return nil, model.ErrStageTemplateNotFound
			},
		}
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, tmpl)
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.PATCH("/stage-templates/:templateId", mockAuthMiddleware(userID), h.UpdateStageTemplate)
		req, _ := http.NewRequest(http.MethodPatch, "/stage-templates/missing", bytes.NewBufferString(`{"name":"New"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, string(model.CodeStageTemplateNotFound), decodeErr(t, w).ErrorCode)
	})

	t.Run("400 when name blanked out", func(t *testing.T) {
		tmpl := &ConfigStageTemplateRepo{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return &model.StageTemplate{ID: tid, UserID: uid, Name: "Old", Order: 1}, nil
			},
		}
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, tmpl)
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.PATCH("/stage-templates/:templateId", mockAuthMiddleware(userID), h.UpdateStageTemplate)
		req, _ := http.NewRequest(http.MethodPatch, "/stage-templates/t1", bytes.NewBufferString(`{"name":"   "}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, string(model.CodeStageNameRequired), decodeErr(t, w).ErrorCode)
	})
}

// ---- DeleteStageTemplate -----------------------------------------------------

func TestJobHandler_DeleteStageTemplate(t *testing.T) {
	userID := "user-123"

	t.Run("deletes template successfully", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.DELETE("/stage-templates/:templateId", mockAuthMiddleware(userID), h.DeleteStageTemplate)
		req, _ := http.NewRequest(http.MethodDelete, "/stage-templates/t1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "deleted successfully")
	})

	t.Run("401 without auth", func(t *testing.T) {
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.DELETE("/stage-templates/:templateId", h.DeleteStageTemplate)
		req, _ := http.NewRequest(http.MethodDelete, "/stage-templates/t1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("409 when template is in use", func(t *testing.T) {
		tmpl := &ConfigStageTemplateRepo{
			DeleteFunc: func(ctx context.Context, uid, tid string) error {
				return model.ErrStageTemplateInUse
			},
		}
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, tmpl)
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.DELETE("/stage-templates/:templateId", mockAuthMiddleware(userID), h.DeleteStageTemplate)
		req, _ := http.NewRequest(http.MethodDelete, "/stage-templates/t1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Equal(t, string(model.CodeStageTemplateInUse), decodeErr(t, w).ErrorCode)
	})

	t.Run("404 when template not found", func(t *testing.T) {
		tmpl := &ConfigStageTemplateRepo{
			DeleteFunc: func(ctx context.Context, uid, tid string) error {
				return model.ErrStageTemplateNotFound
			},
		}
		svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, tmpl)
		h := NewJobHandler(svc)
		router := setupTestRouter()
		router.DELETE("/stage-templates/:templateId", mockAuthMiddleware(userID), h.DeleteStageTemplate)
		req, _ := http.NewRequest(http.MethodDelete, "/stage-templates/missing", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
