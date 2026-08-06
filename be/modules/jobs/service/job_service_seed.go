package service

import (
	"context"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
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

// SeedDefaultStageTemplates creates the starter templates for a new user.
func (s *JobService) SeedDefaultStageTemplates(ctx context.Context, userID string) error {
	for _, tpl := range defaultStageTemplates {
		template := &model.StageTemplate{
			UserID: userID,
			Name:   tpl.Name,
			Order:  tpl.Order,
			Phase:  tpl.Phase,
		}
		if err := s.templateRepo.Create(ctx, template); err != nil {
			return fmt.Errorf("seed template %q: %w", tpl.Name, err)
		}
	}
	return nil
}
