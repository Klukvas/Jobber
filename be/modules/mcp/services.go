// Package mcp exposes the paid, PAT-authenticated assistant tool surface.
package mcp

import (
	"context"

	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	companyPorts "github.com/andreypavlenko/jobber/modules/companies/ports"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	jobPorts "github.com/andreypavlenko/jobber/modules/jobs/ports"
)

type JobReader interface {
	List(context.Context, string, *jobPorts.ListOptions) ([]*jobModel.JobDTO, int, error)
	GetByID(context.Context, string, string) (*jobModel.JobDTO, error)
	ListStageTemplates(context.Context, string, int, int) ([]*jobModel.StageTemplateDTO, int, error)
}

type JobWriter interface {
	Create(context.Context, string, *jobModel.CreateJobRequest) (*jobModel.JobDTO, error)
	Move(context.Context, string, string, *jobModel.MoveJobRequest) (*jobModel.JobDTO, error)
	Update(context.Context, string, string, *jobModel.UpdateJobRequest) (*jobModel.JobDTO, error)
}

type Jobs interface {
	JobReader
	JobWriter
}

// Company names are resolved through the existing company service; jobs store IDs.
type Companies interface {
	List(context.Context, string, *companyPorts.ListOptions) ([]*companyModel.CompanyDTO, int, error)
	Create(context.Context, string, *companyModel.CreateCompanyRequest) (*companyModel.CompanyDTO, error)
}

type PaidPlanChecker interface {
	RequirePaidPlan(context.Context, string) error
}

// JobLimitChecker preflights compound creates before a Company can be persisted.
type JobLimitChecker interface {
	CheckLimit(context.Context, string, string) error
}
