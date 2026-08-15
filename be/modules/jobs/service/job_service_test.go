package service

import (
	"context"
	"errors"
	"testing"
	"time"

	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCompanyRepository implements companyPorts.CompanyRepository for testing
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

// MockCommentRepository implements commentPorts.CommentRepository for testing
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

// newServiceWithTemplates builds a JobService wired with a template repo that
// exposes a single first pipeline column, so Create can place a card in the
// default column. Pass extra repos as needed for a given test.
func newServiceWithTemplates(repo ports.JobRepository, templateRepo ports.StageTemplateRepository) *JobService {
	return NewJobService(nil, repo, nil, templateRepo, defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, nil, nil)
}

func TestJobService_Create(t *testing.T) {
	userID := "user-123"
	tmplRepo := func() *MockStageTemplateRepository { return singleColumnTemplateRepo("stage-wishlist", "Wishlist") }

	t.Run("creates job successfully", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, job *model.Job) error {
				job.ID = "job-1"
				job.CreatedAt = time.Now()
				job.UpdatedAt = time.Now()
				return nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, tmplRepo())
		req := &model.CreateJobRequest{Title: "Software Engineer"}

		result, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "job-1", result.ID)
		assert.Equal(t, "Software Engineer", result.Title)
	})

	t.Run("returns error for empty title", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := newServiceWithTemplates(mockRepo, tmplRepo())
		req := &model.CreateJobRequest{Title: "   "}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrJobTitleRequired, err)
	})

	t.Run("trims whitespace from title", func(t *testing.T) {
		var createdJob *model.Job

		mockRepo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, job *model.Job) error {
				createdJob = job
				job.ID = "job-1"
				return nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, tmplRepo())
		req := &model.CreateJobRequest{Title: "  Software Engineer  "}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "Software Engineer", createdJob.Title)
	})

	t.Run("creates job with optional fields", func(t *testing.T) {
		var createdJob *model.Job
		companyID := "company-1"
		source := "LinkedIn"
		url := "https://linkedin.com/jobs/123"
		notes := "Interesting role"

		mockRepo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, job *model.Job) error {
				createdJob = job
				job.ID = "job-1"
				return nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, tmplRepo())
		req := &model.CreateJobRequest{
			Title:     "Software Engineer",
			CompanyID: &companyID,
			Source:    &source,
			URL:       &url,
			Notes:     &notes,
		}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, &companyID, createdJob.CompanyID)
		assert.Equal(t, &source, createdJob.Source)
		assert.Equal(t, &url, createdJob.URL)
		assert.Equal(t, &notes, createdJob.Notes)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockRepo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, job *model.Job) error {
				return expectedError
			},
		}

		svc := newServiceWithTemplates(mockRepo, tmplRepo())
		req := &model.CreateJobRequest{Title: "Software Engineer"}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})

	t.Run("places card in the first pipeline column with no applied_at", func(t *testing.T) {
		var createdJob *model.Job
		mockRepo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, job *model.Job) error {
				createdJob = job
				job.ID = "job-1"
				return nil
			},
		}
		svc := newServiceWithTemplates(mockRepo, tmplRepo())
		req := &model.CreateJobRequest{Title: "Test Job"}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		require.NotNil(t, createdJob.CurrentStageTemplateID)
		assert.Equal(t, "stage-wishlist", *createdJob.CurrentStageTemplateID)
		assert.Nil(t, createdJob.AppliedAt)
	})

	t.Run("explicit placement past first column stamps applied_at", func(t *testing.T) {
		var createdJob *model.Job
		mockRepo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, job *model.Job) error {
				createdJob = job
				job.ID = "job-1"
				return nil
			},
		}
		templateRepo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, templateID string) (*model.StageTemplate, error) {
				return &model.StageTemplate{ID: templateID, UserID: uid, Name: "Applied", Order: 1}, nil
			},
		}
		svc := newServiceWithTemplates(mockRepo, templateRepo)
		appliedID := "stage-applied"
		req := &model.CreateJobRequest{Title: "Test Job", StageTemplateID: &appliedID}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		require.NotNil(t, createdJob.CurrentStageTemplateID)
		assert.Equal(t, appliedID, *createdJob.CurrentStageTemplateID)
		assert.NotNil(t, createdJob.AppliedAt)
	})

	t.Run("returns error for both resume types set", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := newServiceWithTemplates(mockRepo, tmplRepo())
		resumeID := "resume-1"
		resumeBuilderID := "rb-1"
		req := &model.CreateJobRequest{
			Title:           "Test Job",
			ResumeID:        &resumeID,
			ResumeBuilderID: &resumeBuilderID,
		}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrBothResumeTypesSet, err)
	})

	t.Run("returns error when user has no pipeline columns", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		emptyTemplates := &MockStageTemplateRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
				return nil, 0, nil
			},
		}
		svc := newServiceWithTemplates(mockRepo, emptyTemplates)
		req := &model.CreateJobRequest{Title: "Test Job"}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
	})
}

func TestJobService_GetByID(t *testing.T) {
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
				assert.Equal(t, userID, uid)
				assert.Equal(t, jobID, jid)
				return expectedJob, nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		result, err := svc.GetByID(context.Background(), userID, jobID)

		require.NoError(t, err)
		assert.Equal(t, expectedJob.ID, result.ID)
		assert.Equal(t, expectedJob.Title, result.Title)
	})

	t.Run("returns error when job not found", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		result, err := svc.GetByID(context.Background(), userID, jobID)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrJobNotFound, err)
	})
}

func TestJobService_List(t *testing.T) {
	userID := "user-123"

	t.Run("returns jobs list", func(t *testing.T) {
		expectedJobs := []*model.JobDTO{
			{ID: "job-1", Title: "Software Engineer"},
			{ID: "job-2", Title: "Product Manager"},
		}

		mockRepo := &MockJobRepository{
			ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, 20, opts.Limit)
				assert.Equal(t, 0, opts.Offset)
				return expectedJobs, 2, nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		result, total, err := svc.List(context.Background(), userID, &ports.ListOptions{Limit: 20, Offset: 0, Status: ""})

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 2, total)
	})

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
				return []*model.JobDTO{}, 0, nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		result, total, err := svc.List(context.Background(), userID, &ports.ListOptions{Limit: 20, Offset: 0})

		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, 0, total)
	})

	t.Run("passes sort parameters", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
				assert.Equal(t, "title", opts.SortBy)
				assert.Equal(t, "asc", opts.SortDir)
				return []*model.JobDTO{}, 0, nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		_, _, err := svc.List(context.Background(), userID, &ports.ListOptions{Limit: 20, Offset: 0, SortBy: "title", SortDir: "asc"})

		require.NoError(t, err)
	})

	t.Run("passes archived filter values through untouched", func(t *testing.T) {
		for _, status := range []string{"", "active", "archived", "all"} {
			status := status
			var seen string
			mockRepo := &MockJobRepository{
				ListFunc: func(ctx context.Context, uid string, opts *ports.ListOptions) ([]*model.JobDTO, int, error) {
					seen = opts.Status
					return []*model.JobDTO{}, 0, nil
				},
			}
			svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
			_, _, err := svc.List(context.Background(), userID, &ports.ListOptions{Limit: 20, Status: status})
			require.NoError(t, err, "status=%q should not cause an error", status)
			assert.Equal(t, status, seen, "service must pass the filter through unchanged")
		}
	})
}

func TestJobService_Update(t *testing.T) {
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

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		newTitle := "New Title"
		req := &model.UpdateJobRequest{Title: &newTitle}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		require.NoError(t, err)
		assert.Equal(t, "New Title", result.Title)
	})

	t.Run("returns error for empty title", func(t *testing.T) {
		existingJob := &model.Job{ID: jobID, UserID: userID, Title: "Old Title"}

		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob, nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		emptyTitle := "   "
		req := &model.UpdateJobRequest{Title: &emptyTitle}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrJobTitleRequired, err)
	})

	t.Run("archives a job", func(t *testing.T) {
		existingJob := &model.Job{ID: jobID, UserID: userID, Title: "Job Title"}
		var savedJob *model.Job
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob, nil
			},
			UpdateFunc: func(ctx context.Context, job *model.Job) error {
				savedJob = job
				return nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		archive := true
		req := &model.UpdateJobRequest{IsArchived: &archive}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		require.NoError(t, err)
		assert.True(t, savedJob.IsArchived)
		assert.True(t, result.IsArchived)
	})

	t.Run("unarchives a job", func(t *testing.T) {
		existingJob := &model.Job{ID: jobID, UserID: userID, Title: "Job Title", IsArchived: true}
		var savedJob *model.Job
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob, nil
			},
			UpdateFunc: func(ctx context.Context, job *model.Job) error {
				savedJob = job
				return nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		unarchive := false
		req := &model.UpdateJobRequest{IsArchived: &unarchive}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		require.NoError(t, err)
		assert.False(t, savedJob.IsArchived)
		assert.False(t, result.IsArchived)
	})

	t.Run("returns error when job not found", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		newTitle := "New Title"
		req := &model.UpdateJobRequest{Title: &newTitle}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrJobNotFound, err)
	})
}

func TestJobService_Delete(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	t.Run("deletes job successfully", func(t *testing.T) {
		var deletedJobID string

		mockRepo := &MockJobRepository{
			DeleteFunc: func(ctx context.Context, uid, jid string) error {
				deletedJobID = jid
				return nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		err := svc.Delete(context.Background(), userID, jobID)

		require.NoError(t, err)
		assert.Equal(t, jobID, deletedJobID)
	})

	t.Run("returns error when job not found", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			DeleteFunc: func(ctx context.Context, uid, jid string) error {
				return model.ErrJobNotFound
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		err := svc.Delete(context.Background(), userID, jobID)

		assert.Equal(t, model.ErrJobNotFound, err)
	})
}

// MockCacheInvalidator implements CacheInvalidator for testing
type MockCacheInvalidator struct {
	InvalidateByJobFunc func(ctx context.Context, jobID string) error
	CalledWith          string
	CallCount           int
}

func (m *MockCacheInvalidator) InvalidateByJob(ctx context.Context, jobID string) error {
	m.CalledWith = jobID
	m.CallCount++
	if m.InvalidateByJobFunc != nil {
		return m.InvalidateByJobFunc(ctx, jobID)
	}
	return nil
}

func TestJobService_Update_CacheInvalidation(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	existingJob := func() *model.Job {
		return &model.Job{ID: jobID, UserID: userID, Title: "Job Title"}
	}

	newSvc := func(mockRepo *MockJobRepository, cache CacheInvalidator) *JobService {
		return NewJobService(nil, mockRepo, nil, &MockStageTemplateRepository{}, defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, nil, cache)
	}

	t.Run("invalidates cache when description changes", func(t *testing.T) {
		cache := &MockCacheInvalidator{}
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob(), nil
			},
			UpdateFunc: func(ctx context.Context, job *model.Job) error { return nil },
		}

		svc := newSvc(mockRepo, cache)
		desc := "New description"
		req := &model.UpdateJobRequest{Description: &desc}

		_, err := svc.Update(context.Background(), userID, jobID, req)

		require.NoError(t, err)
		assert.Equal(t, 1, cache.CallCount)
		assert.Equal(t, jobID, cache.CalledWith)
	})

	t.Run("does not invalidate cache when description unchanged", func(t *testing.T) {
		cache := &MockCacheInvalidator{}
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob(), nil
			},
			UpdateFunc: func(ctx context.Context, job *model.Job) error { return nil },
		}

		svc := newSvc(mockRepo, cache)
		newTitle := "New Title"
		req := &model.UpdateJobRequest{Title: &newTitle}

		_, err := svc.Update(context.Background(), userID, jobID, req)

		require.NoError(t, err)
		assert.Equal(t, 0, cache.CallCount)
	})

	t.Run("invalidation error does not break update", func(t *testing.T) {
		cache := &MockCacheInvalidator{
			InvalidateByJobFunc: func(ctx context.Context, jobID string) error {
				return errors.New("cache unavailable")
			},
		}
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob(), nil
			},
			UpdateFunc: func(ctx context.Context, job *model.Job) error { return nil },
		}

		svc := newSvc(mockRepo, cache)
		desc := "New description"
		req := &model.UpdateJobRequest{Description: &desc}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestJobService_Delete_CacheInvalidation(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	newSvc := func(mockRepo *MockJobRepository, cache CacheInvalidator) *JobService {
		return NewJobService(nil, mockRepo, nil, &MockStageTemplateRepository{}, defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, nil, cache)
	}

	t.Run("invalidates cache before delete", func(t *testing.T) {
		cache := &MockCacheInvalidator{}
		mockRepo := &MockJobRepository{
			DeleteFunc: func(ctx context.Context, uid, jid string) error { return nil },
		}

		svc := newSvc(mockRepo, cache)
		err := svc.Delete(context.Background(), userID, jobID)

		require.NoError(t, err)
		assert.Equal(t, 1, cache.CallCount)
		assert.Equal(t, jobID, cache.CalledWith)
	})

	t.Run("invalidation error does not break delete", func(t *testing.T) {
		cache := &MockCacheInvalidator{
			InvalidateByJobFunc: func(ctx context.Context, jobID string) error {
				return errors.New("cache unavailable")
			},
		}
		mockRepo := &MockJobRepository{
			DeleteFunc: func(ctx context.Context, uid, jid string) error { return nil },
		}

		svc := newSvc(mockRepo, cache)
		err := svc.Delete(context.Background(), userID, jobID)

		require.NoError(t, err)
	})
}

// MockLimitChecker implements LimitChecker for testing
type MockLimitChecker struct {
	CheckLimitFunc func(ctx context.Context, userID, resource string) error
}

func (m *MockLimitChecker) CheckLimit(ctx context.Context, userID, resource string) error {
	if m.CheckLimitFunc != nil {
		return m.CheckLimitFunc(ctx, userID, resource)
	}
	return nil
}

func TestJobService_Create_LimitChecker(t *testing.T) {
	tmplRepo := func() *MockStageTemplateRepository { return singleColumnTemplateRepo("stage-wishlist", "Wishlist") }

	t.Run("returns error when limit reached", func(t *testing.T) {
		lc := &MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return errors.New("limit reached")
			},
		}
		mockRepo := &MockJobRepository{}
		svc := NewJobService(nil, mockRepo, nil, tmplRepo(), defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, lc, nil)

		result, err := svc.Create(context.Background(), "user-123", &model.CreateJobRequest{Title: "Test"})

		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Equal(t, "limit reached", err.Error())
	})

	t.Run("passes when limit checker allows", func(t *testing.T) {
		lc := &MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, resource string) error {
				assert.Equal(t, "jobs", resource)
				return nil
			},
		}
		mockRepo := &MockJobRepository{
			CreateFunc: func(_ context.Context, job *model.Job) error {
				job.ID = "job-1"
				return nil
			},
		}
		svc := NewJobService(nil, mockRepo, nil, tmplRepo(), defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, lc, nil)

		result, err := svc.Create(context.Background(), "user-123", &model.CreateJobRequest{Title: "Test"})

		require.NoError(t, err)
		assert.Equal(t, "job-1", result.ID)
	})
}

func TestJobService_Create_CompanyValidation(t *testing.T) {
	tmplRepo := func() *MockStageTemplateRepository { return singleColumnTemplateRepo("stage-wishlist", "Wishlist") }

	t.Run("returns error when company not found", func(t *testing.T) {
		companyID := "company-invalid"
		companyRepo := &MockCompanyRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*companyModel.Company, error) {
				return nil, errors.New("company not found")
			},
		}
		mockRepo := &MockJobRepository{}
		svc := NewJobService(nil, mockRepo, nil, tmplRepo(), companyRepo, nil, nil, defaultMockCommentRepo, nil, nil, nil)

		result, err := svc.Create(context.Background(), "user-123", &model.CreateJobRequest{
			Title:     "Test",
			CompanyID: &companyID,
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrCompanyNotFound)
	})

	t.Run("skips company validation when empty company ID", func(t *testing.T) {
		emptyCompanyID := ""
		mockRepo := &MockJobRepository{
			CreateFunc: func(_ context.Context, job *model.Job) error {
				job.ID = "job-1"
				return nil
			},
		}
		svc := newServiceWithTemplates(mockRepo, tmplRepo())

		result, err := svc.Create(context.Background(), "user-123", &model.CreateJobRequest{
			Title:     "Test",
			CompanyID: &emptyCompanyID,
		})

		require.NoError(t, err)
		assert.Equal(t, "job-1", result.ID)
	})
}

func TestJobService_Update_CompanyValidation(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	t.Run("returns error when new company not found", func(t *testing.T) {
		existingJob := &model.Job{ID: jobID, UserID: userID, Title: "Job Title"}
		companyID := "company-invalid"
		companyRepo := &MockCompanyRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*companyModel.Company, error) {
				return nil, errors.New("company not found")
			},
		}
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Job, error) {
				return existingJob, nil
			},
		}
		svc := NewJobService(nil, mockRepo, nil, &MockStageTemplateRepository{}, companyRepo, nil, nil, defaultMockCommentRepo, nil, nil, nil)

		result, err := svc.Update(context.Background(), userID, jobID, &model.UpdateJobRequest{
			CompanyID: &companyID,
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrCompanyNotFound)
	})

	t.Run("allows clearing company ID with empty string", func(t *testing.T) {
		existingJob := &model.Job{ID: jobID, UserID: userID, Title: "Job Title"}
		emptyCompanyID := ""
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Job, error) {
				return existingJob, nil
			},
			UpdateFunc: func(_ context.Context, job *model.Job) error {
				return nil
			},
		}
		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})

		result, err := svc.Update(context.Background(), userID, jobID, &model.UpdateJobRequest{
			CompanyID: &emptyCompanyID,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("updates all optional fields", func(t *testing.T) {
		existingJob := &model.Job{ID: jobID, UserID: userID, Title: "Job Title"}
		source := "LinkedIn"
		url := "https://linkedin.com/jobs/123"
		notes := "Great opportunity"
		desc := "Job description"

		var updatedJob *model.Job
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Job, error) {
				return existingJob, nil
			},
			UpdateFunc: func(_ context.Context, job *model.Job) error {
				updatedJob = job
				return nil
			},
		}
		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})

		result, err := svc.Update(context.Background(), userID, jobID, &model.UpdateJobRequest{
			Source:      &source,
			URL:         &url,
			Notes:       &notes,
			Description: &desc,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, &source, updatedJob.Source)
		assert.Equal(t, &url, updatedJob.URL)
		assert.Equal(t, &notes, updatedJob.Notes)
		assert.Equal(t, &desc, updatedJob.Description)
	})

	t.Run("returns error when repo update fails", func(t *testing.T) {
		existingJob := &model.Job{ID: jobID, UserID: userID, Title: "Job Title"}
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Job, error) {
				return existingJob, nil
			},
			UpdateFunc: func(_ context.Context, _ *model.Job) error {
				return errors.New("update failed")
			},
		}
		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})

		newTitle := "New Title"
		result, err := svc.Update(context.Background(), userID, jobID, &model.UpdateJobRequest{
			Title: &newTitle,
		})

		assert.Nil(t, result)
		assert.Error(t, err)
	})
}

func TestJob_ToDTO(t *testing.T) {
	now := time.Now()
	companyID := "company-1"
	source := "LinkedIn"
	stageID := "stage-wishlist"

	job := &model.Job{
		ID:                     "job-1",
		UserID:                 "user-123",
		CompanyID:              &companyID,
		Title:                  "Software Engineer",
		Source:                 &source,
		IsArchived:             true,
		CurrentStageTemplateID: &stageID,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	dto := job.ToDTO()

	assert.Equal(t, job.ID, dto.ID)
	assert.Equal(t, job.CompanyID, dto.CompanyID)
	assert.Equal(t, job.Title, dto.Title)
	assert.Equal(t, job.Source, dto.Source)
	assert.True(t, dto.IsArchived)
	assert.Equal(t, job.CurrentStageTemplateID, dto.CurrentStageTemplateID)
	assert.Equal(t, job.CreatedAt, dto.CreatedAt)
	assert.Nil(t, dto.CompanyName) // Set by repository
}

func TestJobService_ToggleFavorite(t *testing.T) {
	userID := "user-123"
	jobID := "job-456"

	t.Run("returns true when toggled to favorite", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ToggleFavoriteFunc: func(ctx context.Context, uid, jid string) (bool, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, jobID, jid)
				return true, nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		result, err := svc.ToggleFavorite(context.Background(), userID, jobID)

		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("returns false when toggled off", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ToggleFavoriteFunc: func(ctx context.Context, uid, jid string) (bool, error) {
				return false, nil
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		result, err := svc.ToggleFavorite(context.Background(), userID, jobID)

		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("returns error when job not found", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ToggleFavoriteFunc: func(ctx context.Context, uid, jid string) (bool, error) {
				return false, model.ErrJobNotFound
			},
		}

		svc := newServiceWithTemplates(mockRepo, &MockStageTemplateRepository{})
		_, err := svc.ToggleFavorite(context.Background(), userID, jobID)

		assert.ErrorIs(t, err, model.ErrJobNotFound)
	})
}

func TestJobService_ReorderStageTemplates(t *testing.T) {
	userID := "user-123"

	t.Run("delegates ordered ids to the repository", func(t *testing.T) {
		var gotUser string
		var gotIDs []string
		templateRepo := &MockStageTemplateRepository{
			ReorderFunc: func(ctx context.Context, uid string, orderedIDs []string) error {
				gotUser = uid
				gotIDs = orderedIDs
				return nil
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		err := svc.ReorderStageTemplates(context.Background(), userID, []string{"c", "a", "b"})

		require.NoError(t, err)
		assert.Equal(t, userID, gotUser)
		assert.Equal(t, []string{"c", "a", "b"}, gotIDs)
	})

	t.Run("propagates a reorder mismatch error", func(t *testing.T) {
		templateRepo := &MockStageTemplateRepository{
			ReorderFunc: func(ctx context.Context, uid string, orderedIDs []string) error {
				return model.ErrReorderMismatch
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		err := svc.ReorderStageTemplates(context.Background(), userID, []string{"a"})

		assert.ErrorIs(t, err, model.ErrReorderMismatch)
	})
}
