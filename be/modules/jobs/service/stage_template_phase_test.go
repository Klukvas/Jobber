package service

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockStageTemplateRepository implements ports.StageTemplateRepository
type MockStageTemplateRepository struct {
	CreateFunc  func(ctx context.Context, template *model.StageTemplate) error
	GetByIDFunc func(ctx context.Context, userID, templateID string) (*model.StageTemplate, error)
	UpdateFunc  func(ctx context.Context, template *model.StageTemplate) error
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
	return nil, model.ErrStageTemplateNotFound
}

func (m *MockStageTemplateRepository) List(ctx context.Context, userID string, limit, offset int) ([]*model.StageTemplate, int, error) {
	return nil, 0, nil
}

func (m *MockStageTemplateRepository) Update(ctx context.Context, template *model.StageTemplate) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, template)
	}
	return nil
}

func (m *MockStageTemplateRepository) Delete(ctx context.Context, userID, templateID string) error {
	return nil
}

func newTemplateService(repo *MockStageTemplateRepository) *JobService {
	return NewJobService(nil, nil, nil, repo, nil, nil, nil, nil, nil, nil, nil)
}

func TestCreateStageTemplate_Phase(t *testing.T) {
	userID := "user-123"

	t.Run("defaults to in_progress when phase omitted", func(t *testing.T) {
		var created *model.StageTemplate
		repo := &MockStageTemplateRepository{
			CreateFunc: func(ctx context.Context, template *model.StageTemplate) error {
				created = template
				return nil
			},
		}

		dto, err := newTemplateService(repo).CreateStageTemplate(context.Background(), userID, &model.CreateStageTemplateRequest{
			Name:  "Tech Interview",
			Order: 3,
		})

		require.NoError(t, err)
		assert.Equal(t, model.PhaseInProgress, created.Phase)
		assert.Equal(t, model.PhaseInProgress, dto.Phase)
	})

	t.Run("accepts an explicit valid phase", func(t *testing.T) {
		var created *model.StageTemplate
		repo := &MockStageTemplateRepository{
			CreateFunc: func(ctx context.Context, template *model.StageTemplate) error {
				created = template
				return nil
			},
		}

		_, err := newTemplateService(repo).CreateStageTemplate(context.Background(), userID, &model.CreateStageTemplateRequest{
			Name:  "Negotiating",
			Order: 1,
			Phase: model.PhaseOffer,
		})

		require.NoError(t, err)
		assert.Equal(t, model.PhaseOffer, created.Phase)
	})

	t.Run("rejects an unknown phase", func(t *testing.T) {
		svc := newTemplateService(&MockStageTemplateRepository{})

		dto, err := svc.CreateStageTemplate(context.Background(), userID, &model.CreateStageTemplateRequest{
			Name:  "Ghosted",
			Phase: model.Phase("rejected"), // terminal — never a template phase
		})

		assert.Nil(t, dto)
		assert.ErrorIs(t, err, model.ErrInvalidPhase)
	})
}

func TestUpdateStageTemplate_Phase(t *testing.T) {
	userID := "user-123"

	existing := func() *model.StageTemplate {
		return &model.StageTemplate{
			ID:     "tpl-1",
			UserID: userID,
			Name:   "Screening",
			Order:  2,
			Phase:  model.PhaseInProgress,
		}
	}

	t.Run("moves template to another phase", func(t *testing.T) {
		var updated *model.StageTemplate
		repo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return existing(), nil
			},
			UpdateFunc: func(ctx context.Context, template *model.StageTemplate) error {
				updated = template
				return nil
			},
		}

		phase := model.PhaseApplied
		dto, err := newTemplateService(repo).UpdateStageTemplate(context.Background(), userID, "tpl-1", &model.UpdateStageTemplateRequest{
			Phase: &phase,
		})

		require.NoError(t, err)
		assert.Equal(t, model.PhaseApplied, updated.Phase)
		assert.Equal(t, model.PhaseApplied, dto.Phase)
	})

	t.Run("rejects an unknown phase on update", func(t *testing.T) {
		repo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return existing(), nil
			},
		}

		bad := model.Phase("nope")
		dto, err := newTemplateService(repo).UpdateStageTemplate(context.Background(), userID, "tpl-1", &model.UpdateStageTemplateRequest{
			Phase: &bad,
		})

		assert.Nil(t, dto)
		assert.ErrorIs(t, err, model.ErrInvalidPhase)
	})

	t.Run("keeps phase when not provided", func(t *testing.T) {
		var updated *model.StageTemplate
		repo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return existing(), nil
			},
			UpdateFunc: func(ctx context.Context, template *model.StageTemplate) error {
				updated = template
				return nil
			},
		}

		newName := "Phone Screening"
		_, err := newTemplateService(repo).UpdateStageTemplate(context.Background(), userID, "tpl-1", &model.UpdateStageTemplateRequest{
			Name: &newName,
		})

		require.NoError(t, err)
		assert.Equal(t, model.PhaseInProgress, updated.Phase)
	})
}

func TestStatusForPhase(t *testing.T) {
	assert.Equal(t, model.StatusSaved, model.StatusForPhase(model.PhaseWishlist))
	assert.Equal(t, model.StatusApplied, model.StatusForPhase(model.PhaseApplied))
	assert.Equal(t, model.StatusApplied, model.StatusForPhase(model.PhaseInProgress))
	assert.Equal(t, model.StatusOffer, model.StatusForPhase(model.PhaseOffer))
}
