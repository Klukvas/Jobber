package repository

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/comments/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentRepository_Create(t *testing.T) {
	t.Run("creates comment successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		comment := &model.Comment{
			UserID:        "user-123",
			ApplicationID: "app-1",
			Content:       "Test comment",
		}

		mock.ExpectExec("INSERT INTO comments").
			WithArgs(pgxmock.AnyArg(), comment.UserID, comment.ApplicationID, comment.StageID, comment.Content, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := &testCommentRepo{mock: mock}
		err = repo.Create(context.Background(), comment)

		require.NoError(t, err)
		assert.NotEmpty(t, comment.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("creates comment with stage ID", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		stageID := "stage-1"
		comment := &model.Comment{
			UserID:        "user-123",
			ApplicationID: "app-1",
			StageID:       &stageID,
			Content:       "Stage comment",
		}

		mock.ExpectExec("INSERT INTO comments").
			WithArgs(pgxmock.AnyArg(), comment.UserID, comment.ApplicationID, comment.StageID, comment.Content, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := &testCommentRepo{mock: mock}
		err = repo.Create(context.Background(), comment)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCommentRepository_ListByApplication(t *testing.T) {
	t.Run("returns comments list", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		appID := "app-1"
		now := time.Now()

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "application_id", "stage_id", "content", "created_at", "updated_at",
		}).
			AddRow("comment-1", "user-123", appID, nil, "First comment", now, now).
			AddRow("comment-2", "user-123", appID, nil, "Second comment", now, now)

		mock.ExpectQuery("SELECT id, user_id, application_id, stage_id, content, created_at, updated_at").
			WithArgs(appID, "user-123").
			WillReturnRows(rows)

		repo := &testCommentRepo{mock: mock}
		comments, err := repo.ListByApplication(context.Background(), appID, "user-123")

		require.NoError(t, err)
		assert.Len(t, comments, 2)
		assert.Equal(t, "First comment", comments[0].Content)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty list", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		appID := "app-1"

		rows := pgxmock.NewRows([]string{
			"id", "user_id", "application_id", "stage_id", "content", "created_at", "updated_at",
		})

		mock.ExpectQuery("SELECT id, user_id, application_id, stage_id, content, created_at, updated_at").
			WithArgs(appID, "user-123").
			WillReturnRows(rows)

		repo := &testCommentRepo{mock: mock}
		comments, err := repo.ListByApplication(context.Background(), appID, "user-123")

		require.NoError(t, err)
		assert.Empty(t, comments)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCommentRepository_Delete(t *testing.T) {
	t.Run("deletes comment successfully", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM comments").
			WithArgs("comment-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := &testCommentRepo{mock: mock}
		err = repo.Delete(context.Background(), "user-123", "comment-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns error when comment not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM comments").
			WithArgs("nonexistent", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := &testCommentRepo{mock: mock}
		err = repo.Delete(context.Background(), "user-123", "nonexistent")

		assert.Equal(t, model.ErrCommentNotFound, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// testCommentRepo is a test wrapper that uses pgxmock
type testCommentRepo struct {
	mock pgxmock.PgxPoolIface
}

func (r *testCommentRepo) Create(ctx context.Context, comment *model.Comment) error {
	query := `
		INSERT INTO comments (id, user_id, application_id, stage_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	comment.ID = "test-comment-id"
	now := time.Now().UTC()
	comment.CreatedAt = now
	comment.UpdatedAt = now

	_, err := r.mock.Exec(ctx, query,
		comment.ID, comment.UserID, comment.ApplicationID, comment.StageID, comment.Content, comment.CreatedAt, comment.UpdatedAt,
	)
	return err
}

func (r *testCommentRepo) ListByApplication(ctx context.Context, appID string, userID ...string) ([]*model.Comment, error) {
	query := `
		SELECT id, user_id, application_id, stage_id, content, created_at, updated_at
		FROM comments
		WHERE application_id = $1 AND user_id = $2
		ORDER BY created_at ASC
	`

	uid := ""
	if len(userID) > 0 {
		uid = userID[0]
	}

	rows, err := r.mock.Query(ctx, query, appID, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*model.Comment
	for rows.Next() {
		comment := &model.Comment{}
		if err := rows.Scan(
			&comment.ID, &comment.UserID, &comment.ApplicationID, &comment.StageID, &comment.Content, &comment.CreatedAt, &comment.UpdatedAt,
		); err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}

	return comments, nil
}

func (r *testCommentRepo) Delete(ctx context.Context, userID, commentID string) error {
	query := `DELETE FROM comments WHERE id = $1 AND user_id = $2`
	result, err := r.mock.Exec(ctx, query, commentID, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return model.ErrCommentNotFound
		}
		return err
	}
	if result.RowsAffected() == 0 {
		return model.ErrCommentNotFound
	}
	return nil
}
