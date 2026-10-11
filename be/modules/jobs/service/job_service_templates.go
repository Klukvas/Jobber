package service

import (
	"context"
	"strings"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
)

// Stage templates — the user's pipeline columns.

// CreateStageTemplate adds a column to the user's pipeline. Without an explicit
// kind the column is "other": a column whose meaning nobody stated must not
// start counting towards response or interview metrics on its own.
func (s *JobService) CreateStageTemplate(ctx context.Context, userID string, req *model.CreateStageTemplateRequest) (*model.StageTemplateDTO, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, model.ErrStageNameRequired
	}

	kind := model.StageKindOther
	if req.Kind != nil {
		parsed, err := model.ParseStageKind(*req.Kind)
		if err != nil {
			return nil, err
		}
		kind = parsed
	}

	template := &model.StageTemplate{
		UserID: userID,
		Name:   strings.TrimSpace(req.Name),
		Order:  req.Order,
		Kind:   kind,
	}

	if err := s.templateRepo.Create(ctx, template); err != nil {
		return nil, err
	}
	return template.ToDTO(), nil
}

// ReorderStageTemplates sets a new order for the user's pipeline columns.
// Kinds are untouched: dragging a column does not change what it means.
func (s *JobService) ReorderStageTemplates(ctx context.Context, userID string, stageIDs []string) error {
	return s.templateRepo.Reorder(ctx, userID, stageIDs)
}

func (s *JobService) ListStageTemplates(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplateDTO, int, error) {
	templates, total, err := s.templateRepo.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*model.StageTemplateDTO, len(templates))
	for i, t := range templates {
		dtos[i] = t.ToDTO()
	}
	return dtos, total, nil
}

// UpdateStageTemplate renames, reorders or reclassifies a column. Only an
// explicit kind changes the classification — a rename keeps it, which is the
// whole point of storing the meaning separately from the label.
//
// Only the fields the request carries are written, and the stored row comes
// back from the write itself. The column is never read, mutated in memory and
// written back whole: that let a rename and a reclassification issued at the
// same time overwrite each other, so one of the two edits silently vanished.
func (s *JobService) UpdateStageTemplate(ctx context.Context, userID, templateID string, req *model.UpdateStageTemplateRequest) (*model.StageTemplateDTO, error) {
	patch := &model.StageTemplatePatch{Order: req.Order}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, model.ErrStageNameRequired
		}
		patch.Name = &name
	}
	if req.Kind != nil {
		kind, err := model.ParseStageKind(*req.Kind)
		if err != nil {
			return nil, err
		}
		patch.Kind = &kind
	}

	template, err := s.templateRepo.Update(ctx, userID, templateID, patch)
	if err != nil {
		return nil, err
	}
	return template.ToDTO(), nil
}

func (s *JobService) DeleteStageTemplate(ctx context.Context, userID, templateID string) error {
	return s.templateRepo.Delete(ctx, userID, templateID)
}
