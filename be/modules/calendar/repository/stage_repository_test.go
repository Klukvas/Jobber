package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/modules/calendar/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStageRepository_SetCalendarEventID(t *testing.T) {
	t.Run("sets event id", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE job_stages SET calendar_event_id").
			WithArgs("stage-1", "event-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := NewStageRepository(mock)
		require.NoError(t, repo.SetCalendarEventID(context.Background(), "stage-1", "event-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrStageNotFound when no rows affected", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE job_stages SET calendar_event_id").
			WithArgs("stage-1", "event-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := NewStageRepository(mock)
		err = repo.SetCalendarEventID(context.Background(), "stage-1", "event-1")
		assert.ErrorIs(t, err, model.ErrStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE job_stages SET calendar_event_id").
			WithArgs("stage-1", "event-1").
			WillReturnError(errors.New("boom"))

		repo := NewStageRepository(mock)
		err = repo.SetCalendarEventID(context.Background(), "stage-1", "event-1")
		assert.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStageRepository_ClearCalendarEventID(t *testing.T) {
	t.Run("clears event id", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE job_stages SET calendar_event_id = NULL").
			WithArgs("stage-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := NewStageRepository(mock)
		require.NoError(t, repo.ClearCalendarEventID(context.Background(), "stage-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrStageNotFound when no rows affected", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE job_stages SET calendar_event_id = NULL").
			WithArgs("stage-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := NewStageRepository(mock)
		err = repo.ClearCalendarEventID(context.Background(), "stage-1")
		assert.ErrorIs(t, err, model.ErrStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE job_stages SET calendar_event_id = NULL").
			WithArgs("stage-1").
			WillReturnError(errors.New("boom"))

		repo := NewStageRepository(mock)
		err = repo.ClearCalendarEventID(context.Background(), "stage-1")
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStageRepository_GetCalendarEventID(t *testing.T) {
	t.Run("returns event id", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		eventID := "event-1"
		rows := pgxmock.NewRows([]string{"calendar_event_id"}).AddRow(&eventID)
		mock.ExpectQuery("SELECT calendar_event_id FROM job_stages").
			WithArgs("stage-1").
			WillReturnRows(rows)

		repo := NewStageRepository(mock)
		got, err := repo.GetCalendarEventID(context.Background(), "stage-1")
		require.NoError(t, err)
		assert.Equal(t, "event-1", got)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrStageNotFound when no rows", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT calendar_event_id FROM job_stages").
			WithArgs("stage-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewStageRepository(mock)
		got, err := repo.GetCalendarEventID(context.Background(), "stage-1")
		assert.Empty(t, got)
		assert.ErrorIs(t, err, model.ErrStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrEventNotFound when event id is NULL", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"calendar_event_id"}).AddRow((*string)(nil))
		mock.ExpectQuery("SELECT calendar_event_id FROM job_stages").
			WithArgs("stage-1").
			WillReturnRows(rows)

		repo := NewStageRepository(mock)
		got, err := repo.GetCalendarEventID(context.Background(), "stage-1")
		assert.Empty(t, got)
		assert.ErrorIs(t, err, model.ErrEventNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrEventNotFound when event id is empty string", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		empty := ""
		rows := pgxmock.NewRows([]string{"calendar_event_id"}).AddRow(&empty)
		mock.ExpectQuery("SELECT calendar_event_id FROM job_stages").
			WithArgs("stage-1").
			WillReturnRows(rows)

		repo := NewStageRepository(mock)
		got, err := repo.GetCalendarEventID(context.Background(), "stage-1")
		assert.Empty(t, got)
		assert.ErrorIs(t, err, model.ErrEventNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT calendar_event_id FROM job_stages").
			WithArgs("stage-1").
			WillReturnError(errors.New("boom"))

		repo := NewStageRepository(mock)
		_, err = repo.GetCalendarEventID(context.Background(), "stage-1")
		assert.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStageRepository_GetStageUserID(t *testing.T) {
	t.Run("returns owning user id", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"user_id"}).AddRow("user-1")
		mock.ExpectQuery("SELECT j.user_id").
			WithArgs("stage-1").
			WillReturnRows(rows)

		repo := NewStageRepository(mock)
		got, err := repo.GetStageUserID(context.Background(), "stage-1")
		require.NoError(t, err)
		assert.Equal(t, "user-1", got)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrStageNotFound when no rows", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT j.user_id").
			WithArgs("stage-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewStageRepository(mock)
		got, err := repo.GetStageUserID(context.Background(), "stage-1")
		assert.Empty(t, got)
		assert.ErrorIs(t, err, model.ErrStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT j.user_id").
			WithArgs("stage-1").
			WillReturnError(errors.New("boom"))

		repo := NewStageRepository(mock)
		_, err = repo.GetStageUserID(context.Background(), "stage-1")
		assert.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrStageNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
