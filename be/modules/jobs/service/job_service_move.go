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
// completes the card's active job_stages row (if any), appends a new active
// job_stages row for the target column (so history/funnel record the real
// path), points the job at the target column, and stamps applied_at when the
// target column means the application is out (see job_service_applied.go).
// Everything runs in one transaction under a job-row lock.
//
// The "already in this column" check happens inside the transaction, on the
// state read under that lock: deciding it from the copy fetched beforehand made
// a move race with another one and return success without writing anything.
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

	now := time.Now().UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	// Serialize with AddStage/UpdateStage/DeleteStage on this job.
	state, err := lockJob(ctx, tx, userID, jobID)
	if err != nil {
		return nil, err
	}

	// Already in this column: nothing to write. Return the card as it stands
	// now — the locked read may differ from what the caller saw.
	if state.InColumn(template.ID) {
		job.CurrentStageTemplateID = state.CurrentStageTemplateID
		job.CurrentStageID = state.CurrentStageID
		job.AppliedAt = state.AppliedAt
		// Release the lock before the fresh read: asking the pool for a second
		// connection while this transaction still holds one is how a burst of
		// concurrent moves starves the pool.
		_ = tx.Rollback(ctx)
		return s.buildJobDTO(ctx, userID, s.rereadJob(ctx, userID, job), false)
	}

	// Complete whatever stage was open — history stays intact.
	if err := completeOpenStages(ctx, tx, jobID, now, nil); err != nil {
		return nil, err
	}

	order, err := nextStageOrder(ctx, tx, jobID)
	if err != nil {
		return nil, err
	}

	newStageID := uuid.New().String()
	if _, err := tx.Exec(ctx,
		`INSERT INTO job_stages (id, job_id, stage_template_id, status, "order", started_at, completed_at, created_at)
		 VALUES ($1, $2, $3, 'active', $4, $5, NULL, $5)`,
		newStageID, jobID, template.ID, order, now,
	); err != nil {
		return nil, fmt.Errorf("failed to create stage: %w", err)
	}

	appliedAt := appliedAtAfterMove(state.AppliedAt, template, now)

	if _, err := tx.Exec(ctx,
		`UPDATE jobs SET current_stage_id = $2, current_stage_template_id = $3, applied_at = $4, updated_at = $5 WHERE id = $1`,
		jobID, newStageID, template.ID, appliedAt, now,
	); err != nil {
		return nil, fmt.Errorf("failed to update job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Reflect the new state on the in-memory job, the fallback for the DTO if
	// the re-read below cannot be served.
	job.CurrentStageID = &newStageID
	tmplID := template.ID
	job.CurrentStageTemplateID = &tmplID
	job.AppliedAt = appliedAt

	s.log.Info("job moved to column",
		zap.String("job_id", job.ID),
		zap.String("stage", template.Name),
		zap.String("user_id", userID))

	// The lock is gone with the commit, so the extra read cannot hold a
	// connection hostage.
	return s.buildJobDTO(ctx, userID, s.rereadJob(ctx, userID, job), false)
}

// rereadJob re-reads a card once its pipeline write has committed (or rolled
// back as a no-op) and the row lock is released.
//
// The copy Move fetched before the transaction is a snapshot of the whole row.
// A PATCH committing in that window changes title/source/description in the
// database, and answering the move from the snapshot hands the client those old
// values — which the frontend writes straight into its cache, so a rename
// disappears on screen while the stored row is perfectly correct.
//
// A failed re-read is not worth failing a committed move over: the locally
// updated copy still describes the pipeline change that was just written.
func (s *JobService) rereadJob(ctx context.Context, userID string, local *model.Job) *model.Job {
	fresh, err := s.repo.GetByID(ctx, userID, local.ID)
	if err != nil {
		s.log.Warn("failed to re-read job after move; answering from the local copy",
			zap.String("job_id", local.ID),
			zap.Error(err))
		return local
	}
	return fresh
}
