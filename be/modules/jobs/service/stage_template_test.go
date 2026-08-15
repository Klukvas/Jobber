package service

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the stage-template lifecycle that runs purely through the
// template repository (no *pgxpool.Pool), exercising the phase-free
// StageTemplate model after the single-pipeline refactor.

func TestJobService_CreateStageTemplate(t *testing.T) {
	userID := "user-123"

	t.Run("creates a column and trims the name", func(t *testing.T) {
		var created *model.StageTemplate
		templateRepo := &MockStageTemplateRepository{
			CreateFunc: func(ctx context.Context, template *model.StageTemplate) error {
				created = template
				template.ID = "stage-1"
				return nil
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		dto, err := svc.CreateStageTemplate(context.Background(), userID, &model.CreateStageTemplateRequest{
			Name:  "  Screening  ",
			Order: 2,
		})

		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, "Screening", created.Name)
		assert.Equal(t, 2, created.Order)
		assert.Equal(t, userID, created.UserID)
		assert.Equal(t, "Screening", dto.Name)
		assert.Equal(t, 2, dto.Order)
	})

	t.Run("rejects an empty name", func(t *testing.T) {
		svc := newServiceWithTemplates(&MockJobRepository{}, &MockStageTemplateRepository{})

		dto, err := svc.CreateStageTemplate(context.Background(), userID, &model.CreateStageTemplateRequest{Name: "   "})

		assert.Nil(t, dto)
		assert.ErrorIs(t, err, model.ErrStageNameRequired)
	})

	t.Run("propagates a duplicate-name error", func(t *testing.T) {
		templateRepo := &MockStageTemplateRepository{
			CreateFunc: func(ctx context.Context, template *model.StageTemplate) error {
				return model.ErrStageTemplateNameExists
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		_, err := svc.CreateStageTemplate(context.Background(), userID, &model.CreateStageTemplateRequest{Name: "Applied"})

		assert.ErrorIs(t, err, model.ErrStageTemplateNameExists)
	})
}

func TestJobService_ListStageTemplates(t *testing.T) {
	userID := "user-123"

	t.Run("returns the pipeline columns in order", func(t *testing.T) {
		templateRepo := &MockStageTemplateRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
				return []*model.StageTemplate{
					{ID: "s0", UserID: uid, Name: "Wishlist", Order: 0},
					{ID: "s1", UserID: uid, Name: "Applied", Order: 1},
				}, 2, nil
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		dtos, total, err := svc.ListStageTemplates(context.Background(), userID, 20, 0)

		require.NoError(t, err)
		assert.Equal(t, 2, total)
		require.Len(t, dtos, 2)
		assert.Equal(t, "Wishlist", dtos[0].Name)
		assert.Equal(t, 0, dtos[0].Order)
		assert.Equal(t, "Applied", dtos[1].Name)
		assert.Equal(t, 1, dtos[1].Order)
	})
}

func TestJobService_UpdateStageTemplate(t *testing.T) {
	userID := "user-123"
	templateID := "stage-1"

	t.Run("updates name and order", func(t *testing.T) {
		existing := &model.StageTemplate{ID: templateID, UserID: userID, Name: "Old", Order: 3}
		var saved *model.StageTemplate
		templateRepo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return existing, nil
			},
			UpdateFunc: func(ctx context.Context, template *model.StageTemplate) error {
				saved = template
				return nil
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		newName := "Final Interview"
		newOrder := 4
		dto, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{
			Name:  &newName,
			Order: &newOrder,
		})

		require.NoError(t, err)
		require.NotNil(t, saved)
		assert.Equal(t, "Final Interview", saved.Name)
		assert.Equal(t, 4, saved.Order)
		assert.Equal(t, "Final Interview", dto.Name)
		assert.Equal(t, 4, dto.Order)
	})

	t.Run("rejects clearing the name to whitespace", func(t *testing.T) {
		existing := &model.StageTemplate{ID: templateID, UserID: userID, Name: "Old"}
		templateRepo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return existing, nil
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		blank := "   "
		_, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{Name: &blank})

		assert.ErrorIs(t, err, model.ErrStageNameRequired)
	})

	t.Run("returns not found when the column is missing", func(t *testing.T) {
		templateRepo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return nil, model.ErrStageTemplateNotFound
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		newName := "X"
		_, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{Name: &newName})

		assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
	})
}

func TestJobService_DeleteStageTemplate(t *testing.T) {
	userID := "user-123"

	t.Run("deletes a column", func(t *testing.T) {
		var deleted string
		templateRepo := &MockStageTemplateRepository{
			DeleteFunc: func(ctx context.Context, uid, tid string) error {
				deleted = tid
				return nil
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		err := svc.DeleteStageTemplate(context.Background(), userID, "stage-1")

		require.NoError(t, err)
		assert.Equal(t, "stage-1", deleted)
	})

	t.Run("propagates an in-use error", func(t *testing.T) {
		templateRepo := &MockStageTemplateRepository{
			DeleteFunc: func(ctx context.Context, uid, tid string) error {
				return model.ErrStageTemplateInUse
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		err := svc.DeleteStageTemplate(context.Background(), userID, "stage-1")

		assert.ErrorIs(t, err, model.ErrStageTemplateInUse)
	})
}
