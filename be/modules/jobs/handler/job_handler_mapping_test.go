package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/andreypavlenko/jobber/modules/jobs/service"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLimitChecker implements service.LimitChecker.
type mockLimitChecker struct {
	err error
}

func (m *mockLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	return m.err
}

// newLimitedJobService wires a JobService with a limit checker + configurable
// template repo (the default first-column placement still works).
func newLimitedJobService(jobRepo *MockJobRepository, limit *mockLimitChecker) *service.JobService {
	return service.NewJobService(nil, jobRepo, &MockJobStageRepository{}, &ConfigStageTemplateRepo{}, defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, limit, nil)
}

func TestJobHandler_Create_PlanLimitReached(t *testing.T) {
	userID := "user-123"
	svc := newLimitedJobService(&MockJobRepository{}, &mockLimitChecker{err: subModel.ErrLimitReached})
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.POST("/jobs", mockAuthMiddleware(userID), h.Create)

	req, _ := http.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"title":"Engineer"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "PLAN_LIMIT_REACHED", decodeErr(t, w).ErrorCode)
}

func TestJobHandler_Create_GenericServiceError(t *testing.T) {
	userID := "user-123"
	// A non-sentinel error from the limit checker maps to the default 500.
	svc := newLimitedJobService(&MockJobRepository{}, &mockLimitChecker{err: assertAnErr})
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.POST("/jobs", mockAuthMiddleware(userID), h.Create)

	req, _ := http.NewRequest(http.MethodPost, "/jobs", bytes.NewBufferString(`{"title":"Engineer"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, string(model.CodeInternalError), decodeErr(t, w).ErrorCode)
}

func TestJobHandler_List_ServiceError(t *testing.T) {
	userID := "user-123"
	jobRepo := &MockJobRepository{
		ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
			return nil, 0, assertAnErr
		},
	}
	svc := newTestJobService(jobRepo)
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.GET("/jobs", mockAuthMiddleware(userID), h.List)

	req, _ := http.NewRequest(http.MethodGet, "/jobs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "INTERNAL_ERROR", decodeErr(t, w).ErrorCode)
}

func TestJobHandler_List_InvalidSort(t *testing.T) {
	userID := "user-123"
	svc := newTestJobService(&MockJobRepository{})
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.GET("/jobs", mockAuthMiddleware(userID), h.List)

	req, _ := http.NewRequest(http.MethodGet, "/jobs?sort=bogus:sideways", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_SORT", decodeErr(t, w).ErrorCode)
}

func TestJobHandler_List_InvalidPagination(t *testing.T) {
	userID := "user-123"
	svc := newTestJobService(&MockJobRepository{})
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.GET("/jobs", mockAuthMiddleware(userID), h.List)

	req, _ := http.NewRequest(http.MethodGet, "/jobs?limit=notanumber", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "INVALID_PAGINATION_PARAMS", decodeErr(t, w).ErrorCode)
}

func TestJobHandler_List_SearchTruncatedAndForwarded(t *testing.T) {
	userID := "user-123"
	longSearch := strings.Repeat("é", 150) // 150 runes -> truncated to 100
	var seenSearch string
	jobRepo := &MockJobRepository{
		ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
			seenSearch = opts.Search
			return []*model.JobDTO{}, 0, nil
		},
	}
	svc := newTestJobService(jobRepo)
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.GET("/jobs", mockAuthMiddleware(userID), h.List)

	req, _ := http.NewRequest(http.MethodGet, "/jobs?search="+longSearch, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 100, len([]rune(seenSearch)), "search must be rune-truncated to 100")
}

func TestJobHandler_Update_GenericServiceError(t *testing.T) {
	userID := "user-123"
	jobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
			return &model.Job{ID: jid, UserID: uid, Title: "T"}, nil
		},
		UpdateFunc: func(ctx context.Context, job *model.Job) error {
			return assertAnErr
		},
	}
	svc := newTestJobService(jobRepo)
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.PATCH("/jobs/:id", mockAuthMiddleware(userID), h.Update)

	req, _ := http.NewRequest(http.MethodPatch, "/jobs/j1", bytes.NewBufferString(`{"title":"New"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, string(model.CodeInternalError), decodeErr(t, w).ErrorCode)
}

func TestJobHandler_Delete_GenericServiceError(t *testing.T) {
	userID := "user-123"
	jobRepo := &MockJobRepository{
		DeleteFunc: func(ctx context.Context, uid, jid string) error {
			return assertAnErr
		},
	}
	svc := newTestJobService(jobRepo)
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.DELETE("/jobs/:id", mockAuthMiddleware(userID), h.Delete)

	req, _ := http.NewRequest(http.MethodDelete, "/jobs/j1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, string(model.CodeInternalError), decodeErr(t, w).ErrorCode)
}

func TestJobHandler_Move_ServiceError(t *testing.T) {
	userID := "user-123"
	// Job exists but the referenced stage template is missing -> service maps to
	// STAGE_TEMPLATE_NOT_FOUND (404).
	jobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
			return &model.Job{ID: jid, UserID: uid, Title: "T"}, nil
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
	router.POST("/jobs/:id/move", mockAuthMiddleware(userID), h.Move)

	req, _ := http.NewRequest(http.MethodPost, "/jobs/j1/move", bytes.NewBufferString(`{"stage_template_id":"missing"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, string(model.CodeStageTemplateNotFound), decodeErr(t, w).ErrorCode)
}

func TestJobHandler_Get_GenericServiceError(t *testing.T) {
	userID := "user-123"
	jobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
			return nil, assertAnErr
		},
	}
	svc := newTestJobService(jobRepo)
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.GET("/jobs/:id", mockAuthMiddleware(userID), h.Get)

	req, _ := http.NewRequest(http.MethodGet, "/jobs/j1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, string(model.CodeInternalError), decodeErr(t, w).ErrorCode)
}

func TestJobHandler_ReorderStageTemplates_ServiceError(t *testing.T) {
	userID := "user-123"
	tmpl := &ConfigStageTemplateRepo{
		ReorderFunc: func(ctx context.Context, uid string, ids []string) error {
			return model.ErrReorderMismatch
		},
	}
	svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, tmpl)
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.POST("/stage-templates/reorder", mockAuthMiddleware(userID), h.ReorderStageTemplates)

	req, _ := http.NewRequest(http.MethodPost, "/stage-templates/reorder", bytes.NewBufferString(`{"stage_ids":["a","b"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, string(model.CodeReorderMismatch), decodeErr(t, w).ErrorCode)
}

func TestJobHandler_ReorderStageTemplates_Unauthorized(t *testing.T) {
	svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, &ConfigStageTemplateRepo{})
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.POST("/stage-templates/reorder", h.ReorderStageTemplates)

	req, _ := http.NewRequest(http.MethodPost, "/stage-templates/reorder", bytes.NewBufferString(`{"stage_ids":["a"]}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJobHandler_NameExistsMapsToConflict(t *testing.T) {
	userID := "user-123"
	// Creating a template whose name collides maps to 409.
	tmpl := &ConfigStageTemplateRepo{
		CreateFunc: func(ctx context.Context, tpl *model.StageTemplate) error {
			return model.ErrStageTemplateNameExists
		},
	}
	svc := newStageJobService(&MockJobRepository{}, &MockJobStageRepository{}, tmpl)
	h := NewJobHandler(svc)
	router := setupTestRouter()
	router.POST("/stage-templates", mockAuthMiddleware(userID), h.CreateStageTemplate)

	req, _ := http.NewRequest(http.MethodPost, "/stage-templates", bytes.NewBufferString(`{"name":"Applied"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, string(model.CodeStageTemplateNameExists), decodeErr(t, w).ErrorCode)
}
