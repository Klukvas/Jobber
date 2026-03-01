package service

import (
	"context"
	"log"
	"strings"

	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
)

// LimitChecker checks subscription limits before resource creation.
type LimitChecker interface {
	CheckLimit(ctx context.Context, userID, resource string) error
}

// CacheInvalidator invalidates match-score cache when source data changes.
type CacheInvalidator interface {
	InvalidateByJob(ctx context.Context, jobID string) error
}

// JobService handles job business logic
type JobService struct {
	repo             ports.JobRepository
	companyRepo      companyPorts.CompanyRepository
	limitChecker     LimitChecker
	cacheInvalidator CacheInvalidator
}

// NewJobService creates a new job service
func NewJobService(repo ports.JobRepository, companyRepo companyPorts.CompanyRepository, limitChecker LimitChecker, cacheInvalidator CacheInvalidator) *JobService {
	return &JobService{
		repo:             repo,
		companyRepo:      companyRepo,
		limitChecker:     limitChecker,
		cacheInvalidator: cacheInvalidator,
	}
}

// Create creates a new job
func (s *JobService) Create(ctx context.Context, userID string, req *model.CreateJobRequest) (*model.JobDTO, error) {
	// Check subscription limit
	if s.limitChecker != nil {
		if err := s.limitChecker.CheckLimit(ctx, userID, "jobs"); err != nil {
			return nil, err
		}
	}

	// Validate
	if strings.TrimSpace(req.Title) == "" {
		return nil, model.ErrJobTitleRequired
	}

	// Validate company ownership if provided
	if req.CompanyID != nil && *req.CompanyID != "" {
		if _, err := s.companyRepo.GetByID(ctx, userID, *req.CompanyID); err != nil {
			return nil, model.ErrCompanyNotFound
		}
	}

	job := &model.Job{
		UserID:      userID,
		CompanyID:   req.CompanyID,
		Title:       strings.TrimSpace(req.Title),
		Source:      req.Source,
		URL:         req.URL,
		Notes:       req.Notes,
		Description: req.Description,
	}

	if err := s.repo.Create(ctx, job); err != nil {
		return nil, err
	}

	return job.ToDTO(), nil
}

// GetByID retrieves a job by ID
func (s *JobService) GetByID(ctx context.Context, userID, jobID string) (*model.JobDTO, error) {
	job, err := s.repo.GetByID(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	return job.ToDTO(), nil
}

// List retrieves jobs for a user with pagination, filtering, and sorting
func (s *JobService) List(ctx context.Context, userID string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error) {
	return s.repo.List(ctx, userID, limit, offset, status, sortBy, sortOrder)
}

// Update updates a job
func (s *JobService) Update(ctx context.Context, userID, jobID string, req *model.UpdateJobRequest) (*model.JobDTO, error) {
	// Get existing job
	job, err := s.repo.GetByID(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.CompanyID != nil {
		// Validate company ownership if a non-empty company ID is provided
		if *req.CompanyID != "" {
			if _, err := s.companyRepo.GetByID(ctx, userID, *req.CompanyID); err != nil {
				return nil, model.ErrCompanyNotFound
			}
		}
		job.CompanyID = req.CompanyID
	}
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			return nil, model.ErrJobTitleRequired
		}
		job.Title = strings.TrimSpace(*req.Title)
	}
	if req.Source != nil {
		job.Source = req.Source
	}
	if req.URL != nil {
		job.URL = req.URL
	}
	if req.Notes != nil {
		job.Notes = req.Notes
	}
	descriptionChanged := false
	if req.Description != nil {
		oldDesc := ""
		if job.Description != nil {
			oldDesc = *job.Description
		}
		descriptionChanged = *req.Description != oldDesc
		job.Description = req.Description
	}
	if req.Status != nil {
		// Validate status
		if *req.Status != "active" && *req.Status != "archived" {
			return nil, model.ErrInvalidJobStatus
		}
		job.Status = *req.Status
	}

	if err := s.repo.Update(ctx, job); err != nil {
		return nil, err
	}

	// Invalidate match-score cache when description changes
	if descriptionChanged && s.cacheInvalidator != nil {
		if err := s.cacheInvalidator.InvalidateByJob(ctx, jobID); err != nil {
			log.Printf("[WARN] match score cache invalidation failed for job=%s: %v", jobID, err)
		}
	}

	return job.ToDTO(), nil
}

// ToggleFavorite toggles the favorite status of a job
func (s *JobService) ToggleFavorite(ctx context.Context, userID, jobID string) (bool, error) {
	return s.repo.ToggleFavorite(ctx, userID, jobID)
}

// Delete deletes a job
func (s *JobService) Delete(ctx context.Context, userID, jobID string) error {
	// Invalidate match-score cache before deleting (FK CASCADE is a safety net)
	if s.cacheInvalidator != nil {
		if err := s.cacheInvalidator.InvalidateByJob(ctx, jobID); err != nil {
			log.Printf("[WARN] match score cache invalidation failed for job=%s: %v", jobID, err)
		}
	}

	return s.repo.Delete(ctx, userID, jobID)
}
