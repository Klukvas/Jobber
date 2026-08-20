package service

import (
	"context"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Move places a card into a pipeline column — the single write path for
// pipeline position (board drag or the detail-page stage selector). It
// completes the current active job_stages row (if any), appends a new active
// job_stages row for the target column (so history/funnel record the real
// path), points the job at the target column, and — the first time a card
// advances past its first column — stamps applied_at. Everything runs in one
// transaction under a job-row lock.
func (s *JobService) Move(ctx context.Context, userID, jobID string, req *model.MoveJobRequest) (*model.JobDTO, error) {
	if req.StageTemplateID == "" {
		return nil, model.ErrInvalidMoveTarget
	}

	job, err := s.repo.GetByID(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}
	template, err := s.templateRepo.GetByID(ctx, userID, req.StageTemplateID)
	if err != nil {
		return nil, err
	}

	// No-op: already in this column.
	if job.CurrentStageTemplateID != nil && *job.CurrentStageTemplateID == template.ID {
		return s.buildJobDTO(ctx, userID, job, false)
	}

	now := time.Now().UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	// Serialize with AddStage/UpdateStage/DeleteStage on this job.
	lockTag, err := tx.Exec(ctx, `SELECT id FROM jobs WHERE id = $1 FOR UPDATE`, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock job: %w", err)
	}
	if lockTag.RowsAffected() == 0 {
		return nil, model.ErrJobNotFound
	}

	// Complete the current active stage — history stays intact.
	if job.CurrentStageID != nil && *job.CurrentStageID != "" {
		if _, err := tx.Exec(ctx,
			`UPDATE job_stages SET status = 'completed', completed_at = $2 WHERE id = $1 AND status != 'completed'`,
			*job.CurrentStageID, now,
		); err != nil {
			return nil, fmt.Errorf("failed to complete current stage: %w", err)
		}
	}

	// Next order = max existing + 1 (under the lock).
	var order int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX("order"), -1) + 1 FROM job_stages WHERE job_id = $1`, job.ID).Scan(&order); err != nil {
		return nil, fmt.Errorf("failed to compute stage order: %w", err)
	}

	newStageID := uuid.New().String()
	if _, err := tx.Exec(ctx,
		`INSERT INTO job_stages (id, job_id, stage_template_id, status, "order", started_at, completed_at, created_at)
		 VALUES ($1, $2, $3, 'active', $4, $5, NULL, $5)`,
		newStageID, job.ID, template.ID, order, now,
	); err != nil {
		return nil, fmt.Errorf("failed to create stage: %w", err)
	}

	appliedAt := job.AppliedAt
	if appliedAt == nil && template.Order > 0 {
		appliedAt = &now
	}

	if _, err := tx.Exec(ctx,
		`UPDATE jobs SET current_stage_id = $2, current_stage_template_id = $3, applied_at = $4, updated_at = $5 WHERE id = $1`,
		job.ID, newStageID, template.ID, appliedAt, now,
	); err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Reflect the new state on the in-memory job for the enriched DTO.
	job.CurrentStageID = &newStageID
	tmplID := template.ID
	job.CurrentStageTemplateID = &tmplID
	job.AppliedAt = appliedAt

	s.log.Info("job moved to column",
		zap.String("job_id", job.ID),
		zap.String("stage", template.Name),
		zap.String("user_id", userID))

	return s.buildJobDTO(ctx, userID, job, false)
}
