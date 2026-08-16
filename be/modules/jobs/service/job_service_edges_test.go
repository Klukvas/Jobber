package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Small edge-case tests closing the remaining branch gaps in the non-
// transactional service methods.

func TestJobService_Create_ExplicitTemplateError(t *testing.T) {
	// An explicit stage_template_id that fails to resolve propagates the error.
	tmplRepo := &MockStageTemplateRepository{
		GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return nil, model.ErrStageTemplateNotFound
		},
	}
	svc := newServiceWithTemplates(&MockJobRepository{}, tmplRepo)

	explicit := "missing-column"
	_, err := svc.Create(context.Background(), "user-1", &model.CreateJobRequest{
		Title:           "Engineer",
		StageTemplateID: &explicit,
	})
	assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
}

func TestJobService_Create_FirstColumnListError(t *testing.T) {
	// firstStageTemplate surfaces a repository List error (not just "empty").
	tmplRepo := &MockStageTemplateRepository{
		ListFunc: func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
			return nil, 0, errors.New("list boom")
		},
	}
	svc := newServiceWithTemplates(&MockJobRepository{}, tmplRepo)

	_, err := svc.Create(context.Background(), "user-1", &model.CreateJobRequest{Title: "Engineer"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list boom")
}

func TestJobService_ListStageTemplates_Error(t *testing.T) {
	tmplRepo := &MockStageTemplateRepository{
		ListFunc: func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
			return nil, 0, errors.New("list boom")
		},
	}
	svc := newServiceWithTemplates(&MockJobRepository{}, tmplRepo)

	dtos, total, err := svc.ListStageTemplates(context.Background(), "user-1", 20, 0)
	require.Error(t, err)
	assert.Nil(t, dtos)
	assert.Equal(t, 0, total)
}

func TestJobService_UpdateStageTemplate_OrderOnly(t *testing.T) {
	// Updating only the order leaves the name untouched.
	existing := &model.StageTemplate{ID: "s1", UserID: "user-1", Name: "Screening", Order: 2}
	var saved *model.StageTemplate
	tmplRepo := &MockStageTemplateRepository{
		GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) { return existing, nil },
		UpdateFunc: func(ctx context.Context, tmpl *model.StageTemplate) error {
			saved = tmpl
			return nil
		},
	}
	svc := newServiceWithTemplates(&MockJobRepository{}, tmplRepo)

	newOrder := 5
	dto, err := svc.UpdateStageTemplate(context.Background(), "user-1", "s1", &model.UpdateStageTemplateRequest{Order: &newOrder})

	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, "Screening", saved.Name) // unchanged
	assert.Equal(t, 5, saved.Order)
	assert.Equal(t, 5, dto.Order)
}

func TestJobService_UpdateStageTemplate_RepoUpdateError(t *testing.T) {
	existing := &model.StageTemplate{ID: "s1", UserID: "user-1", Name: "Screening"}
	tmplRepo := &MockStageTemplateRepository{
		GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) { return existing, nil },
		UpdateFunc:  func(ctx context.Context, tmpl *model.StageTemplate) error { return errors.New("update boom") },
	}
	svc := newServiceWithTemplates(&MockJobRepository{}, tmplRepo)

	name := "Final"
	_, err := svc.UpdateStageTemplate(context.Background(), "user-1", "s1", &model.UpdateStageTemplateRequest{Name: &name})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update boom")
}

func TestJobService_CreateStageTemplate_PersistsOrder(t *testing.T) {
	// Guards the Create path's repo error propagation distinctly from name checks.
	tmplRepo := &MockStageTemplateRepository{
		CreateFunc: func(ctx context.Context, tmpl *model.StageTemplate) error { return errors.New("create boom") },
	}
	svc := newServiceWithTemplates(&MockJobRepository{}, tmplRepo)

	_, err := svc.CreateStageTemplate(context.Background(), "user-1", &model.CreateStageTemplateRequest{Name: "Applied", Order: 3})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create boom")
}

func TestJobService_Update_ResumeSelectionError(t *testing.T) {
	// A resume-selection failure during Update propagates (both types set here).
	existing := &model.Job{ID: "job-1", UserID: "user-1", Title: "Engineer"}
	jobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) { return existing, nil },
	}
	svc := newServiceWithTemplates(jobRepo, &MockStageTemplateRepository{})

	rID := "r-1"
	rbID := "rb-1"
	_, err := svc.Update(context.Background(), "user-1", "job-1", &model.UpdateJobRequest{
		ResumeID:        &rID,
		ResumeBuilderID: &rbID,
	})
	assert.ErrorIs(t, err, model.ErrBothResumeTypesSet)
}

func TestJobService_Create_ResumeNotFound(t *testing.T) {
	// Attaching a foreign/missing uploaded resume on Create is rejected.
	tmplRepo := singleColumnTemplateRepo("stage-wishlist", "Wishlist")
	resumeRepo := &ownedResumeRepo{ownErr: errors.New("not owned")}
	svc := NewJobService(nil, &MockJobRepository{}, nil, tmplRepo, defaultMockCompanyRepo,
		resumeRepo, &ownedRBRepo{}, defaultMockCommentRepo, nil, nil, nil)

	rID := "r-foreign"
	_, err := svc.Create(context.Background(), "user-1", &model.CreateJobRequest{Title: "Engineer", ResumeID: &rID})
	assert.ErrorIs(t, err, model.ErrResumeNotFound)
}
