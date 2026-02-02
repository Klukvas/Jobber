package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/resumes/model"
)

// ResumeWithCount holds a resume and its application count
type ResumeWithCount struct {
	Resume            *model.Resume
	ApplicationsCount int
}

type ResumeRepository interface {
	Create(ctx context.Context, resume *model.Resume) error
	GetByID(ctx context.Context, userID, resumeID string) (*model.Resume, error)
	List(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*ResumeWithCount, int, error)
	Update(ctx context.Context, resume *model.Resume) error
	Delete(ctx context.Context, userID, resumeID string) error
}
