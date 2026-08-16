package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/reminders/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errDB = errors.New("boom: db failure")

func newReminderRepo(t *testing.T) (*ReminderRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewReminderRepository(mock), mock
}

func TestReminderRepository_Create(t *testing.T) {
	t.Run("creates reminder successfully", func(t *testing.T) {
		repo, mock := newReminderRepo(t)
		rem := &model.Reminder{UserID: "user-123", JobID: "job-1", RemindAt: time.Now(), Message: "call back"}

		mock.ExpectExec("INSERT INTO reminders").
			WithArgs(pgxmock.AnyArg(), rem.UserID, rem.JobID, rem.StageID, rem.RemindAt, rem.Message, rem.IsDone, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), rem)

		require.NoError(t, err)
		assert.NotEmpty(t, rem.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns job not found when zero rows affected (ownership guard)", func(t *testing.T) {
		repo, mock := newReminderRepo(t)
		rem := &model.Reminder{UserID: "user-123", JobID: "foreign-job", RemindAt: time.Now(), Message: "x"}

		mock.ExpectExec("INSERT INTO reminders").
			WithArgs(pgxmock.AnyArg(), rem.UserID, rem.JobID, rem.StageID, rem.RemindAt, rem.Message, rem.IsDone, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		err := repo.Create(context.Background(), rem)

		assert.ErrorIs(t, err, model.ErrJobNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newReminderRepo(t)
		rem := &model.Reminder{UserID: "user-123", JobID: "job-1", RemindAt: time.Now(), Message: "x"}

		mock.ExpectExec("INSERT INTO reminders").
			WithArgs(pgxmock.AnyArg(), rem.UserID, rem.JobID, rem.StageID, rem.RemindAt, rem.Message, rem.IsDone, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errDB)

		err := repo.Create(context.Background(), rem)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

var reminderCols = []string{"id", "user_id", "job_id", "stage_id", "remind_at", "message", "is_done", "created_at", "updated_at"}

func TestReminderRepository_ListByUser(t *testing.T) {
	t.Run("returns reminders ordered by remind_at", func(t *testing.T) {
		repo, mock := newReminderRepo(t)
		now := time.Now()

		mock.ExpectQuery("FROM reminders WHERE user_id = ").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows(reminderCols).
				AddRow("rem-1", "user-123", "job-1", nil, now, "First", false, now, now).
				AddRow("rem-2", "user-123", "job-2", nil, now, "Second", true, now, now))

		reminders, err := repo.ListByUser(context.Background(), "user-123")

		require.NoError(t, err)
		require.Len(t, reminders, 2)
		assert.Equal(t, "First", reminders[0].Message)
		assert.True(t, reminders[1].IsDone)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice when none", func(t *testing.T) {
		repo, mock := newReminderRepo(t)

		mock.ExpectQuery("FROM reminders WHERE user_id = ").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows(reminderCols))

		reminders, err := repo.ListByUser(context.Background(), "user-123")

		require.NoError(t, err)
		assert.Empty(t, reminders)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates query error", func(t *testing.T) {
		repo, mock := newReminderRepo(t)

		mock.ExpectQuery("FROM reminders WHERE user_id = ").
			WithArgs("user-123").
			WillReturnError(errDB)

		reminders, err := repo.ListByUser(context.Background(), "user-123")

		assert.Nil(t, reminders)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates scan error", func(t *testing.T) {
		repo, mock := newReminderRepo(t)
		now := time.Now()

		mock.ExpectQuery("FROM reminders WHERE user_id = ").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows(reminderCols).
				AddRow("rem-1", "user-123", "job-1", nil, "not-a-time", "First", false, now, now))

		reminders, err := repo.ListByUser(context.Background(), "user-123")

		assert.Nil(t, reminders)
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReminderRepository_ListByJob(t *testing.T) {
	t.Run("returns reminders for job scoped to user", func(t *testing.T) {
		repo, mock := newReminderRepo(t)
		now := time.Now()

		mock.ExpectQuery("FROM reminders WHERE user_id = .* AND job_id = ").
			WithArgs("user-123", "job-1").
			WillReturnRows(pgxmock.NewRows(reminderCols).
				AddRow("rem-1", "user-123", "job-1", nil, now, "Only", false, now, now))

		reminders, err := repo.ListByJob(context.Background(), "user-123", "job-1")

		require.NoError(t, err)
		require.Len(t, reminders, 1)
		assert.Equal(t, "job-1", reminders[0].JobID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates query error", func(t *testing.T) {
		repo, mock := newReminderRepo(t)

		mock.ExpectQuery("FROM reminders WHERE user_id = .* AND job_id = ").
			WithArgs("user-123", "job-1").
			WillReturnError(errDB)

		reminders, err := repo.ListByJob(context.Background(), "user-123", "job-1")

		assert.Nil(t, reminders)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReminderRepository_GetByID(t *testing.T) {
	t.Run("returns reminder successfully", func(t *testing.T) {
		repo, mock := newReminderRepo(t)
		now := time.Now()
		stageID := "stage-1"

		mock.ExpectQuery("FROM reminders WHERE id = ").
			WithArgs("rem-1", "user-123").
			WillReturnRows(pgxmock.NewRows(reminderCols).
				AddRow("rem-1", "user-123", "job-1", &stageID, now, "Msg", false, now, now))

		rem, err := repo.GetByID(context.Background(), "user-123", "rem-1")

		require.NoError(t, err)
		assert.Equal(t, "rem-1", rem.ID)
		require.NotNil(t, rem.StageID)
		assert.Equal(t, "stage-1", *rem.StageID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found on ErrNoRows", func(t *testing.T) {
		repo, mock := newReminderRepo(t)

		mock.ExpectQuery("FROM reminders WHERE id = ").
			WithArgs("nope", "user-123").
			WillReturnError(pgx.ErrNoRows)

		rem, err := repo.GetByID(context.Background(), "user-123", "nope")

		assert.Nil(t, rem)
		assert.ErrorIs(t, err, model.ErrReminderNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		repo, mock := newReminderRepo(t)

		mock.ExpectQuery("FROM reminders WHERE id = ").
			WithArgs("rem-1", "user-123").
			WillReturnError(errDB)

		rem, err := repo.GetByID(context.Background(), "user-123", "rem-1")

		assert.Nil(t, rem)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReminderRepository_Update(t *testing.T) {
	t.Run("updates reminder successfully", func(t *testing.T) {
		repo, mock := newReminderRepo(t)
		rem := &model.Reminder{ID: "rem-1", UserID: "user-123", Message: "Updated", RemindAt: time.Now(), IsDone: true}

		mock.ExpectExec("UPDATE reminders").
			WithArgs(rem.ID, rem.Message, rem.RemindAt, rem.IsDone, pgxmock.AnyArg(), rem.UserID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		err := repo.Update(context.Background(), rem)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newReminderRepo(t)
		rem := &model.Reminder{ID: "nope", UserID: "user-123", Message: "X", RemindAt: time.Now()}

		mock.ExpectExec("UPDATE reminders").
			WithArgs(rem.ID, rem.Message, rem.RemindAt, rem.IsDone, pgxmock.AnyArg(), rem.UserID).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		err := repo.Update(context.Background(), rem)

		assert.ErrorIs(t, err, model.ErrReminderNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newReminderRepo(t)
		rem := &model.Reminder{ID: "rem-1", UserID: "user-123", Message: "X", RemindAt: time.Now()}

		mock.ExpectExec("UPDATE reminders").
			WithArgs(rem.ID, rem.Message, rem.RemindAt, rem.IsDone, pgxmock.AnyArg(), rem.UserID).
			WillReturnError(errDB)

		err := repo.Update(context.Background(), rem)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestReminderRepository_Delete(t *testing.T) {
	t.Run("deletes reminder successfully", func(t *testing.T) {
		repo, mock := newReminderRepo(t)

		mock.ExpectExec("DELETE FROM reminders").
			WithArgs("rem-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), "user-123", "rem-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newReminderRepo(t)

		mock.ExpectExec("DELETE FROM reminders").
			WithArgs("nope", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), "user-123", "nope")

		assert.ErrorIs(t, err, model.ErrReminderNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newReminderRepo(t)

		mock.ExpectExec("DELETE FROM reminders").
			WithArgs("rem-1", "user-123").
			WillReturnError(errDB)

		err := repo.Delete(context.Background(), "user-123", "rem-1")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
