package service

import (
	"context"
	"strings"

	"github.com/andreypavlenko/jobber/modules/companies/model"
	"github.com/andreypavlenko/jobber/modules/companies/ports"
)

// CompanyService handles company business logic
type CompanyService struct {
	repo ports.CompanyRepository
}

// NewCompanyService creates a new company service
func NewCompanyService(repo ports.CompanyRepository) *CompanyService {
	return &CompanyService{repo: repo}
}

// Create creates a new company
func (s *CompanyService) Create(ctx context.Context, userID string, req *model.CreateCompanyRequest) (*model.CompanyDTO, error) {
	// Validate
	if strings.TrimSpace(req.Name) == "" {
		return nil, model.ErrCompanyNameRequired
	}

	company := &model.Company{
		UserID:   userID,
		Name:     strings.TrimSpace(req.Name),
		Location: req.Location,
		Notes:    req.Notes,
	}

	if err := s.repo.Create(ctx, company); err != nil {
		return nil, err
	}

	// Return enriched DTO
	return s.repo.GetByIDEnriched(ctx, userID, company.ID)
}

// GetByID retrieves a company by ID with enriched fields
func (s *CompanyService) GetByID(ctx context.Context, userID, companyID string) (*model.CompanyDTO, error) {
	return s.repo.GetByIDEnriched(ctx, userID, companyID)
}

// List retrieves companies for a user with pagination and enriched fields
func (s *CompanyService) List(ctx context.Context, userID string, opts *ports.ListOptions) ([]*model.CompanyDTO, int, error) {
	return s.repo.List(ctx, userID, opts)
}

// Update updates a company
func (s *CompanyService) Update(ctx context.Context, userID, companyID string, req *model.UpdateCompanyRequest) (*model.CompanyDTO, error) {
	// Get existing company
	company, err := s.repo.GetByID(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}

	// Update fields
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, model.ErrCompanyNameRequired
		}
		company.Name = strings.TrimSpace(*req.Name)
	}
	if req.Location != nil {
		company.Location = req.Location
	}
	if req.Notes != nil {
		company.Notes = req.Notes
	}

	if err := s.repo.Update(ctx, company); err != nil {
		return nil, err
	}

	// Return enriched DTO
	return s.repo.GetByIDEnriched(ctx, userID, companyID)
}

// Delete deletes a company after checking for related data
func (s *CompanyService) Delete(ctx context.Context, userID, companyID string) error {
	// Check if company exists first
	_, err := s.repo.GetByID(ctx, userID, companyID)
	if err != nil {
		return err
	}

	// Note: We don't prevent deletion, but the frontend will warn users
	// about related jobs/applications using GetRelatedJobsAndApplicationsCount
	return s.repo.Delete(ctx, userID, companyID)
}

// GetRelatedJobsAndApplicationsCount gets counts of related data for delete warning
func (s *CompanyService) GetRelatedJobsAndApplicationsCount(ctx context.Context, userID, companyID string) (jobsCount, appsCount int, err error) {
	return s.repo.GetRelatedJobsAndApplicationsCount(ctx, userID, companyID)
}
