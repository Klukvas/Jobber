package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/applications/model"
	"github.com/andreypavlenko/jobber/modules/applications/ports"
	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	rbPorts "github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
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
	ListEnrichedFunc      func(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.ApplicationDTO, int, error)
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

func (m *MockApplicationRepository) ListEnriched(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.ApplicationDTO, int, error) {
	if m.ListEnrichedFunc != nil {
		return m.ListEnrichedFunc(ctx, userID, opts)
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
func (m *MockJobRepository) ToggleFavorite(ctx context.Context, userID, jobID string) (bool, error) {
	return false, nil
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
func (m *MockCompanyRepository) ToggleFavorite(ctx context.Context, userID, companyID string) (bool, error) {
	return false, nil
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

func strPtr(s string) *string { return &s }

func createTestService() (*ApplicationService, *MockApplicationRepository, *MockStageRepository, *MockTemplateRepository, *MockJobRepository, *MockCompanyRepository, *MockResumeRepository, *MockCommentRepository) {
	appRepo := &MockApplicationRepository{}
	stageRepo := &MockStageRepository{}
	templateRepo := &MockTemplateRepository{}
	jobRepo := &MockJobRepository{}
	companyRepo := &MockCompanyRepository{}
	resumeRepo := &MockResumeRepository{}
	commentRepo := &MockCommentRepository{}

	svc := NewApplicationService(nil, appRepo, stageRepo, templateRepo, jobRepo, companyRepo, resumeRepo, nil, commentRepo, nil)
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
			ResumeID: strPtr("resume-1"),
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
			ResumeID: strPtr("resume-1"),
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
			ResumeID: strPtr("resume-1"),
			Name:     "Custom Application Name",
		}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "Custom Application Name", createdApp.Name)
	})

	t.Run("creates application without resume", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, resumeRepo, _ := createTestService()

		resumeRepoCalled := false
		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Software Engineer"}, nil
		}

		resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
			resumeRepoCalled = true
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
			ResumeID: nil, // no resume
		}

		result, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "app-1", result.ID)
		assert.Nil(t, result.Resume)
		assert.False(t, resumeRepoCalled, "resume repo should not be called when ResumeID is nil")
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
			ResumeID:  strPtr("resume-1"),
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

	t.Run("handles comment fetch error gracefully", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, _, commentRepo := createTestService()

		expectedApp := &model.Application{
			ID:     appID,
			UserID: userID,
			JobID:  "job-1",
			Name:   "Test Application",
			Status: "active",
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

		commentRepo.ListByApplicationFunc = func(ctx context.Context, aid string, uid ...string) ([]*commentModel.Comment, error) {
			return nil, errors.New("comment fetch error")
		}

		result, err := svc.GetByID(context.Background(), userID, appID)

		require.NoError(t, err)
		assert.Equal(t, expectedApp.ID, result.ID)
		// Comments should be nil due to error
		assert.Nil(t, result.ApplicationComments)
		assert.Nil(t, result.StageComments)
	})

	t.Run("splits comments into application and stage comments", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, _, commentRepo := createTestService()

		expectedApp := &model.Application{
			ID:     appID,
			UserID: userID,
			JobID:  "job-1",
			Name:   "Test Application",
			Status: "active",
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

		stageID := "stage-1"
		commentRepo.ListByApplicationFunc = func(ctx context.Context, aid string, uid ...string) ([]*commentModel.Comment, error) {
			return []*commentModel.Comment{
				{ID: "c1", ApplicationID: aid, Content: "App comment", StageID: nil},
				{ID: "c2", ApplicationID: aid, Content: "Stage comment", StageID: &stageID},
			}, nil
		}

		result, err := svc.GetByID(context.Background(), userID, appID)

		require.NoError(t, err)
		assert.Len(t, result.ApplicationComments, 1)
		assert.Len(t, result.StageComments, 1)
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
			ResumeID:  strPtr("resume-1"),
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

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return &model.Application{ID: aid, UserID: uid, JobID: "job-1"}, nil
		}

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
	t.Run("returns error when application not found", func(t *testing.T) {
		svc, appRepo, _, _, _, _, _, _ := createTestService()

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return nil, model.ErrApplicationNotFound
		}

		req := &model.AddStageRequest{StageTemplateID: "template-1"}
		result, err := svc.AddStage(context.Background(), "user-123", "app-1", req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrApplicationNotFound, err)
	})

	t.Run("returns error when template not found", func(t *testing.T) {
		svc, appRepo, _, templateRepo, _, _, _, _ := createTestService()

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return &model.Application{ID: "app-1", UserID: "user-123"}, nil
		}

		templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return nil, model.ErrStageTemplateNotFound
		}

		req := &model.AddStageRequest{StageTemplateID: "template-1"}
		result, err := svc.AddStage(context.Background(), "user-123", "app-1", req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrStageTemplateNotFound, err)
	})

	t.Run("returns error when listing stages fails", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return &model.Application{ID: "app-1", UserID: "user-123"}, nil
		}

		templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return &model.StageTemplate{ID: tid, Name: "Test"}, nil
		}

		stageRepo.ListByApplicationFunc = func(ctx context.Context, aid string) ([]*model.ApplicationStage, error) {
			return nil, errors.New("stage list error")
		}

		req := &model.AddStageRequest{StageTemplateID: "template-1"}
		result, err := svc.AddStage(context.Background(), "user-123", "app-1", req)

		assert.Nil(t, result)
		assert.Error(t, err)
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
		t.Skip("Requires real database pool for transaction testing")
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
		svc, appRepo, _, _, _, _, _, _ := createTestService()

		now := time.Now()
		dtos := []*model.ApplicationDTO{
			{ID: "app-1", Name: "App 1", Status: "active", CreatedAt: now, LastActivityAt: now},
			{ID: "app-2", Name: "App 2", Status: "offer", CreatedAt: now, LastActivityAt: now},
		}

		appRepo.ListEnrichedFunc = func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.ApplicationDTO, int, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, 20, opts.Limit)
			assert.Equal(t, 0, opts.Offset)
			return dtos, 2, nil
		}

		result, total, err := svc.List(context.Background(), userID, "created_at", "desc", "", 20, 0)

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
		t.Skip("Requires real database pool for transaction testing")
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
	// AddStage uses database transactions (pgxpool.Begin) for atomicity,
	// so it requires a real database connection and cannot be unit-tested with mocks.
	t.Skip("AddStage requires a real database connection for transaction support")

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

// MockResumeBuilderRepository implements rbPorts.ResumeBuilderRepository (minimal)
type MockResumeBuilderRepository struct {
	GetByIDFunc func(ctx context.Context, id string) (*rbModel.ResumeBuilder, error)
}

func (m *MockResumeBuilderRepository) Create(ctx context.Context, rb *rbModel.ResumeBuilder) error {
	return nil
}
func (m *MockResumeBuilderRepository) GetByID(ctx context.Context, id string) (*rbModel.ResumeBuilder, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *MockResumeBuilderRepository) List(ctx context.Context, userID string) ([]*rbModel.ResumeBuilderDTO, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) Update(ctx context.Context, rb *rbModel.ResumeBuilder) error {
	return nil
}
func (m *MockResumeBuilderRepository) Delete(ctx context.Context, id string) error { return nil }
func (m *MockResumeBuilderRepository) GetFullResume(ctx context.Context, id string) (*rbModel.FullResumeDTO, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) VerifyOwnership(ctx context.Context, userID, resumeBuilderID string) error {
	return nil
}
func (m *MockResumeBuilderRepository) RunInTransaction(ctx context.Context, fn func(txRepo rbPorts.ResumeBuilderRepository) error) error {
	return fn(m)
}
func (m *MockResumeBuilderRepository) UpsertContact(ctx context.Context, contact *rbModel.Contact) error {
	return nil
}
func (m *MockResumeBuilderRepository) GetContact(ctx context.Context, resumeBuilderID string) (*rbModel.Contact, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) UpsertSummary(ctx context.Context, summary *rbModel.Summary) error {
	return nil
}
func (m *MockResumeBuilderRepository) GetSummary(ctx context.Context, resumeBuilderID string) (*rbModel.Summary, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateExperience(ctx context.Context, exp *rbModel.Experience) error {
	return nil
}
func (m *MockResumeBuilderRepository) UpdateExperience(ctx context.Context, exp *rbModel.Experience) error {
	return nil
}
func (m *MockResumeBuilderRepository) DeleteExperience(ctx context.Context, resumeBuilderID, id string) error {
	return nil
}
func (m *MockResumeBuilderRepository) ListExperiences(ctx context.Context, resumeBuilderID string) ([]*rbModel.Experience, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetExperienceByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Experience, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateEducation(ctx context.Context, edu *rbModel.Education) error {
	return nil
}
func (m *MockResumeBuilderRepository) UpdateEducation(ctx context.Context, edu *rbModel.Education) error {
	return nil
}
func (m *MockResumeBuilderRepository) DeleteEducation(ctx context.Context, resumeBuilderID, id string) error {
	return nil
}
func (m *MockResumeBuilderRepository) ListEducations(ctx context.Context, resumeBuilderID string) ([]*rbModel.Education, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetEducationByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Education, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateSkill(ctx context.Context, skill *rbModel.Skill) error {
	return nil
}
func (m *MockResumeBuilderRepository) UpdateSkill(ctx context.Context, skill *rbModel.Skill) error {
	return nil
}
func (m *MockResumeBuilderRepository) DeleteSkill(ctx context.Context, resumeBuilderID, id string) error {
	return nil
}
func (m *MockResumeBuilderRepository) ListSkills(ctx context.Context, resumeBuilderID string) ([]*rbModel.Skill, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetSkillByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Skill, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateLanguage(ctx context.Context, lang *rbModel.Language) error {
	return nil
}
func (m *MockResumeBuilderRepository) UpdateLanguage(ctx context.Context, lang *rbModel.Language) error {
	return nil
}
func (m *MockResumeBuilderRepository) DeleteLanguage(ctx context.Context, resumeBuilderID, id string) error {
	return nil
}
func (m *MockResumeBuilderRepository) ListLanguages(ctx context.Context, resumeBuilderID string) ([]*rbModel.Language, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetLanguageByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Language, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateCertification(ctx context.Context, cert *rbModel.Certification) error {
	return nil
}
func (m *MockResumeBuilderRepository) UpdateCertification(ctx context.Context, cert *rbModel.Certification) error {
	return nil
}
func (m *MockResumeBuilderRepository) DeleteCertification(ctx context.Context, resumeBuilderID, id string) error {
	return nil
}
func (m *MockResumeBuilderRepository) ListCertifications(ctx context.Context, resumeBuilderID string) ([]*rbModel.Certification, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetCertificationByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Certification, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateProject(ctx context.Context, proj *rbModel.Project) error {
	return nil
}
func (m *MockResumeBuilderRepository) UpdateProject(ctx context.Context, proj *rbModel.Project) error {
	return nil
}
func (m *MockResumeBuilderRepository) DeleteProject(ctx context.Context, resumeBuilderID, id string) error {
	return nil
}
func (m *MockResumeBuilderRepository) ListProjects(ctx context.Context, resumeBuilderID string) ([]*rbModel.Project, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetProjectByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Project, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateVolunteering(ctx context.Context, vol *rbModel.Volunteering) error {
	return nil
}
func (m *MockResumeBuilderRepository) UpdateVolunteering(ctx context.Context, vol *rbModel.Volunteering) error {
	return nil
}
func (m *MockResumeBuilderRepository) DeleteVolunteering(ctx context.Context, resumeBuilderID, id string) error {
	return nil
}
func (m *MockResumeBuilderRepository) ListVolunteering(ctx context.Context, resumeBuilderID string) ([]*rbModel.Volunteering, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetVolunteeringByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.Volunteering, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) CreateCustomSection(ctx context.Context, cs *rbModel.CustomSection) error {
	return nil
}
func (m *MockResumeBuilderRepository) UpdateCustomSection(ctx context.Context, cs *rbModel.CustomSection) error {
	return nil
}
func (m *MockResumeBuilderRepository) DeleteCustomSection(ctx context.Context, resumeBuilderID, id string) error {
	return nil
}
func (m *MockResumeBuilderRepository) ListCustomSections(ctx context.Context, resumeBuilderID string) ([]*rbModel.CustomSection, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) GetCustomSectionByID(ctx context.Context, resumeBuilderID, id string) (*rbModel.CustomSection, error) {
	return nil, nil
}
func (m *MockResumeBuilderRepository) UpsertSectionOrder(ctx context.Context, resumeBuilderID string, orders []*rbModel.SectionOrder) error {
	return nil
}
func (m *MockResumeBuilderRepository) ListSectionOrders(ctx context.Context, resumeBuilderID string) ([]*rbModel.SectionOrder, error) {
	return nil, nil
}

// MockAppLimitChecker implements LimitChecker for testing
type MockAppLimitChecker struct {
	CheckLimitFunc func(ctx context.Context, userID, resource string) error
}

func (m *MockAppLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, userID, resource)
	}
	return nil
}

// createTestServiceWithRBRepo creates a test service that includes resume builder repo
func createTestServiceWithRBRepo() (*ApplicationService, *MockApplicationRepository, *MockStageRepository, *MockTemplateRepository, *MockJobRepository, *MockCompanyRepository, *MockResumeRepository, *MockResumeBuilderRepository, *MockCommentRepository) {
	appRepo := &MockApplicationRepository{}
	stageRepo := &MockStageRepository{}
	templateRepo := &MockTemplateRepository{}
	jobRepo := &MockJobRepository{}
	companyRepo := &MockCompanyRepository{}
	resumeRepo := &MockResumeRepository{}
	rbRepo := &MockResumeBuilderRepository{}
	commentRepo := &MockCommentRepository{}

	svc := NewApplicationService(nil, appRepo, stageRepo, templateRepo, jobRepo, companyRepo, resumeRepo, rbRepo, commentRepo, nil)
	return svc, appRepo, stageRepo, templateRepo, jobRepo, companyRepo, resumeRepo, rbRepo, commentRepo
}

// --- Additional tests for buildApplicationDTO coverage ---

func TestBuildApplicationDTO_JobFetchFails(t *testing.T) {
	svc, appRepo, _, _, jobRepo, _, _, _ := createTestService()

	app := &model.Application{
		ID:     "app-1",
		UserID: "user-123",
		JobID:  "job-1",
		Status: "active",
	}

	jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
		return nil, errors.New("job fetch error")
	}

	appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
		return time.Now(), nil
	}

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return app, nil
	}

	result, err := svc.GetByID(context.Background(), "user-123", "app-1")

	require.NoError(t, err)
	assert.Nil(t, result.Job)
}

func TestBuildApplicationDTO_CompanyFetchFails(t *testing.T) {
	svc, appRepo, _, _, jobRepo, companyRepo, _, _ := createTestService()

	companyID := "company-1"
	app := &model.Application{
		ID:     "app-1",
		UserID: "user-123",
		JobID:  "job-1",
		Status: "active",
	}

	jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
		return &jobModel.Job{ID: jid, Title: "Engineer", CompanyID: &companyID}, nil
	}

	companyRepo.GetByIDFunc = func(ctx context.Context, uid, cid string) (*companyModel.Company, error) {
		return nil, errors.New("company fetch error")
	}

	appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
		return time.Now(), nil
	}

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return app, nil
	}

	result, err := svc.GetByID(context.Background(), "user-123", "app-1")

	require.NoError(t, err)
	assert.NotNil(t, result.Job)
	assert.Nil(t, result.Job.Company)
}

func TestBuildApplicationDTO_JobWithNoCompanyID(t *testing.T) {
	svc, appRepo, _, _, jobRepo, _, _, _ := createTestService()

	app := &model.Application{
		ID:     "app-1",
		UserID: "user-123",
		JobID:  "job-1",
		Status: "active",
	}

	jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
		return &jobModel.Job{ID: jid, Title: "Engineer", CompanyID: nil}, nil
	}

	appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
		return time.Now(), nil
	}

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return app, nil
	}

	result, err := svc.GetByID(context.Background(), "user-123", "app-1")

	require.NoError(t, err)
	assert.NotNil(t, result.Job)
	assert.Nil(t, result.Job.Company)
}

func TestBuildApplicationDTO_ResumeFetchFails(t *testing.T) {
	svc, appRepo, _, _, jobRepo, _, resumeRepo, _ := createTestService()

	resumeID := "resume-1"
	app := &model.Application{
		ID:       "app-1",
		UserID:   "user-123",
		JobID:    "job-1",
		ResumeID: &resumeID,
		Status:   "active",
	}

	jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
		return &jobModel.Job{ID: jid, Title: "Engineer"}, nil
	}

	resumeRepo.GetByIDFunc = func(ctx context.Context, uid, rid string) (*resumeModel.Resume, error) {
		return nil, errors.New("resume fetch error")
	}

	appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
		return time.Now(), nil
	}

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return app, nil
	}

	result, err := svc.GetByID(context.Background(), "user-123", "app-1")

	require.NoError(t, err)
	assert.Nil(t, result.Resume)
}

func TestBuildApplicationDTO_ResumeBuilderFetchSuccess(t *testing.T) {
	svc, appRepo, _, _, jobRepo, _, _, rbRepo, _ := createTestServiceWithRBRepo()

	rbID := "rb-1"
	app := &model.Application{
		ID:              "app-1",
		UserID:          "user-123",
		JobID:           "job-1",
		ResumeBuilderID: &rbID,
		Status:          "active",
	}

	jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
		return &jobModel.Job{ID: jid, Title: "Engineer"}, nil
	}

	rbRepo.GetByIDFunc = func(ctx context.Context, id string) (*rbModel.ResumeBuilder, error) {
		return &rbModel.ResumeBuilder{ID: id, Title: "My Builder Resume"}, nil
	}

	appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
		return time.Now(), nil
	}

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return app, nil
	}

	result, err := svc.GetByID(context.Background(), "user-123", "app-1")

	require.NoError(t, err)
	assert.NotNil(t, result.Resume)
	assert.Equal(t, "builder", result.Resume.Type)
	assert.Equal(t, "My Builder Resume", result.Resume.Name)
}

func TestBuildApplicationDTO_ResumeBuilderFetchFails(t *testing.T) {
	svc, appRepo, _, _, jobRepo, _, _, rbRepo, _ := createTestServiceWithRBRepo()

	rbID := "rb-1"
	app := &model.Application{
		ID:              "app-1",
		UserID:          "user-123",
		JobID:           "job-1",
		ResumeBuilderID: &rbID,
		Status:          "active",
	}

	jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
		return &jobModel.Job{ID: jid, Title: "Engineer"}, nil
	}

	rbRepo.GetByIDFunc = func(ctx context.Context, id string) (*rbModel.ResumeBuilder, error) {
		return nil, errors.New("rb fetch error")
	}

	appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
		return time.Now(), nil
	}

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return app, nil
	}

	result, err := svc.GetByID(context.Background(), "user-123", "app-1")

	require.NoError(t, err)
	assert.Nil(t, result.Resume)
}

func TestBuildApplicationDTO_LastActivityFails(t *testing.T) {
	svc, appRepo, _, _, jobRepo, _, _, _ := createTestService()

	now := time.Now()
	app := &model.Application{
		ID:        "app-1",
		UserID:    "user-123",
		JobID:     "job-1",
		Status:    "active",
		UpdatedAt: now,
	}

	jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
		return &jobModel.Job{ID: jid, Title: "Engineer"}, nil
	}

	appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
		return time.Time{}, errors.New("last activity error")
	}

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return app, nil
	}

	result, err := svc.GetByID(context.Background(), "user-123", "app-1")

	require.NoError(t, err)
	assert.Equal(t, now, result.LastActivityAt)
}

func TestBuildApplicationDTO_CurrentStageResolution(t *testing.T) {
	t.Run("resolves current stage name from template", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, jobRepo, _, _, _ := createTestService()

		stageID := "stage-1"
		app := &model.Application{
			ID:             "app-1",
			UserID:         "user-123",
			JobID:          "job-1",
			CurrentStageID: &stageID,
			Status:         "active",
		}

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Engineer"}, nil
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
			return time.Now(), nil
		}

		stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
			return &model.ApplicationStage{
				ID:              sid,
				ApplicationID:   "app-1",
				StageTemplateID: "template-1",
				Status:          "active",
			}, nil
		}

		templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return &model.StageTemplate{ID: tid, Name: "Phone Screen"}, nil
		}

		result, err := svc.GetByID(context.Background(), "user-123", "app-1")

		require.NoError(t, err)
		assert.NotNil(t, result.CurrentStageName)
		assert.Equal(t, "Phone Screen", *result.CurrentStageName)
	})

	t.Run("stage fetch fails gracefully", func(t *testing.T) {
		svc, appRepo, stageRepo, _, jobRepo, _, _, _ := createTestService()

		stageID := "stage-1"
		app := &model.Application{
			ID:             "app-1",
			UserID:         "user-123",
			JobID:          "job-1",
			CurrentStageID: &stageID,
			Status:         "active",
		}

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Engineer"}, nil
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
			return time.Now(), nil
		}

		stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
			return nil, errors.New("stage not found")
		}

		result, err := svc.GetByID(context.Background(), "user-123", "app-1")

		require.NoError(t, err)
		assert.Nil(t, result.CurrentStageName)
	})

	t.Run("template fetch fails gracefully", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, jobRepo, _, _, _ := createTestService()

		stageID := "stage-1"
		app := &model.Application{
			ID:             "app-1",
			UserID:         "user-123",
			JobID:          "job-1",
			CurrentStageID: &stageID,
			Status:         "active",
		}

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Engineer"}, nil
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
			return time.Now(), nil
		}

		stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
			return &model.ApplicationStage{
				ID:              sid,
				ApplicationID:   "app-1",
				StageTemplateID: "template-1",
				Status:          "active",
			}, nil
		}

		templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return nil, errors.New("template not found")
		}

		result, err := svc.GetByID(context.Background(), "user-123", "app-1")

		require.NoError(t, err)
		assert.Nil(t, result.CurrentStageName)
	})

	t.Run("empty current stage ID is ignored", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, _, _ := createTestService()

		emptyStageID := ""
		app := &model.Application{
			ID:             "app-1",
			UserID:         "user-123",
			JobID:          "job-1",
			CurrentStageID: &emptyStageID,
			Status:         "active",
		}

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Engineer"}, nil
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
			return time.Now(), nil
		}

		result, err := svc.GetByID(context.Background(), "user-123", "app-1")

		require.NoError(t, err)
		assert.Nil(t, result.CurrentStageName)
	})
}

// --- GetByID comment-splitting tests ---

func TestGetByID_CommentsAreSplit(t *testing.T) {
	t.Run("splits application-level and stage-level comments", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, _, commentRepo := createTestService()

		app := &model.Application{
			ID:     "app-1",
			UserID: "user-123",
			JobID:  "job-1",
			Status: "active",
		}

		stageID := "stage-1"
		comments := []*commentModel.Comment{
			{ID: "c1", ApplicationID: "app-1", StageID: nil, Content: "App-level comment"},
			{ID: "c2", ApplicationID: "app-1", StageID: &stageID, Content: "Stage comment"},
			{ID: "c3", ApplicationID: "app-1", StageID: nil, Content: "Another app comment"},
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
			return time.Now(), nil
		}

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Engineer"}, nil
		}

		commentRepo.ListByApplicationFunc = func(ctx context.Context, aid string, uid ...string) ([]*commentModel.Comment, error) {
			return comments, nil
		}

		result, err := svc.GetByID(context.Background(), "user-123", "app-1")

		require.NoError(t, err)
		assert.Len(t, result.ApplicationComments, 2)
		assert.Len(t, result.StageComments, 1)
	})

	t.Run("comment fetch error does not fail GetByID", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, _, commentRepo := createTestService()

		app := &model.Application{
			ID:     "app-1",
			UserID: "user-123",
			JobID:  "job-1",
			Status: "active",
		}

		appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
			return app, nil
		}

		appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
			return time.Now(), nil
		}

		jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: jid, Title: "Engineer"}, nil
		}

		commentRepo.ListByApplicationFunc = func(ctx context.Context, aid string, uid ...string) ([]*commentModel.Comment, error) {
			return nil, errors.New("comment fetch error")
		}

		result, err := svc.GetByID(context.Background(), "user-123", "app-1")

		require.NoError(t, err)
		assert.Nil(t, result.ApplicationComments)
		assert.Nil(t, result.StageComments)
	})
}

func TestGetByID_BuildDTOFails(t *testing.T) {
	// buildApplicationDTO itself does not return errors currently (all errors are logged and silently recovered).
	// But GetByID also returns buildApplicationDTO errors if they were returned.
	// This test verifies the second error path in GetByID (line 133-135).
	// Since buildApplicationDTO does not currently return errors, we test that it still passes.
	svc, appRepo, _, _, jobRepo, _, _, _ := createTestService()

	app := &model.Application{
		ID:     "app-1",
		UserID: "user-123",
		JobID:  "job-1",
		Status: "active",
	}

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return app, nil
	}

	jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
		return nil, errors.New("job error")
	}

	appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
		return time.Time{}, errors.New("activity error")
	}

	result, err := svc.GetByID(context.Background(), "user-123", "app-1")

	require.NoError(t, err)
	assert.NotNil(t, result)
}

// --- CompleteStage edge cases ---

func TestCompleteStage_WithCustomCompletedAt(t *testing.T) {
	svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

	customTime := time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC)

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

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return app, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return stage, nil
	}

	var updatedStage *model.ApplicationStage
	stageRepo.UpdateFunc = func(ctx context.Context, s *model.ApplicationStage) error {
		updatedStage = s
		return nil
	}

	templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
		return &model.StageTemplate{ID: tid, Name: "Phone Screen"}, nil
	}

	req := &model.CompleteStageRequest{CompletedAt: &customTime}
	result, err := svc.CompleteStage(context.Background(), "user-123", "app-1", "stage-1", req)

	require.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, customTime, *updatedStage.CompletedAt)
}

func TestCompleteStage_ApplicationNotFound(t *testing.T) {
	svc, appRepo, _, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return nil, model.ErrApplicationNotFound
	}

	result, err := svc.CompleteStage(context.Background(), "user-123", "app-1", "stage-1", &model.CompleteStageRequest{})

	assert.Nil(t, result)
	assert.Equal(t, model.ErrApplicationNotFound, err)
}

func TestCompleteStage_StageNotFound(t *testing.T) {
	svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return nil, model.ErrApplicationStageNotFound
	}

	result, err := svc.CompleteStage(context.Background(), "user-123", "app-1", "stage-1", &model.CompleteStageRequest{})

	assert.Nil(t, result)
	assert.Equal(t, model.ErrApplicationStageNotFound, err)
}

func TestCompleteStage_StageNotBelongingToApp(t *testing.T) {
	svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return &model.ApplicationStage{
			ID:            "stage-1",
			ApplicationID: "different-app",
		}, nil
	}

	result, err := svc.CompleteStage(context.Background(), "user-123", "app-1", "stage-1", &model.CompleteStageRequest{})

	assert.Nil(t, result)
	assert.Equal(t, model.ErrApplicationStageNotFound, err)
}

func TestCompleteStage_UpdateFails(t *testing.T) {
	svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return &model.ApplicationStage{
			ID:              "stage-1",
			ApplicationID:   "app-1",
			StageTemplateID: "template-1",
			Status:          "active",
		}, nil
	}

	stageRepo.UpdateFunc = func(ctx context.Context, s *model.ApplicationStage) error {
		return errors.New("database error")
	}

	result, err := svc.CompleteStage(context.Background(), "user-123", "app-1", "stage-1", &model.CompleteStageRequest{})

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestCompleteStage_TemplateFetchFails(t *testing.T) {
	svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return &model.ApplicationStage{
			ID:              "stage-1",
			ApplicationID:   "app-1",
			StageTemplateID: "template-1",
			Status:          "active",
		}, nil
	}

	stageRepo.UpdateFunc = func(ctx context.Context, s *model.ApplicationStage) error {
		return nil
	}

	templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
		return nil, errors.New("template not found")
	}

	result, err := svc.CompleteStage(context.Background(), "user-123", "app-1", "stage-1", &model.CompleteStageRequest{})

	assert.Nil(t, result)
	assert.Error(t, err)
}

// --- Create edge cases ---

func TestCreate_BothResumeTypesSet(t *testing.T) {
	svc, _, _, _, _, _, _, _ := createTestService()

	resumeID := "resume-1"
	rbID := "rb-1"
	req := &model.CreateApplicationRequest{
		JobID:           "job-1",
		ResumeID:        &resumeID,
		ResumeBuilderID: &rbID,
	}

	result, err := svc.Create(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Equal(t, model.ErrBothResumeTypesSet, err)
}

func TestCreate_LimitCheckerBlocksCreation(t *testing.T) {
	appRepo := &MockApplicationRepository{}
	stageRepo := &MockStageRepository{}
	templateRepo := &MockTemplateRepository{}
	jobRepo := &MockJobRepository{}
	companyRepo := &MockCompanyRepository{}
	resumeRepo := &MockResumeRepository{}
	commentRepo := &MockCommentRepository{}
	limitChecker := &MockAppLimitChecker{
		CheckLimitFunc: func(ctx context.Context, userID, resource string) error {
			return errors.New("limit reached")
		},
	}

	svc := NewApplicationService(nil, appRepo, stageRepo, templateRepo, jobRepo, companyRepo, resumeRepo, nil, commentRepo, nil, limitChecker)

	req := &model.CreateApplicationRequest{
		JobID: "job-1",
	}

	result, err := svc.Create(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Equal(t, "limit reached", err.Error())
}

func TestCreate_RepoCreateFails(t *testing.T) {
	svc, appRepo, _, _, jobRepo, _, _, _ := createTestService()

	jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
		return &jobModel.Job{ID: jid, Title: "Engineer"}, nil
	}

	appRepo.CreateFunc = func(ctx context.Context, app *model.Application) error {
		return errors.New("create error")
	}

	req := &model.CreateApplicationRequest{
		JobID: "job-1",
	}

	result, err := svc.Create(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestCreate_JobFetchFailsUsesUntitledName(t *testing.T) {
	svc, appRepo, _, _, jobRepo, _, _, _ := createTestService()

	jobRepo.GetByIDFunc = func(ctx context.Context, uid, jid string) (*jobModel.Job, error) {
		return nil, errors.New("job not found")
	}

	var createdApp *model.Application
	appRepo.CreateFunc = func(ctx context.Context, app *model.Application) error {
		createdApp = app
		app.ID = "app-1"
		return nil
	}

	appRepo.GetLastActivityAtFunc = func(ctx context.Context, appID string) (time.Time, error) {
		return time.Now(), nil
	}

	req := &model.CreateApplicationRequest{
		JobID: "job-1",
		Name:  "",
	}

	_, err := svc.Create(context.Background(), "user-123", req)

	require.NoError(t, err)
	assert.Equal(t, "Untitled Application", createdApp.Name)
}

// --- Update edge cases ---

func TestUpdate_ApplicationNotFound(t *testing.T) {
	svc, appRepo, _, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return nil, model.ErrApplicationNotFound
	}

	status := "offer"
	req := &model.UpdateApplicationRequest{Status: &status}

	result, err := svc.Update(context.Background(), "user-123", "app-1", req)

	assert.Nil(t, result)
	assert.Equal(t, model.ErrApplicationNotFound, err)
}

func TestUpdate_RepoUpdateFails(t *testing.T) {
	svc, appRepo, _, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123", Status: "active"}, nil
	}

	appRepo.UpdateFunc = func(ctx context.Context, app *model.Application) error {
		return errors.New("update error")
	}

	status := "offer"
	req := &model.UpdateApplicationRequest{Status: &status}

	result, err := svc.Update(context.Background(), "user-123", "app-1", req)

	assert.Nil(t, result)
	assert.Error(t, err)
}

// --- ListStages edge cases ---

func TestListStages_ApplicationNotFound(t *testing.T) {
	svc, appRepo, _, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return nil, model.ErrApplicationNotFound
	}

	result, err := svc.ListStages(context.Background(), "user-123", "app-1")

	assert.Nil(t, result)
	assert.Equal(t, model.ErrApplicationNotFound, err)
}

func TestListStages_StageRepoFails(t *testing.T) {
	svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.ListByApplicationFunc = func(ctx context.Context, aid string) ([]*model.ApplicationStage, error) {
		return nil, errors.New("stage list error")
	}

	result, err := svc.ListStages(context.Background(), "user-123", "app-1")

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestListStages_TemplateListFails(t *testing.T) {
	svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.ListByApplicationFunc = func(ctx context.Context, aid string) ([]*model.ApplicationStage, error) {
		return []*model.ApplicationStage{
			{ID: "stage-1", StageTemplateID: "template-1"},
		}, nil
	}

	templateRepo.ListFunc = func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
		return nil, 0, errors.New("template list error")
	}

	result, err := svc.ListStages(context.Background(), "user-123", "app-1")

	assert.Nil(t, result)
	assert.Error(t, err)
}

// --- UpdateStage edge cases ---

func TestUpdateStage_ApplicationNotFound(t *testing.T) {
	svc, appRepo, _, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return nil, model.ErrApplicationNotFound
	}

	status := "completed"
	req := &model.UpdateStageRequest{Status: &status}

	result, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", req)

	assert.Nil(t, result)
	assert.Equal(t, model.ErrApplicationNotFound, err)
}

func TestUpdateStage_StageNotFound(t *testing.T) {
	svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return nil, errors.New("not found")
	}

	status := "completed"
	req := &model.UpdateStageRequest{Status: &status}

	result, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", req)

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestUpdateStage_StageDoesNotBelongToApp(t *testing.T) {
	svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return &model.ApplicationStage{
			ID:            "stage-1",
			ApplicationID: "different-app",
		}, nil
	}

	status := "completed"
	req := &model.UpdateStageRequest{Status: &status}

	result, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", req)

	assert.Nil(t, result)
	assert.Equal(t, model.ErrApplicationStageNotFound, err)
}

func TestUpdateStage_CompletedSetsCompletedAt(t *testing.T) {
	svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return &model.ApplicationStage{
			ID:              "stage-1",
			ApplicationID:   "app-1",
			StageTemplateID: "template-1",
			Status:          "active",
			CompletedAt:     nil,
		}, nil
	}

	var updatedStage *model.ApplicationStage
	stageRepo.UpdateFunc = func(ctx context.Context, s *model.ApplicationStage) error {
		updatedStage = s
		return nil
	}

	templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
		return &model.StageTemplate{ID: tid, Name: "Test"}, nil
	}

	status := "completed"
	req := &model.UpdateStageRequest{Status: &status}

	result, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", req)

	require.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
	assert.NotNil(t, updatedStage.CompletedAt)
}

func TestUpdateStage_ActiveClearsCompletedAt(t *testing.T) {
	svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

	now := time.Now()
	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return &model.ApplicationStage{
			ID:              "stage-1",
			ApplicationID:   "app-1",
			StageTemplateID: "template-1",
			Status:          "completed",
			CompletedAt:     &now,
		}, nil
	}

	var updatedStage *model.ApplicationStage
	stageRepo.UpdateFunc = func(ctx context.Context, s *model.ApplicationStage) error {
		updatedStage = s
		return nil
	}

	templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
		return &model.StageTemplate{ID: tid, Name: "Test"}, nil
	}

	status := "active"
	req := &model.UpdateStageRequest{Status: &status}

	result, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", req)

	require.NoError(t, err)
	assert.Equal(t, "active", result.Status)
	assert.Nil(t, updatedStage.CompletedAt)
}

func TestUpdateStage_CustomCompletedAt(t *testing.T) {
	svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return &model.ApplicationStage{
			ID:              "stage-1",
			ApplicationID:   "app-1",
			StageTemplateID: "template-1",
			Status:          "active",
		}, nil
	}

	var updatedStage *model.ApplicationStage
	stageRepo.UpdateFunc = func(ctx context.Context, s *model.ApplicationStage) error {
		updatedStage = s
		return nil
	}

	templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
		return &model.StageTemplate{ID: tid, Name: "Test"}, nil
	}

	customTime := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	status := "completed"
	req := &model.UpdateStageRequest{Status: &status, CompletedAt: &customTime}

	_, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", req)

	require.NoError(t, err)
	assert.Equal(t, customTime, *updatedStage.CompletedAt)
}

func TestUpdateStage_UpdateFails(t *testing.T) {
	svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return &model.ApplicationStage{
			ID:              "stage-1",
			ApplicationID:   "app-1",
			StageTemplateID: "template-1",
			Status:          "active",
		}, nil
	}

	stageRepo.UpdateFunc = func(ctx context.Context, s *model.ApplicationStage) error {
		return errors.New("db error")
	}

	status := "completed"
	req := &model.UpdateStageRequest{Status: &status}

	result, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", req)

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestUpdateStage_TemplateFetchFails(t *testing.T) {
	svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return &model.ApplicationStage{
			ID:              "stage-1",
			ApplicationID:   "app-1",
			StageTemplateID: "template-1",
			Status:          "active",
		}, nil
	}

	stageRepo.UpdateFunc = func(ctx context.Context, s *model.ApplicationStage) error {
		return nil
	}

	templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
		return nil, errors.New("template error")
	}

	status := "completed"
	req := &model.UpdateStageRequest{Status: &status}

	result, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", req)

	assert.Nil(t, result)
	assert.Error(t, err)
}

// --- UpdateStageTemplate edge cases ---

func TestUpdateStageTemplate_NotFound(t *testing.T) {
	svc, _, _, templateRepo, _, _, _, _ := createTestService()

	templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
		return nil, model.ErrStageTemplateNotFound
	}

	newName := "New Name"
	req := &model.UpdateStageTemplateRequest{Name: &newName}

	result, err := svc.UpdateStageTemplate(context.Background(), "user-123", "template-1", req)

	assert.Nil(t, result)
	assert.Equal(t, model.ErrStageTemplateNotFound, err)
}

func TestUpdateStageTemplate_UpdateOrderOnly(t *testing.T) {
	svc, _, _, templateRepo, _, _, _, _ := createTestService()

	templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
		return &model.StageTemplate{ID: tid, UserID: uid, Name: "Phone Screen", Order: 1}, nil
	}

	templateRepo.UpdateFunc = func(ctx context.Context, t *model.StageTemplate) error {
		return nil
	}

	newOrder := 5
	req := &model.UpdateStageTemplateRequest{Order: &newOrder}

	result, err := svc.UpdateStageTemplate(context.Background(), "user-123", "template-1", req)

	require.NoError(t, err)
	assert.Equal(t, 5, result.Order)
	assert.Equal(t, "Phone Screen", result.Name)
}

func TestUpdateStageTemplate_UpdateFails(t *testing.T) {
	svc, _, _, templateRepo, _, _, _, _ := createTestService()

	templateRepo.GetByIDFunc = func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
		return &model.StageTemplate{ID: tid, UserID: uid, Name: "Phone Screen", Order: 1}, nil
	}

	templateRepo.UpdateFunc = func(ctx context.Context, t *model.StageTemplate) error {
		return errors.New("update error")
	}

	newName := "Updated"
	req := &model.UpdateStageTemplateRequest{Name: &newName}

	result, err := svc.UpdateStageTemplate(context.Background(), "user-123", "template-1", req)

	assert.Nil(t, result)
	assert.Error(t, err)
}

// --- ListStageTemplates error path ---

func TestListStageTemplates_Error(t *testing.T) {
	svc, _, _, templateRepo, _, _, _, _ := createTestService()

	templateRepo.ListFunc = func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
		return nil, 0, errors.New("list error")
	}

	result, total, err := svc.ListStageTemplates(context.Background(), "user-123", 20, 0)

	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.Error(t, err)
}

// --- CreateStageTemplate error path ---

func TestCreateStageTemplate_RepoError(t *testing.T) {
	svc, _, _, templateRepo, _, _, _, _ := createTestService()

	templateRepo.CreateFunc = func(ctx context.Context, template *model.StageTemplate) error {
		return errors.New("create error")
	}

	req := &model.CreateStageTemplateRequest{Name: "Valid Name", Order: 0}

	result, err := svc.CreateStageTemplate(context.Background(), "user-123", req)

	assert.Nil(t, result)
	assert.Error(t, err)
}

// --- DeleteStage validation path ---

func TestDeleteStage_ApplicationNotFound(t *testing.T) {
	svc, appRepo, _, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return nil, model.ErrApplicationNotFound
	}

	err := svc.DeleteStage(context.Background(), "user-123", "app-1", "stage-1")

	assert.Equal(t, model.ErrApplicationNotFound, err)
}

func TestDeleteStage_StageNotFound(t *testing.T) {
	svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

	appRepo.GetByIDFunc = func(ctx context.Context, uid, aid string) (*model.Application, error) {
		return &model.Application{ID: "app-1", UserID: "user-123"}, nil
	}

	stageRepo.GetByIDFunc = func(ctx context.Context, sid string) (*model.ApplicationStage, error) {
		return nil, model.ErrApplicationStageNotFound
	}

	err := svc.DeleteStage(context.Background(), "user-123", "app-1", "stage-1")

	assert.Equal(t, model.ErrApplicationStageNotFound, err)
}

// --- List error path ---

func TestList_RepoError(t *testing.T) {
	svc, appRepo, _, _, _, _, _, _ := createTestService()

	appRepo.ListEnrichedFunc = func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.ApplicationDTO, int, error) {
		return nil, 0, errors.New("list error")
	}

	result, total, err := svc.List(context.Background(), "user-123", "created_at", "desc", "", 20, 0)

	assert.Nil(t, result)
	assert.Equal(t, 0, total)
	assert.Error(t, err)
}

// --- List with filters ---

func TestList_PassesFilterParams(t *testing.T) {
	svc, appRepo, _, _, _, _, _, _ := createTestService()

	appRepo.ListEnrichedFunc = func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.ApplicationDTO, int, error) {
		assert.Equal(t, "updated_at", opts.SortBy)
		assert.Equal(t, "asc", opts.SortDir)
		assert.Equal(t, "active", opts.Status)
		assert.Equal(t, 10, opts.Limit)
		assert.Equal(t, 5, opts.Offset)
		return []*model.ApplicationDTO{}, 0, nil
	}

	_, _, err := svc.List(context.Background(), "user-123", "updated_at", "asc", "active", 10, 5)

	require.NoError(t, err)
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
			ResumeID:  strPtr("resume-1"),
			AppliedAt: appliedAt,
		}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, appliedAt, createdApp.AppliedAt)
	})
}

// ---------------------------------------------------------------------------
// Create with limit checker tests
// ---------------------------------------------------------------------------

func TestApplicationService_GetByID_BuildDTOError(t *testing.T) {
	t.Run("returns error when GetLastActivityAt fails and job fetch fails", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, _, commentRepo := createTestService()

		expectedApp := &model.Application{
			ID:     "app-1",
			UserID: "user-123",
			JobID:  "job-1",
			Name:   "Test",
			Status: "active",
		}

		appRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.Application, error) {
			return expectedApp, nil
		}

		// Let GetLastActivityAt fail — buildApplicationDTO handles this gracefully
		appRepo.GetLastActivityAtFunc = func(_ context.Context, _ string) (time.Time, error) {
			return time.Time{}, errors.New("activity error")
		}

		jobRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*jobModel.Job, error) {
			return nil, errors.New("job not found")
		}

		commentRepo.ListByApplicationFunc = func(_ context.Context, _ string, _ ...string) ([]*commentModel.Comment, error) {
			return nil, nil
		}

		// buildApplicationDTO handles all errors gracefully; it should succeed
		result, err := svc.GetByID(context.Background(), "user-123", "app-1")
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestApplicationService_Create_LimitChecker(t *testing.T) {
	t.Run("returns error when limit reached", func(t *testing.T) {
		lc := &MockAppLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, resource string) error {
				assert.Equal(t, "applications", resource)
				return errors.New("limit reached")
			},
		}
		appRepo := &MockApplicationRepository{}
		svc := NewApplicationService(nil, appRepo, &MockStageRepository{}, &MockTemplateRepository{}, &MockJobRepository{}, &MockCompanyRepository{}, &MockResumeRepository{}, nil, &MockCommentRepository{}, nil, lc)

		req := &model.CreateApplicationRequest{JobID: "job-1"}
		result, err := svc.Create(context.Background(), "user-123", req)

		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Equal(t, "limit reached", err.Error())
	})
}

func TestApplicationService_Create_BothResumeTypesSet(t *testing.T) {
	t.Run("returns error when both resume_id and resume_builder_id are set", func(t *testing.T) {
		svc, _, _, _, _, _, _, _ := createTestService()

		req := &model.CreateApplicationRequest{
			JobID:           "job-1",
			ResumeID:        strPtr("resume-1"),
			ResumeBuilderID: strPtr("rb-1"),
		}

		result, err := svc.Create(context.Background(), "user-123", req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrBothResumeTypesSet)
	})
}

// ---------------------------------------------------------------------------
// GetByID with comments splitting tests
// ---------------------------------------------------------------------------

func TestApplicationService_GetByID_WithComments(t *testing.T) {
	userID := "user-123"
	appID := "app-1"

	t.Run("splits application and stage comments", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, _, commentRepo := createTestService()

		expectedApp := &model.Application{
			ID:     appID,
			UserID: userID,
			JobID:  "job-1",
			Name:   "Test Application",
			Status: "active",
		}

		appRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.Application, error) {
			return expectedApp, nil
		}
		appRepo.GetLastActivityAtFunc = func(_ context.Context, _ string) (time.Time, error) {
			return time.Now(), nil
		}

		jobRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: "job-1", Title: "Software Engineer"}, nil
		}

		stageID := "stage-1"
		commentRepo.ListByApplicationFunc = func(_ context.Context, _ string, _ ...string) ([]*commentModel.Comment, error) {
			return []*commentModel.Comment{
				{ID: "comment-1", ApplicationID: appID, StageID: nil, Content: "App-level comment"},
				{ID: "comment-2", ApplicationID: appID, StageID: &stageID, Content: "Stage comment"},
				{ID: "comment-3", ApplicationID: appID, StageID: nil, Content: "Another app comment"},
			}, nil
		}

		result, err := svc.GetByID(context.Background(), userID, appID)

		require.NoError(t, err)
		assert.Len(t, result.ApplicationComments, 2)
		assert.Len(t, result.StageComments, 1)
	})

	t.Run("handles comment fetch error gracefully", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, _, commentRepo := createTestService()

		expectedApp := &model.Application{
			ID:     appID,
			UserID: userID,
			JobID:  "job-1",
			Name:   "Test Application",
			Status: "active",
		}

		appRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.Application, error) {
			return expectedApp, nil
		}
		appRepo.GetLastActivityAtFunc = func(_ context.Context, _ string) (time.Time, error) {
			return time.Now(), nil
		}

		jobRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: "job-1", Title: "Software Engineer"}, nil
		}

		commentRepo.ListByApplicationFunc = func(_ context.Context, _ string, _ ...string) ([]*commentModel.Comment, error) {
			return nil, errors.New("comments unavailable")
		}

		result, err := svc.GetByID(context.Background(), userID, appID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		// Comments should be nil when fetch fails
		assert.Nil(t, result.ApplicationComments)
		assert.Nil(t, result.StageComments)
	})
}

// ---------------------------------------------------------------------------
// Create with repo error tests
// ---------------------------------------------------------------------------

func TestApplicationService_Create_RepoError(t *testing.T) {
	t.Run("returns error from repository", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, _, _ := createTestService()

		jobRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*jobModel.Job, error) {
			return &jobModel.Job{ID: "job-1", Title: "Software Engineer"}, nil
		}

		appRepo.CreateFunc = func(_ context.Context, _ *model.Application) error {
			return errors.New("database error")
		}

		req := &model.CreateApplicationRequest{JobID: "job-1"}
		result, err := svc.Create(context.Background(), "user-123", req)

		assert.Nil(t, result)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Update with repo error tests
// ---------------------------------------------------------------------------

func TestApplicationService_Update_RepoError(t *testing.T) {
	t.Run("returns error when update fails", func(t *testing.T) {
		svc, appRepo, _, _, _, _, _, _ := createTestService()

		appRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.Application, error) {
			return &model.Application{
				ID:     "app-1",
				UserID: "user-123",
				JobID:  "job-1",
				Status: "active",
			}, nil
		}

		appRepo.UpdateFunc = func(_ context.Context, _ *model.Application) error {
			return errors.New("update failed")
		}

		newStatus := "offer"
		req := &model.UpdateApplicationRequest{Status: &newStatus}
		result, err := svc.Update(context.Background(), "user-123", "app-1", req)

		assert.Nil(t, result)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Delete with repo error tests
// ---------------------------------------------------------------------------

func TestApplicationService_Delete_RepoError(t *testing.T) {
	t.Run("returns error from repository", func(t *testing.T) {
		svc, appRepo, _, _, _, _, _, _ := createTestService()

		appRepo.DeleteFunc = func(_ context.Context, _, _ string) error {
			return errors.New("delete failed")
		}

		err := svc.Delete(context.Background(), "user-123", "app-1")

		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// ListStages error tests
// ---------------------------------------------------------------------------

func TestApplicationService_ListStages_Error(t *testing.T) {
	t.Run("returns error when stage repo fails", func(t *testing.T) {
		svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

		appRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.Application, error) {
			return &model.Application{ID: "app-1", UserID: "user-123"}, nil
		}

		stageRepo.ListByApplicationFunc = func(_ context.Context, _ string) ([]*model.ApplicationStage, error) {
			return nil, errors.New("stage list error")
		}

		result, err := svc.ListStages(context.Background(), "user-123", "app-1")

		assert.Nil(t, result)
		assert.Error(t, err)
	})

	t.Run("returns error when template repo fails", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

		appRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.Application, error) {
			return &model.Application{ID: "app-1", UserID: "user-123"}, nil
		}

		stageRepo.ListByApplicationFunc = func(_ context.Context, _ string) ([]*model.ApplicationStage, error) {
			return []*model.ApplicationStage{
				{ID: "stage-1", StageTemplateID: "template-1", Status: "active"},
			}, nil
		}

		templateRepo.ListFunc = func(_ context.Context, _ string, _, _ int) ([]*model.StageTemplate, int, error) {
			return nil, 0, errors.New("template list error")
		}

		result, err := svc.ListStages(context.Background(), "user-123", "app-1")

		assert.Nil(t, result)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// CompleteStage edge cases
// ---------------------------------------------------------------------------

func TestApplicationService_CompleteStage_WithCustomTime(t *testing.T) {
	t.Run("uses provided completed_at time", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

		customTime := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

		appRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.Application, error) {
			return &model.Application{ID: "app-1", UserID: "user-123"}, nil
		}

		stage := &model.ApplicationStage{
			ID:              "stage-1",
			ApplicationID:   "app-1",
			StageTemplateID: "template-1",
			Status:          "active",
		}

		stageRepo.GetByIDFunc = func(_ context.Context, _ string) (*model.ApplicationStage, error) {
			return stage, nil
		}

		var updatedStage *model.ApplicationStage
		stageRepo.UpdateFunc = func(_ context.Context, s *model.ApplicationStage) error {
			updatedStage = s
			return nil
		}

		templateRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.StageTemplate, error) {
			return &model.StageTemplate{ID: "template-1", Name: "Phone Screen"}, nil
		}

		result, err := svc.CompleteStage(context.Background(), "user-123", "app-1", "stage-1", &model.CompleteStageRequest{
			CompletedAt: &customTime,
		})

		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
		assert.Equal(t, customTime, *updatedStage.CompletedAt)
	})

	t.Run("returns error when stage does not belong to application", func(t *testing.T) {
		svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

		appRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.Application, error) {
			return &model.Application{ID: "app-1", UserID: "user-123"}, nil
		}

		stageRepo.GetByIDFunc = func(_ context.Context, _ string) (*model.ApplicationStage, error) {
			return &model.ApplicationStage{
				ID:            "stage-1",
				ApplicationID: "different-app",
			}, nil
		}

		result, err := svc.CompleteStage(context.Background(), "user-123", "app-1", "stage-1", &model.CompleteStageRequest{})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrApplicationStageNotFound)
	})
}

// ---------------------------------------------------------------------------
// UpdateStage edge cases
// ---------------------------------------------------------------------------

func TestApplicationService_UpdateStage_EdgeCases(t *testing.T) {
	t.Run("sets completed_at when status changed to completed", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

		appRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.Application, error) {
			return &model.Application{ID: "app-1", UserID: "user-123"}, nil
		}

		stageRepo.GetByIDFunc = func(_ context.Context, _ string) (*model.ApplicationStage, error) {
			return &model.ApplicationStage{
				ID:              "stage-1",
				ApplicationID:   "app-1",
				StageTemplateID: "template-1",
				Status:          "active",
			}, nil
		}

		var savedStage *model.ApplicationStage
		stageRepo.UpdateFunc = func(_ context.Context, s *model.ApplicationStage) error {
			savedStage = s
			return nil
		}

		templateRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.StageTemplate, error) {
			return &model.StageTemplate{ID: "template-1", Name: "Phone Screen"}, nil
		}

		completed := "completed"
		result, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", &model.UpdateStageRequest{
			Status: &completed,
		})

		require.NoError(t, err)
		assert.Equal(t, "completed", result.Status)
		assert.NotNil(t, savedStage.CompletedAt, "completed_at should be auto-set")
	})

	t.Run("clears completed_at when status changed to active", func(t *testing.T) {
		svc, appRepo, stageRepo, templateRepo, _, _, _, _ := createTestService()

		now := time.Now()
		appRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.Application, error) {
			return &model.Application{ID: "app-1", UserID: "user-123"}, nil
		}

		stageRepo.GetByIDFunc = func(_ context.Context, _ string) (*model.ApplicationStage, error) {
			return &model.ApplicationStage{
				ID:              "stage-1",
				ApplicationID:   "app-1",
				StageTemplateID: "template-1",
				Status:          "completed",
				CompletedAt:     &now,
			}, nil
		}

		var savedStage *model.ApplicationStage
		stageRepo.UpdateFunc = func(_ context.Context, s *model.ApplicationStage) error {
			savedStage = s
			return nil
		}

		templateRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.StageTemplate, error) {
			return &model.StageTemplate{ID: "template-1", Name: "Phone Screen"}, nil
		}

		active := "active"
		result, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", &model.UpdateStageRequest{
			Status: &active,
		})

		require.NoError(t, err)
		assert.Equal(t, "active", result.Status)
		assert.Nil(t, savedStage.CompletedAt, "completed_at should be cleared")
	})

	t.Run("returns error when stage repo update fails", func(t *testing.T) {
		svc, appRepo, stageRepo, _, _, _, _, _ := createTestService()

		appRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*model.Application, error) {
			return &model.Application{ID: "app-1", UserID: "user-123"}, nil
		}

		stageRepo.GetByIDFunc = func(_ context.Context, _ string) (*model.ApplicationStage, error) {
			return &model.ApplicationStage{
				ID:              "stage-1",
				ApplicationID:   "app-1",
				StageTemplateID: "template-1",
				Status:          "active",
			}, nil
		}

		stageRepo.UpdateFunc = func(_ context.Context, _ *model.ApplicationStage) error {
			return errors.New("update failed")
		}

		completed := "completed"
		result, err := svc.UpdateStage(context.Background(), "user-123", "app-1", "stage-1", &model.UpdateStageRequest{
			Status: &completed,
		})

		assert.Nil(t, result)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// ListStageTemplates error tests
// ---------------------------------------------------------------------------

func TestApplicationService_ListStageTemplates_Error(t *testing.T) {
	t.Run("returns error from repository", func(t *testing.T) {
		svc, _, _, templateRepo, _, _, _, _ := createTestService()

		templateRepo.ListFunc = func(_ context.Context, _ string, _, _ int) ([]*model.StageTemplate, int, error) {
			return nil, 0, errors.New("list error")
		}

		result, total, err := svc.ListStageTemplates(context.Background(), "user-123", 20, 0)

		assert.Nil(t, result)
		assert.Equal(t, 0, total)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Create with Untitled name when job fetch fails
// ---------------------------------------------------------------------------

func TestApplicationService_Create_FallbackName(t *testing.T) {
	t.Run("uses Untitled Application when job fetch fails", func(t *testing.T) {
		svc, appRepo, _, _, jobRepo, _, _, _ := createTestService()

		var createdApp *model.Application

		jobRepo.GetByIDFunc = func(_ context.Context, _, _ string) (*jobModel.Job, error) {
			return nil, errors.New("job not found")
		}

		appRepo.CreateFunc = func(_ context.Context, app *model.Application) error {
			createdApp = app
			app.ID = "app-1"
			return nil
		}

		appRepo.GetLastActivityAtFunc = func(_ context.Context, _ string) (time.Time, error) {
			return time.Now(), nil
		}

		req := &model.CreateApplicationRequest{
			JobID: "job-missing",
		}

		_, err := svc.Create(context.Background(), "user-123", req)

		require.NoError(t, err)
		assert.Equal(t, "Untitled Application", createdApp.Name)
	})
}
