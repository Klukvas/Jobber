package service

import (
	"context"
	"errors"
	"testing"
	"time"

	commentModel "github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/jackc/pgx/v5"
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

// expLock is the "lock the card and read its authoritative state" query every
// pipeline write starts with. The returned state is what the write decides on,
// so each test says which column/stage/applied_at the locked row carries.
func expLock(m pgxmock.PgxPoolIface) *pgxmock.ExpectedQuery {
	return expLockedJob(m, nil, nil, nil)
}

func expLockedJob(m pgxmock.PgxPoolIface, templateID, stageID *string, appliedAt *time.Time) *pgxmock.ExpectedQuery {
	return m.ExpectQuery(`SELECT current_stage_template_id, current_stage_id, applied_at`).
		WithArgs(stagesJobID, stagesUserID).
		WillReturnRows(pgxmock.NewRows([]string{"current_stage_template_id", "current_stage_id", "applied_at"}).
			AddRow(templateID, stageID, appliedAt))
}

// expReadStage is the in-transaction read of one history row.
func expReadStage(m pgxmock.PgxPoolIface, stageID string, templateID, status string, order int) *pgxmock.ExpectedQuery {
	return m.ExpectQuery(`SELECT stage_template_id, status, "order", started_at, completed_at, created_at`).
		WithArgs(stageID, stagesJobID).
		WillReturnRows(pgxmock.NewRows([]string{"stage_template_id", "status", "order", "started_at", "completed_at", "created_at"}).
			AddRow(templateID, status, order, time.Now().UTC(), nil, time.Now().UTC()))
}

// expReadTemplate is the in-transaction read of a pipeline column. A write
// holding the job lock resolves the column on its own connection, so the
// expectation belongs on the pool mock and not on the template repo. kind is
// left unset, the way the repo mocks leave it, so the applied_at rule still
// falls back to the column's position.
func expReadTemplate(m pgxmock.PgxPoolIface, templateID, name string, order int) *pgxmock.ExpectedQuery {
	return m.ExpectQuery(`SELECT name, "order", kind FROM stage_templates`).
		WithArgs(templateID, stagesUserID).
		WillReturnRows(pgxmock.NewRows([]string{"name", "order", "kind"}).AddRow(name, order, ""))
}

func expMaxOrder(m pgxmock.PgxPoolIface, order int) *pgxmock.ExpectedQuery {
	return m.ExpectQuery(`SELECT COALESCE\(MAX`).
		WithArgs(stagesJobID).
		WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(order))
}

// expCompleteOpenExcept is the same statement with one row spared — used when a
// history row is re-activated and becomes the card's only open stage.
func expCompleteOpenExcept(m pgxmock.PgxPoolIface, keepStageID string) *pgxmock.ExpectedExec {
	return m.ExpectExec(`UPDATE job_stages SET status = 'completed', completed_at = \$2`).
		WithArgs(stagesJobID, pgxmock.AnyArg(), strPtrArg(keepStageID)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
}

// expCompleteOpen is the set-wise "close every open history row" statement that
// keeps a card down to one active stage.
func expCompleteOpen(m pgxmock.PgxPoolIface) *pgxmock.ExpectedExec {
	return m.ExpectExec(`UPDATE job_stages SET status = 'completed', completed_at = \$2`).
		WithArgs(stagesJobID, pgxmock.AnyArg(), nilStrPtrArg()).
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
func (r *recordingCommentRepo) Update(ctx context.Context, userID, commentID, content string) (*commentModel.Comment, error) {
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
	Update(ctx context.Context, userID, commentID, content string) (*commentModel.Comment, error)
	Delete(ctx context.Context, userID, commentID string) error
}

// -------------------- AddStage --------------------

func TestJobService_AddStage(t *testing.T) {
	t.Run("appends the first stage, stamps applied_at and returns the enriched DTO", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expCompleteOpen(mock)
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

		// The current stage is read inside the transaction, from the pointer the
		// locked row carries — not from the copy fetched before the lock. Its
		// column, for the log line, comes off the same connection.
		mock.ExpectBegin()
		expLockedJob(mock, nil, &currentStageID, nil)
		expReadStage(mock, currentStageID, "tmpl-prev", "active", 1)
		expReadTemplate(mock, "tmpl-prev", "Screening", 2)
		expCompleteOpen(mock)
		expMaxOrder(mock, 2)
		expInsertStage(mock)
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, appliedTemplateRepo(), nil)

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
		expCompleteOpen(mock)
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
		expCompleteOpen(mock)
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
		expCompleteOpen(mock)
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
		mock.ExpectQuery(`SELECT current_stage_template_id, current_stage_id, applied_at`).
			WithArgs(stagesJobID, stagesUserID).WillReturnError(errors.New("lock timeout"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to lock job")
	})

	t.Run("wraps an order-computation error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(stagesJobID).WillReturnError(errors.New("query failed"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to compute stage order")
	})

	t.Run("tolerates a dangling current-stage pointer", func(t *testing.T) {
		mock := newPool(t)
		currentStageID := "stage-deleted-concurrently"
		job := &model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &currentStageID}

		mock.ExpectBegin()
		expLockedJob(mock, nil, &currentStageID, nil)
		// The row the pointer names is gone (a concurrent delete) — there is
		// nothing to close, and the add must still go through.
		mock.ExpectQuery(`SELECT stage_template_id, status, "order"`).
			WithArgs(currentStageID, stagesJobID).WillReturnError(pgx.ErrNoRows)
		expCompleteOpen(mock)
		expMaxOrder(mock, 0)
		expInsertStage(mock)
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, appliedTemplateRepo(), nil)

		dto, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})

		require.NoError(t, err)
		require.NotNil(t, dto)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps a current-stage read error", func(t *testing.T) {
		mock := newPool(t)
		currentStageID := "stage-current"
		job := &model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &currentStageID}

		mock.ExpectBegin()
		expLockedJob(mock, nil, &currentStageID, nil)
		mock.ExpectQuery(`SELECT stage_template_id, status, "order"`).
			WithArgs(currentStageID, stagesJobID).WillReturnError(errors.New("read failed"))

		svc := svcWith(mock, jobRepoReturning(job), nil, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read stage")
	})

	t.Run("tolerates a column deleted under the card, but not a failing read", func(t *testing.T) {
		// A column the card still has history in may be gone — only the log
		// line loses detail, and the append goes through.
		mock := newPool(t)
		currentStageID := "stage-current"
		job := &model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &currentStageID}

		mock.ExpectBegin()
		expLockedJob(mock, nil, &currentStageID, nil)
		expReadStage(mock, currentStageID, "tmpl-deleted", "active", 1)
		mock.ExpectQuery(`SELECT name, "order", kind FROM stage_templates`).
			WithArgs("tmpl-deleted", stagesUserID).WillReturnError(pgx.ErrNoRows)
		expCompleteOpen(mock)
		expMaxOrder(mock, 2)
		expInsertStage(mock)
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, appliedTemplateRepo(), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())

		// A failing read is different: the statement aborted the transaction,
		// so the write must not be carried on and committed.
		mock = newPool(t)
		mock.ExpectBegin()
		expLockedJob(mock, nil, &currentStageID, nil)
		expReadStage(mock, currentStageID, "tmpl-deleted", "active", 1)
		mock.ExpectQuery(`SELECT name, "order", kind FROM stage_templates`).
			WithArgs("tmpl-deleted", stagesUserID).WillReturnError(errors.New("connection reset"))

		svc = svcWith(mock, jobRepoReturning(job), nil, appliedTemplateRepo(), nil)

		_, err = svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read stage template")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps an insert-stage error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expCompleteOpen(mock)
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
		expCompleteOpen(mock)
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
		expCompleteOpen(mock)
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
		expReadStage(mock, stageID, stagesTmplID, "active", 0)
		mock.ExpectExec(`UPDATE job_stages SET status = \$2, completed_at = \$3 WHERE id = \$1`).WithArgs(
			stageID, "completed", pgxmock.AnyArg(), stagesJobID,
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		expReadTemplate(mock, stagesTmplID, "Applied", 1)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

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
		expReadStage(mock, stageID, stagesTmplID, "completed", 0)
		// completed_at is passed as a (nil) *time.Time, so match it structurally.
		mock.ExpectExec(`UPDATE job_stages SET status = \$2, completed_at = \$3 WHERE id = \$1`).WithArgs(
			stageID, "active", pgxmock.AnyArg(), stagesJobID,
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		expReadTemplate(mock, stagesTmplID, "Applied", 1)
		// Re-activating a row makes it the card's only open stage and the card's
		// current position.
		expCompleteOpenExcept(mock, stageID)
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

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
		expReadStage(mock, stageID, stagesTmplID, "active", 0)
		expUpdateStageRow(mock)
		expReadTemplate(mock, stagesTmplID, "Applied", 1)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		when := timeNowUTC()
		dto, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{CompletedAt: &when})

		require.NoError(t, err)
		require.NotNil(t, dto.CompletedAt)
		assert.Equal(t, when, *dto.CompletedAt)
	})

	t.Run("rejects an invalid status without opening a transaction", func(t *testing.T) {
		mock := newPool(t)
		// No Begin expected: a bogus status never reaches the database.

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		bogus := "bogus"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &bogus})
		assert.ErrorIs(t, err, model.ErrInvalidStatus)
		require.NoError(t, mock.ExpectationsWereMet())
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

	t.Run("returns not found when the stage is gone", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		mock.ExpectQuery(`SELECT stage_template_id, status, "order"`).
			WithArgs(stageID, stagesJobID).WillReturnError(pgx.ErrNoRows)

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a stage of another job is not reachable", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		// The in-transaction read is scoped by job_id, so a foreign stage id
		// simply matches no row.
		mock.ExpectQuery(`SELECT stage_template_id, status, "order"`).
			WithArgs("stage-of-other-job", stagesJobID).WillReturnError(pgx.ErrNoRows)

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, "stage-of-other-job", &model.UpdateStageRequest{Status: &status})
		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not-found when the UPDATE affects no rows", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expReadStage(mock, stageID, stagesTmplID, "active", 0)
		mock.ExpectExec(`UPDATE job_stages SET status = \$2, completed_at = \$3 WHERE id = \$1`).WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

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
		expReadStage(mock, stageID, stagesTmplID, "active", 0)
		mock.ExpectExec(`UPDATE job_stages SET status = \$2, completed_at = \$3 WHERE id = \$1`).WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnError(errors.New("update boom"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update stage")
	})

	t.Run("returns the template lookup error after the update", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expReadStage(mock, stageID, stagesTmplID, "active", 0)
		expUpdateStageRow(mock)
		// The column was deleted while the card still had history in it: the
		// in-transaction read finds no row, and the write is not committed.
		mock.ExpectQuery(`SELECT name, "order", kind FROM stage_templates`).
			WithArgs(stagesTmplID, stagesUserID).WillReturnError(pgx.ErrNoRows)

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps a commit error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLock(mock)
		expReadStage(mock, stageID, stagesTmplID, "active", 0)
		expUpdateStageRow(mock)
		expReadTemplate(mock, stagesTmplID, "Applied", 1)
		mock.ExpectCommit().WillReturnError(errors.New("commit boom"))

		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil, appliedTemplateRepo(), nil)

		status := "completed"
		_, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID, &model.UpdateStageRequest{Status: &status})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
	})
}

// -------------------- in-transaction column reads --------------------

// A pipeline write holds a pooled connection and the card's row lock for the
// whole transaction. Resolving a column through the pool-backed repository in
// that window asks the pool for a second connection while the lock holder is
// the one that needs it to finish: a burst of concurrent stage writes starves
// the pool, and a single-connection pool deadlocks outright.
//
// These tests pin where the column comes from. Counting repository calls is the
// only way to state it deterministically here — the pool mock cannot tell the
// difference between a connection the transaction owns and a second one, which
// is exactly what the constrained-pool integration test in
// tests/e2e/jobs_pool_test.go covers.
func TestJobService_StageWritesResolveColumnsOnTheTransaction(t *testing.T) {
	// countingTemplateRepo yields an "Applied" column and counts how often the
	// pool-backed repository is reached for.
	countingTemplateRepo := func(calls *int) *MockStageTemplateRepository {
		return &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				*calls++
				return &model.StageTemplate{ID: tid, UserID: uid, Name: "Applied", Order: 1}, nil
			},
		}
	}

	t.Run("AddStage reads the repository once, before it opens the transaction", func(t *testing.T) {
		mock := newPool(t)
		currentStageID := "stage-current"

		mock.ExpectBegin()
		expLockedJob(mock, nil, &currentStageID, nil)
		expReadStage(mock, currentStageID, "tmpl-prev", "active", 1)
		// The column the card is leaving — on the transaction's connection.
		expReadTemplate(mock, "tmpl-prev", "Screening", 2)
		expCompleteOpen(mock)
		expMaxOrder(mock, 2)
		expInsertStage(mock)
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit()

		var repoCalls int
		job := &model.Job{ID: stagesJobID, UserID: stagesUserID, CurrentStageID: &currentStageID}
		svc := svcWith(mock, jobRepoReturning(job), nil, countingTemplateRepo(&repoCalls), nil)

		_, err := svc.AddStage(context.Background(), stagesUserID, stagesJobID, &model.AddStageRequest{StageTemplateID: stagesTmplID})

		require.NoError(t, err)
		assert.Equal(t, 1, repoCalls,
			"only the ownership check of the target column may go through the pool, and it runs before Begin")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("UpdateStage resolves the column without touching the repository", func(t *testing.T) {
		mock := newPool(t)
		stageID := "stage-1"

		mock.ExpectBegin()
		expLock(mock)
		expReadStage(mock, stageID, stagesTmplID, "completed", 0)
		expUpdateStageRow(mock)
		expReadTemplate(mock, stagesTmplID, "Applied", 1)
		expCompleteOpenExcept(mock, stageID)
		expUpdateJobCurrentStage(mock)
		mock.ExpectCommit()

		var repoCalls int
		svc := svcWith(mock, jobRepoReturning(&model.Job{ID: stagesJobID, UserID: stagesUserID}), nil,
			countingTemplateRepo(&repoCalls), nil)

		active := "active"
		dto, err := svc.UpdateStage(context.Background(), stagesUserID, stagesJobID, stageID,
			&model.UpdateStageRequest{Status: &active})

		require.NoError(t, err)
		assert.Equal(t, "Applied", dto.StageName, "the DTO is still enriched with the column name")
		assert.Zero(t, repoCalls, "the whole update runs inside the transaction, so nothing may hit the pool")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// -------------------- DeleteStage --------------------

func TestJobService_DeleteStage(t *testing.T) {
	stageID := "stage-1"
	otherStageID := "stage-other"
	replacementTmplID := "tmpl-prev"

	// expReplacement is the in-transaction lookup of the history row the card
	// falls back to when its current one is deleted.
	expReplacement := func(m pgxmock.PgxPoolIface, replacement *string) {
		q := m.ExpectQuery(`SELECT id, stage_template_id FROM job_stages`).WithArgs(stagesJobID, stageID)
		if replacement == nil {
			q.WillReturnError(pgx.ErrNoRows)
			return
		}
		// Handed back as a *string: the column is nullable, so that is what the
		// service scans into.
		q.WillReturnRows(pgxmock.NewRows([]string{"id", "stage_template_id"}).AddRow(replacement, &replacementTmplID))
	}
	expDelete := func(m pgxmock.PgxPoolIface) *pgxmock.ExpectedExec {
		return m.ExpectExec(`DELETE FROM job_stages WHERE id = \$1`).
			WithArgs(stageID, stagesJobID).
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
	}
	job := func() *model.Job { return &model.Job{ID: stagesJobID, UserID: stagesUserID} }

	t.Run("deletes a non-current stage without recalculating the pointer", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		// The locked row points at another stage, so no recalculation runs.
		expLockedJob(mock, nil, &otherStageID, nil)
		expReadStage(mock, stageID, stagesTmplID, "completed", 0)
		expDelete(mock)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job()), nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("repoints current_stage_id and current_stage_template_id when deleting the current stage", func(t *testing.T) {
		mock := newPool(t)
		replacement := "s-prev"
		mock.ExpectBegin()
		// The pointer read under the lock — not the copy fetched before it — is
		// what decides whether a recalculation is needed.
		expLockedJob(mock, nil, &stageID, nil)
		expReadStage(mock, stageID, stagesTmplID, "active", 1)
		expReplacement(mock, &replacement)
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3, updated_at = \$4 WHERE id = \$1`).WithArgs(
			stagesJobID, strPtrArg(replacement), strPtrArg(replacementTmplID), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		expDelete(mock)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job()), nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("sets current_stage_id and current_stage_template_id to NULL when no replacement remains", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLockedJob(mock, nil, &stageID, nil)
		expReadStage(mock, stageID, stagesTmplID, "active", 0)
		expReplacement(mock, nil)
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3, updated_at = \$4 WHERE id = \$1`).WithArgs(
			stagesJobID, nilStrPtrArg(), nilStrPtrArg(), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		expDelete(mock)
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job()), nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns the job lookup error", func(t *testing.T) {
		svc := svcWith(newPool(t), &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				return nil, model.ErrJobNotFound
			},
		}, nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		assert.ErrorIs(t, err, model.ErrJobNotFound)
	})

	t.Run("returns not found when the stage is already gone", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLockedJob(mock, nil, &otherStageID, nil)
		mock.ExpectQuery(`SELECT stage_template_id, status, "order"`).
			WithArgs(stageID, stagesJobID).WillReturnError(pgx.ErrNoRows)
		mock.ExpectRollback()

		svc := svcWith(mock, jobRepoReturning(job()), nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a stage of another job is not reachable", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLockedJob(mock, nil, &otherStageID, nil)
		// The read is scoped by job_id, so a foreign stage matches no row.
		mock.ExpectQuery(`SELECT stage_template_id, status, "order"`).
			WithArgs("stage-of-other-job", stagesJobID).WillReturnError(pgx.ErrNoRows)
		mock.ExpectRollback()

		svc := svcWith(mock, jobRepoReturning(job()), nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, "stage-of-other-job")
		assert.ErrorIs(t, err, model.ErrJobStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps a begin error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin().WillReturnError(errors.New("no conn"))

		svc := svcWith(mock, jobRepoReturning(job()), nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")
	})

	t.Run("wraps a lock error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT current_stage_template_id, current_stage_id, applied_at`).
			WithArgs(stagesJobID, stagesUserID).WillReturnError(errors.New("lock timeout"))

		svc := svcWith(mock, jobRepoReturning(job()), nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to lock job")
	})

	t.Run("wraps a replacement-lookup error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLockedJob(mock, nil, &stageID, nil)
		expReadStage(mock, stageID, stagesTmplID, "active", 0)
		mock.ExpectQuery(`SELECT id, stage_template_id FROM job_stages`).
			WithArgs(stagesJobID, stageID).WillReturnError(errors.New("list failed"))

		svc := svcWith(mock, jobRepoReturning(job()), nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find replacement stage")
	})

	t.Run("wraps a recalculation update error", func(t *testing.T) {
		mock := newPool(t)
		replacement := "s-prev"
		mock.ExpectBegin()
		expLockedJob(mock, nil, &stageID, nil)
		expReadStage(mock, stageID, stagesTmplID, "active", 0)
		expReplacement(mock, &replacement)
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3, updated_at = \$4 WHERE id = \$1`).WithArgs(
			pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnError(errors.New("update boom"))

		svc := svcWith(mock, jobRepoReturning(job()), nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update job current stage")
	})

	t.Run("wraps a delete error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLockedJob(mock, nil, &otherStageID, nil)
		expReadStage(mock, stageID, stagesTmplID, "completed", 0)
		mock.ExpectExec(`DELETE FROM job_stages WHERE id = \$1`).
			WithArgs(stageID, stagesJobID).WillReturnError(errors.New("delete boom"))

		svc := svcWith(mock, jobRepoReturning(job()), nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete stage")
	})

	t.Run("wraps a commit error", func(t *testing.T) {
		mock := newPool(t)
		mock.ExpectBegin()
		expLockedJob(mock, nil, &otherStageID, nil)
		expReadStage(mock, stageID, stagesTmplID, "completed", 0)
		expDelete(mock)
		mock.ExpectCommit().WillReturnError(errors.New("commit boom"))

		svc := svcWith(mock, jobRepoReturning(job()), nil, appliedTemplateRepo(), nil)

		err := svc.DeleteStage(context.Background(), stagesUserID, stagesJobID, stageID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
	})
}
