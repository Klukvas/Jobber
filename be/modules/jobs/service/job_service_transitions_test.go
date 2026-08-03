package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	rbPorts "github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

func TestApplyStatusTransition(t *testing.T) {
	past := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name              string
		job               model.Job
		newStatus         string
		explicitAppliedAt *time.Time
		wantErr           error
		wantStatus        string
		wantAppliedNil    bool
		wantAppliedEqual  *time.Time
	}{
		{
			name:           "invalid status rejected",
			job:            model.Job{Status: "saved"},
			newStatus:      "active", // legacy value is not a valid pipeline status
			wantErr:        model.ErrInvalidJobStatus,
			wantStatus:     "saved",
			wantAppliedNil: true,
		},
		{
			name:           "leaving saved stamps applied_at",
			job:            model.Job{Status: "saved"},
			newStatus:      "applied",
			wantStatus:     "applied",
			wantAppliedNil: false,
		},
		{
			name:              "leaving saved honors explicit applied_at",
			job:               model.Job{Status: "saved"},
			newStatus:         "applied",
			explicitAppliedAt: &past,
			wantStatus:        "applied",
			wantAppliedEqual:  &past,
		},
		{
			name:           "returning to saved clears applied_at",
			job:            model.Job{Status: "applied", AppliedAt: &past},
			newStatus:      "saved",
			wantStatus:     "saved",
			wantAppliedNil: true,
		},
		{
			name:             "applied to offer keeps applied_at",
			job:              model.Job{Status: "applied", AppliedAt: &past},
			newStatus:        "offer",
			wantStatus:       "offer",
			wantAppliedEqual: &past,
		},
		{
			name:           "archiving stamps applied_at (any non-saved status counts as applied)",
			job:            model.Job{Status: "saved"},
			newStatus:      "archived",
			wantStatus:     "archived",
			wantAppliedNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := tt.job
			err := applyStatusTransition(&job, tt.newStatus, tt.explicitAppliedAt)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, job.Status)
			if tt.wantAppliedNil {
				assert.Nil(t, job.AppliedAt)
			} else {
				require.NotNil(t, job.AppliedAt)
			}
			if tt.wantAppliedEqual != nil {
				assert.Equal(t, tt.wantAppliedEqual.UTC(), job.AppliedAt.UTC())
			}
		})
	}
}

// ownedResumeRepo / ownedRBRepo: embed the port interface (unimplemented methods
// panic if hit) and override only the ownership lookups used by
// resolveResumeSelection. ownErr simulates a foreign / missing resume.
type ownedResumeRepo struct {
	resumePorts.ResumeRepository
	ownErr error
}

func (m *ownedResumeRepo) GetByID(ctx context.Context, userID, resumeID string) (*resumeModel.Resume, error) {
	if m.ownErr != nil {
		return nil, m.ownErr
	}
	return &resumeModel.Resume{ID: resumeID, UserID: userID}, nil
}

type ownedRBRepo struct {
	rbPorts.ResumeBuilderRepository
	ownErr error
}

func (m *ownedRBRepo) VerifyOwnership(ctx context.Context, userID, resumeBuilderID string) error {
	return m.ownErr
}

func TestResolveResumeSelection(t *testing.T) {
	notOwned := errors.New("not owned")

	tests := []struct {
		name            string
		job             model.Job
		resumeID        *string
		resumeBuilderID *string
		resumeOwnErr    error
		builderOwnErr   error
		wantErr         error
		wantResume      *string
		wantBuilder     *string
	}{
		{
			name:            "both set is rejected",
			resumeID:        strPtr("r-1"),
			resumeBuilderID: strPtr("rb-1"),
			wantErr:         model.ErrBothResumeTypesSet,
		},
		{
			name:       "setting owned resume works",
			resumeID:   strPtr("r-1"),
			wantResume: strPtr("r-1"),
		},
		{
			name:         "setting a foreign resume is rejected (IDOR guard)",
			resumeID:     strPtr("r-foreign"),
			resumeOwnErr: notOwned,
			wantErr:      model.ErrResumeNotFound,
		},
		{
			name:            "setting owned builder clears uploaded resume",
			job:             model.Job{ResumeID: strPtr("r-1")},
			resumeBuilderID: strPtr("rb-1"),
			wantBuilder:     strPtr("rb-1"),
		},
		{
			name:            "setting a foreign builder is rejected (IDOR guard)",
			resumeBuilderID: strPtr("rb-foreign"),
			builderOwnErr:   notOwned,
			wantErr:         model.ErrResumeNotFound,
		},
		{
			name:       "setting owned resume clears builder",
			job:        model.Job{ResumeBuilderID: strPtr("rb-1")},
			resumeID:   strPtr("r-2"),
			wantResume: strPtr("r-2"),
		},
		{
			name:     "empty string clears resume without ownership lookup",
			job:      model.Job{ResumeID: strPtr("r-1")},
			resumeID: strPtr(""),
		},
		{
			name:       "nil pointers leave selection untouched",
			job:        model.Job{ResumeID: strPtr("r-1")},
			wantResume: strPtr("r-1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewJobService(nil, &MockJobRepository{}, nil, nil, defaultMockCompanyRepo,
				&ownedResumeRepo{ownErr: tt.resumeOwnErr},
				&ownedRBRepo{ownErr: tt.builderOwnErr},
				defaultMockCommentRepo, nil, nil, nil)
			job := tt.job
			err := svc.resolveResumeSelection(context.Background(), "user-1", &job, tt.resumeID, tt.resumeBuilderID)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantResume != nil {
				require.NotNil(t, job.ResumeID)
				assert.Equal(t, *tt.wantResume, *job.ResumeID)
			} else {
				assert.Nil(t, job.ResumeID)
			}
			if tt.wantBuilder != nil {
				require.NotNil(t, job.ResumeBuilderID)
				assert.Equal(t, *tt.wantBuilder, *job.ResumeBuilderID)
			} else {
				assert.Nil(t, job.ResumeBuilderID)
			}
		})
	}
}

func TestJobService_Update_StatusTransitions(t *testing.T) {
	userID := "user-123"
	past := time.Date(2026, 7, 10, 9, 0, 0, 0, time.UTC)

	newSvc := func(job *model.Job, updated **model.Job) *JobService {
		repo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				j := *job
				return &j, nil
			},
			UpdateFunc: func(ctx context.Context, j *model.Job) error {
				*updated = j
				return nil
			},
		}
		return NewJobService(nil, repo, nil, nil, defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, nil, nil)
	}

	t.Run("saved to applied stamps applied_at", func(t *testing.T) {
		var updated *model.Job
		svc := newSvc(&model.Job{ID: "j1", UserID: userID, Title: "T", Status: "saved"}, &updated)

		dto, err := svc.Update(context.Background(), userID, "j1", &model.UpdateJobRequest{Status: strPtr("applied")})

		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "applied", updated.Status)
		assert.NotNil(t, updated.AppliedAt)
		assert.Equal(t, "applied", dto.Status)
		assert.NotNil(t, dto.AppliedAt)
	})

	t.Run("applied back to saved clears applied_at", func(t *testing.T) {
		var updated *model.Job
		svc := newSvc(&model.Job{ID: "j1", UserID: userID, Title: "T", Status: "applied", AppliedAt: &past}, &updated)

		dto, err := svc.Update(context.Background(), userID, "j1", &model.UpdateJobRequest{Status: strPtr("saved")})

		require.NoError(t, err)
		require.NotNil(t, updated)
		assert.Equal(t, "saved", updated.Status)
		assert.Nil(t, updated.AppliedAt)
		assert.Nil(t, dto.AppliedAt)
	})

	t.Run("legacy active status is rejected on update", func(t *testing.T) {
		var updated *model.Job
		svc := newSvc(&model.Job{ID: "j1", UserID: userID, Title: "T", Status: "saved"}, &updated)

		_, err := svc.Update(context.Background(), userID, "j1", &model.UpdateJobRequest{Status: strPtr("active")})

		assert.ErrorIs(t, err, model.ErrInvalidJobStatus)
	})

	t.Run("both resume kinds rejected on update", func(t *testing.T) {
		var updated *model.Job
		svc := newSvc(&model.Job{ID: "j1", UserID: userID, Title: "T", Status: "applied", AppliedAt: &past}, &updated)

		_, err := svc.Update(context.Background(), userID, "j1", &model.UpdateJobRequest{
			ResumeID:        strPtr("r-1"),
			ResumeBuilderID: strPtr("rb-1"),
		})

		assert.ErrorIs(t, err, model.ErrBothResumeTypesSet)
	})

	t.Run("create with applied status stamps applied_at", func(t *testing.T) {
		var created *model.Job
		repo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, j *model.Job) error {
				created = j
				j.ID = "j-new"
				return nil
			},
		}
		svc := NewJobService(nil, repo, nil, nil, defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, nil, nil)

		_, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{
			Title:  "Engineer",
			Status: strPtr("applied"),
		})

		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, "applied", created.Status)
		assert.NotNil(t, created.AppliedAt)
	})

	t.Run("create default is saved without applied_at", func(t *testing.T) {
		var created *model.Job
		repo := &MockJobRepository{
			CreateFunc: func(ctx context.Context, j *model.Job) error {
				created = j
				j.ID = "j-new"
				return nil
			},
		}
		svc := NewJobService(nil, repo, nil, nil, defaultMockCompanyRepo, nil, nil, defaultMockCommentRepo, nil, nil, nil)

		_, err := svc.Create(context.Background(), userID, &model.CreateJobRequest{Title: "Engineer"})

		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, string(model.StatusSaved), created.Status)
		assert.Nil(t, created.AppliedAt)
	})
}
