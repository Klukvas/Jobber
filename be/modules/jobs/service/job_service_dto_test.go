package service

import (
	"context"
	"errors"
	"testing"
	"time"

	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	companyModel "github.com/andreypavlenko/jobber/modules/companies/model"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	rbPorts "github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	resumePorts "github.com/andreypavlenko/jobber/modules/resumes/ports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests drive buildJobDTO through GetByID, exercising each enrichment
// branch: company name, uploaded resume, resume-builder, last-activity fallback,
// current-column name, and the job/stage comment split.

// enrichResumeRepo returns a resume by ID; ownErr simulates a lookup failure.
type enrichResumeRepo struct {
	resumePorts.ResumeRepository
	resume *resumeModel.Resume
	err    error
}

func (r *enrichResumeRepo) GetByID(ctx context.Context, userID, resumeID string) (*resumeModel.Resume, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.resume != nil {
		return r.resume, nil
	}
	return &resumeModel.Resume{ID: resumeID, UserID: userID, Title: "My Resume"}, nil
}

// enrichRBRepo returns a resume builder by ID; err simulates a lookup failure.
type enrichRBRepo struct {
	rbPorts.ResumeBuilderRepository
	rb  *rbModel.ResumeBuilder
	err error
}

func (r *enrichRBRepo) GetByID(ctx context.Context, userID, id string) (*rbModel.ResumeBuilder, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.rb != nil {
		return r.rb, nil
	}
	return &rbModel.ResumeBuilder{ID: id, Title: "Builder Resume"}, nil
}

// commentListRepo returns a fixed set of comments for ListByJob.
type commentListRepo struct {
	comments []*commentModel.Comment
	err      error
}

func (r *commentListRepo) Create(ctx context.Context, comment *commentModel.Comment) error {
	return nil
}
func (r *commentListRepo) ListByJob(ctx context.Context, jobID, userID string) ([]*commentModel.Comment, error) {
	return r.comments, r.err
}
func (r *commentListRepo) Delete(ctx context.Context, userID, commentID string) error { return nil }

func TestBuildJobDTO_Enrichment(t *testing.T) {
	userID := "user-123"
	jobID := "job-1"

	t.Run("enriches company name, current column, uploaded resume and last activity", func(t *testing.T) {
		companyID := "company-1"
		resumeID := "resume-1"
		stageTmplID := "tmpl-applied"
		lastActivity := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

		job := &model.Job{
			ID: jobID, UserID: userID, Title: "Engineer",
			CompanyID:              &companyID,
			ResumeID:               &resumeID,
			CurrentStageTemplateID: &stageTmplID,
		}
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) { return job, nil },
			GetLastActivityAtFunc: func(ctx context.Context, jid string) (time.Time, error) {
				return lastActivity, nil
			},
		}
		companyRepo := &MockCompanyRepository{
			GetByIDFunc: func(ctx context.Context, uid, cid string) (*companyModel.Company, error) {
				return &companyModel.Company{ID: cid, UserID: uid, Name: "Acme Corp"}, nil
			},
		}
		tmplRepo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return &model.StageTemplate{ID: tid, UserID: uid, Name: "Applied", Order: 1}, nil
			},
		}
		resumeRepo := &enrichResumeRepo{resume: &resumeModel.Resume{ID: resumeID, UserID: userID, Title: "Backend CV"}}

		svc := NewJobService(nil, jobRepo, nil, tmplRepo, companyRepo, resumeRepo, &enrichRBRepo{}, &commentListRepo{}, nil, nil, nil)

		dto, err := svc.GetByID(context.Background(), userID, jobID)

		require.NoError(t, err)
		require.NotNil(t, dto.CompanyName)
		assert.Equal(t, "Acme Corp", *dto.CompanyName)
		require.NotNil(t, dto.CurrentStageName)
		assert.Equal(t, "Applied", *dto.CurrentStageName)
		require.NotNil(t, dto.Resume)
		assert.Equal(t, "Backend CV", dto.Resume.Name)
		assert.Equal(t, "uploaded", dto.Resume.Type)
		assert.Equal(t, lastActivity, dto.LastActivityAt)
	})

	t.Run("enriches a resume-builder attachment", func(t *testing.T) {
		rbID := "rb-1"
		job := &model.Job{ID: jobID, UserID: userID, Title: "Engineer", ResumeBuilderID: &rbID}
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) { return job, nil },
		}
		rbRepo := &enrichRBRepo{rb: &rbModel.ResumeBuilder{ID: rbID, Title: "Interactive CV"}}

		svc := NewJobService(nil, jobRepo, nil, &MockStageTemplateRepository{}, defaultMockCompanyRepo,
			&enrichResumeRepo{}, rbRepo, &commentListRepo{}, nil, nil, nil)

		dto, err := svc.GetByID(context.Background(), userID, jobID)

		require.NoError(t, err)
		require.NotNil(t, dto.Resume)
		assert.Equal(t, "Interactive CV", dto.Resume.Name)
		assert.Equal(t, "builder", dto.Resume.Type)
	})

	t.Run("splits job-level and stage-level comments", func(t *testing.T) {
		stageID := "stage-1"
		job := &model.Job{ID: jobID, UserID: userID, Title: "Engineer"}
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) { return job, nil },
		}
		comments := &commentListRepo{
			comments: []*commentModel.Comment{
				{ID: "c1", JobID: jobID, Content: "job note"},                 // StageID nil -> job-level
				{ID: "c2", JobID: jobID, StageID: &stageID, Content: "stage"}, // stage-level
			},
		}
		svc := NewJobService(nil, jobRepo, nil, &MockStageTemplateRepository{}, defaultMockCompanyRepo,
			&enrichResumeRepo{}, &enrichRBRepo{}, comments, nil, nil, nil)

		dto, err := svc.GetByID(context.Background(), userID, jobID)

		require.NoError(t, err)
		require.Len(t, dto.JobComments, 1)
		assert.Equal(t, "c1", dto.JobComments[0].ID)
		require.Len(t, dto.StageComments, 1)
		assert.Equal(t, "c2", dto.StageComments[0].ID)
	})

	t.Run("falls back to updated_at when last-activity lookup fails", func(t *testing.T) {
		updatedAt := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
		job := &model.Job{ID: jobID, UserID: userID, Title: "Engineer", UpdatedAt: updatedAt}
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) { return job, nil },
			GetLastActivityAtFunc: func(ctx context.Context, jid string) (time.Time, error) {
				return time.Time{}, errors.New("activity lookup failed")
			},
		}
		svc := NewJobService(nil, jobRepo, nil, &MockStageTemplateRepository{}, defaultMockCompanyRepo,
			&enrichResumeRepo{}, &enrichRBRepo{}, &commentListRepo{}, nil, nil, nil)

		dto, err := svc.GetByID(context.Background(), userID, jobID)

		require.NoError(t, err)
		assert.Equal(t, updatedAt, dto.LastActivityAt)
	})

	t.Run("tolerates enrichment lookup failures without erroring", func(t *testing.T) {
		companyID := "company-1"
		resumeID := "resume-1"
		stageTmplID := "tmpl-applied"
		job := &model.Job{
			ID: jobID, UserID: userID, Title: "Engineer",
			CompanyID:              &companyID,
			ResumeID:               &resumeID,
			CurrentStageTemplateID: &stageTmplID,
		}
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) { return job, nil },
		}
		companyRepo := &MockCompanyRepository{
			GetByIDFunc: func(ctx context.Context, uid, cid string) (*companyModel.Company, error) {
				return nil, errors.New("company gone")
			},
		}
		tmplRepo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return nil, errors.New("template gone")
			},
		}
		resumeRepo := &enrichResumeRepo{err: errors.New("resume gone")}

		svc := NewJobService(nil, jobRepo, nil, tmplRepo, companyRepo, resumeRepo, &enrichRBRepo{}, &commentListRepo{}, nil, nil, nil)

		dto, err := svc.GetByID(context.Background(), userID, jobID)

		require.NoError(t, err) // enrichment failures are logged, not fatal
		require.NotNil(t, dto)
		assert.Nil(t, dto.CompanyName)
		assert.Nil(t, dto.CurrentStageName)
		assert.Nil(t, dto.Resume)
	})

	t.Run("tolerates a resume-builder lookup failure", func(t *testing.T) {
		rbID := "rb-1"
		job := &model.Job{ID: jobID, UserID: userID, Title: "Engineer", ResumeBuilderID: &rbID}
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) { return job, nil },
		}
		rbRepo := &enrichRBRepo{err: errors.New("builder gone")}

		svc := NewJobService(nil, jobRepo, nil, &MockStageTemplateRepository{}, defaultMockCompanyRepo,
			&enrichResumeRepo{}, rbRepo, &commentListRepo{}, nil, nil, nil)

		dto, err := svc.GetByID(context.Background(), userID, jobID)

		require.NoError(t, err)
		assert.Nil(t, dto.Resume)
	})

	t.Run("tolerates a comment lookup failure", func(t *testing.T) {
		job := &model.Job{ID: jobID, UserID: userID, Title: "Engineer"}
		jobRepo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) { return job, nil },
		}
		comments := &commentListRepo{err: errors.New("comments unavailable")}
		svc := NewJobService(nil, jobRepo, nil, &MockStageTemplateRepository{}, defaultMockCompanyRepo,
			&enrichResumeRepo{}, &enrichRBRepo{}, comments, nil, nil, nil)

		dto, err := svc.GetByID(context.Background(), userID, jobID)

		require.NoError(t, err)
		assert.Empty(t, dto.JobComments)
		assert.Empty(t, dto.StageComments)
	})
}
