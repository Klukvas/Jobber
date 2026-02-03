package service

import (
	"context"
	"strings"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/andreypavlenko/jobber/modules/jobs/ports"
)

// JobService handles job business logic
type JobService struct {
	repo ports.JobRepository
}

// NewJobService creates a new job service
func NewJobService(repo ports.JobRepository) *JobService {
	return &JobService{repo: repo}
}

// Create creates a new job
func (s *JobService) Create(ctx context.Context, userID string, req *model.CreateJobRequest) (*model.JobDTO, error) {
	// Validate
	if strings.TrimSpace(req.Title) == "" {
		return nil, model.ErrJobTitleRequired
	}

	job := &model.Job{
		UserID:    userID,
		CompanyID: req.CompanyID,
		Title:     strings.TrimSpace(req.Title),
		Source:    req.Source,
		URL:       req.URL,
		Notes:     req.Notes,
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
	// Repository now returns JobDTO directly with enriched data
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

	return job.ToDTO(), nil
}

// Delete deletes a job
func (s *JobService) Delete(ctx context.Context, userID, jobID string) error {
	return s.repo.Delete(ctx, userID, jobID)
}
