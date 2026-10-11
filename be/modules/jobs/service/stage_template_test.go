package service

import (
	"context"
	"errors"
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

	t.Run("defaults a column with no kind to other", func(t *testing.T) {
		var created *model.StageTemplate
		templateRepo := &MockStageTemplateRepository{
			CreateFunc: func(ctx context.Context, template *model.StageTemplate) error {
				created = template
				return nil
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		dto, err := svc.CreateStageTemplate(context.Background(), userID, &model.CreateStageTemplateRequest{Name: "Follow-up"})

		require.NoError(t, err)
		assert.Equal(t, model.StageKindOther, created.Kind,
			"a column nobody classified must not start counting towards response metrics")
		assert.Equal(t, model.StageKindOther, dto.Kind)
	})

	t.Run("stores the requested kind", func(t *testing.T) {
		var created *model.StageTemplate
		templateRepo := &MockStageTemplateRepository{
			CreateFunc: func(ctx context.Context, template *model.StageTemplate) error {
				created = template
				return nil
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		kind := "interview"
		dto, err := svc.CreateStageTemplate(context.Background(), userID, &model.CreateStageTemplateRequest{
			Name: "Onsite loop",
			Kind: &kind,
		})

		require.NoError(t, err)
		assert.Equal(t, model.StageKindInterview, created.Kind)
		assert.Equal(t, model.StageKindInterview, dto.Kind)
	})

	t.Run("rejects an unknown kind", func(t *testing.T) {
		svc := newServiceWithTemplates(&MockJobRepository{}, &MockStageTemplateRepository{})

		kind := "ghosted"
		dto, err := svc.CreateStageTemplate(context.Background(), userID, &model.CreateStageTemplateRequest{
			Name: "Ghosted",
			Kind: &kind,
		})

		assert.Nil(t, dto)
		assert.ErrorIs(t, err, model.ErrInvalidStageKind)
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

	// patchRecorder wires a template repo whose Update records the patch it was
	// given and answers with stored + patch, the way the field-scoped UPDATE
	// does. What the patch leaves nil is what the statement never touches. The
	// returned getter reads the recorded patch back after the call.
	patchRecorder := func(t *testing.T, stored *model.StageTemplate) (*MockStageTemplateRepository, func() *model.StageTemplatePatch) {
		t.Helper()
		var recorded *model.StageTemplatePatch
		repo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return nil, errors.New("UpdateStageTemplate must not read the column before writing it")
			},
			UpdateFunc: func(ctx context.Context, uid, tid string, patch *model.StageTemplatePatch) (*model.StageTemplate, error) {
				recorded = patch
				return applyStageTemplatePatch(stored, patch), nil
			},
		}
		return repo, func() *model.StageTemplatePatch { return recorded }
	}

	t.Run("updates name and order", func(t *testing.T) {
		templateRepo, recordedPatch := patchRecorder(t, &model.StageTemplate{ID: templateID, UserID: userID, Name: "Old", Order: 3})
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		newName := "Final Interview"
		newOrder := 4
		dto, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{
			Name:  &newName,
			Order: &newOrder,
		})

		require.NoError(t, err)
		patch := recordedPatch()
		require.NotNil(t, patch)
		require.NotNil(t, patch.Name)
		assert.Equal(t, "Final Interview", *patch.Name)
		require.NotNil(t, patch.Order)
		assert.Equal(t, 4, *patch.Order)
		assert.Equal(t, "Final Interview", dto.Name)
		assert.Equal(t, 4, dto.Order)
	})

	t.Run("trims the new name", func(t *testing.T) {
		templateRepo, recordedPatch := patchRecorder(t, &model.StageTemplate{ID: templateID, UserID: userID, Name: "Old"})
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		padded := "  Onsite loop  "
		dto, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{Name: &padded})

		require.NoError(t, err)
		patch := recordedPatch()
		require.NotNil(t, patch.Name)
		assert.Equal(t, "Onsite loop", *patch.Name)
		assert.Equal(t, "Onsite loop", dto.Name)
	})

	t.Run("renaming and reordering leave the kind out of the write", func(t *testing.T) {
		templateRepo, recordedPatch := patchRecorder(t, &model.StageTemplate{
			ID: templateID, UserID: userID, Name: "Technical Interview", Order: 3, Kind: model.StageKindInterview,
		})
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		newName := "Onsite loop"
		newOrder := 1
		dto, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{
			Name:  &newName,
			Order: &newOrder,
		})

		require.NoError(t, err)
		assert.Nil(t, recordedPatch().Kind,
			"a rename must not write the kind at all — otherwise it carries a stale one over a concurrent reclassification")
		assert.Equal(t, model.StageKindInterview, dto.Kind,
			"a renamed interview column is still an interview column")
	})

	t.Run("reclassifies a column when a kind is sent", func(t *testing.T) {
		templateRepo, recordedPatch := patchRecorder(t, &model.StageTemplate{
			ID: templateID, UserID: userID, Name: "Recruiter ping", Kind: model.StageKindOther,
		})
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		kind := "response"
		dto, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{Kind: &kind})

		require.NoError(t, err)
		patch := recordedPatch()
		require.NotNil(t, patch.Kind)
		assert.Equal(t, model.StageKindResponse, *patch.Kind)
		assert.Equal(t, model.StageKindResponse, dto.Kind)
		assert.Nil(t, patch.Name, "a reclassification must not write the label")
		assert.Nil(t, patch.Order, "a reclassification must not write the position")
		assert.Equal(t, "Recruiter ping", dto.Name, "the label is untouched")
	})

	t.Run("an empty request writes no column but updated_at", func(t *testing.T) {
		templateRepo, recordedPatch := patchRecorder(t, &model.StageTemplate{
			ID: templateID, UserID: userID, Name: "Screening", Order: 2, Kind: model.StageKindResponse,
		})
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		dto, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{})

		require.NoError(t, err)
		patch := recordedPatch()
		require.NotNil(t, patch)
		assert.Nil(t, patch.Name)
		assert.Nil(t, patch.Order)
		assert.Nil(t, patch.Kind)
		assert.Equal(t, "Screening", dto.Name)
		assert.Equal(t, 2, dto.Order)
		assert.Equal(t, model.StageKindResponse, dto.Kind)
	})

	t.Run("rejects an unknown kind", func(t *testing.T) {
		templateRepo := &MockStageTemplateRepository{
			UpdateFunc: func(ctx context.Context, uid, tid string, patch *model.StageTemplatePatch) (*model.StageTemplate, error) {
				t.Fatalf("an invalid kind must not reach the repository")
				return nil, nil
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		kind := "ghosted"
		_, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{Kind: &kind})

		assert.ErrorIs(t, err, model.ErrInvalidStageKind)
	})

	t.Run("rejects clearing the name to whitespace", func(t *testing.T) {
		templateRepo := &MockStageTemplateRepository{
			UpdateFunc: func(ctx context.Context, uid, tid string, patch *model.StageTemplatePatch) (*model.StageTemplate, error) {
				t.Fatalf("a blank name must not reach the repository")
				return nil, nil
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		blank := "   "
		_, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{Name: &blank})

		assert.ErrorIs(t, err, model.ErrStageNameRequired)
	})

	t.Run("returns not found when the column is missing", func(t *testing.T) {
		// The write itself reports it: no row matched the id/owner pair.
		templateRepo := &MockStageTemplateRepository{
			UpdateFunc: func(ctx context.Context, uid, tid string, patch *model.StageTemplatePatch) (*model.StageTemplate, error) {
				return nil, model.ErrStageTemplateNotFound
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		newName := "X"
		_, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{Name: &newName})

		assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
	})

	t.Run("scopes the write to the caller", func(t *testing.T) {
		var gotUser, gotTemplate string
		templateRepo := &MockStageTemplateRepository{
			UpdateFunc: func(ctx context.Context, uid, tid string, patch *model.StageTemplatePatch) (*model.StageTemplate, error) {
				gotUser, gotTemplate = uid, tid
				return &model.StageTemplate{ID: tid, UserID: uid, Name: "Screening"}, nil
			},
		}
		svc := newServiceWithTemplates(&MockJobRepository{}, templateRepo)

		newOrder := 2
		_, err := svc.UpdateStageTemplate(context.Background(), userID, templateID, &model.UpdateStageTemplateRequest{Order: &newOrder})

		require.NoError(t, err)
		assert.Equal(t, userID, gotUser)
		assert.Equal(t, templateID, gotTemplate)
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
