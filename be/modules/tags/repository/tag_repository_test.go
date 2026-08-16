package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/tags/model"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errDB = errors.New("boom: db failure")

func newTagRepo(t *testing.T) (*TagRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewTagRepository(mock), mock
}

func TestTagRepository_Create(t *testing.T) {
	t.Run("creates tag successfully", func(t *testing.T) {
		repo, mock := newTagRepo(t)
		color := "#fff"
		tag := &model.Tag{UserID: "user-123", Name: "Backend", Color: &color}

		mock.ExpectExec("INSERT INTO tags").
			WithArgs(pgxmock.AnyArg(), tag.UserID, tag.Name, tag.Color, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), tag)

		require.NoError(t, err)
		assert.NotEmpty(t, tag.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps unique violation to ErrTagNameExists", func(t *testing.T) {
		repo, mock := newTagRepo(t)
		tag := &model.Tag{UserID: "user-123", Name: "Dup"}

		mock.ExpectExec("INSERT INTO tags").
			WithArgs(pgxmock.AnyArg(), tag.UserID, tag.Name, tag.Color, pgxmock.AnyArg()).
			WillReturnError(&pgconn.PgError{Code: uniqueViolationCode})

		err := repo.Create(context.Background(), tag)

		assert.ErrorIs(t, err, model.ErrTagNameExists)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates other db error", func(t *testing.T) {
		repo, mock := newTagRepo(t)
		tag := &model.Tag{UserID: "user-123", Name: "X"}

		mock.ExpectExec("INSERT INTO tags").
			WithArgs(pgxmock.AnyArg(), tag.UserID, tag.Name, tag.Color, pgxmock.AnyArg()).
			WillReturnError(errDB)

		err := repo.Create(context.Background(), tag)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTagRepository_List(t *testing.T) {
	cols := []string{"id", "user_id", "name", "color", "created_at"}

	t.Run("returns tags for user", func(t *testing.T) {
		repo, mock := newTagRepo(t)
		now := time.Now()

		c1, c2 := "#111", "#222"
		mock.ExpectQuery("SELECT id, user_id, name, color, created_at FROM tags").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows(cols).
				AddRow("tag-1", "user-123", "Backend", &c1, now).
				AddRow("tag-2", "user-123", "Frontend", &c2, now))

		tags, err := repo.List(context.Background(), "user-123")

		require.NoError(t, err)
		require.Len(t, tags, 2)
		assert.Equal(t, "Backend", tags[0].Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns empty slice when none", func(t *testing.T) {
		repo, mock := newTagRepo(t)

		mock.ExpectQuery("SELECT id, user_id, name, color, created_at FROM tags").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows(cols))

		tags, err := repo.List(context.Background(), "user-123")

		require.NoError(t, err)
		assert.Empty(t, tags)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates query error", func(t *testing.T) {
		repo, mock := newTagRepo(t)

		mock.ExpectQuery("FROM tags").
			WithArgs("user-123").
			WillReturnError(errDB)

		tags, err := repo.List(context.Background(), "user-123")

		assert.Nil(t, tags)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates scan error", func(t *testing.T) {
		repo, mock := newTagRepo(t)

		mock.ExpectQuery("FROM tags").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows(cols).
				AddRow("tag-1", "user-123", "Backend", nil, "not-a-time"))

		tags, err := repo.List(context.Background(), "user-123")

		assert.Nil(t, tags)
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTagRepository_Delete(t *testing.T) {
	t.Run("deletes tag successfully", func(t *testing.T) {
		repo, mock := newTagRepo(t)

		mock.ExpectExec("DELETE FROM tags").
			WithArgs("tag-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), "user-123", "tag-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newTagRepo(t)

		mock.ExpectExec("DELETE FROM tags").
			WithArgs("nonexistent", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), "user-123", "nonexistent")

		assert.ErrorIs(t, err, model.ErrTagNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newTagRepo(t)

		mock.ExpectExec("DELETE FROM tags").
			WithArgs("tag-1", "user-123").
			WillReturnError(errDB)

		err := repo.Delete(context.Background(), "user-123", "tag-1")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTagRepository_AddRelation(t *testing.T) {
	t.Run("attaches tag successfully", func(t *testing.T) {
		repo, mock := newTagRepo(t)
		rel := &model.TagRelation{TagID: "tag-1", EntityType: "job", EntityID: "job-1"}

		mock.ExpectExec("INSERT INTO tag_relations").
			WithArgs(pgxmock.AnyArg(), rel.TagID, rel.EntityType, rel.EntityID, pgxmock.AnyArg(), "user-123").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.AddRelation(context.Background(), "user-123", rel)

		require.NoError(t, err)
		assert.NotEmpty(t, rel.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("swallows unique violation (idempotent)", func(t *testing.T) {
		repo, mock := newTagRepo(t)
		rel := &model.TagRelation{TagID: "tag-1", EntityType: "job", EntityID: "job-1"}

		mock.ExpectExec("INSERT INTO tag_relations").
			WithArgs(pgxmock.AnyArg(), rel.TagID, rel.EntityType, rel.EntityID, pgxmock.AnyArg(), "user-123").
			WillReturnError(&pgconn.PgError{Code: uniqueViolationCode})

		err := repo.AddRelation(context.Background(), "user-123", rel)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns entity not found when zero rows affected", func(t *testing.T) {
		repo, mock := newTagRepo(t)
		rel := &model.TagRelation{TagID: "tag-1", EntityType: "job", EntityID: "foreign-job"}

		mock.ExpectExec("INSERT INTO tag_relations").
			WithArgs(pgxmock.AnyArg(), rel.TagID, rel.EntityType, rel.EntityID, pgxmock.AnyArg(), "user-123").
			WillReturnResult(pgxmock.NewResult("INSERT", 0))

		err := repo.AddRelation(context.Background(), "user-123", rel)

		assert.ErrorIs(t, err, model.ErrEntityNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates other db error", func(t *testing.T) {
		repo, mock := newTagRepo(t)
		rel := &model.TagRelation{TagID: "tag-1", EntityType: "job", EntityID: "job-1"}

		mock.ExpectExec("INSERT INTO tag_relations").
			WithArgs(pgxmock.AnyArg(), rel.TagID, rel.EntityType, rel.EntityID, pgxmock.AnyArg(), "user-123").
			WillReturnError(errDB)

		err := repo.AddRelation(context.Background(), "user-123", rel)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTagRepository_RemoveRelation(t *testing.T) {
	t.Run("removes relation successfully", func(t *testing.T) {
		repo, mock := newTagRepo(t)

		mock.ExpectExec("DELETE FROM tag_relations").
			WithArgs("user-123", "tag-1", "job", "job-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.RemoveRelation(context.Background(), "user-123", "tag-1", "job", "job-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		repo, mock := newTagRepo(t)

		mock.ExpectExec("DELETE FROM tag_relations").
			WithArgs("user-123", "tag-1", "job", "job-1").
			WillReturnError(errDB)

		err := repo.RemoveRelation(context.Background(), "user-123", "tag-1", "job", "job-1")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTagRepository_ListByEntity(t *testing.T) {
	cols := []string{"id", "user_id", "name", "color", "created_at"}

	t.Run("returns entity tags", func(t *testing.T) {
		repo, mock := newTagRepo(t)
		now := time.Now()

		// Args order: entity_type ($1), entity_id ($2), user_id ($3).
		color := "#111"
		mock.ExpectQuery("INNER JOIN tag_relations").
			WithArgs("job", "job-1", "user-123").
			WillReturnRows(pgxmock.NewRows(cols).
				AddRow("tag-1", "user-123", "Backend", &color, now))

		tags, err := repo.ListByEntity(context.Background(), "user-123", "job", "job-1")

		require.NoError(t, err)
		require.Len(t, tags, 1)
		assert.Equal(t, "Backend", tags[0].Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates query error", func(t *testing.T) {
		repo, mock := newTagRepo(t)

		mock.ExpectQuery("INNER JOIN tag_relations").
			WithArgs("job", "job-1", "user-123").
			WillReturnError(errDB)

		tags, err := repo.ListByEntity(context.Background(), "user-123", "job", "job-1")

		assert.Nil(t, tags)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
