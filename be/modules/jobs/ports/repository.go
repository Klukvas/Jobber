package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
)

// JobRepository defines the interface for job data access
type JobRepository interface {
	Create(ctx context.Context, job *model.Job) error
	GetByID(ctx context.Context, userID, jobID string) (*model.Job, error)
	List(ctx context.Context, userID string, limit, offset int, status, sortBy, sortOrder string) ([]*model.JobDTO, int, error)
	Update(ctx context.Context, job *model.Job) error
	Delete(ctx context.Context, userID, jobID string) error
}
