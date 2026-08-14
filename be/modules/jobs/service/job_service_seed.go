package service

import (
	"context"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/google/uuid"
)

// defaultStageTemplates is the starter pipeline for fresh accounts. Only
// in_progress steps — the unified board's base columns already cover
// Wishlist/Applied/Offer/Rejected, so seeding those would duplicate them.
var defaultStageTemplates = []struct {
	Name  string
	Order int
	Phase model.Phase
}{
	{"Screening", 1, model.PhaseInProgress},
	{"Technical Interview", 2, model.PhaseInProgress},
	{"Final Interview", 3, model.PhaseInProgress},
}

// SeedDefaultStageTemplates creates the starter templates for a new user in a
// single transaction, so a partial failure never leaves an account with only
// some of its default pipeline.
func (s *JobService) SeedDefaultStageTemplates(ctx context.Context, userID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin seed transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is a no-op after commit

	now := time.Now().UTC()
	for _, tpl := range defaultStageTemplates {
		if _, err := tx.Exec(ctx,
			`INSERT INTO stage_templates (id, user_id, name, "order", phase, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
			uuid.New().String(), userID, tpl.Name, tpl.Order, tpl.Phase, now,
		); err != nil {
			return fmt.Errorf("seed template %q: %w", tpl.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	return nil
}
