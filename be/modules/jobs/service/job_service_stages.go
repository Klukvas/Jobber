package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

// Stage history. Every write in this file runs in one transaction and reads the
// card's authoritative state inside it, under the row lock taken by lockJob —
// see that function for why the pre-transaction copy cannot be trusted.

// AddStage adds a new stage to a job following append-only semantics.
func (s *JobService) AddStage(ctx context.Context, userID, jobID string, req *model.AddStageRequest) (*model.JobStageDTO, error) {
	// Ownership of the card and of the target column are checked before the
	// transaction opens: neither can change under us, and a 404 should not
	// have held a row lock.
	if _, err := s.repo.GetByID(ctx, userID, jobID); err != nil {
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

	state, err := lockJob(ctx, tx, userID, jobID)
	if err != nil {
		return nil, err
	}

	// The column the card is leaving, for the log line — read under the lock,
	// so it names the stage this call actually closed.
	previousStageName, err := s.openStageName(ctx, tx, userID, jobID, state.CurrentStageID)
	if err != nil {
		return nil, err
	}

	if err := completeOpenStages(ctx, tx, jobID, now, nil); err != nil {
		return nil, err
	}

	order, err := nextStageOrder(ctx, tx, jobID)
	if err != nil {
		return nil, err
	}

	// Create new stage with "active" status
	newStageID := uuid.New().String()
	createdAt := time.Now().UTC()
	_, err = tx.Exec(ctx,
		`INSERT INTO job_stages (id, job_id, stage_template_id, status, "order", started_at, completed_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		newStageID, jobID, req.StageTemplateID, "active", order, now, nil, createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create stage: %w", err)
	}

	// Update the job's current column + history pointer. Stamp applied_at when
	// the target column means the application is out.
	appliedAt := appliedAtAfterMove(state.AppliedAt, template, now)
	_, err = tx.Exec(ctx,
		`UPDATE jobs SET current_stage_id = $2, current_stage_template_id = $3, applied_at = $4, updated_at = $5 WHERE id = $1`,
		jobID, newStageID, template.ID, appliedAt, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update job current stage: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Create comment if provided (outside tx — non-critical)
	if req.Comment != nil && strings.TrimSpace(*req.Comment) != "" {
		comment := &commentModel.Comment{
			UserID:  userID,
			JobID:   jobID,
			StageID: &newStageID,
			Content: strings.TrimSpace(*req.Comment),
		}
		if err := s.commentRepo.Create(ctx, comment); err != nil {
			s.log.Error("failed to create comment for stage", zap.Error(err))
		}
	}

	// Log the stage change
	if previousStageName != "" {
		s.log.Info("stage changed",
			zap.String("job_id", jobID),
			zap.String("previous_stage", previousStageName),
			zap.String("new_stage", template.Name),
			zap.String("user_id", userID))
	} else {
		s.log.Info("stage added",
			zap.String("job_id", jobID),
			zap.String("stage", template.Name),
			zap.String("user_id", userID))
	}

	// Build the stage DTO for the response
	stage := &model.JobStage{
		ID:              newStageID,
		JobID:           jobID,
		StageTemplateID: req.StageTemplateID,
		Status:          "active",
		Order:           order,
		StartedAt:       now,
		CreatedAt:       createdAt,
	}

	return stage.ToDTO(template.Name), nil
}

// ListStages lists all stages of a job
func (s *JobService) ListStages(ctx context.Context, userID, jobID string) ([]*model.JobStageDTO, error) {
	// Verify job belongs to user
	_, err := s.repo.GetByID(ctx, userID, jobID)
	if err != nil {
		return nil, err
	}

	stages, err := s.stageRepo.ListByJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	// Get all templates to enrich DTOs (without pagination for internal use)
	templates, _, err := s.templateRepo.List(ctx, userID, pipelineColumnLimit, 0)
	if err != nil {
		return nil, err
	}

	templateMap := make(map[string]string)
	for _, t := range templates {
		templateMap[t.ID] = t.Name
	}

	dtos := make([]*model.JobStageDTO, len(stages))
	for i, stage := range stages {
		stageName := templateMap[stage.StageTemplateID]
		dtos[i] = stage.ToDTO(stageName)
	}

	return dtos, nil
}

// UpdateStage updates a history row's status and completion timestamp.
//
// Re-activating a row is a real state change, not just a label: the card is
// back in that column. So it closes whatever else was active, points the card at
// the re-activated row and its column, and stamps applied_at if that column
// means the application is out. Anything else would leave the card with two
// active stages, or with a current_stage_id and a current column that disagree.
func (s *JobService) UpdateStage(ctx context.Context, userID, jobID, stageID string, req *model.UpdateStageRequest) (*model.JobStageDTO, error) {
	// Verify job belongs to user
	if _, err := s.repo.GetByID(ctx, userID, jobID); err != nil {
		return nil, err
	}

	if req.Status != nil && !validStageStatuses[*req.Status] {
		return nil, model.ErrInvalidStatus
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	state, err := lockJob(ctx, tx, userID, jobID)
	if err != nil {
		return nil, err
	}

	// Read the row inside the transaction: a stage fetched before the lock may
	// already have been completed or deleted by a concurrent move.
	stage, err := readStage(ctx, tx, jobID, stageID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	// A row that was already the open one stays where it is; only a real
	// transition back to "active" moves the card.
	wasActive := stage.Status == "active"
	applyStageStatusUpdate(stage, req, now)
	reactivated := stage.Status == "active" && !wasActive

	// Update within the transaction (under the job lock taken above)
	res, err := tx.Exec(ctx,
		`UPDATE job_stages SET status = $2, completed_at = $3 WHERE id = $1 AND job_id = $4`,
		stage.ID, stage.Status, stage.CompletedAt, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update stage: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil, model.ErrJobStageNotFound
	}

	// Get template for DTO (also the column the card moves to when the row is
	// re-activated). Read through tx: the pool must not be asked for a second
	// connection while this one holds the job lock — see readStageTemplate.
	template, err := readStageTemplate(ctx, tx, userID, stage.StageTemplateID)
	if err != nil {
		return nil, err
	}

	if reactivated {
		if err := completeOpenStages(ctx, tx, jobID, now, &stage.ID); err != nil {
			return nil, err
		}
		appliedAt := appliedAtAfterMove(state.AppliedAt, template, now)
		if _, err := tx.Exec(ctx,
			`UPDATE jobs SET current_stage_id = $2, current_stage_template_id = $3, applied_at = $4, updated_at = $5 WHERE id = $1`,
			jobID, stage.ID, template.ID, appliedAt, now,
		); err != nil {
			return nil, fmt.Errorf("failed to update job current stage: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("stage status updated",
		zap.String("job_id", jobID),
		zap.String("stage_id", stageID),
		zap.String("new_status", stage.Status),
		zap.String("user_id", userID))

	return stage.ToDTO(template.Name), nil
}

// validStageStatuses are the history-row statuses a client may set.
var validStageStatuses = map[string]bool{
	"pending":   true,
	"active":    true,
	"completed": true,
	"skipped":   true,
	"cancelled": true,
}

// applyStageStatusUpdate folds the request into the stage row: an explicit
// completed_at wins, completing a stage without one stamps now, and moving back
// to a non-final status clears it. Pure — the caller persists the result.
func applyStageStatusUpdate(stage *model.JobStage, req *model.UpdateStageRequest, now time.Time) {
	if req.Status != nil {
		stage.Status = *req.Status
	}

	switch {
	case req.CompletedAt != nil:
		stage.CompletedAt = req.CompletedAt
	case req.Status != nil && *req.Status == "completed" && stage.CompletedAt == nil:
		stage.CompletedAt = &now
	}

	if req.Status != nil && *req.Status != "completed" && *req.Status != "skipped" && *req.Status != "cancelled" {
		stage.CompletedAt = nil
	}
}

// DeleteStage deletes a stage from a job's history. When the deleted row is the
// card's current one, the pointer is recalculated from the rows that remain —
// all of it under the job lock, so a concurrent AddStage cannot have the
// pointer moved out from under the recalculation.
func (s *JobService) DeleteStage(ctx context.Context, userID, jobID, stageID string) error {
	// Verify job belongs to user (read, outside tx)
	if _, err := s.repo.GetByID(ctx, userID, jobID); err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	state, err := lockJob(ctx, tx, userID, jobID)
	if err != nil {
		return err
	}

	if _, err := readStage(ctx, tx, jobID, stageID); err != nil {
		return err
	}

	// If this is the current stage, repoint the job first (the FK forbids
	// deleting a row the job still points at).
	if state.CurrentStageID != nil && *state.CurrentStageID == stageID {
		replacement, err := s.replacementStage(ctx, tx, jobID, stageID)
		if err != nil {
			return err
		}
		// current_stage_template_id (the column the card is shown in) must follow
		// current_stage_id, otherwise the header keeps showing the deleted stage.
		if _, err := tx.Exec(ctx,
			`UPDATE jobs SET current_stage_id = $2, current_stage_template_id = $3, updated_at = $4 WHERE id = $1`,
			jobID, replacement.id, replacement.templateID, time.Now().UTC(),
		); err != nil {
			return fmt.Errorf("failed to update job current stage: %w", err)
		}
	}

	// Delete the stage within the transaction
	if _, err := tx.Exec(ctx,
		`DELETE FROM job_stages WHERE id = $1 AND job_id = $2`,
		stageID, jobID,
	); err != nil {
		return fmt.Errorf("failed to delete stage: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("stage deleted",
		zap.String("job_id", jobID),
		zap.String("stage_id", stageID),
		zap.String("user_id", userID))
	return nil
}

// replacementStage is the history row a card falls back to; both fields are nil
// when no row remains.
type replacementStage struct {
	id         *string
	templateID *string
}

// replacementStage picks the history row a card falls back to when its
// current one is deleted: the most recent active or completed row that remains.
// Read inside the transaction so it sees exactly the rows the DELETE will
// leave behind.
func (s *JobService) replacementStage(ctx context.Context, tx pgx.Tx, jobID, deletedStageID string) (replacementStage, error) {
	var r replacementStage
	err := tx.QueryRow(ctx,
		`SELECT id, stage_template_id FROM job_stages
		 WHERE job_id = $1 AND id <> $2 AND status IN ('active', 'completed')
		 ORDER BY "order" DESC, created_at DESC
		 LIMIT 1`,
		jobID, deletedStageID,
	).Scan(&r.id, &r.templateID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return replacementStage{}, fmt.Errorf("failed to find replacement stage: %w", err)
	}
	return r, nil
}

// readStage loads one history row of a job inside the transaction.
func readStage(ctx context.Context, tx pgx.Tx, jobID, stageID string) (*model.JobStage, error) {
	stage := &model.JobStage{ID: stageID, JobID: jobID}
	err := tx.QueryRow(ctx,
		`SELECT stage_template_id, status, "order", started_at, completed_at, created_at
		 FROM job_stages WHERE id = $1 AND job_id = $2`,
		stageID, jobID,
	).Scan(&stage.StageTemplateID, &stage.Status, &stage.Order, &stage.StartedAt, &stage.CompletedAt, &stage.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrJobStageNotFound
		}
		return nil, fmt.Errorf("failed to read stage: %w", err)
	}
	return stage, nil
}

// openStageName returns the column name of the card's current history row when
// that row is still open, and "" when there is nothing to close — including
// when the pointer dangles because a concurrent delete removed the row.
func (s *JobService) openStageName(ctx context.Context, tx pgx.Tx, userID, jobID string, currentStageID *string) (string, error) {
	if currentStageID == nil || *currentStageID == "" {
		return "", nil
	}

	stage, err := readStage(ctx, tx, jobID, *currentStageID)
	if errors.Is(err, model.ErrJobStageNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if stage.Status == "completed" {
		return "", nil
	}

	// Read on the transaction's connection, not through the pool — see
	// readStageTemplate.
	template, err := readStageTemplate(ctx, tx, userID, stage.StageTemplateID)
	if errors.Is(err, model.ErrStageTemplateNotFound) {
		// Only the log line loses detail if the column is gone.
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return template.Name, nil
}
