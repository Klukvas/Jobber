package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/jackc/pgx/v5"
)

// lockedJobState is a card's pipeline state as read inside a transaction while
// holding the row lock.
type lockedJobState struct {
	CurrentStageTemplateID *string
	CurrentStageID         *string
	AppliedAt              *time.Time
}

// InColumn reports whether the card already sits in the given column.
func (s *lockedJobState) InColumn(templateID string) bool {
	return s.CurrentStageTemplateID != nil && *s.CurrentStageTemplateID == templateID
}

// lockJob locks the user's card and returns its pipeline state as of that
// moment.
//
// Every pipeline write starts here instead of trusting the copy fetched before
// the transaction. Between that read and the lock another request can move the
// card, so decisions taken on the stale copy are simply wrong: a move looks
// like a no-op and silently does nothing, applied_at gets stamped a second
// time, or current_stage_id is recalculated from a pointer that has already
// changed. The lock also serializes AddStage/UpdateStage/DeleteStage/Move on
// the same card against each other.
//
// Scoping the lock to user_id (not just the id) keeps another user's card
// unreachable here even if the caller's ownership check were ever dropped, and
// turns a deleted card into ErrJobNotFound rather than a silent no-op.
func lockJob(ctx context.Context, tx pgx.Tx, userID, jobID string) (*lockedJobState, error) {
	state := &lockedJobState{}
	err := tx.QueryRow(ctx,
		`SELECT current_stage_template_id, current_stage_id, applied_at
		 FROM jobs WHERE id = $1 AND user_id = $2 FOR UPDATE`,
		jobID, userID,
	).Scan(&state.CurrentStageTemplateID, &state.CurrentStageID, &state.AppliedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to lock job: %w", err)
	}
	return state, nil
}

// completeOpenStages closes a card's ACTIVE history rows, except an optionally
// excluded one, keeping the invariant that a card has at most one active stage.
//
// Closing the whole active set rather than only the row current_stage_id points
// at is what makes the invariant hold: a card that already carries two active
// rows — from a crash between the insert and the pointer update, or from a
// client that re-activated an old entry — is healed by the next move instead of
// double-counting forever.
//
// A 'pending' row is deliberately left alone. Pending means "planned, not
// started": the user can schedule rounds ahead of time from the stage status
// dialog ("Stage has not started yet"), several may exist at once, and the
// invariant only ever spoke about active rows. Completing them on the next move
// marked interviews the card had not even reached as done and dated them to the
// move. 'skipped' and 'cancelled' are final for the same reason — the user set
// them on purpose.
func completeOpenStages(ctx context.Context, tx pgx.Tx, jobID string, at time.Time, exceptStageID *string) error {
	if _, err := tx.Exec(ctx,
		`UPDATE job_stages SET status = 'completed', completed_at = $2
		 WHERE job_id = $1
		   AND status = 'active'
		   AND ($3::uuid IS NULL OR id <> $3::uuid)`,
		jobID, at, exceptStageID,
	); err != nil {
		return fmt.Errorf("failed to complete current stage: %w", err)
	}
	return nil
}

// readStageTemplate loads one of the user's pipeline columns on the
// transaction's own connection.
//
// The pool-backed repository must not be used while a pipeline write is open:
// the transaction already holds a connection and the card's row lock, so
// asking the pool for a second one makes the lock holder queue behind the very
// writers that are waiting for its lock. With one connection that deadlocks
// outright; with a production-sized pool it takes only as many simultaneous
// writers as there are connections. Reading through tx keeps the write on the
// single connection it already owns.
//
// Scoped by user_id like the repository, so another user's column is
// unreachable here and a deleted one reports ErrStageTemplateNotFound. Only
// the fields a pipeline write decides on are selected: the name for the DTO
// and the log line, kind and order for the applied_at rule.
func readStageTemplate(ctx context.Context, tx pgx.Tx, userID, templateID string) (*model.StageTemplate, error) {
	template := &model.StageTemplate{ID: templateID, UserID: userID}
	// kind is scanned as a plain string and converted, the same way the
	// repository does it, so the model keeps its own enum type.
	var kind string
	err := tx.QueryRow(ctx,
		`SELECT name, "order", kind FROM stage_templates WHERE id = $1 AND user_id = $2`,
		templateID, userID,
	).Scan(&template.Name, &template.Order, &kind)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrStageTemplateNotFound
		}
		return nil, fmt.Errorf("failed to read stage template: %w", err)
	}
	template.Kind = model.StageKind(kind)
	return template, nil
}

// nextStageOrder returns the next history order for a card: max existing + 1,
// read under the job lock. MAX (not COUNT) keeps orders unique after a middle
// stage is deleted.
func nextStageOrder(ctx context.Context, tx pgx.Tx, jobID string) (int, error) {
	var order int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(MAX("order"), -1) + 1 FROM job_stages WHERE job_id = $1`, jobID,
	).Scan(&order); err != nil {
		return 0, fmt.Errorf("failed to compute stage order: %w", err)
	}
	return order, nil
}
