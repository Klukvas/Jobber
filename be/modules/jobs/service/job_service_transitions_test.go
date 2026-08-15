package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	rbPorts "github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func strPtr(s string) *string { return &s }

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
