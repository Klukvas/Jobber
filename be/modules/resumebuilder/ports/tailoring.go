package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

// TailoringTransaction keeps the clone and Job link in the same database transaction.
type TailoringTransaction interface {
	ResumeBuilderRepository
	GetTailoringJob(ctx context.Context, userID, jobID string) (*model.TailoringJob, error)
	LockTailoringSource(ctx context.Context, userID, resumeID string) (*model.FullResumeDTO, error)
	CreateWithID(ctx context.Context, resume *model.ResumeBuilder) error
	LinkTailoredResume(ctx context.Context, userID, jobID, resumeID string) error
	CheckResumeBuilderCapacity(ctx context.Context, userID string, limit int) error
}

type ResumeCreationRepository interface {
	RunResumeCreationTransaction(ctx context.Context, userID string, limit int, fn func(ResumeBuilderRepository) error) error
}

type TailoringRepository interface {
	GetTailoringJob(ctx context.Context, userID, jobID string) (*model.TailoringJob, error)
	RunTailoringTransaction(ctx context.Context, userID string, fn func(TailoringTransaction) error) error
}
