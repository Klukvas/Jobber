package ports

import (
	"context"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

type SnapshotRepository interface {
	RunSnapshotTransaction(context.Context, func(SnapshotTransaction) error) error
	ListSnapshots(context.Context, string, string) ([]*model.ResumeSnapshotDTO, error)
}

// SnapshotTransaction keeps the undo copy and full replacement in one commit.
// LockResume must precede loading or writing a resume's snapshot state.
type SnapshotTransaction interface {
	ResumeBuilderRepository
	LockResume(context.Context, string, string) error
	GetSnapshot(context.Context, string, string) (*model.ResumeSnapshot, error)
	HasRecentSnapshot(context.Context, string, string, time.Time) (bool, error)
	InsertSnapshot(context.Context, *model.ResumeSnapshot) error
	PruneSnapshots(context.Context, string, int) error
	ClearResumeSections(context.Context, string) error
}
