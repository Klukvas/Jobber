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
	// Create writes a complete resume. Both create methods fill `resume` in as
	// it is stored — its id, if it had none, and its timestamps — because the
	// caller builds its response from the same value.
	Create(ctx context.Context, resume *model.Resume) error
	// CreateFinalizedUpload writes a verified upload and enforces the plan's
	// resume allowance in the same transaction. A negative maxResumes means the
	// plan is unlimited. Returns model.ErrResumeLimitReached when the allowance
	// was already full, and model.ErrResumeAlreadyExists when this upload has
	// been finalized before.
	CreateFinalizedUpload(ctx context.Context, resume *model.Resume, maxResumes int) error
	GetByID(ctx context.Context, userID, resumeID string) (*model.Resume, error)
	List(ctx context.Context, userID string, limit, offset int, sortBy, sortDir string) ([]*ResumeWithCount, int, error)
	Update(ctx context.Context, resume *model.Resume) error
	Delete(ctx context.Context, userID, resumeID string) error
}
