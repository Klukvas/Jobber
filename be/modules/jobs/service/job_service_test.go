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

var defaultMockCompanyRepo = &MockCompanyRepository{}

// MockJobRepository implements ports.JobRepository
type MockJobRepository struct {
	CreateFunc  func(ctx context.Context, job *model.Job) error
	GetByIDFunc func(ctx context.Context, userID, jobID string) (*model.Job, error)
	ListFunc    func(ctx context.Context, userID string, limit, offset int, status, sortBy, sortOrder, boardColumn string) ([]*model.JobDTO, int, error)
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

func (m *MockJobRepository) List(ctx context.Context, userID string, limit, offset int, status, sortBy, sortOrder, boardColumn string) ([]*model.JobDTO, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, limit, offset, status, sortBy, sortOrder, boardColumn)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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
		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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

	t.Run("creates job with board_column", func(t *testing.T) {
		var createdJob *model.Job

		mockRepo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, job *model.Job) error {
				createdJob = job
				job.ID = "job-1"
				return nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
		boardColumn := "applied"
		req := &model.CreateJobRequest{
			Title:       "Software Engineer",
			BoardColumn: &boardColumn,
		}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "applied", createdJob.BoardColumn)
	})

	t.Run("defaults board_column to wishlist", func(t *testing.T) {
		var createdJob *model.Job

		mockRepo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, job *model.Job) error {
				createdJob = job
				job.ID = "job-1"
				return nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
		req := &model.CreateJobRequest{Title: "Software Engineer"}

		_, err := svc.Create(context.Background(), userID, req)

		require.NoError(t, err)
		assert.Equal(t, "wishlist", createdJob.BoardColumn)
	})

	t.Run("returns error for invalid board_column", func(t *testing.T) {
		mockRepo := &MockJobRepository{}
		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
		invalidColumn := "invalid-column"
		req := &model.CreateJobRequest{
			Title:       "Software Engineer",
			BoardColumn: &invalidColumn,
		}

		result, err := svc.Create(context.Background(), userID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrInvalidBoardColumn, err)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockRepo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, job *model.Job) error {
				return expectedError
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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
			ListFunc: func(ctx context.Context, uid string, limit, offset int, status, sortBy, sortOrder, boardColumn string) ([]*model.JobDTO, int, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, 20, limit)
				assert.Equal(t, 0, offset)
				return expectedJobs, 2, nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
		result, total, err := svc.List(context.Background(), userID, 20, 0, "active", "", "", "")

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, 2, total)
	})

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int, status, sortBy, sortOrder, boardColumn string) ([]*model.JobDTO, int, error) {
				return []*model.JobDTO{}, 0, nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
		result, total, err := svc.List(context.Background(), userID, 20, 0, "active", "", "", "")

		require.NoError(t, err)
		assert.Empty(t, result)
		assert.Equal(t, 0, total)
	})

	t.Run("passes sort parameters", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int, status, sortBy, sortOrder, boardColumn string) ([]*model.JobDTO, int, error) {
				assert.Equal(t, "title", sortBy)
				assert.Equal(t, "asc", sortOrder)
				return []*model.JobDTO{}, 0, nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
		_, _, err := svc.List(context.Background(), userID, 20, 0, "active", "title", "asc", "")

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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
		newStatus := "archived"
		req := &model.UpdateJobRequest{Status: &newStatus}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		require.NoError(t, err)
		assert.Equal(t, "archived", result.Status)
	})

	t.Run("updates board_column successfully", func(t *testing.T) {
		existingJob := &model.Job{
			ID:          jobID,
			UserID:      userID,
			Title:       "Job Title",
			Status:      "active",
			BoardColumn: "wishlist",
		}

		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob, nil
			},
			UpdateFunc: func(ctx context.Context, job *model.Job) error {
				return nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
		newColumn := "interview"
		req := &model.UpdateJobRequest{BoardColumn: &newColumn}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		require.NoError(t, err)
		assert.Equal(t, "interview", result.BoardColumn)
	})

	t.Run("returns error for invalid board_column on update", func(t *testing.T) {
		existingJob := &model.Job{
			ID:          jobID,
			UserID:      userID,
			Title:       "Job Title",
			Status:      "active",
			BoardColumn: "wishlist",
		}

		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return existingJob, nil
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
		invalidColumn := "invalid"
		req := &model.UpdateJobRequest{BoardColumn: &invalidColumn}

		result, err := svc.Update(context.Background(), userID, jobID, req)

		assert.Nil(t, result)
		assert.Equal(t, model.ErrInvalidBoardColumn, err)
	})

	t.Run("returns error when job not found", func(t *testing.T) {
		mockRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
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

		svc := NewJobService(mockRepo, defaultMockCompanyRepo, nil)
		err := svc.Delete(context.Background(), userID, jobID)

		assert.Equal(t, model.ErrJobNotFound, err)
	})
}

func TestJob_ToDTO(t *testing.T) {
	now := time.Now()
	companyID := "company-1"
	source := "LinkedIn"

	job := &model.Job{
		ID:          "job-1",
		UserID:      "user-123",
		CompanyID:   &companyID,
		Title:       "Software Engineer",
		Source:      &source,
		Status:      "active",
		BoardColumn: "applied",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	dto := job.ToDTO()

	assert.Equal(t, job.ID, dto.ID)
	assert.Equal(t, job.CompanyID, dto.CompanyID)
	assert.Equal(t, job.Title, dto.Title)
	assert.Equal(t, job.Source, dto.Source)
	assert.Equal(t, job.Status, dto.Status)
	assert.Equal(t, "applied", dto.BoardColumn)
	assert.Equal(t, job.CreatedAt, dto.CreatedAt)
	assert.Nil(t, dto.CompanyName) // Set by repository
	assert.Equal(t, 0, dto.ApplicationsCount) // Set by repository
}
