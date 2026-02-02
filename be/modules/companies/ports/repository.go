package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/companies/model"
)

// ListOptions defines options for listing companies
type ListOptions struct {
	Limit   int
	Offset  int
	SortBy  string // "name", "last_activity", "applications_count"
	SortDir string // "asc", "desc"
}

// CompanyRepository defines the interface for company data access
type CompanyRepository interface {
	Create(ctx context.Context, company *model.Company) error
	GetByID(ctx context.Context, userID, companyID string) (*model.Company, error)
	GetByIDEnriched(ctx context.Context, userID, companyID string) (*model.CompanyDTO, error)
	List(ctx context.Context, userID string, opts *ListOptions) ([]*model.CompanyDTO, int, error)
	Update(ctx context.Context, company *model.Company) error
	Delete(ctx context.Context, userID, companyID string) error
	GetRelatedJobsAndApplicationsCount(ctx context.Context, userID, companyID string) (jobsCount, appsCount int, err error)
}
