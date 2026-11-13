package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/google/uuid"
)

const (
	SnapshotWindow = 30 * time.Minute
	SnapshotLimit  = 20
)

type SnapshotService struct {
	repo ports.SnapshotRepository
	now  func() time.Time
}

func NewSnapshotService(repo ports.SnapshotRepository) *SnapshotService {
	return &SnapshotService{repo: repo, now: time.Now}
}

func (s *SnapshotService) SnapshotBeforeWrite(ctx context.Context, userID, resumeID, source string) error {
	if strings.TrimSpace(source) == "" || utf8.RuneCountInString(source) > 20 {
		return model.ErrInvalidSnapshotSource
	}
	return s.repo.RunSnapshotTransaction(ctx, func(tx ports.SnapshotTransaction) error {
		if err := tx.LockResume(ctx, userID, resumeID); err != nil {
			return err
		}
		full, err := tx.GetFullResume(ctx, userID, resumeID)
		if err != nil {
			return fmt.Errorf("load resume before snapshot: %w", err)
		}
		now := s.now().UTC()
		exists, err := tx.HasRecentSnapshot(ctx, resumeID, source, now.Add(-SnapshotWindow))
		if err != nil || exists {
			return err
		}
		return storeResumeSnapshot(ctx, tx, userID, resumeID, source, full, now)
	})
}

func (s *SnapshotService) ListSnapshots(ctx context.Context, userID, resumeID string) ([]*model.ResumeSnapshotDTO, error) {
	return s.repo.ListSnapshots(ctx, userID, resumeID)
}

func (s *SnapshotService) RestoreSnapshot(ctx context.Context, userID, snapshotID string) (*model.FullResumeDTO, error) {
	var restored *model.FullResumeDTO
	err := s.repo.RunSnapshotTransaction(ctx, func(tx ports.SnapshotTransaction) error {
		snapshot, err := tx.GetSnapshot(ctx, userID, snapshotID)
		if err != nil {
			return err
		}
		if err := tx.LockResume(ctx, userID, snapshot.ResumeBuilderID); err != nil {
			return err
		}
		full, err := decodeResumeSnapshot(snapshot)
		if err != nil {
			return err
		}
		if err := s.snapshotCurrentForRestore(ctx, tx, userID, snapshot.ResumeBuilderID); err != nil {
			return err
		}
		if err := replaceResumeContent(ctx, tx, userID, snapshot.ResumeBuilderID, full); err != nil {
			return err
		}
		restored, err = tx.GetFullResume(ctx, userID, snapshot.ResumeBuilderID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("restore resume snapshot: %w", err)
	}
	return restored, nil
}

func (s *SnapshotService) snapshotCurrentForRestore(ctx context.Context, tx ports.SnapshotTransaction, userID, resumeID string) error {
	current, err := tx.GetFullResume(ctx, userID, resumeID)
	if err != nil {
		return fmt.Errorf("load resume before restore: %w", err)
	}
	// Restores always create an undo copy, including successive restores in one window.
	return storeResumeSnapshot(ctx, tx, userID, resumeID, "restore", current, s.now().UTC())
}

func storeResumeSnapshot(ctx context.Context, tx ports.SnapshotTransaction, userID, resumeID, source string, full *model.FullResumeDTO, now time.Time) error {
	content, err := encodeResumeSnapshot(full)
	if err != nil {
		return fmt.Errorf("encode resume snapshot: %w", err)
	}
	snapshot := &model.ResumeSnapshot{ID: uuid.NewString(), ResumeBuilderID: resumeID,
		UserID: userID, Source: source, Content: content, CreatedAt: now}
	if err := tx.InsertSnapshot(ctx, snapshot); err != nil {
		return err
	}
	return tx.PruneSnapshots(ctx, resumeID, SnapshotLimit)
}
