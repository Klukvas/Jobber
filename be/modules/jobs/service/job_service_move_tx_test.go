package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests drive Move's transactional body (past the validation and no-op
// fast paths already covered in job_service_move_test.go) through the pgxmock
// pool injected via the txBeginner seam.

// sequencedJobRepo hands out the given rows one per GetByID call, repeating the
// last one — a stand-in for a row another request rewrote between Move's
// pre-transaction read and its re-read after the commit.
func sequencedJobRepo(rows ...*model.Job) *MockJobRepository {
	call := 0
	return &MockJobRepository{
		GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
			row := rows[min(call, len(rows)-1)]
			call++
			return row, nil
		},
	}
}

func TestMove_Transaction(t *testing.T) {
	userID := stagesUserID
	jobID := stagesJobID
	targetTmpl := "tmpl-screening"

	// targetTemplateRepo yields a target column (order 2) distinct from the
	// card's current column so Move takes the write path.
	targetTemplateRepo := func() *MockStageTemplateRepository {
		return &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return &model.StageTemplate{ID: tid, UserID: uid, Name: "Screening", Order: 2}, nil
			},
		}
	}

	t.Run("moves a card with a prior active stage, stamps applied_at, returns enriched DTO", func(t *testing.T) {
		mock := newPool(t)
		currentStageID := "stage-current"
		currentTmpl := "tmpl-wishlist"
		job := &model.Job{
			ID: jobID, UserID: userID, Title: "Engineer",
			CurrentStageID:         &currentStageID,
			CurrentStageTemplateID: &currentTmpl,
		}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, &currentStageID, nil)
		// Close whatever stage was open — set-wise, so a card cannot end up
		// with two active rows.
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(1))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(
			pgxmock.AnyArg(), jobID, targetTmpl, 1, pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3`).WithArgs(
			jobID, pgxmock.AnyArg(), targetTmpl, pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)

		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})

		require.NoError(t, err)
		require.NotNil(t, dto)
		require.NotNil(t, dto.CurrentStageTemplateID)
		assert.Equal(t, targetTmpl, *dto.CurrentStageTemplateID)
		require.NotNil(t, dto.CurrentStageName)
		assert.Equal(t, "Screening", *dto.CurrentStageName)
		// applied_at is stamped because the target column order (2) > 0.
		assert.NotNil(t, dto.AppliedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("moves a card with no prior stage (skips the complete step)", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-wishlist"
		job := &model.Job{
			ID: jobID, UserID: userID, Title: "Engineer",
			CurrentStageTemplateID: &currentTmpl, // no CurrentStageID
		}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, nil)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(0))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE jobs SET current_stage_id`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)

		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})

		require.NoError(t, err)
		require.NotNil(t, dto)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("does not stamp applied_at when moving into the first column", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-applied"
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &currentTmpl}

		firstColumnRepo := &MockStageTemplateRepository{
			GetByIDFunc: func(ctx context.Context, uid, tid string) (*model.StageTemplate, error) {
				return &model.StageTemplate{ID: tid, UserID: uid, Name: "Wishlist", Order: 0}, nil
			},
		}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, nil)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(0))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		// applied_at (4th arg) stays nil because target order == 0.
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3`).WithArgs(
			jobID, pgxmock.AnyArg(), "tmpl-wishlist", nilTimePtrArg(), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, firstColumnRepo, nil)

		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: "tmpl-wishlist"})

		require.NoError(t, err)
		assert.Nil(t, dto.AppliedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a stale copy claiming another column cannot force a redundant write", func(t *testing.T) {
		mock := newPool(t)
		staleTmpl := "tmpl-wishlist"
		// The caller's copy says the card is still in the wishlist, but it has
		// already been moved to the target column by another request. Deciding
		// on the stale copy would append a second history row for a column the
		// card is already in.
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &staleTmpl}

		mock.ExpectBegin()
		expLockedJob(mock, &targetTmpl, nil, nil)
		mock.ExpectRollback()

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)

		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})

		require.NoError(t, err)
		require.NotNil(t, dto.CurrentStageTemplateID)
		assert.Equal(t, targetTmpl, *dto.CurrentStageTemplateID, "the response reports the state under the lock")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a stale copy claiming the target column cannot swallow the move", func(t *testing.T) {
		mock := newPool(t)
		staleTmpl := targetTmpl
		// The caller's copy says the card already sits in the target, but the
		// locked row says otherwise (another request moved it away). The move
		// must actually happen instead of returning a silent no-op.
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &staleTmpl}
		elsewhere := "tmpl-rejected"

		mock.ExpectBegin()
		expLockedJob(mock, &elsewhere, nil, nil)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(3))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(
			pgxmock.AnyArg(), jobID, targetTmpl, 3, pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3`).WithArgs(
			jobID, pgxmock.AnyArg(), targetTmpl, pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)

		_, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("keeps the applied_at found under the lock instead of stamping a second one", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-wishlist"
		// The caller's copy has no applied_at, but a concurrent move already
		// stamped one. Re-stamping would reset the user's application date.
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &currentTmpl}
		stamped := time.Date(2026, time.January, 7, 12, 0, 0, 0, time.UTC)

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, &stamped)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(1))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(
			pgxmock.AnyArg(), jobID, targetTmpl, 1, pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3`).WithArgs(
			jobID, pgxmock.AnyArg(), targetTmpl, timePtrArg(stamped), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)

		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})

		require.NoError(t, err)
		require.NotNil(t, dto.AppliedAt)
		assert.True(t, dto.AppliedAt.Equal(stamped))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps a begin error", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-wishlist"
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &currentTmpl}
		mock.ExpectBegin().WillReturnError(errors.New("no conn"))

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)
		_, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")
	})

	t.Run("wraps a lock error", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-wishlist"
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &currentTmpl}
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT current_stage_template_id, current_stage_id, applied_at`).
			WithArgs(jobID, userID).WillReturnError(errors.New("lock timeout"))

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)
		_, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to lock job")
	})

	t.Run("wraps a complete-current-stage error", func(t *testing.T) {
		mock := newPool(t)
		currentStageID := "stage-current"
		currentTmpl := "tmpl-wishlist"
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageID: &currentStageID, CurrentStageTemplateID: &currentTmpl}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, &currentStageID, nil)
		mock.ExpectExec(`UPDATE job_stages SET status = 'completed'`).
			WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("complete boom"))

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)
		_, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to complete current stage")
	})

	t.Run("wraps an order-computation error", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-wishlist"
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &currentTmpl}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, nil)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnError(errors.New("query boom"))

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)
		_, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to compute stage order")
	})

	t.Run("wraps an insert-stage error", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-wishlist"
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &currentTmpl}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, nil)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(0))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("insert boom"))

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)
		_, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create stage")
	})

	t.Run("wraps an update-job error", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-wishlist"
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &currentTmpl}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, nil)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(0))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE jobs SET current_stage_id`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("update boom"))

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)
		_, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update job")
	})

	t.Run("wraps a commit error", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-wishlist"
		job := &model.Job{ID: jobID, UserID: userID, CurrentStageTemplateID: &currentTmpl}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, nil)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(0))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE jobs SET current_stage_id`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit().WillReturnError(errors.New("commit boom"))

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)
		_, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
	})

	// Move answers with the card as the database holds it after the write, not
	// as the pre-transaction snapshot described it. A PATCH that commits in that
	// window is otherwise reported back with its edit missing, and the frontend
	// caches that stale copy — the rename vanishes on screen while the stored
	// row is correct. These two tests simulate the PATCH by having the repo hand
	// out the stale copy first and the updated row afterwards.
	t.Run("the response carries a rename that landed during the move", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-wishlist"
		stale := &model.Job{ID: jobID, UserID: userID, Title: "Engineer", CurrentStageTemplateID: &currentTmpl}
		// What the row holds after the commit: this move's column plus the
		// rename that landed while the move was in flight.
		stored := &model.Job{ID: jobID, UserID: userID, Title: "Senior Engineer", CurrentStageTemplateID: &targetTmpl}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, nil)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(1))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(
			pgxmock.AnyArg(), jobID, targetTmpl, 1, pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3`).WithArgs(
			jobID, pgxmock.AnyArg(), targetTmpl, pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		svc := svcWith(mock, sequencedJobRepo(stale, stored), nil, targetTemplateRepo(), nil)

		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})

		require.NoError(t, err)
		assert.Equal(t, "Senior Engineer", dto.Title, "the move must not hand back a pre-write snapshot")
		require.NotNil(t, dto.CurrentStageTemplateID)
		assert.Equal(t, targetTmpl, *dto.CurrentStageTemplateID, "the pipeline change it just wrote is still reported")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a no-op move also answers from the fresh row", func(t *testing.T) {
		mock := newPool(t)
		stale := &model.Job{ID: jobID, UserID: userID, Title: "Engineer", CurrentStageTemplateID: &targetTmpl}
		renamed := &model.Job{ID: jobID, UserID: userID, Title: "Senior Engineer", CurrentStageTemplateID: &targetTmpl}

		mock.ExpectBegin()
		expLockedJob(mock, &targetTmpl, nil, nil)
		mock.ExpectRollback()

		svc := svcWith(mock, sequencedJobRepo(stale, renamed), nil, targetTemplateRepo(), nil)

		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})

		require.NoError(t, err)
		assert.Equal(t, "Senior Engineer", dto.Title)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a failed re-read still reports the committed move", func(t *testing.T) {
		mock := newPool(t)
		currentTmpl := "tmpl-wishlist"
		stale := &model.Job{ID: jobID, UserID: userID, Title: "Engineer", CurrentStageTemplateID: &currentTmpl}

		mock.ExpectBegin()
		expLockedJob(mock, &currentTmpl, nil, nil)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(1))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(
			pgxmock.AnyArg(), jobID, targetTmpl, 1, pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE jobs SET current_stage_id = \$2, current_stage_template_id = \$3`).WithArgs(
			jobID, pgxmock.AnyArg(), targetTmpl, pgxmock.AnyArg(), pgxmock.AnyArg(),
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		calls := 0
		repo := &MockJobRepository{
			GetByIDFunc: func(ctx context.Context, uid, jid string) (*model.Job, error) {
				calls++
				if calls == 1 {
					return stale, nil
				}
				return nil, errors.New("card vanished")
			},
		}

		svc := svcWith(mock, repo, nil, targetTemplateRepo(), nil)

		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})

		require.NoError(t, err, "the write is already committed — a read failure must not undo the answer")
		require.NotNil(t, dto.CurrentStageTemplateID)
		assert.Equal(t, targetTmpl, *dto.CurrentStageTemplateID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("moves when the card has no current column set", func(t *testing.T) {
		mock := newPool(t)
		// The locked row carries no column → not a no-op; takes the write path.
		job := &model.Job{ID: jobID, UserID: userID}

		mock.ExpectBegin()
		expLockedJob(mock, nil, nil, nil)
		expCompleteOpen(mock)
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(0))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE jobs SET current_stage_id`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)
		dto, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})
		require.NoError(t, err)
		require.NotNil(t, dto)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
