package service

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Move is the single write path for a card's pipeline position. Its transactional
// body runs against a *pgxpool.Pool that cannot be mocked here, so these tests
// cover the branches that resolve before the transaction begins: validation, the
// pre-transaction lookups, and the no-op fast path (already in the target column),
// which returns an enriched DTO without any writes.

func TestMove_Validation(t *testing.T) {
	userID := "user-123"
	stageID := "stage-applied"

	job := &model.Job{ID: "job-1", UserID: userID, CurrentStageTemplateID: &stageID}
	jobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
			return job, nil
		},
	}

	newSvc := func() *JobService {
		return newServiceWithTemplates(jobRepo, &MockStageTemplateRepository{})
	}

	t.Run("rejects empty stage template id", func(t *testing.T) {
		_, err := newSvc().Move(context.Background(), userID, "job-1", &model.MoveJobRequest{
			StageTemplateID: "",
		})
		assert.ErrorIs(t, err, model.ErrInvalidMoveTarget)
	})

	t.Run("propagates job lookup error", func(t *testing.T) {
		failing := newServiceWithTemplates(&MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}, &MockStageTemplateRepository{})

		_, err := failing.Move(context.Background(), userID, "job-1", &model.MoveJobRequest{
			StageTemplateID: "stage-applied",
		})
		assert.ErrorIs(t, err, model.ErrJobNotFound)
	})

	t.Run("propagates stage template lookup error", func(t *testing.T) {
		templateRepo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, templateID string) (*model.StageTemplate, error) {
				return nil, model.ErrStageTemplateNotFound
			},
		}
		svc := newServiceWithTemplates(jobRepo, templateRepo)

		_, err := svc.Move(context.Background(), userID, "job-1", &model.MoveJobRequest{
			StageTemplateID: "missing",
		})
		assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
	})
}

func TestMove_NoOp(t *testing.T) {
	userID := "user-123"
	stageID := "stage-applied"

	// Card already sits in the target column: Move must return an enriched DTO
	// with no writes (no *pgxpool.Pool is available in this test).
	job := &model.Job{
		ID:                     "job-1",
		UserID:                 userID,
		Title:                  "Engineer",
		CurrentStageTemplateID: &stageID,
	}

	jobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
			return job, nil
		},
		UpdateFunc: func(ctx context.Context, j *model.Job) error {
			t.Fatalf("no-op Move must not update the job")
			return nil
		},
	}
	templateRepo := &MockStageTemplateRepository{
		GetByIDFunc: func(ctx context.Context, uid, templateID string) (*model.StageTemplate, error) {
			return &model.StageTemplate{ID: templateID, UserID: uid, Name: "Applied", Order: 1}, nil
		},
	}

	svc := newServiceWithTemplates(jobRepo, templateRepo)

	dto, err := svc.Move(context.Background(), userID, "job-1", &model.MoveJobRequest{
		StageTemplateID: stageID,
	})

	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, "job-1", dto.ID)
	require.NotNil(t, dto.CurrentStageTemplateID)
	assert.Equal(t, stageID, *dto.CurrentStageTemplateID)
	// The enriched DTO carries the current column's name.
	require.NotNil(t, dto.CurrentStageName)
	assert.Equal(t, "Applied", *dto.CurrentStageName)
}
