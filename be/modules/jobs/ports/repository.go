package ports

import (
	"context"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
)

// ListOptions represents options for listing jobs
type ListOptions struct {
	Limit   int
	Offset  int
	SortBy  string // "created_at", "title", "company_name", "last_activity", "status", "applied_at"
	SortDir string // "asc", "desc"
	// Status filter. "" and the legacy "active" mean "everything except archived"
	// (backward compatible with pre-merge clients incl. the Chrome extension),
	// "all" means no filter, otherwise an exact pipeline status.
	Status string
	// Search is a case-insensitive substring matched against the job title and
	// the linked company name. Empty means no search filter.
	Search string
}

// JobRepository defines the interface for job data access
type JobRepository interface {
	Create(ctx context.Context, job *model.Job) error
	GetByID(ctx context.Context, userID, jobID string) (*model.Job, error)
	List(ctx context.Context, userID string, opts *ListOptions) ([]*model.JobDTO, int, error)
	Update(ctx context.Context, job *model.Job) error
	Delete(ctx context.Context, userID, jobID string) error
	ToggleFavorite(ctx context.Context, userID, jobID string) (bool, error)
	GetLastActivityAt(ctx context.Context, userID, jobID string) (time.Time, error)
}

// StageTemplateRepository defines the interface for stage template data access
type StageTemplateRepository interface {
	Create(ctx context.Context, template *model.StageTemplate) error
	GetByID(ctx context.Context, userID, templateID string) (*model.StageTemplate, error)
	List(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error)
	Update(ctx context.Context, template *model.StageTemplate) error
	Reorder(ctx context.Context, userID string, orderedIDs []string) error
	Delete(ctx context.Context, userID, templateID string) error
}

// JobStageRepository defines the interface for job stage data access
type JobStageRepository interface {
	Create(ctx context.Context, stage *model.JobStage) error
	GetByID(ctx context.Context, stageID, jobID string) (*model.JobStage, error)
	ListByJob(ctx context.Context, jobID string) ([]*model.JobStage, error)
	Update(ctx context.Context, stage *model.JobStage) error
	Delete(ctx context.Context, stageID, jobID string) error
}
