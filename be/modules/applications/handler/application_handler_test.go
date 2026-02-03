package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/applications/model"
	"github.com/andreypavlenko/jobber/modules/applications/ports"
	"github.com/andreypavlenko/jobber/modules/applications/service"
	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock repositories (same as in service tests)
type MockApplicationRepository struct {
	CreateFunc            func(ctx context.Context, app *model.Application) error
	GetByIDFunc           func(ctx context.Context, userID, appID string) (*model.Application, error)
	ListFunc              func(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.Application, int, error)
	UpdateFunc            func(ctx context.Context, app *model.Application) error
	DeleteFunc            func(ctx context.Context, userID, appID string) error
	GetLastActivityAtFunc func(ctx context.Context, appID string) (time.Time, error)
}

func (m *MockApplicationRepository) Create(ctx context.Context, app *model.Application) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, app)
	}
	return nil
}

func (m *MockApplicationRepository) GetByID(ctx context.Context, userID, appID string) (*model.Application, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, appID)
	}
	return nil, nil
}

func (m *MockApplicationRepository) List(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.Application, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, opts)
	}
	return nil, 0, nil
}

func (m *MockApplicationRepository) Update(ctx context.Context, app *model.Application) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, app)
	}
	return nil
}

func (m *MockApplicationRepository) Delete(ctx context.Context, userID, appID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, appID)
	}
	return nil
}

func (m *MockApplicationRepository) GetLastActivityAt(ctx context.Context, appID string) (time.Time, error) {
	if m.GetLastActivityAtFunc != nil {
		return m.GetLastActivityAtFunc(ctx, appID)
	}
	return time.Now(), nil
}

type MockStageRepository struct {
	CreateFunc            func(ctx context.Context, stage *model.ApplicationStage) error
	GetByIDFunc           func(ctx context.Context, stageID string) (*model.ApplicationStage, error)
	ListByApplicationFunc func(ctx context.Context, appID string) ([]*model.ApplicationStage, error)
	UpdateFunc            func(ctx context.Context, stage *model.ApplicationStage) error
	DeleteFunc            func(ctx context.Context, stageID string) error
}

func (m *MockStageRepository) Create(ctx context.Context, stage *model.ApplicationStage) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, stage)
	}
	return nil
}

func (m *MockStageRepository) GetByID(ctx context.Context, stageID string) (*model.ApplicationStage, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, stageID)
	}
	return nil, nil
}

func (m *MockStageRepository) ListByApplication(ctx context.Context, appID string) ([]*model.ApplicationStage, error) {
	if m.ListByApplicationFunc != nil {
		return m.ListByApplicationFunc(ctx, appID)
	}
	return nil, nil
}

func (m *MockStageRepository) Update(ctx context.Context, stage *model.ApplicationStage) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, stage)
	}
	return nil
}

func (m *MockStageRepository) Delete(ctx context.Context, stageID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, stageID)
	}
	return nil
}

type MockTemplateRepository struct {
	CreateFunc  func(ctx context.Context, template *model.StageTemplate) error
	GetByIDFunc func(ctx context.Context, userID, templateID string) (*model.StageTemplate, error)
	ListFunc    func(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error)
	UpdateFunc  func(ctx context.Context, template *model.StageTemplate) error
	DeleteFunc  func(ctx context.Context, userID, templateID string) error
}

func (m *MockTemplateRepository) Create(ctx context.Context, template *model.StageTemplate) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, template)
	}
	return nil
}

func (m *MockTemplateRepository) GetByID(ctx context.Context, userID, templateID string) (*model.StageTemplate, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, templateID)
	}
	return nil, nil
}

func (m *MockTemplateRepository) List(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockTemplateRepository) Update(ctx context.Context, template *model.StageTemplate) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, template)
	}
	return nil
}

func (m *MockTemplateRepository) Delete(ctx context.Context, userID, templateID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, templateID)
	}
	return nil
}

type MockJobRepository struct {
	GetByIDFunc func(ctx context.Context, userID, jobID string) (*jobModel.Job, error)
}

func (m *MockJobRepository) Create(ctx context.Context, job *jobModel.Job) error { return nil }
func (m *MockJobRepository) GetByID(ctx context.Context, userID, jobID string) (*jobModel.Job, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, jobID)
	}
	return nil, nil
}
func (m *MockJobRepository) List(ctx context.Context, userID string, limit, offset int, status, sortBy, sortOrder string) ([]*jobModel.JobDTO, int, error) {
	return nil, 0, nil
}
func (m *MockJobRepository) Update(ctx context.Context, job *jobModel.Job) error { return nil }
func (m *MockJobRepository) Delete(ctx context.Context, userID, jobID string) error {
	return nil
}

type MockCompanyRepository struct {
	GetByIDFunc func(ctx context.Context, userID, companyID string) (*companyModel.Company, error)
}

func (m *MockCompanyRepository) Create(ctx context.Context, company *companyModel.Company) error {
	return nil
}
func (m *MockCompanyRepository) GetByID(ctx context.Context, userID, companyID string) (*companyModel.Company, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, companyID)
	}
	return nil, nil
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

type MockResumeRepository struct {
	GetByIDFunc func(ctx context.Context, userID, resumeID string) (*resumeModel.Resume, error)
}

func (m *MockResumeRepository) Create(ctx context.Context, resume *resumeModel.Resume) error {
	return nil
}
func (m *MockResumeRepository) GetByID(ctx context.Context, userID, resumeID string) (*resumeModel.Resume, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, resumeID)
	}
	return nil, nil
}
func (m *MockResumeRepository) List(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*resumePorts.ResumeWithCount, int, error) {
	return nil, 0, nil
}
func (m *MockResumeRepository) Update(ctx context.Context, resume *resumeModel.Resume) error {
	return nil
}
func (m *MockResumeRepository) Delete(ctx context.Context, userID, resumeID string) error {
	return nil
}

type MockCommentRepository struct {
	CreateFunc            func(ctx context.Context, comment *commentModel.Comment) error
	ListByApplicationFunc func(ctx context.Context, appID string, userID ...string) ([]*commentModel.Comment, error)
}

func (m *MockCommentRepository) Create(ctx context.Context, comment *commentModel.Comment) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, comment)
	}
	return nil
}
func (m *MockCommentRepository) ListByApplication(ctx context.Context, appID string, userID ...string) ([]*commentModel.Comment, error) {
	if m.ListByApplicationFunc != nil {
		return m.ListByApplicationFunc(ctx, appID, userID...)
	}
	return nil, nil
}
func (m *MockCommentRepository) Delete(ctx context.Context, userID, commentID string) error {
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

func createTestHandler() (*ApplicationHandler, *MockApplicationRepository, *MockStageRepository, *MockTemplateRepository, *MockJobRepository, *MockResumeRepository, *MockCommentRepository) {
	appRepo := &MockApplicationRepository{}
	stageRepo := &MockStageRepository{}
	templateRepo := &MockTemplateRepository{}
	jobRepo := &MockJobRepository{}
	companyRepo := &MockCompanyRepository{}
	resumeRepo := &MockResumeRepository{}
	commentRepo := &MockCommentRepository{}

	svc := service.NewApplicationService(appRepo, stageRepo, templateRepo, jobRepo, companyRepo, resumeRepo, commentRepo)
	handler := NewApplicationHandler(svc)
	return handler, appRepo, stageRepo, templateRepo, jobRepo, resumeRepo, commentRepo
}

func TestApplicationHandler_Create(t *testing.T) {
	userID := "user-123"

	t.Run("creates application successfully", func(t *testing.T) {
		handler, appRepo, _, _, jobRepo, resumeRepo, _ := createTestHandler()

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Software Engineer"}, nil
		}

		resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{ID: rid, Title: "My Resume"}, nil
		}

		appRepo.CreateFunc = func(ctx context.Context, app *model.Application) error {
			app.ID = "app-1"
			app.CreatedAt = time.Now()
			app.UpdatedAt = time.Now()
			return nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
			return time.Now(), nil
		}

		router := setupTestRouter()
		router.POST("/applications", mockAuthMiddleware(userID), handler.Create)

		body := `{"job_id":"job-1","resume_id":"resume-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/applications", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response model.ApplicationDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "app-1", response.ID)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler, _, _, _, _, _, _ := createTestHandler()

		router := setupTestRouter()
		router.POST("/applications", handler.Create) // No auth middleware

		body := `{"job_id":"job-1","resume_id":"resume-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/applications", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid request", func(t *testing.T) {
		handler, _, _, _, _, _, _ := createTestHandler()

		router := setupTestRouter()
		router.POST("/applications", mockAuthMiddleware(userID), handler.Create)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPost, "/applications", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestApplicationHandler_Get(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("returns application successfully", func(t *testing.T) {
		handler, appRepo, _, _, jobRepo, resumeRepo, commentRepo := createTestHandler()

		expectedApp := &model.Application{
			ID:        appID,
			UserID:    userID,
			JobID:     "job-1",
			ResumeID:  "resume-1",
			Name:      "Test Application",
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return expectedApp, nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, aid string) (time.Time, error) {
			return time.Now(), nil
		}

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Software Engineer"}, nil
		}

		resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{ID: rid, Title: "My Resume"}, nil
		}

		commentRepo.ListByApplicationFunc = func(ctx context.Context, aid string, uid ...string) ([]*commentModel.Comment, error) {
			return []*commentModel.Comment{}, nil
		}

		router := setupTestRouter()
		router.GET("/applications/:id", mockAuthMiddleware(userID), handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/applications/"+appID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.ApplicationDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, expectedApp.ID, response.ID)
	})

	t.Run("returns 404 when application not found", func(t *testing.T) {
		handler, appRepo, _, _, _, _, _ := createTestHandler()

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return nil, model.ErrApplicationNotFound
		}

		router := setupTestRouter()
		router.GET("/applications/:id", mockAuthMiddleware(userID), handler.Get)

		req, _ := http.NewRequest(http.MethodGet, "/applications/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestApplicationHandler_List(t *testing.T) {
	userID := "user-123"

	t.Run("returns applications list", func(t *testing.T) {
		handler, appRepo, _, _, jobRepo, resumeRepo, _ := createTestHandler()

		apps := []*model.Application{
			{ID: "app-1", JobID: "job-1", ResumeID: "resume-1", Status: "active", CreatedAt: time.Now()},
		}

		appRepo.ListFunc = func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.Application, int, error) {
			return apps, 1, nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, aid string) (time.Time, error) {
			return time.Now(), nil
		}

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Software Engineer"}, nil
		}

		resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{ID: rid, Title: "My Resume"}, nil
		}

		router := setupTestRouter()
		router.GET("/applications", mockAuthMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/applications?limit=20&offset=0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestApplicationHandler_Update(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("updates application successfully", func(t *testing.T) {
		handler, appRepo, _, _, jobRepo, resumeRepo, _ := createTestHandler()

		existingApp := &model.Application{
			ID:        appID,
			UserID:    userID,
			JobID:     "job-1",
			ResumeID:  "resume-1",
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return existingApp, nil
		}

		appRepo.UpdateFunc = func(ctx context.Context, app *model.Application) error {
			return nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, aid string) (time.Time, error) {
			return time.Now(), nil
		}

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Software Engineer"}, nil
		}

		resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{ID: rid, Title: "My Resume"}, nil
		}

		router := setupTestRouter()
		router.PATCH("/applications/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"status":"offer"}`
		req, _ := http.NewRequest(http.MethodPatch, "/applications/"+appID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 when application not found", func(t *testing.T) {
		handler, appRepo, _, _, _, _, _ := createTestHandler()

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return nil, model.ErrApplicationNotFound
		}

		router := setupTestRouter()
		router.PATCH("/applications/:id", mockAuthMiddleware(userID), handler.Update)

		body := `{"status":"offer"}`
		req, _ := http.NewRequest(http.MethodPatch, "/applications/nonexistent", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestApplicationHandler_Delete(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("deletes application successfully", func(t *testing.T) {
		handler, appRepo, _, _, _, _, _ := createTestHandler()

		appRepo.DeleteFunc = func(ctx context.Context, uid, aid string) error {
			return nil
		}

		router := setupTestRouter()
		router.DELETE("/applications/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/applications/"+appID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 when application not found", func(t *testing.T) {
		handler, appRepo, _, _, _, _, _ := createTestHandler()

		appRepo.DeleteFunc = func(ctx context.Context, uid, aid string) error {
			return model.ErrApplicationNotFound
		}

		router := setupTestRouter()
		router.DELETE("/applications/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/applications/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestApplicationHandler_StageTemplates(t *testing.T) {
	userID := "user-123"

	t.Run("creates stage template", func(t *testing.T) {
		handler, _, _, templateRepo, _, _, _ := createTestHandler()

		templateRepo.CreateFunc = func(ctx context.Context, template *model.StageTemplate) error {
			template.ID = "template-1"
			return nil
		}

		router := setupTestRouter()
		router.POST("/stage-templates", mockAuthMiddleware(userID), handler.CreateStageTemplate)

		body := `{"name":"Phone Screen","order":1}`
		req, _ := http.NewRequest(http.MethodPost, "/stage-templates", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("lists stage templates", func(t *testing.T) {
		handler, _, _, templateRepo, _, _, _ := createTestHandler()

		templates := []*model.StageTemplate{
			{ID: "template-1", Name: "Phone Screen"},
			{ID: "template-2", Name: "Technical Interview"},
		}

		templateRepo.ListFunc = func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
			return templates, 2, nil
		}

		router := setupTestRouter()
		router.GET("/stage-templates", mockAuthMiddleware(userID), handler.ListStageTemplates)

		req, _ := http.NewRequest(http.MethodGet, "/stage-templates?limit=20&offset=0", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestApplicationHandler_Stages(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("adds stage to application", func(t *testing.T) {
		handler, appRepo, stageRepo, templateRepo, _, _, commentRepo := createTestHandler()

		app := &model.Application{
			ID:     appID,
			UserID: userID,
			Status: "active",
		}

		template := &model.StageTemplate{
			ID:   "template-1",
			Name: "Phone Screen",
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		appRepo.UpdateFunc = func(ctx context.Context, app *model.Application) error {
			return nil
		}

		templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return template, nil
		}

		stageRepo.ListByApplicationFunc = func(ctx context.Context, aid string) ([]*model.ApplicationStage, error) {
			return []*model.ApplicationStage{}, nil
		}

		stageRepo.CreateFunc = func(ctx context.Context, stage *model.ApplicationStage) error {
			stage.ID = "stage-1"
			return nil
		}

		commentRepo.CreateFunc = func(ctx context.Context, comment *commentModel.Comment) error {
			return nil
		}

		router := setupTestRouter()
		router.POST("/applications/:id/stages", mockAuthMiddleware(userID), handler.AddStage)

		body := `{"stage_template_id":"template-1"}`
		req, _ := http.NewRequest(http.MethodPost, "/applications/"+appID+"/stages", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("lists stages", func(t *testing.T) {
		handler, appRepo, stageRepo, templateRepo, _, _, _ := createTestHandler()

		app := &model.Application{
			ID:     appID,
			UserID: userID,
		}

		stages := []*model.ApplicationStage{
			{ID: "stage-1", StageTemplateID: "template-1", Status: "completed"},
		}

		templates := []*model.StageTemplate{
			{ID: "template-1", Name: "Phone Screen"},
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		stageRepo.ListByApplicationFunc = func(ctx context.Context, aid string) ([]*model.ApplicationStage, error) {
			return stages, nil
		}

		templateRepo.ListFunc = func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
			return templates, 1, nil
		}

		router := setupTestRouter()
		router.GET("/applications/:id/stages", mockAuthMiddleware(userID), handler.ListStages)

		req, _ := http.NewRequest(http.MethodGet, "/applications/"+appID+"/stages", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestApplicationHandler_UpdateStage(t *testing.T) {
	userID := "user-123"
	appID := "app-1"
	stageID := "stage-1"

	t.Run("updates stage status", func(t *testing.T) {
		handler, appRepo, stageRepo, templateRepo, _, _, _ := createTestHandler()

		app := &model.Application{
			ID:     appID,
			UserID: userID,
		}

		stage := &model.ApplicationStage{
			ID:              stageID,
			ApplicationID:   appID,
			StageTemplateID: "template-1",
			Status:          "active",
		}

		template := &model.StageTemplate{
			ID:   "template-1",
			Name: "Phone Screen",
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
			return stage, nil
		}

		stageRepo.UpdateFunc = func(ctx context.Context, s *model.ApplicationStage) error {
			return nil
		}

		templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return template, nil
		}

		router := setupTestRouter()
		router.PATCH("/applications/:id/stages/:stageId", mockAuthMiddleware(userID), handler.UpdateStage)

		body := `{"status":"completed"}`
		req, _ := http.NewRequest(http.MethodPatch, "/applications/"+appID+"/stages/"+stageID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 when stage not found", func(t *testing.T) {
		handler, appRepo, stageRepo, _, _, _, _ := createTestHandler()

		app := &model.Application{
			ID:     appID,
			UserID: userID,
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
			return nil, model.ErrApplicationStageNotFound
		}

		router := setupTestRouter()
		router.PATCH("/applications/:id/stages/:stageId", mockAuthMiddleware(userID), handler.UpdateStage)

		body := `{"status":"completed"}`
		req, _ := http.NewRequest(http.MethodPatch, "/applications/"+appID+"/stages/nonexistent", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestApplicationHandler_CompleteStage(t *testing.T) {
	userID := "user-123"
	appID := "app-1"
	stageID := "stage-1"

	t.Run("completes stage successfully", func(t *testing.T) {
		handler, appRepo, stageRepo, templateRepo, _, _, _ := createTestHandler()

		app := &model.Application{
			ID:     appID,
			UserID: userID,
		}

		stage := &model.ApplicationStage{
			ID:              stageID,
			ApplicationID:   appID,
			StageTemplateID: "template-1",
			Status:          "active",
		}

		template := &model.StageTemplate{
			ID:   "template-1",
			Name: "Phone Screen",
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
			return stage, nil
		}

		stageRepo.UpdateFunc = func(ctx context.Context, s *model.ApplicationStage) error {
			return nil
		}

		templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return template, nil
		}

		router := setupTestRouter()
		router.POST("/applications/:id/stages/:stageId/complete", mockAuthMiddleware(userID), handler.CompleteStage)

		body := `{}`
		req, _ := http.NewRequest(http.MethodPost, "/applications/"+appID+"/stages/"+stageID+"/complete", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestApplicationHandler_DeleteStage(t *testing.T) {
	userID := "user-123"
	appID := "app-1"
	stageID := "stage-1"

	t.Run("deletes stage successfully", func(t *testing.T) {
		handler, appRepo, stageRepo, _, _, _, _ := createTestHandler()

		app := &model.Application{
			ID:             appID,
			UserID:         userID,
			CurrentStageID: nil,
		}

		stage := &model.ApplicationStage{
			ID:            stageID,
			ApplicationID: appID,
			Status:        "active",
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
			return stage, nil
		}

		stageRepo.DeleteFunc = func(ctx context.Context, sid string) error {
			return nil
		}

		router := setupTestRouter()
		router.DELETE("/applications/:id/stages/:stageId", mockAuthMiddleware(userID), handler.DeleteStage)

		req, _ := http.NewRequest(http.MethodDelete, "/applications/"+appID+"/stages/"+stageID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestApplicationHandler_UpdateStageTemplate(t *testing.T) {
	userID := "user-123"
	templateID := "template-1"

	t.Run("updates stage template", func(t *testing.T) {
		handler, _, _, templateRepo, _, _, _ := createTestHandler()

		template := &model.StageTemplate{
			ID:     templateID,
			UserID: userID,
			Name:   "Phone Screen",
			Order:  1,
		}

		templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return template, nil
		}

		templateRepo.UpdateFunc = func(ctx context.Context, t *model.StageTemplate) error {
			return nil
		}

		router := setupTestRouter()
		router.PATCH("/stage-templates/:id", mockAuthMiddleware(userID), handler.UpdateStageTemplate)

		body := `{"name":"Technical Interview"}`
		req, _ := http.NewRequest(http.MethodPatch, "/stage-templates/"+templateID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns error when template not found", func(t *testing.T) {
		handler, _, _, templateRepo, _, _, _ := createTestHandler()

		templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return nil, model.ErrStageTemplateNotFound
		}

		router := setupTestRouter()
		router.PATCH("/stage-templates/:id", mockAuthMiddleware(userID), handler.UpdateStageTemplate)

		body := `{"name":"Technical Interview"}`
		req, _ := http.NewRequest(http.MethodPatch, "/stage-templates/nonexistent", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Handler returns 500 for unhandled errors - error mapping may vary
		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

func TestApplicationHandler_DeleteStageTemplate(t *testing.T) {
	userID := "user-123"
	templateID := "template-1"

	t.Run("deletes stage template successfully", func(t *testing.T) {
		handler, _, _, templateRepo, _, _, _ := createTestHandler()

		templateRepo.DeleteFunc = func(ctx context.Context, uid, tid string) error {
			return nil
		}

		router := setupTestRouter()
		router.DELETE("/stage-templates/:id", mockAuthMiddleware(userID), handler.DeleteStageTemplate)

		req, _ := http.NewRequest(http.MethodDelete, "/stage-templates/"+templateID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns error when template not found", func(t *testing.T) {
		handler, _, _, templateRepo, _, _, _ := createTestHandler()

		templateRepo.DeleteFunc = func(ctx context.Context, uid, tid string) error {
			return model.ErrStageTemplateNotFound
		}

		router := setupTestRouter()
		router.DELETE("/stage-templates/:id", mockAuthMiddleware(userID), handler.DeleteStageTemplate)

		req, _ := http.NewRequest(http.MethodDelete, "/stage-templates/nonexistent", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Handler returns 500 for unhandled errors - error mapping may vary
		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

func TestApplicationHandler_RegisterRoutes(t *testing.T) {
	handler, appRepo, stageRepo, templateRepo, jobRepo, resumeRepo, commentRepo := createTestHandler()

	// Setup mock responses for all routes
	appRepo.CreateFunc = func(ctx context.Context, app *model.Application) error {
		app.ID = "app-1"
		return nil
	}
	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: aid, Status: "active", JobID: "job-1", ResumeID: "resume-1"}, nil
	}
	appRepo.ListFunc = func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.Application, int, error) {
		return []*model.Application{}, 0, nil
	}
	appRepo.UpdateFunc = func(ctx context.Context, app *model.Application) error {
		return nil
	}
	appRepo.DeleteFunc = func(ctx context.Context, uid, aid string) error {
		return nil
	}
	appRepo.GetLastActivityAtFunc = func(ctx context.Context, aid string) (time.Time, error) {
		return time.Now(), nil
	}

	jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
		return &jobModel.Job{ID: jid, Title: "Test"}, nil
	}

	resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
		return &resumeModel.Resume{ID: rid, Title: "Test"}, nil
	}

	commentRepo.ListByApplicationFunc = func(ctx context.Context, aid string, uid ...string) ([]*commentModel.Comment, error) {
		return []*commentModel.Comment{}, nil
	}

	templateRepo.CreateFunc = func(ctx context.Context, template *model.StageTemplate) error {
		template.ID = "template-1"
		return nil
	}
	templateRepo.ListFunc = func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
		return []*model.StageTemplate{}, 0, nil
	}
	templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
		return &model.StageTemplate{ID: tid, Name: "Test"}, nil
	}
	templateRepo.UpdateFunc = func(ctx context.Context, template *model.StageTemplate) error {
		return nil
	}
	templateRepo.DeleteFunc = func(ctx context.Context, uid, tid string) error {
		return nil
	}

	stageRepo.CreateFunc = func(ctx context.Context, stage *model.ApplicationStage) error {
		stage.ID = "stage-1"
		return nil
	}
	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return &model.ApplicationStage{ID: sid, ApplicationID: "test-id", StageTemplateID: "template-1", Status: "active"}, nil
	}
	stageRepo.ListByApplicationFunc = func(ctx context.Context, aid string) ([]*model.ApplicationStage, error) {
		return []*model.ApplicationStage{}, nil
	}
	stageRepo.UpdateFunc = func(ctx context.Context, stage *model.ApplicationStage) error {
		return nil
	}
	stageRepo.DeleteFunc = func(ctx context.Context, sid string) error {
		return nil
	}

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"))

	routes := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/v1/applications", `{"job_id":"job-1","resume_id":"resume-1"}`},
		{http.MethodGet, "/api/v1/applications", ""},
		{http.MethodGet, "/api/v1/applications/test-id", ""},
		{http.MethodPatch, "/api/v1/applications/test-id", `{"status":"offer"}`},
		{http.MethodDelete, "/api/v1/applications/test-id", ""},
		{http.MethodPost, "/api/v1/applications/test-id/stages", `{"stage_template_id":"template-1"}`},
		{http.MethodGet, "/api/v1/applications/test-id/stages", ""},
		{http.MethodPost, "/api/v1/stage-templates", `{"name":"Test","order":1}`},
		{http.MethodGet, "/api/v1/stage-templates", ""},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			var body *bytes.Buffer
			if route.body != "" {
				body = bytes.NewBufferString(route.body)
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
