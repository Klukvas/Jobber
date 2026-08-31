package service

import (
	"context"
	"errors"
	"testing"

	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests exercise the transactional stage-history operations (AddStage,
// UpdateStage, DeleteStage) and the read-only ListStages. The transactional
// bodies run against a pgxmock pool injected through the txBeginner seam; the
// job/stage/template lookups still go through the hand-written repo mocks.
//
// pgxmock enforces exact argument counts on every Exec/Query, so the small
// exp* helpers below encode the correct WithArgs shape for each SQL statement
// and return the expectation so callers can override the outcome
// (WillReturnError / a specific row count).

const (
	stagesUserID = "user-123"
	stagesJobID  = "job-1"
	stagesTmplID = "tmpl-applied"
)

// -------------------- expectation helpers --------------------

func expLock(m pgxmock.PgxPoolIface) *pgxmock.ExpectedExec {
	return m.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).
		WithArgs(stagesJobID).
		WillReturnResult(pgxmock.NewResult("SELECT", 1))
}

func expMaxOrder(m pgxmock.PgxPoolIface, order int) *pgxmock.ExpectedQuery {
	return m.ExpectQuery(`SELECT COALESCE\(MAX`).
		WithArgs(stagesJobID).
		WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(order))
}

func expCompleteCurrent(m pgxmock.PgxPoolIface, currentStageID string) *pgxmock.ExpectedExec {
	return m.ExpectExec(`UPDATE job_stages SET status = \$2, completed_at = \$3 WHERE id = \$1`).
		WithArgs(currentStageID, "completed", pgxmock.AnyArg(), stagesJobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
}

func expInsertStage(m pgxmock.PgxPoolIface) *pgxmock.ExpectedExec {
	return m.ExpectExec(`INSERT INTO job_stages`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
}

func expUpdateJobCurrentStage(m pgxmock.PgxPoolIface) *pgxmock.ExpectedExec {
	return m.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
}

func expUpdateStageRow(m pgxmock.PgxPoolIface) *pgxmock.ExpectedExec {
	return m.ExpectExec(`UPDATE job_stages SET status = \$2, completed_at = \$3 WHERE id = \$1`).
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
}

// -------------------- shared repo builders / mocks --------------------

// recordingCommentRepo captures Create calls so tests can assert the optional
// stage comment is persisted (or that a Create failure is swallowed).
type recordingCommentRepo struct {
	created   []*commentModel.Comment
	createErr error
}

func (r *recordingCommentRepo) Create(ctx context.Context, comment *commentModel.Comment) error {
	r.created = append(r.created, comment)
	return r.createErr
}
func (r *recordingCommentRepo) ListByJob(ctx context.Context, jobID, userID string) ([]*commentModel.Comment, error) {
	return nil, nil
}
func (r *recordingCommentRepo) Delete(ctx context.Context, userID, commentID string) error {
	return nil
}

func newPool(t *testing.T) pgxmock.PgxPoolIface {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return mock
}

// jobRepoReturning builds a job repo whose GetByID yields the given job.
func jobRepoReturning(job *model.Job) *MockJobRepository {
	return &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
			return job, nil
		},
	}
}

// appliedTemplateRepo builds a template repo whose GetByID yields an
// "Applied" (order 1) column.
func appliedTemplateRepo() *MockStageTemplateRepository {
	return &MockStageTemplateRepository{
		GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
			return &model.StageTemplate{ID: tid, UserID: uid, Name: "Applied", Order: 1}, nil
		},
	}
}

// svcWith wires a JobService with the given pool and repos, defaulting the rest.
func svcWith(pool txBeginner, jobRepo *MockJobRepository, stageRepo *MockJobStageRepository,
	tmplRepo *MockStageTemplateRepository, commentRepo commentRepoIface) *JobService {
	var cr commentRepoIface = defaultMockCommentRepo
	if commentRepo != nil {
		cr = commentRepo
	}
	if stageRepo == nil {
		stageRepo = &MockJobStageRepository{}
	}
	if tmplRepo == nil {
		tmplRepo = appliedTemplateRepo()
	}
	return NewJobService(pool, jobRepo, stageRepo, tmplRepo, defaultMockCompanyRepo, nil, nil, cr, nil, nil, nil)
}

// commentRepoIface mirrors the comments port used by the service, so both the
// default no-op mock and recordingCommentRepo satisfy svcWith.
type commentRepoIface interface {
	Create(ctx context.Context, comment *commentModel.Comment) error
	ListByJob(ctx context.Context, jobID, userID string) ([]*commentModel.Comment, error)
	Delete(ctx context.Context, userID, commentID string) error
}

// -------------------- AddStage --------------------

func TestJobService_AddStage(t *testing.T) {
	t.Run("appends the first stage, stamps applied_at and returns the enriched DTO", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expMaxOrder(mock, 0)
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(
			pgxmock.AnyArg(), stagesJobID, stagesTmplID, "active", 0, pgxmock.AnyArg(), nil, pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		dto, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})

		require.NoError(t, err)
		require.NotNil(t, dto)
		assert.Equal(t, stagesJobID, dto.JobID)
		assert.Equal(t, stagesTmplID, dto.StageTemplateID)
		assert.Equal(t, "active", dto.Status)
		assert.Equal(t, "Applied", dto.StageName)
		assert.Equal(t, 0, dto.Order)
		assert.NotEmpty(t, dto.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("completes the current active stage before appending", func(t *testing.T) {
		mock := newPool(t)
		currentStageID := "stage-current"
		job := &model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &currentStageID}

		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				assert.Equal(t, currentStageID, sid)
				return &model.JobStage{ID: sid, JobID: jid, StageTemplateID: "tmpl-prev", Status: "active"}, nil
			},
		}
		tmplRepo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				name, order := "Applied", 1
				if tid == "tmpl-prev" {
					name, order = "Screening", 2
				}
				return &model.StageTemplate{ID: tid, UserID: uid, Name: name, Order: order}, nil
			},
		}

		mock.ExpectBegin()
		expLock(mock)
		expMaxOrder(mock, 2)
		expCompleteCurrent(mock, currentStageID)
		expInsertStage(mock)
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), stageRepo, tmplRepo, nil)

		dto, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})

		require.NoError(t, err)
		// The mocked MAX query returns 2 directly (the "+1" lives in the SQL the
		// mock does not evaluate), so the appended stage's order is 2.
		assert.Equal(t, 2, dto.Order)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("persists an optional comment on the new stage", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expMaxOrder(mock, 0)
		expInsertStage(mock)
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit()

		commentRepo := &recordingCommentRepo{}
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), commentRepo)

		comment := "  Started interviewing  "
		dto, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{
			StageTemplateID: stagesTmplID,
			Comment:         &comment,
		})

		require.NoError(t, err)
		require.Len(t, commentRepo.created, 1)
		assert.Equal(t, "Started interviewing", commentRepo.created[0].Content) // trimmed
		require.NotNil(t, commentRepo.created[0].StageID)
		assert.Equal(t, dto.ID, *commentRepo.created[0].StageID)
		assert.Equal(t, stagesJobID, commentRepo.created[0].JobID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("swallows a comment-create failure without failing the stage add", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expMaxOrder(mock, 0)
		expInsertStage(mock)
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit()

		commentRepo := &recordingCommentRepo{createErr: errors.New("comment insert failed")}
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), commentRepo)

		comment := "note"
		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{
			StageTemplateID: stagesTmplID,
			Comment:         &comment,
		})

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("skips a blank comment", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expMaxOrder(mock, 0)
		expInsertStage(mock)
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit()

		commentRepo := &recordingCommentRepo{}
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), commentRepo)

		blank := "   "
		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{
			StageTemplateID: stagesTmplID,
			Comment:         &blank,
		})

		require.NoError(t, err)
		assert.Empty(t, commentRepo.created)
	})

	t.Run("returns the job lookup error (wrong owner / not found)", func(t *testing.T) {
		svc := svcWith(newPool(t), &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}, nil, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		assert.ErrorIs(t, err, model.ErrJobNotFound)
	})

	t.Run("returns the template lookup error", func(t *testing.T) {
		tmplRepo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return nil, model.ErrStageTemplateNotFound
			},
		}
		svc := svcWith(newPool(t), jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, tmplRepo, nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: "missing"})
		assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
	})

	t.Run("wraps a begin-transaction error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin().WillReturnError(errors.New("no connection"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")
	})

	t.Run("wraps a lock-row error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(stagesJobID).WillReturnError(errors.New("lock timeout"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to lock job")
	})

	t.Run("wraps an order-computation error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(stagesJobID).WillReturnError(errors.New("query failed"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to compute stage order")
	})

	t.Run("returns the current-stage lookup error", func(t *testing.T) {
		mock := newPool(t)
		currentStageID := "stage-current"
		job := &model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &currentStageID}

		mock.ExpectBegin()
		expLock(mock)
		expMaxOrder(mock, 0)

		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return nil, model.ErrJobStageNotFound
			},
		}
		svc := svcWith(mock, jobRepoReturning(job), stageRepo, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
	})

	t.Run("wraps an insert-stage error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expMaxOrder(mock, 0)
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnError(errors.New("insert failed"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create stage")
	})

	t.Run("wraps an update-job error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expMaxOrder(mock, 0)
		expInsertStage(mock)
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3`).WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnError(errors.New("update failed"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update job current stage")
	})

	t.Run("wraps a commit error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expMaxOrder(mock, 0)
		expInsertStage(mock)
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
	})
}

// -------------------- ListStages --------------------

func TestJobService_ListStages(t *testing.T) {
	t.Run("returns stages enriched with template names", func(t *testing.T) {
		stageRepo := &MockJobStageRepository{
			ListByJobFunc: func(ctx context.Context, jid string) ([]*model.JobStage, error) {
				return []*model.JobStage{
					{ID: "s1", JobID: jid, StageTemplateID: "t-wish", Status: "completed", Order: 0},
					{ID: "s2", JobID: jid, StageTemplateID: "t-applied", Status: "active", Order: 1},
				}, nil
			},
		}
		tmplRepo := &MockStageTemplateRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
				return []*model.StageTemplate{
					{ID: "t-wish", Name: "Wishlist"},
					{ID: "t-applied", Name: "Applied"},
				}, 2, nil
			},
		}
		svc := svcWith(nil, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), stageRepo, tmplRepo, nil)

		dtos, err := svc.ListStages(context.Background(), stagesUserID, stagesJobID)

		require.NoError(t, err)
		require.Len(t, dtos, 2)
		assert.Equal(t, "Wishlist", dtos[0].StageName)
		assert.Equal(t, "Applied", dtos[1].StageName)
	})

	t.Run("leaves the name empty when its template is missing", func(t *testing.T) {
		stageRepo := &MockJobStageRepository{
			ListByJobFunc: func(ctx context.Context, jid string) ([]*model.JobStage, error) {
				return []*model.JobStage{{ID: "s1", JobID: jid, StageTemplateID: "gone"}}, nil
			},
		}
		tmplRepo := &MockStageTemplateRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
				return []*model.StageTemplate{}, 0, nil
			},
		}
		svc := svcWith(nil, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), stageRepo, tmplRepo, nil)

		dtos, err := svc.ListStages(context.Background(), stagesUserID, stagesJobID)
		require.NoError(t, err)
		require.Len(t, dtos, 1)
		assert.Empty(t, dtos[0].StageName)
	})

	t.Run("returns the job lookup error", func(t *testing.T) {
		svc := svcWith(nil, &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}, nil, &MockStageTemplateRepository{}, nil)

		_, err := svc.ListStages(context.Background(), stagesUserID, stagesJobID)
		assert.ErrorIs(t, err, model.ErrJobNotFound)
	})

	t.Run("propagates a stage-list error", func(t *testing.T) {
		stageRepo := &MockJobStageRepository{
			ListByJobFunc: func(ctx context.Context, jid string) ([]*model.JobStage, error) {
				return nil, errors.New("db down")
			},
		}
		svc := svcWith(nil, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), stageRepo, &MockStageTemplateRepository{}, nil)

		_, err := svc.ListStages(context.Background(), stagesUserID, stagesJobID)
		assert.Error(t, err)
	})

	t.Run("propagates a template-list error", func(t *testing.T) {
		stageRepo := &MockJobStageRepository{
			ListByJobFunc: func(ctx context.Context, jid string) ([]*model.JobStage, error) {
				return []*model.JobStage{{ID: "s1", JobID: jid}}, nil
			},
		}
		tmplRepo := &MockStageTemplateRepository{
			ListFunc: func(ctx context.Context, uid string, limit, offset int) ([]*model.StageTemplate, int, error) {
				return nil, 0, errors.New("templates unavailable")
			},
		}
		svc := svcWith(nil, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), stageRepo, tmplRepo, nil)

		_, err := svc.ListStages(context.Background(), stagesUserID, stagesJobID)
		assert.Error(t, err)
	})
}

// -------------------- UpdateStage --------------------

func TestJobService_UpdateStage(t *testing.T) {
	stageID := "stage-1"

	baseStageRepo := func() *MockJobStageRepository {
		return &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return &model.JobStage{ID: sid, JobID: jid, StageTemplateID: stagesTmplID, Status: "active"}, nil
			},
		}
	}

	t.Run("updates status and stamps completed_at when moving to completed", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		mock.ExpectExec(`UPDATE job_stages SET status = \$2, completed_at = \$3 WHERE id = \$1`).WithArgs(
			stageID, "completed", pgxmock.AnyArg(), stagesJobID,
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), baseStageRepo(), appliedTemplateRepo(), nil)

		completed := "completed"
		dto, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &completed})

		require.NoError(t, err)
		assert.Equal(t, "completed", dto.Status)
		require.NotNil(t, dto.CompletedAt)
		assert.Equal(t, "Applied", dto.StageName)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("clears completed_at when moving back to a non-terminal status", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		// completed_at is passed as a (nil) *time.Time, so match it structurally.
		mock.ExpectExec(`UPDATE job_stages SET status = \$2, completed_at = \$3 WHERE id = \$1`).WithArgs(
			stageID, "active", pgxmock.AnyArg(), stagesJobID,
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				completed := timeNowUTC()
				return &model.JobStage{ID: sid, JobID: jid, StageTemplateID: stagesTmplID, Status: "completed", CompletedAt: &completed}, nil
			},
		}
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), stageRepo, appliedTemplateRepo(), nil)

		active := "active"
		dto, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &active})

		require.NoError(t, err)
		assert.Equal(t, "active", dto.Status)
		assert.Nil(t, dto.CompletedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("honors an explicitly supplied completed_at", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expUpdateStageRow(mock)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), baseStageRepo(), appliedTemplateRepo(), nil)

		when := timeNowUTC()
		dto, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{CompletedAt: &when})

		require.NoError(t, err)
		require.NotNil(t, dto.CompletedAt)
		assert.Equal(t, when, *dto.CompletedAt)
	})

	t.Run("rejects an invalid status", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		// No UPDATE/Commit expected: the invalid status short-circuits.

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), baseStageRepo(), appliedTemplateRepo(), nil)

		bogus := "bogus"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &bogus})
		assert.ErrorIs(t, err, model.ErrInvalidStatus)
	})

	t.Run("returns the job lookup error before opening a transaction", func(t *testing.T) {
		svc := svcWith(newPool(t), &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}, baseStageRepo(), appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		assert.ErrorIs(t, err, model.ErrJobNotFound)
	})

	t.Run("returns the stage lookup error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)

		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return nil, model.ErrJobStageNotFound
			},
		}
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), stageRepo, appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
	})

	t.Run("rejects a stage that belongs to another job", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)

		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return &model.JobStage{ID: sid, JobID: "other-job", StageTemplateID: stagesTmplID, Status: "active"}, nil
			},
		}
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), stageRepo, appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
	})

	t.Run("returns not-found when the UPDATE affects no rows", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		mock.ExpectExec(`UPDATE job_stages SET status = \$2, completed_at = \$3 WHERE id = \$1`).WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), baseStageRepo(), appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
	})

	t.Run("wraps a begin error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin().WillReturnError(errors.New("no conn"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), baseStageRepo(), appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")
	})

	t.Run("wraps an update error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		mock.ExpectExec(`UPDATE job_stages SET status = \$2, completed_at = \$3 WHERE id = \$1`).WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnError(errors.New("update boom"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), baseStageRepo(), appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update stage")
	})

	t.Run("returns the template lookup error after the update", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expUpdateStageRow(mock)

		tmplRepo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return nil, model.ErrStageTemplateNotFound
			},
		}
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), baseStageRepo(), tmplRepo, nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
	})

	t.Run("wraps a commit error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expUpdateStageRow(mock)
		mock.ExpectCommit().WillReturnError(errors.New("commit boom"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), baseStageRepo(), appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
	})
}

// -------------------- DeleteStage --------------------

func TestJobService_DeleteStage(t *testing.T) {
	stageID := "stage-1"

	nonCurrentStageRepo := func() *MockJobStageRepository {
		return &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return &model.JobStage{ID: sid, JobID: jid, Status: "completed"}, nil
			},
		}
	}

	t.Run("deletes a non-current stage without recalculating the pointer", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		mock.ExpectExec(`DELETE FROM job_stages WHERE id = \$1`).WithArgs(stageID, stagesJobID).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectCommit()

		// current_stage_id points elsewhere, so no recalculation UPDATE runs.
		other := "stage-other"
		job := &model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &other}
		svc := svcWith(mock, jobRepoReturning(job), nonCurrentStageRepo(), appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("recalculates current_stage_id when deleting the current stage", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		// The replacement is the most recent active/completed stage (excluding the
		// deleted one); the service passes it as a *string, so match by dereference.
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, updated_at = \$3 WHERE id = \$1`).WithArgs(
			stagesJobID, strPtrArg("s-prev"), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`DELETE FROM job_stages WHERE id = \$1`).WithArgs(stageID, stagesJobID).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectCommit()

		currentID := stageID
		job := &model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &currentID}
		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return &model.JobStage{ID: sid, JobID: jid, Status: "active"}, nil
			},
			ListByJobFunc: func(ctx context.Context, jid string) ([]*model.JobStage, error) {
				return []*model.JobStage{
					{ID: "s-prev", JobID: jid, Status: "completed", Order: 0},
					{ID: stageID, JobID: jid, Status: "active", Order: 1},
				}, nil
			},
		}
		svc := svcWith(mock, jobRepoReturning(job), stageRepo, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sets current_stage_id to NULL when no replacement remains", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		// No eligible replacement remains, so current_stage_id is set to a nil *string.
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, updated_at = \$3 WHERE id = \$1`).WithArgs(
			stagesJobID, nilStrPtrArg(), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec(`DELETE FROM job_stages WHERE id = \$1`).WithArgs(stageID, stagesJobID).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectCommit()

		currentID := stageID
		job := &model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &currentID}
		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return &model.JobStage{ID: sid, JobID: jid, Status: "active"}, nil
			},
			ListByJobFunc: func(ctx context.Context, jid string) ([]*model.JobStage, error) {
				// Only the stage being deleted, plus a pending one that isn't eligible.
				return []*model.JobStage{
					{ID: stageID, JobID: jid, Status: "active", Order: 0},
					{ID: "s-pending", JobID: jid, Status: "pending", Order: 1},
				}, nil
			},
		}
		svc := svcWith(mock, jobRepoReturning(job), stageRepo, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns the job lookup error", func(t *testing.T) {
		svc := svcWith(newPool(t), &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}, nonCurrentStageRepo(), appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		assert.ErrorIs(t, err, model.ErrJobNotFound)
	})

	t.Run("returns the stage lookup error", func(t *testing.T) {
		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return nil, model.ErrJobStageNotFound
			},
		}
		svc := svcWith(newPool(t), jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), stageRepo, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
	})

	t.Run("rejects a stage that belongs to another job", func(t *testing.T) {
		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return &model.JobStage{ID: sid, JobID: "other-job", Status: "active"}, nil
			},
		}
		svc := svcWith(newPool(t), jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), stageRepo, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
	})

	t.Run("wraps a begin error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin().WillReturnError(errors.New("no conn"))

		other := "stage-other"
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &other}),
			nonCurrentStageRepo(), appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")
	})

	t.Run("wraps a lock error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(stagesJobID).WillReturnError(errors.New("lock timeout"))

		other := "stage-other"
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &other}),
			nonCurrentStageRepo(), appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to lock job")
	})

	t.Run("propagates a stage-list error during recalculation", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)

		currentID := stageID
		job := &model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &currentID}
		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return &model.JobStage{ID: sid, JobID: jid, Status: "active"}, nil
			},
			ListByJobFunc: func(ctx context.Context, jid string) ([]*model.JobStage, error) {
				return nil, errors.New("list failed")
			},
		}
		svc := svcWith(mock, jobRepoReturning(job), stageRepo, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "list failed")
	})

	t.Run("wraps a recalculation update error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, updated_at = \$3 WHERE id = \$1`).WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnError(errors.New("update boom"))

		currentID := stageID
		job := &model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &currentID}
		stageRepo := &MockJobStageRepository{
			GetByIDFunc: func(ctx context.Context, sid, jid string) (*model.JobStage, error) {
				return &model.JobStage{ID: sid, JobID: jid, Status: "active"}, nil
			},
			ListByJobFunc: func(ctx context.Context, jid string) ([]*model.JobStage, error) {
				return []*model.JobStage{{ID: stageID, JobID: jid, Status: "active"}}, nil
			},
		}
		svc := svcWith(mock, jobRepoReturning(job), stageRepo, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update job current stage")
	})

	t.Run("wraps a delete error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		mock.ExpectExec(`DELETE FROM job_stages WHERE id = \$1`).WithArgs(stageID, stagesJobID).WillReturnError(errors.New("delete boom"))

		other := "stage-other"
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &other}),
			nonCurrentStageRepo(), appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete stage")
	})

	t.Run("wraps a commit error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		mock.ExpectExec(`DELETE FROM job_stages WHERE id = \$1`).WithArgs(stageID, stagesJobID).WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectCommit().WillReturnError(errors.New("commit boom"))

		other := "stage-other"
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &other}),
			nonCurrentStageRepo(), appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
	})
}
