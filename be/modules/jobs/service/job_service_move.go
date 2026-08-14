package service

import (
	"context"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// movePhaseStatus maps a phase-target of a unified-board move to the derived
// job status.
func movePhaseStatus(phase model.Phase) (string, error) {
	if !model.IsValidPhase(phase) {
		return "", model.ErrInvalidPhase
	}
	return string(model.StatusForPhase(phase)), nil
}

// Move atomically repositions a card on the unified board. Dropping on a
// stage column sets the stage AND derives the status from the stage's phase;
// dropping on a phase base column completes the current stage (if any),
// clears the current-stage pointer and sets the derived status. Everything
// runs in one transaction so the two axes can never drift apart mid-drag.
func (s *JobService) Move(ctx context.Context, userID, jobID string, req *model.MoveJobRequest) (*model.JobDTO, error) {
	job, err := s.repo.GetByID(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}

	switch req.Target.Type {
	case "stage":
		if req.Target.StageTemplateID == nil || *req.Target.StageTemplateID == "" {
			return nil, model.ErrInvalidMoveTarget
		}
		template, err := s.templateRepo.GetByID(ctx, userID, *req.Target.StageTemplateID)
		if err != nil {
			return nil, err
		}
		return s.moveToStage(ctx, userID, job, template)
	case "phase":
		if req.Target.Phase == nil {
			return nil, model.ErrInvalidMoveTarget
		}
		newStatus, err := movePhaseStatus(*req.Target.Phase)
		if err != nil {
			return nil, err
		}
		return s.moveToPhaseBase(ctx, userID, job, newStatus)
	default:
		return nil, model.ErrInvalidMoveTarget
	}
}

func (s *JobService) moveToStage(ctx context.Context, userID string, job *model.Job, template *model.StageTemplate) (*model.JobDTO, error) {
	newStatus := string(model.StatusForPhase(template.Phase))

	// No-op guard: dropping a card back on its own column must not spawn a
	// duplicate stage entry.
	if job.CurrentStageID != nil && *job.CurrentStageID != "" && job.Status == newStatus {
		current, err := s.stageRepo.GetByID(ctx, *job.CurrentStageID, job.ID)
		if err == nil && current.StageTemplateID == template.ID && current.Status != "completed" {
			return s.buildJobDTO(ctx, userID, job, false)
		}
	}

	now := time.Now().UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	// Serialize concurrent stage writes on this job (same as AddStage)
	if _, err := tx.Exec(ctx, `SELECT id FROM jobs WHERE id = $1 FOR UPDATE`, job.ID); err != nil {
		return nil, fmt.Errorf("failed to lock job: %w", err)
	}

	var order int
	if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX("order"), -1) + 1 FROM job_stages WHERE job_id = $1`, job.ID).Scan(&order); err != nil {
		return nil, fmt.Errorf("failed to compute stage order: %w", err)
	}

	// Complete the current active stage (if any)
	if job.CurrentStageID != nil && *job.CurrentStageID != "" {
		if _, err := tx.Exec(ctx,
			`UPDATE job_stages SET status = 'completed', completed_at = $2 WHERE id = $1 AND status != 'completed'`,
			*job.CurrentStageID, now,
		); err != nil {
			return nil, fmt.Errorf("failed to complete current stage: %w", err)
		}
	}

	newStageID := uuid.New().String()
	if _, err := tx.Exec(ctx,
		`INSERT INTO job_stages (id, job_id, stage_template_id, status, "order", started_at, completed_at, created_at)
		 VALUES ($1, $2, $3, 'active', $4, $5, NULL, $6)`,
		newStageID, job.ID, template.ID, order, now, now,
	); err != nil {
		return nil, fmt.Errorf("failed to create stage: %w", err)
	}

	if err := applyStatusTransition(job, newStatus, nil); err != nil {
		return nil, err
	}
	job.CurrentStageID = &newStageID

	if _, err := tx.Exec(ctx,
		`UPDATE jobs SET current_stage_id = $2, status = $3, applied_at = $4, updated_at = $5 WHERE id = $1`,
		job.ID, newStageID, job.Status, job.AppliedAt, now,
	); err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("job moved to stage",
		zap.String("job_id", job.ID),
		zap.String("stage", template.Name),
		zap.String("phase", string(template.Phase)),
		zap.String("status", job.Status),
		zap.String("user_id", userID))

	return s.buildJobDTO(ctx, userID, job, false)
}

func (s *JobService) moveToPhaseBase(ctx context.Context, userID string, job *model.Job, newStatus string) (*model.JobDTO, error) {
	hasStage := job.CurrentStageID != nil && *job.CurrentStageID != ""

	// No-op guard: already in this base column
	if job.Status == newStatus && !hasStage {
		return s.buildJobDTO(ctx, userID, job, false)
	}

	now := time.Now().UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	if _, err := tx.Exec(ctx, `SELECT id FROM jobs WHERE id = $1 FOR UPDATE`, job.ID); err != nil {
		return nil, fmt.Errorf("failed to lock job: %w", err)
	}

	// Leaving a stage for a base column completes it — history stays intact
	if hasStage {
		if _, err := tx.Exec(ctx,
			`UPDATE job_stages SET status = 'completed', completed_at = $2 WHERE id = $1 AND status != 'completed'`,
			*job.CurrentStageID, now,
		); err != nil {
			return nil, fmt.Errorf("failed to complete current stage: %w", err)
		}
	}

	if err := applyStatusTransition(job, newStatus, nil); err != nil {
		return nil, err
	}
	job.CurrentStageID = nil

	if _, err := tx.Exec(ctx,
		`UPDATE jobs SET current_stage_id = NULL, status = $2, applied_at = $3, updated_at = $4 WHERE id = $1`,
		job.ID, job.Status, job.AppliedAt, now,
	); err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("job moved to phase base",
		zap.String("job_id", job.ID),
		zap.String("status", job.Status))

	return s.buildJobDTO(ctx, userID, job, false)
}
