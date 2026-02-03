package service

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/applications/model"
	"github.com/andreypavlenko/jobber/modules/applications/ports"
	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock repositories
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

func createTestService() (*ApplicationService, *MockApplicationRepository, *MockStageRepository, *MockTemplateRepository, *MockJobRepository, *MockCompanyRepository, *MockResumeRepository, *MockCommentRepository) {
	appRepo := &MockApplicationRepository{}
	stageRepo := &MockStageRepository{}
	templateRepo := &MockTemplateRepository{}
	jobRepo := &MockJobRepository{}
	companyRepo := &MockCompanyRepository{}
	resumeRepo := &MockResumeRepository{}
	commentRepo := &MockCommentRepository{}

	svc := NewApplicationService(appRepo, stageRepo, templateRepo, jobRepo, companyRepo, resumeRepo, commentRepo)
	return svc, appRepo, stageRepo, templateRepo, jobRepo, companyRepo, resumeRepo, commentRepo
}

func TestApplicationService_Create(t *testing.T) {
	userID := "user-123"

	t.Run("creates application successfully", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, resumeRepo, _ := createTestService()

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

		req := &model.CreateApplicationRequest{
			JobID:    "job-1",
			ResumeID: "resume-1",
		}

		result, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "app-1", result.ID)
		assert.Equal(t, "active", result.Status)
	})

	t.Run("uses job title as name when not provided", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, resumeRepo, _ := createTestService()

		var createdApp *model.Application

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Software Engineer"}, nil
		}

		resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{ID: rid, Title: "My Resume"}, nil
		}

		appRepo.CreateFunc = func(ctx context.Context, app *model.Application) error {
			createdApp = app
			app.ID = "app-1"
			return nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
			return time.Now(), nil
		}

		req := &model.CreateApplicationRequest{
			JobID:    "job-1",
			ResumeID: "resume-1",
			Name:     "", // Empty name
		}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "Software Engineer", createdApp.Name)
	})

	t.Run("uses provided name when given", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, resumeRepo, _ := createTestService()

		var createdApp *model.Application

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Software Engineer"}, nil
		}

		resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{ID: rid, Title: "My Resume"}, nil
		}

		appRepo.CreateFunc = func(ctx context.Context, app *model.Application) error {
			createdApp = app
			app.ID = "app-1"
			return nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
			return time.Now(), nil
		}

		req := &model.CreateApplicationRequest{
			JobID:    "job-1",
			ResumeID: "resume-1",
			Name:     "Custom Application Name",
		}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "Custom Application Name", createdApp.Name)
	})
}

func TestApplicationService_GetByID(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("returns application successfully", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, resumeRepo, commentRepo := createTestService()

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

		result, err := svc.GetByID(context.Background(), userID, appID)

		require.NoError(t, err)
		assert.Equal(t, expectedApp.ID, result.ID)
		assert.Equal(t, expectedApp.Name, result.Name)
	})

	t.Run("returns error when application not found", func(t *testing.T) {
		svc, appRepo, _, _, _, _, _, _ := createTestService()

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return nil, model.ErrApplicationNotFound
		}

		result, err := svc.GetByID(context.Background(), userID, appID)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrApplicationNotFound, err)
	})
}

func TestApplicationService_Update(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("updates application status successfully", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, resumeRepo, _ := createTestService()

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

		newStatus := "offer"
		req := &model.UpdateApplicationRequest{Status: &newStatus}

		result, err := svc.Update(context.Background(), userID, appID, req)

		require.NoError(t, err)
		assert.Equal(t, "offer", result.Status)
	})

	t.Run("returns error for invalid status", func(t *testing.T) {
		svc, appRepo, _, _, _, _, _, _ := createTestService()

		existingApp := &model.Application{
			ID:     appID,
			UserID: userID,
			Status: "active",
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return existingApp, nil
		}

		invalidStatus := "invalid-status"
		req := &model.UpdateApplicationRequest{Status: &invalidStatus}

		result, err := svc.Update(context.Background(), userID, appID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrInvalidStatus, err)
	})
}

func TestApplicationService_Delete(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("deletes application successfully", func(t *testing.T) {
		svc, appRepo, _, _, _, _, _, _ := createTestService()

		var deletedAppID string

		appRepo.DeleteFunc = func(ctx context.Context, uid, aid string) error {
			deletedAppID = aid
			return nil
		}

		err := svc.Delete(context.Background(), userID, appID)

		require.NoError(t, err)
		assert.Equal(t, appID, deletedAppID)
	})
}

func TestApplicationService_CreateStageTemplate(t *testing.T) {
	userID := "user-123"

	t.Run("creates stage template successfully", func(t *testing.T) {
		svc, _, _, templateRepo, _, _, _, _ := createTestService()

		templateRepo.CreateFunc = func(ctx context.Context, template *model.StageTemplate) error {
			template.ID = "template-1"
			template.CreatedAt = time.Now()
			template.UpdatedAt = time.Now()
			return nil
		}

		req := &model.CreateStageTemplateRequest{
			Name:  "Phone Screen",
			Order: 1,
		}

		result, err := svc.CreateStageTemplate(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "template-1", result.ID)
		assert.Equal(t, "Phone Screen", result.Name)
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		svc, _, _, _, _, _, _, _ := createTestService()

		req := &model.CreateStageTemplateRequest{
			Name: "   ",
		}

		result, err := svc.CreateStageTemplate(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrStageNameRequired, err)
	})
}

func TestApplicationService_ListStageTemplates(t *testing.T) {
	userID := "user-123"

	t.Run("returns templates list", func(t *testing.T) {
		svc, _, _, templateRepo, _, _, _, _ := createTestService()

		expectedTemplates := []*model.StageTemplate{
			{ID: "template-1", Name: "Phone Screen", Order: 1},
			{ID: "template-2", Name: "Technical Interview", Order: 2},
		}

		templateRepo.ListFunc = func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
			return expectedTemplates, 2, nil
		}

		result, total, err := svc.ListStageTemplates(context.Background(), userID, 20, 0)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 2, total)
	})
}

func TestApplicationService_AddStage(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("adds stage successfully", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, _, _, _, commentRepo := createTestService()

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

		req := &model.AddStageRequest{
			StageTemplateID: "template-1",
		}

		result, err := svc.AddStage(context.Background(), userID, appID, req)

		require.NoError(t, err)
		assert.Equal(t, "stage-1", result.ID)
		assert.Equal(t, "active", result.Status)
	})
}

func TestApplicationService_ListStages(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("returns stages list", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

		app := &model.Application{
			ID:     appID,
			UserID: userID,
		}

		stages := []*model.ApplicationStage{
			{ID: "stage-1", StageTemplateID: "template-1", Status: "completed"},
			{ID: "stage-2", StageTemplateID: "template-2", Status: "active"},
		}

		templates := []*model.StageTemplate{
			{ID: "template-1", Name: "Phone Screen"},
			{ID: "template-2", Name: "Technical Interview"},
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		stageRepo.ListByApplicationFunc = func(ctx context.Context, aid string) ([]*model.ApplicationStage, error) {
			return stages, nil
		}

		templateRepo.ListFunc = func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
			return templates, 2, nil
		}

		result, err := svc.ListStages(context.Background(), userID, appID)

		require.NoError(t, err)
		assert.Len(t, result, 2)
	})
}

func TestApplicationService_UpdateStage(t *testing.T) {
	userID := "user-123"
	appID := "app-1"
	stageID := "stage-1"

	t.Run("updates stage status successfully", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

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

		newStatus := "completed"
		req := &model.UpdateStageRequest{Status: &newStatus}

		result, err := svc.UpdateStage(context.Background(), userID, appID, stageID, req)

		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
	})

	t.Run("returns error for invalid status", func(t *testing.T) {
		svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

		app := &model.Application{
			ID:     appID,
			UserID: userID,
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

		invalidStatus := "invalid-status"
		req := &model.UpdateStageRequest{Status: &invalidStatus}

		result, err := svc.UpdateStage(context.Background(), userID, appID, stageID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrInvalidStatus, err)
	})
}

func TestApplicationService_DeleteStage(t *testing.T) {
	userID := "user-123"
	appID := "app-1"
	stageID := "stage-1"

	t.Run("deletes stage successfully", func(t *testing.T) {
		svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

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

		err := svc.DeleteStage(context.Background(), userID, appID, stageID)

		require.NoError(t, err)
	})

	t.Run("returns error when stage doesn't belong to application", func(t *testing.T) {
		svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

		app := &model.Application{
			ID:     appID,
			UserID: userID,
		}

		stage := &model.ApplicationStage{
			ID:            stageID,
			ApplicationID: "different-app", // Different application
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
			return stage, nil
		}

		err := svc.DeleteStage(context.Background(), userID, appID, stageID)

		assert.Equal(t, model.ErrApplicationStageNotFound, err)
	})
}

func TestApplicationStatus_Validation(t *testing.T) {
	validStatuses := []string{"active", "on_hold", "rejected", "offer", "archived"}
	invalidStatuses := []string{"invalid", "pending", ""}

	for _, status := range validStatuses {
		t.Run("valid_"+status, func(t *testing.T) {
			svc, appRepo, _, _, jobRepo, _, resumeRepo, _ := createTestService()

			existingApp := &model.Application{
				ID:     "app-1",
				UserID: "user-123",
				Status: "active",
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
				return &jobModel.Job{ID: jid, Title: "Test"}, nil
			}

			resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
				return &resumeModel.Resume{ID: rid, Title: "Test"}, nil
			}

			req := &model.UpdateApplicationRequest{Status: &status}
			_, err := svc.Update(context.Background(), "user-123", "app-1", req)

			require.NoError(t, err)
		})
	}

	for _, status := range invalidStatuses {
		t.Run("invalid_"+status, func(t *testing.T) {
			svc, appRepo, _, _, _, _, _, _ := createTestService()

			existingApp := &model.Application{
				ID:     "app-1",
				UserID: "user-123",
				Status: "active",
			}

			appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
				return existingApp, nil
			}

			req := &model.UpdateApplicationRequest{Status: &status}
			_, err := svc.Update(context.Background(), "user-123", "app-1", req)

			assert.Equal(t, model.ErrInvalidStatus, err)
		})
	}
}

func TestStageStatus_Validation(t *testing.T) {
	validStatuses := []string{"pending", "active", "completed", "skipped", "cancelled"}

	for _, status := range validStatuses {
		t.Run("valid_"+status, func(t *testing.T) {
			svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

			app := &model.Application{
				ID:     "app-1",
				UserID: "user-123",
			}

			stage := &model.ApplicationStage{
				ID:              "stage-1",
				ApplicationID:   "app-1",
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

			req := &model.UpdateStageRequest{Status: &status}
			_, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", req)

			require.NoError(t, err)
		})
	}

	t.Run("invalid status", func(t *testing.T) {
		svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

		app := &model.Application{
			ID:     "app-1",
			UserID: "user-123",
		}

		stage := &model.ApplicationStage{
			ID:            "stage-1",
			ApplicationID: "app-1",
			Status:        "active",
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
			return stage, nil
		}

		invalidStatus := "invalid"
		req := &model.UpdateStageRequest{Status: &invalidStatus}
		_, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", req)

		assert.Equal(t, model.ErrInvalidStatus, err)
	})
}

func TestApplicationService_List(t *testing.T) {
	userID := "user-123"

	t.Run("returns applications list", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, resumeRepo, _ := createTestService()

		apps := []*model.Application{
			{ID: "app-1", UserID: userID, JobID: "job-1", ResumeID: "resume-1", Status: "active", CreatedAt: time.Now()},
			{ID: "app-2", UserID: userID, JobID: "job-2", ResumeID: "resume-2", Status: "offer", CreatedAt: time.Now()},
		}

		appRepo.ListFunc = func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.Application, int, error) {
			return apps, 2, nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
			return time.Now(), nil
		}

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Test Job"}, nil
		}

		resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{ID: rid, Title: "Test Resume"}, nil
		}

		result, total, err := svc.List(context.Background(), userID, "created_at", "desc", 20, 0)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 2, total)
	})
}

func TestApplicationService_CompleteStage(t *testing.T) {
	userID := "user-123"
	appID := "app-1"
	stageID := "stage-1"

	t.Run("completes stage successfully", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

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

		result, err := svc.CompleteStage(context.Background(), userID, appID, stageID, &model.CompleteStageRequest{})

		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
	})
}

func TestApplicationService_UpdateStageTemplate(t *testing.T) {
	userID := "user-123"
	templateID := "template-1"

	t.Run("updates stage template successfully", func(t *testing.T) {
		svc, _, _, templateRepo, _, _, _, _ := createTestService()

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

		newName := "Updated Phone Screen"
		req := &model.UpdateStageTemplateRequest{Name: &newName}

		result, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, req)

		require.NoError(t, err)
		assert.Equal(t, "Updated Phone Screen", result.Name)
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		svc, _, _, templateRepo, _, _, _, _ := createTestService()

		template := &model.StageTemplate{
			ID:     templateID,
			UserID: userID,
			Name:   "Phone Screen",
		}

		templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return template, nil
		}

		emptyName := "   "
		req := &model.UpdateStageTemplateRequest{Name: &emptyName}

		result, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrStageNameRequired, err)
	})
}

func TestApplicationService_DeleteStageTemplate(t *testing.T) {
	userID := "user-123"
	templateID := "template-1"

	t.Run("deletes stage template successfully", func(t *testing.T) {
		svc, _, _, templateRepo, _, _, _, _ := createTestService()

		var deletedID string
		templateRepo.DeleteFunc = func(ctx context.Context, uid, tid string) error {
			deletedID = tid
			return nil
		}

		err := svc.DeleteStageTemplate(context.Background(), userID, templateID)

		require.NoError(t, err)
		assert.Equal(t, templateID, deletedID)
	})
}

func TestApplicationService_DeleteStage_CurrentStage(t *testing.T) {
	userID := "user-123"
	appID := "app-1"
	stageID := "stage-1"

	t.Run("deletes current stage and recalculates", func(t *testing.T) {
		svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

		app := &model.Application{
			ID:             appID,
			UserID:         userID,
			CurrentStageID: &stageID,
		}

		stage := &model.ApplicationStage{
			ID:            stageID,
			ApplicationID: appID,
			Status:        "active",
		}

		remainingStages := []*model.ApplicationStage{
			{ID: "stage-0", ApplicationID: appID, Status: "completed"},
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

		stageRepo.ListByApplicationFunc = func(ctx context.Context, aid string) ([]*model.ApplicationStage, error) {
			return remainingStages, nil
		}

		var updatedApp *model.Application
		appRepo.UpdateFunc = func(ctx context.Context, a *model.Application) error {
			updatedApp = a
			return nil
		}

		err := svc.DeleteStage(context.Background(), userID, appID, stageID)

		require.NoError(t, err)
		// The current stage should be recalculated to the remaining completed stage
		assert.NotNil(t, updatedApp.CurrentStageID)
		assert.Equal(t, "stage-0", *updatedApp.CurrentStageID)
	})
}

func TestApplicationService_AddStage_WithExistingStages(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("adds stage and completes previous", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, _, _, _, commentRepo := createTestService()

		currentStageID := "stage-0"
		app := &model.Application{
			ID:             appID,
			UserID:         userID,
			Status:         "active",
			CurrentStageID: &currentStageID,
		}

		currentStage := &model.ApplicationStage{
			ID:              currentStageID,
			ApplicationID:   appID,
			StageTemplateID: "template-1",
			Status:          "active",
		}

		template := &model.StageTemplate{
			ID:   "template-2",
			Name: "Technical Interview",
		}

		prevTemplate := &model.StageTemplate{
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
			if tid == "template-2" {
				return template, nil
			}
			return prevTemplate, nil
		}

		stageRepo.ListByApplicationFunc = func(ctx context.Context, aid string) ([]*model.ApplicationStage, error) {
			return []*model.ApplicationStage{currentStage}, nil
		}

		stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
			return currentStage, nil
		}

		stageRepo.UpdateFunc = func(ctx context.Context, s *model.ApplicationStage) error {
			return nil
		}

		stageRepo.CreateFunc = func(ctx context.Context, stage *model.ApplicationStage) error {
			stage.ID = "stage-1"
			return nil
		}

		comment := "Starting technical interview"
		commentRepo.CreateFunc = func(ctx context.Context, c *commentModel.Comment) error {
			return nil
		}

		req := &model.AddStageRequest{
			StageTemplateID: "template-2",
			Comment:         &comment,
		}

		result, err := svc.AddStage(context.Background(), userID, appID, req)

		require.NoError(t, err)
		assert.Equal(t, "stage-1", result.ID)
		assert.Equal(t, "active", result.Status)
	})
}

func TestApplicationService_Create_WithAppliedAt(t *testing.T) {
	userID := "user-123"

	t.Run("uses provided applied_at date", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, resumeRepo, _ := createTestService()

		appliedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
		var createdApp *model.Application

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Software Engineer"}, nil
		}

		resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			return &resumeModel.Resume{ID: rid, Title: "My Resume"}, nil
		}

		appRepo.CreateFunc = func(ctx context.Context, app *model.Application) error {
			createdApp = app
			app.ID = "app-1"
			return nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
			return time.Now(), nil
		}

		req := &model.CreateApplicationRequest{
			JobID:     "job-1",
			ResumeID:  "resume-1",
			AppliedAt: appliedAt,
		}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, appliedAt, createdApp.AppliedAt)
	})
}
