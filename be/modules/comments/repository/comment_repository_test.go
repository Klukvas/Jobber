package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errDB = errors.New("boom: db failure")

func newCommentRepo(t *testing.T) (*CommentRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewCommentRepository(mock), mock
}

func TestCommentRepository_Create(t *testing.T) {
	t.Run("creates comment successfully", func(t *testing.T) {
		repo, mock := newCommentRepo(t)
		comment := &model.Comment{UserID: "user-123", JobID: "job-1", Content: "Test comment"}

		mock.ExpectExec("INSERT INTO comments").
			WithArgs(pgxmock.AnyArg(), comment.UserID, comment.JobID, comment.StageID, comment.Content, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), comment)

		require.NoError(t, err)
		assert.NotEmpty(t, comment.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("creates comment with stage id", func(t *testing.T) {
		repo, mock := newCommentRepo(t)
		stageID := "stage-1"
		comment := &model.Comment{UserID: "user-123", JobID: "job-1", StageID: &stageID, Content: "Stage comment"}

		mock.ExpectExec("INSERT INTO comments").
			WithArgs(pgxmock.AnyArg(), comment.UserID, comment.JobID, comment.StageID, comment.Content, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), comment)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns job not found when zero rows affected (ownership guard)", func(t *testing.T) {
		repo, mock := newCommentRepo(t)
		comment := &model.Comment{UserID: "user-123", JobID: "foreign-job", Content: "x"}

		mock.ExpectExec("INSERT INTO comments").
			WithArgs(pgxmock.AnyArg(), comment.UserID, comment.JobID, comment.StageID, comment.Content, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		err := repo.Create(context.Background(), comment)

		assert.ErrorIs(t, err, model.ErrJobNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newCommentRepo(t)
		comment := &model.Comment{UserID: "user-123", JobID: "job-1", Content: "x"}

		mock.ExpectExec("INSERT INTO comments").
			WithArgs(pgxmock.AnyArg(), comment.UserID, comment.JobID, comment.StageID, comment.Content, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errDB)

		err := repo.Create(context.Background(), comment)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCommentRepository_ListByJob(t *testing.T) {
	cols := []string{"id", "user_id", "job_id", "stage_id", "content", "created_at", "updated_at"}

	t.Run("returns comments scoped to job owner", func(t *testing.T) {
		repo, mock := newCommentRepo(t)
		now := time.Now()

		// Real query binds user_id at $1 and job_id at $2.
		mock.ExpectQuery("FROM comments c").
			WithArgs("user-123", "job-1").
			WillReturnRows(pgxmock.NewRows(cols).
				AddRow("comment-1", "user-123", "job-1", nil, "First comment", now, now).
				AddRow("comment-2", "user-123", "job-1", nil, "Second comment", now, now))

		comments, err := repo.ListByJob(context.Background(), "job-1", "user-123")

		require.NoError(t, err)
		require.Len(t, comments, 2)
		assert.Equal(t, "First comment", comments[0].Content)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty list", func(t *testing.T) {
		repo, mock := newCommentRepo(t)

		mock.ExpectQuery("FROM comments c").
			WithArgs("user-123", "job-1").
			WillReturnRows(pgxmock.NewRows(cols))

		comments, err := repo.ListByJob(context.Background(), "job-1", "user-123")

		require.NoError(t, err)
		assert.Empty(t, comments)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates query error", func(t *testing.T) {
		repo, mock := newCommentRepo(t)

		mock.ExpectQuery("FROM comments c").
			WithArgs("user-123", "job-1").
			WillReturnError(errDB)

		comments, err := repo.ListByJob(context.Background(), "job-1", "user-123")

		assert.Nil(t, comments)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates scan error", func(t *testing.T) {
		repo, mock := newCommentRepo(t)
		now := time.Now()

		mock.ExpectQuery("FROM comments c").
			WithArgs("user-123", "job-1").
			WillReturnRows(pgxmock.NewRows(cols).
				AddRow("comment-1", "user-123", "job-1", nil, "x", "not-a-time", now)) // created_at not time.Time -> scan fails

		comments, err := repo.ListByJob(context.Background(), "job-1", "user-123")

		assert.Nil(t, comments)
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCommentRepository_Delete(t *testing.T) {
	t.Run("deletes comment successfully", func(t *testing.T) {
		repo, mock := newCommentRepo(t)

		mock.ExpectExec("DELETE FROM comments").
			WithArgs("comment-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), "user-123", "comment-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newCommentRepo(t)

		mock.ExpectExec("DELETE FROM comments").
			WithArgs("nonexistent", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), "user-123", "nonexistent")

		assert.ErrorIs(t, err, model.ErrCommentNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newCommentRepo(t)

		mock.ExpectExec("DELETE FROM comments").
			WithArgs("comment-1", "user-123").
			WillReturnError(errDB)

		err := repo.Delete(context.Background(), "user-123", "comment-1")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
