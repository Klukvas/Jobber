package service

import (
	"context"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/google/uuid"
)

// defaultPipeline is the starter set of pipeline columns for a fresh account.
// The whole pipeline (steps + outcomes) is one customizable, ordered list — a
// card sits in exactly one column and there is no separate status.
//
// Kind is what analytics reads, so the starter columns ship classified: the
// user can rename "Screening" or move it and the response rate stays the same
// metric.
var defaultPipeline = []struct {
	Name  string
	Order int
	Kind  model.StageKind
}{
	{"Wishlist", 0, model.StageKindSaved},
	{"Applied", 1, model.StageKindApplied},
	{"Screening", 2, model.StageKindResponse},
	{"Technical Interview", 3, model.StageKindInterview},
	{"Final Interview", 4, model.StageKindInterview},
	{"Offer", 5, model.StageKindOffer},
	{"Rejected", 6, model.StageKindRejected},
}

// SeedDefaultStageTemplates creates the starter pipeline for a new user in a
// single transaction, so a partial failure never leaves an account with only
// some of its default columns.
func (s *JobService) SeedDefaultStageTemplates(ctx context.Context, userID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	now := time.Now().UTC()
	for _, col := range defaultPipeline {
		if _, err := tx.Exec(ctx,
			`INSERT INTO stage_templates (id, user_id, name, "order", kind, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
			uuid.New().String(), userID, col.Name, col.Order, string(col.Kind), now,
		); err != nil {
			return fmt.Errorf("seed column %q: %w", col.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	return nil
}
