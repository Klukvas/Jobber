package service

import (
	"context"
	"errors"
	"testing"
	"time"

	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
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

// MockJobRepository implements ports.JobRepository
type MockJobRepository struct {
	CreateFunc         func(ctx context.Context, job *model.Job) error
	GetByIDFunc        func(ctx context.Context, userID, jobID string) (*model.Job, error)
	ListFunc           func(ctx context.Context, userID string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error)
	UpdateFunc         func(ctx context.Context, job *model.Job) error
	DeleteFunc         func(ctx context.Context, userID, jobID string) error
	ToggleFavoriteFunc func(ctx context.Context, userID, jobID string) (bool, error)
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

func (m *MockJobRepository) ToggleFavorite(ctx context.Context, userID, jobID string) (bool, error) {
	if m.ToggleFavoriteFunc != nil {
		return m.ToggleFavoriteFunc(ctx, userID, jobID)
	}
	return false, nil
}

func TestJobService_Create(t *testing.T) {
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
		req := &model.CreateJobRequest{
			Title: "Software Engineer",
		}

		result, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "job-1", result.ID)
		assert.Equal(t, "Software Engineer", result.Title)
	})

	t.Run("returns error for empty title", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
		req := &model.CreateJobRequest{Title: "Software Engineer"}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
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
			Status:    "active",
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
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
			ListFunc: func(ctx context.Context, uid string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, 20, limit)
				assert.Equal(t, 0, offset)
				return expectedJobs, 2, nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
		result, total, err := svc.List(context.Background(), userID, 20, 0, "active", "", "")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 2, total)
	})

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error) {
				return []*model.JobDTO{}, 0, nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
		result, total, err := svc.List(context.Background(), userID, 20, 0, "active", "", "")

		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, 0, total)
	})

	t.Run("passes sort parameters", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error) {
				assert.Equal(t, "title", sortBy)
				assert.Equal(t, "asc", sortOrder)
				return []*model.JobDTO{}, 0, nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
		_, _, err := svc.List(context.Background(), userID, 20, 0, "active", "title", "asc")

		require.NoError(t, err)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
		newTitle := "New Title"
		req := &model.UpdateJobRequest{Title: &newTitle}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		require.NoError(t, err)
		assert.Equal(t, "New Title", result.Title)
	})

	t.Run("returns error for empty title", func(t *testing.T) {
		existingJob := &model.Job{
			ID:     jobID,
			UserID: userID,
			Title:  "Old Title",
			Status: "active",
		}

		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob, nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
		emptyTitle := "   "
		req := &model.UpdateJobRequest{Title: &emptyTitle}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrJobTitleRequired, err)
	})

	t.Run("returns error for invalid status", func(t *testing.T) {
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
		invalidStatus := "invalid-status"
		req := &model.UpdateJobRequest{Status: &invalidStatus}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrInvalidJobStatus, err)
	})

	t.Run("allows valid status update", func(t *testing.T) {
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
			UpdateFunc: func(ctx context.Context, job *model.Job) error {
				return nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
		newStatus := "archived"
		req := &model.UpdateJobRequest{Status: &newStatus}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		require.NoError(t, err)
		assert.Equal(t, "archived", result.Status)
	})

	t.Run("returns error when job not found", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
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
		return &model.Job{
			ID:     jobID,
			UserID: userID,
			Title:  "Job Title",
			Status: "active",
		}
	}

	t.Run("invalidates cache when description changes", func(t *testing.T) {
		cache := &MockCacheInvalidator{}
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob(), nil
			},
			UpdateFunc: func(ctx context.Context, job *model.Job) error { return nil },
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, cache)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, cache)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, cache)
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

	t.Run("invalidates cache before delete", func(t *testing.T) {
		cache := &MockCacheInvalidator{}
		mockRepo := &MockJobRepository{
			DeleteFunc: func(ctx context.Context, uid, jid string) error { return nil },
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, cache)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, cache)
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
	t.Run("returns error when limit reached", func(t *testing.T) {
		lc := &MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return errors.New("limit reached")
			},
		}
		mockRepo := &MockJobRepository{}
		svc := NewJobService(mockRepo, defaultMockCompanyRepo, lc, nil)

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
		svc := NewJobService(mockRepo, defaultMockCompanyRepo, lc, nil)

		result, err := svc.Create(context.Background(), "user-123", &model.CreateJobRequest{Title: "Test"})

		require.NoError(t, err)
		assert.Equal(t, "job-1", result.ID)
	})
}

func TestJobService_Create_CompanyValidation(t *testing.T) {
	t.Run("returns error when company not found", func(t *testing.T) {
		companyID := "company-invalid"
		companyRepo := &MockCompanyRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*companyModel.Company, error) {
				return nil, errors.New("company not found")
			},
		}
		mockRepo := &MockJobRepository{}
		svc := NewJobService(mockRepo, companyRepo, nil, nil)

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
		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)

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
		existingJob := &model.Job{
			ID:     jobID,
			UserID: userID,
			Title:  "Job Title",
			Status: "active",
		}
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
		svc := NewJobService(mockRepo, companyRepo, nil, nil)

		result, err := svc.Update(context.Background(), userID, jobID, &model.UpdateJobRequest{
			CompanyID: &companyID,
		})

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrCompanyNotFound)
	})

	t.Run("allows clearing company ID with empty string", func(t *testing.T) {
		existingJob := &model.Job{
			ID:     jobID,
			UserID: userID,
			Title:  "Job Title",
			Status: "active",
		}
		emptyCompanyID := ""
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Job, error) {
				return existingJob, nil
			},
			UpdateFunc: func(_ context.Context, job *model.Job) error {
				return nil
			},
		}
		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)

		result, err := svc.Update(context.Background(), userID, jobID, &model.UpdateJobRequest{
			CompanyID: &emptyCompanyID,
		})

		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("updates all optional fields", func(t *testing.T) {
		existingJob := &model.Job{
			ID:     jobID,
			UserID: userID,
			Title:  "Job Title",
			Status: "active",
		}
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
		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)

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
		existingJob := &model.Job{
			ID:     jobID,
			UserID: userID,
			Title:  "Job Title",
			Status: "active",
		}
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(_ context.Context, _, _ string) (*model.Job, error) {
				return existingJob, nil
			},
			UpdateFunc: func(_ context.Context, _ *model.Job) error {
				return errors.New("update failed")
			},
		}
		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)

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

	job := &model.Job{
		ID:        "job-1",
		UserID:    "user-123",
		CompanyID: &companyID,
		Title:     "Software Engineer",
		Source:    &source,
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}

	dto := job.ToDTO()

	assert.Equal(t, job.ID, dto.ID)
	assert.Equal(t, job.CompanyID, dto.CompanyID)
	assert.Equal(t, job.Title, dto.Title)
	assert.Equal(t, job.Source, dto.Source)
	assert.Equal(t, job.Status, dto.Status)
	assert.Equal(t, job.CreatedAt, dto.CreatedAt)
	assert.Nil(t, dto.CompanyName)        // Set by repository
	assert.Equal(t, 0, dto.ApplicationsCount) // Set by repository
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil, nil)
		_, err := svc.ToggleFavorite(context.Background(), userID, jobID)

		assert.ErrorIs(t, err, model.ErrJobNotFound)
	})
}
