package service

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/stretchr/testify/assert"
)

func TestMovePhaseStatus(t *testing.T) {
	cases := []struct {
		phase   model.Phase
		status  string
		wantErr bool
	}{
		{model.PhaseWishlist, "saved", false},
		{model.PhaseApplied, "applied", false},
		{model.PhaseInProgress, "applied", false},
		{model.PhaseOffer, "offer", false},
		{model.Phase("rejected"), "rejected", false}, // terminal board column
		{model.Phase("archived"), "", true},          // never a move target
		{model.Phase("nope"), "", true},
	}
	for _, tc := range cases {
		status, err := movePhaseStatus(tc.phase)
		if tc.wantErr {
			assert.Error(t, err, string(tc.phase))
		} else {
			assert.NoError(t, err, string(tc.phase))
			assert.Equal(t, tc.status, status, string(tc.phase))
		}
	}
}

func TestMove_Validation(t *testing.T) {
	userID := "user-123"
	job := &model.Job{ID: "job-1", UserID: userID, Status: "applied"}

	jobRepo := &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
			return job, nil
		},
	}

	newSvc := func() *JobService {
		return NewJobService(nil, jobRepo, nil, &MockStageTemplateRepository{}, defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, nil, nil)
	}

	t.Run("rejects unknown target type", func(t *testing.T) {
		_, err := newSvc().Move(context.Background(), userID, "job-1", &model.MoveJobRequest{
			Target: model.MoveTarget{Type: "column"},
		})
		assert.ErrorIs(t, err, model.ErrInvalidMoveTarget)
	})

	t.Run("rejects stage target without template id", func(t *testing.T) {
		_, err := newSvc().Move(context.Background(), userID, "job-1", &model.MoveJobRequest{
			Target: model.MoveTarget{Type: "stage"},
		})
		assert.ErrorIs(t, err, model.ErrInvalidMoveTarget)
	})

	t.Run("rejects phase target without phase", func(t *testing.T) {
		_, err := newSvc().Move(context.Background(), userID, "job-1", &model.MoveJobRequest{
			Target: model.MoveTarget{Type: "phase"},
		})
		assert.ErrorIs(t, err, model.ErrInvalidMoveTarget)
	})

	t.Run("rejects archived as phase target", func(t *testing.T) {
		phase := model.Phase("archived")
		_, err := newSvc().Move(context.Background(), userID, "job-1", &model.MoveJobRequest{
			Target: model.MoveTarget{Type: "phase", Phase: &phase},
		})
		assert.ErrorIs(t, err, model.ErrInvalidPhase)
	})

	t.Run("no-op when already in the target base column", func(t *testing.T) {
		phase := model.PhaseApplied
		dto, err := newSvc().Move(context.Background(), userID, "job-1", &model.MoveJobRequest{
			Target: model.MoveTarget{Type: "phase", Phase: &phase},
		})
		// job is applied with no current stage → no transaction needed
		assert.NoError(t, err)
		assert.Equal(t, "applied", dto.Status)
	})
}
