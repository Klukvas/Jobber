package service

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
)

// MockStageTemplateRepository implements ports.StageTemplateRepository for the
// service tests. Each method delegates to a *Func field when set, otherwise
// returns a benign default. Recreated here after the single-pipeline refactor
// (the previous shared mock lived in the now-deleted stage_template_phase_test.go)
// and now implements the full interface including Reorder.
type MockStageTemplateRepository struct {
	CreateFunc  func(ctx context.Context, template *model.StageTemplate) error
	GetByIDFunc func(ctx context.Context, userID, templateID string) (*model.StageTemplate, error)
	ListFunc    func(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error)
	UpdateFunc  func(ctx context.Context, template *model.StageTemplate) error
	ReorderFunc func(ctx context.Context, userID string, orderedIDs []string) error
	DeleteFunc  func(ctx context.Context, userID, templateID string) error
}

func (m *MockStageTemplateRepository) Create(ctx context.Context, template *model.StageTemplate) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, template)
	}
	return nil
}

func (m *MockStageTemplateRepository) GetByID(ctx context.Context, userID, templateID string) (*model.StageTemplate, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, templateID)
	}
	return &model.StageTemplate{ID: templateID, UserID: userID}, nil
}

func (m *MockStageTemplateRepository) List(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, userID, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockStageTemplateRepository) Update(ctx context.Context, template *model.StageTemplate) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, template)
	}
	return nil
}

func (m *MockStageTemplateRepository) Reorder(ctx context.Context, userID string, orderedIDs []string) error {
	if m.ReorderFunc != nil {
		return m.ReorderFunc(ctx, userID, orderedIDs)
	}
	return nil
}

func (m *MockStageTemplateRepository) Delete(ctx context.Context, userID, templateID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, templateID)
	}
	return nil
}

// singleColumnTemplateRepo returns a MockStageTemplateRepository whose List
// yields one first-column template (order 0) — the placement a bare Create
// falls back to. GetByID echoes the requested template so explicit placements
// resolve too.
func singleColumnTemplateRepo(firstID, firstName string) *MockStageTemplateRepository {
	return &MockStageTemplateRepository{
		ListFunc: func(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error) {
			return []*model.StageTemplate{
				{ID: firstID, UserID: userID, Name: firstName, Order: 0},
			}, 1, nil
		},
		GetByIDFunc: func(ctx context.Context, userID, templateID string) (*model.StageTemplate, error) {
			return &model.StageTemplate{ID: templateID, UserID: userID, Name: firstName, Order: 0}, nil
		},
	}
}

// MockJobStageRepository implements ports.JobStageRepository for the service
// tests. Only the read methods are exercised by the current suite; writes
// return benign defaults.
type MockJobStageRepository struct {
	CreateFunc    func(ctx context.Context, stage *model.JobStage) error
	GetByIDFunc   func(ctx context.Context, stageID, jobID string) (*model.JobStage, error)
	ListByJobFunc func(ctx context.Context, jobID string) ([]*model.JobStage, error)
	UpdateFunc    func(ctx context.Context, stage *model.JobStage) error
	DeleteFunc    func(ctx context.Context, stageID string) error
}

func (m *MockJobStageRepository) Create(ctx context.Context, stage *model.JobStage) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, stage)
	}
	return nil
}

func (m *MockJobStageRepository) GetByID(ctx context.Context, stageID, jobID string) (*model.JobStage, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, stageID, jobID)
	}
	return &model.JobStage{ID: stageID, JobID: jobID}, nil
}

func (m *MockJobStageRepository) ListByJob(ctx context.Context, jobID string) ([]*model.JobStage, error) {
	if m.ListByJobFunc != nil {
		return m.ListByJobFunc(ctx, jobID)
	}
	return nil, nil
}

func (m *MockJobStageRepository) Update(ctx context.Context, stage *model.JobStage) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, stage)
	}
	return nil
}

func (m *MockJobStageRepository) Delete(ctx context.Context, stageID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, stageID)
	}
	return nil
}
