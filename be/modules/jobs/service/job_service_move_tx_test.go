package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests drive Move's transactional body (past the validation and no-op
// fast paths already covered in job_service_move_test.go) through the pgxmock
// pool injected via the txBeginner seam.

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
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(jobID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		// Complete the current active stage.
		mock.ExpectExec(`UPDATE job_stages SET status = 'completed', completed_at = \$2 WHERE id = \$1 AND status != 'completed'`).WithArgs(
			currentStageID, pgxmock.AnyArg(), jobID,
		).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
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
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(jobID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		// No "UPDATE job_stages SET status = 'completed'" here — nothing to complete.
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
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(jobID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
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
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(jobID).WillReturnError(errors.New("lock timeout"))

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
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(jobID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		mock.ExpectExec(`UPDATE job_stages SET status = 'completed'`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("complete boom"))

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
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(jobID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
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
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(jobID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
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
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(jobID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
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
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(jobID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
		mock.ExpectQuery(`SELECT COALESCE\(MAX`).WithArgs(jobID).WillReturnRows(pgxmock.NewRows([]string{"order"}).AddRow(0))
		mock.ExpectExec(`INSERT INTO job_stages`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec(`UPDATE jobs SET current_stage_id`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit().WillReturnError(errors.New("commit boom"))

		svc := svcWith(mock, jobRepoReturning(job), nil, targetTemplateRepo(), nil)
		_, err := svc.Move(context.Background(), userID, jobID, &model.MoveJobRequest{StageTemplateID: targetTmpl})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
	})

	t.Run("moves when the card has no current column set", func(t *testing.T) {
		mock := newPool(t)
		// CurrentStageTemplateID nil → not a no-op; takes the write path.
		job := &model.Job{ID: jobID, UserID: userID}

		mock.ExpectBegin()
		mock.ExpectExec(`SELECT id FROM jobs WHERE id = \$1 FOR UPDATE`).WithArgs(jobID).WillReturnResult(pgxmock.NewResult("SELECT", 1))
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
