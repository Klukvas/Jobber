package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/contentlibrary/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentLibraryRepository_Create(t *testing.T) {
	t.Run("creates entry and populates generated fields", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow("gen-id", now, now)

		mock.ExpectQuery("INSERT INTO content_library").
			WithArgs("user-1", "Title", "Body", "cover").
			WillReturnRows(rows)

		repo := NewContentLibraryRepository(mock)
		entry := &model.ContentLibraryEntry{
			UserID:   "user-1",
			Title:    "Title",
			Content:  "Body",
			Category: "cover",
		}
		got, err := repo.Create(context.Background(), entry)
		require.NoError(t, err)
		assert.Equal(t, "gen-id", got.ID)
		assert.Equal(t, now, got.CreatedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("INSERT INTO content_library").
			WithArgs("user-1", "Title", "Body", "cover").
			WillReturnError(errors.New("boom"))

		repo := NewContentLibraryRepository(mock)
		got, err := repo.Create(context.Background(), &model.ContentLibraryEntry{
			UserID: "user-1", Title: "Title", Content: "Body", Category: "cover",
		})
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create content library entry")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestContentLibraryRepository_GetByID(t *testing.T) {
	t.Run("returns entry", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "user_id", "title", "content", "category", "created_at", "updated_at",
		}).AddRow("id-1", "user-1", "Title", "Body", "cover", now, now)

		mock.ExpectQuery("SELECT id, user_id, title, content, category, created_at, updated_at").
			WithArgs("id-1").
			WillReturnRows(rows)

		repo := NewContentLibraryRepository(mock)
		got, err := repo.GetByID(context.Background(), "id-1")
		require.NoError(t, err)
		assert.Equal(t, "id-1", got.ID)
		assert.Equal(t, "Title", got.Title)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps not-found error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, title, content, category, created_at, updated_at").
			WithArgs("id-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewContentLibraryRepository(mock)
		got, err := repo.GetByID(context.Background(), "id-1")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "content library entry not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestContentLibraryRepository_List(t *testing.T) {
	t.Run("returns all entries in order", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{
			"id", "user_id", "title", "content", "category", "created_at", "updated_at",
		}).
			AddRow("id-1", "user-1", "T1", "B1", "cover", now, now).
			AddRow("id-2", "user-1", "T2", "B2", "note", now, now)

		mock.ExpectQuery("SELECT id, user_id, title, content, category, created_at, updated_at").
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewContentLibraryRepository(mock)
		got, err := repo.List(context.Background(), "user-1")
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "id-1", got[0].ID)
		assert.Equal(t, "id-2", got[1].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, title, content, category, created_at, updated_at").
			WithArgs("user-1").
			WillReturnError(errors.New("boom"))

		repo := NewContentLibraryRepository(mock)
		got, err := repo.List(context.Background(), "user-1")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list content library entries")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps scan error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		// Wrong column count triggers a scan error.
		rows := pgxmock.NewRows([]string{"id"}).AddRow("id-1")
		mock.ExpectQuery("SELECT id, user_id, title, content, category, created_at, updated_at").
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewContentLibraryRepository(mock)
		got, err := repo.List(context.Background(), "user-1")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to scan content library entry")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestContentLibraryRepository_Update(t *testing.T) {
	t.Run("updates entry and refreshes updated_at", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{"updated_at"}).AddRow(now)
		mock.ExpectQuery("UPDATE content_library SET title").
			WithArgs("Title", "Body", "cover", "id-1", "user-1").
			WillReturnRows(rows)

		repo := NewContentLibraryRepository(mock)
		entry := &model.ContentLibraryEntry{ID: "id-1", UserID: "user-1", Title: "Title", Content: "Body", Category: "cover"}
		got, err := repo.Update(context.Background(), entry)
		require.NoError(t, err)
		assert.Equal(t, now, got.UpdatedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("UPDATE content_library SET title").
			WithArgs("Title", "Body", "cover", "id-1", "user-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewContentLibraryRepository(mock)
		got, err := repo.Update(context.Background(), &model.ContentLibraryEntry{
			ID: "id-1", UserID: "user-1", Title: "Title", Content: "Body", Category: "cover",
		})
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update content library entry")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestContentLibraryRepository_Delete(t *testing.T) {
	t.Run("deletes entry", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM content_library").
			WithArgs("id-1", "user-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := NewContentLibraryRepository(mock)
		require.NoError(t, repo.Delete(context.Background(), "id-1", "user-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not-found when no rows affected", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM content_library").
			WithArgs("id-1", "user-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := NewContentLibraryRepository(mock)
		err = repo.Delete(context.Background(), "id-1", "user-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "content library entry not found")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM content_library").
			WithArgs("id-1", "user-1").
			WillReturnError(errors.New("boom"))

		repo := NewContentLibraryRepository(mock)
		err = repo.Delete(context.Background(), "id-1", "user-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete content library entry")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
