package repository

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/jobs/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStageTemplateRepo(t *testing.T) (*StageTemplateRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	t.Cleanup(mock.Close)
	return NewStageTemplateRepository(mock), mock
}

var stageTemplateCols = []string{"id", "user_id", "name", "order", "kind", "created_at", "updated_at"}

func TestStageTemplateRepository_Create(t *testing.T) {
	t.Run("creates template successfully", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)
		tpl := &model.StageTemplate{UserID: "user-123", Name: "Applied", Order: 0, Kind: model.StageKindApplied}

		mock.ExpectExec("INSERT INTO stage_templates").
			WithArgs(pgxmock.AnyArg(), tpl.UserID, tpl.Name, tpl.Order, string(tpl.Kind), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), tpl)

		require.NoError(t, err)
		assert.NotEmpty(t, tpl.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("defaults an unclassified column to other", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)
		tpl := &model.StageTemplate{UserID: "user-123", Name: "Follow-up", Order: 3}

		mock.ExpectExec("INSERT INTO stage_templates").
			WithArgs(pgxmock.AnyArg(), tpl.UserID, tpl.Name, tpl.Order, string(model.StageKindOther), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		err := repo.Create(context.Background(), tpl)

		require.NoError(t, err)
		assert.Equal(t, model.StageKindOther, tpl.Kind)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps unique violation to ErrStageTemplateNameExists", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)
		tpl := &model.StageTemplate{UserID: "user-123", Name: "Dup", Order: 0, Kind: model.StageKindOther}

		mock.ExpectExec("INSERT INTO stage_templates").
			WithArgs(pgxmock.AnyArg(), tpl.UserID, tpl.Name, tpl.Order, string(tpl.Kind), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(&pgconn.PgError{Code: "23505"})

		err := repo.Create(context.Background(), tpl)

		assert.ErrorIs(t, err, model.ErrStageTemplateNameExists)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates other db error", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)
		tpl := &model.StageTemplate{UserID: "user-123", Name: "X", Order: 0, Kind: model.StageKindOther}

		mock.ExpectExec("INSERT INTO stage_templates").
			WithArgs(pgxmock.AnyArg(), tpl.UserID, tpl.Name, tpl.Order, string(tpl.Kind), pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errDB)

		err := repo.Create(context.Background(), tpl)

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStageTemplateRepository_GetByID(t *testing.T) {
	t.Run("returns template successfully", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)
		now := time.Now()

		mock.ExpectQuery("FROM stage_templates WHERE id = ").
			WithArgs("tpl-1", "user-123").
			WillReturnRows(pgxmock.NewRows(stageTemplateCols).AddRow(
				"tpl-1", "user-123", "Applied", 0, "applied", now, now,
			))

		tpl, err := repo.GetByID(context.Background(), "user-123", "tpl-1")

		require.NoError(t, err)
		assert.Equal(t, "Applied", tpl.Name)
		assert.Equal(t, model.StageKindApplied, tpl.Kind)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found on ErrNoRows", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectQuery("FROM stage_templates WHERE id = ").
			WithArgs("nope", "user-123").
			WillReturnError(pgx.ErrNoRows)

		tpl, err := repo.GetByID(context.Background(), "user-123", "nope")

		assert.Nil(t, tpl)
		assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectQuery("FROM stage_templates WHERE id = ").
			WithArgs("tpl-1", "user-123").
			WillReturnError(errDB)

		tpl, err := repo.GetByID(context.Background(), "user-123", "tpl-1")

		assert.Nil(t, tpl)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStageTemplateRepository_List(t *testing.T) {
	t.Run("returns templates with total", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)
		now := time.Now()

		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery("FROM stage_templates WHERE user_id = ").
			WithArgs("user-123", 20, 0).
			WillReturnRows(pgxmock.NewRows(stageTemplateCols).
				AddRow("tpl-1", "user-123", "Applied", 0, "applied", now, now).
				AddRow("tpl-2", "user-123", "Interview", 1, "interview", now, now))

		templates, total, err := repo.List(context.Background(), "user-123", 20, 0)

		require.NoError(t, err)
		require.Len(t, templates, 2)
		assert.Equal(t, 2, total)
		assert.Equal(t, "Interview", templates[1].Name)
		assert.Equal(t, model.StageKindInterview, templates[1].Kind)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates count error", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnError(errDB)

		templates, total, err := repo.List(context.Background(), "user-123", 20, 0)

		assert.Nil(t, templates)
		assert.Zero(t, total)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates list query error", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectQuery("FROM stage_templates WHERE user_id = ").
			WithArgs("user-123", 20, 0).
			WillReturnError(errDB)

		templates, total, err := repo.List(context.Background(), "user-123", 20, 0)

		assert.Nil(t, templates)
		assert.Zero(t, total)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates scan error", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)
		now := time.Now()

		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("FROM stage_templates WHERE user_id = ").
			WithArgs("user-123", 20, 0).
			WillReturnRows(pgxmock.NewRows(stageTemplateCols).
				AddRow("tpl-1", "user-123", "Applied", "bad-order", "applied", now, now))

		templates, total, err := repo.List(context.Background(), "user-123", 20, 0)

		assert.Nil(t, templates)
		assert.Zero(t, total)
		assert.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStageTemplateRepository_Update(t *testing.T) {
	// updatedRow is what the field-scoped UPDATE returns.
	updatedRow := func(name string, order int, kind string) *pgxmock.Rows {
		now := time.Now()
		return pgxmock.NewRows(stageTemplateCols).AddRow("tpl-1", "user-123", name, order, kind, now, now)
	}
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }
	kindPtr := func(k model.StageKind) *model.StageKind { return &k }

	t.Run("a rename writes the name and nothing else", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		// updated_at ($3) plus name ($4) — "order" and kind are not in the
		// statement, so a concurrent reorder or reclassification survives.
		mock.ExpectQuery(`UPDATE stage_templates SET updated_at = \$3, name = \$4\s`).
			WithArgs("tpl-1", "user-123", pgxmock.AnyArg(), "Onsite loop").
			WillReturnRows(updatedRow("Onsite loop", 3, "interview"))

		tpl, err := repo.Update(context.Background(), "user-123", "tpl-1",
			&model.StageTemplatePatch{Name: strPtr("Onsite loop")})

		require.NoError(t, err)
		require.NotNil(t, tpl)
		assert.Equal(t, "Onsite loop", tpl.Name)
		assert.Equal(t, model.StageKindInterview, tpl.Kind, "the stored kind comes back untouched")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("a reclassification writes the kind and nothing else", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectQuery(`UPDATE stage_templates SET updated_at = \$3, kind = \$4\s`).
			WithArgs("tpl-1", "user-123", pgxmock.AnyArg(), string(model.StageKindResponse)).
			WillReturnRows(updatedRow("Recruiter ping", 4, "response"))

		tpl, err := repo.Update(context.Background(), "user-123", "tpl-1",
			&model.StageTemplatePatch{Kind: kindPtr(model.StageKindResponse)})

		require.NoError(t, err)
		assert.Equal(t, model.StageKindResponse, tpl.Kind)
		assert.Equal(t, "Recruiter ping", tpl.Name, "the stored label comes back untouched")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("writes name, order and kind when all three are patched", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectQuery(`UPDATE stage_templates SET updated_at = \$3, name = \$4, "order" = \$5, kind = \$6`).
			WithArgs("tpl-1", "user-123", pgxmock.AnyArg(), "Renamed", 2, string(model.StageKindInterview)).
			WillReturnRows(updatedRow("Renamed", 2, "interview"))

		tpl, err := repo.Update(context.Background(), "user-123", "tpl-1", &model.StageTemplatePatch{
			Name:  strPtr("Renamed"),
			Order: intPtr(2),
			Kind:  kindPtr(model.StageKindInterview),
		})

		require.NoError(t, err)
		assert.Equal(t, "Renamed", tpl.Name)
		assert.Equal(t, 2, tpl.Order)
		assert.Equal(t, model.StageKindInterview, tpl.Kind)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("an empty patch bumps updated_at only", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectQuery(`UPDATE stage_templates SET updated_at = \$3\s`).
			WithArgs("tpl-1", "user-123", pgxmock.AnyArg()).
			WillReturnRows(updatedRow("Applied", 1, "applied"))

		tpl, err := repo.Update(context.Background(), "user-123", "tpl-1", &model.StageTemplatePatch{})

		require.NoError(t, err)
		assert.Equal(t, "Applied", tpl.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("defaults an unrecognized kind to other", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectQuery("UPDATE stage_templates SET").
			WithArgs("tpl-1", "user-123", pgxmock.AnyArg(), string(model.StageKindOther)).
			WillReturnRows(updatedRow("Applied", 1, "other"))

		tpl, err := repo.Update(context.Background(), "user-123", "tpl-1",
			&model.StageTemplatePatch{Kind: kindPtr(model.StageKind("ghosted"))})

		require.NoError(t, err)
		assert.Equal(t, model.StageKindOther, tpl.Kind)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps unique violation to ErrStageTemplateNameExists", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectQuery("UPDATE stage_templates SET").
			WithArgs("tpl-1", "user-123", pgxmock.AnyArg(), "Dup").
			WillReturnError(&pgconn.PgError{Code: "23505"})

		tpl, err := repo.Update(context.Background(), "user-123", "tpl-1",
			&model.StageTemplatePatch{Name: strPtr("Dup")})

		assert.Nil(t, tpl)
		assert.ErrorIs(t, err, model.ErrStageTemplateNameExists)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when no row matches", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectQuery("UPDATE stage_templates SET").
			WithArgs("nope", "user-123", pgxmock.AnyArg(), "N").
			WillReturnError(pgx.ErrNoRows)

		tpl, err := repo.Update(context.Background(), "user-123", "nope",
			&model.StageTemplatePatch{Name: strPtr("N")})

		assert.Nil(t, tpl)
		assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates other db error", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectQuery("UPDATE stage_templates SET").
			WithArgs("tpl-1", "user-123", pgxmock.AnyArg(), "N").
			WillReturnError(errDB)

		tpl, err := repo.Update(context.Background(), "user-123", "tpl-1",
			&model.StageTemplatePatch{Name: strPtr("N")})

		assert.Nil(t, tpl)
		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStageTemplateRepository_Reorder(t *testing.T) {
	t.Run("reorders successfully within a transaction", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)
		ids := []string{"tpl-1", "tpl-2"}

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectExec("UPDATE stage_templates").
			WithArgs("tpl-1", "user-123", 0, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec("UPDATE stage_templates").
			WithArgs("tpl-2", "user-123", 1, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectCommit()

		err := repo.Reorder(context.Background(), "user-123", ids)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns mismatch when count differs", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(3))
		mock.ExpectRollback()

		err := repo.Reorder(context.Background(), "user-123", []string{"tpl-1", "tpl-2"})

		assert.ErrorIs(t, err, model.ErrReorderMismatch)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns mismatch when an id is not the user's", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectExec("UPDATE stage_templates").
			WithArgs("tpl-1", "user-123", 0, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		mock.ExpectExec("UPDATE stage_templates").
			WithArgs("foreign", "user-123", 1, pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		mock.ExpectRollback()

		err := repo.Reorder(context.Background(), "user-123", []string{"tpl-1", "foreign"})

		assert.ErrorIs(t, err, model.ErrReorderMismatch)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps begin error", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectBegin().WillReturnError(errDB)

		err := repo.Reorder(context.Background(), "user-123", []string{"tpl-1"})

		require.Error(t, err)
		assert.ErrorIs(t, err, errDB)
		assert.Contains(t, err.Error(), "begin reorder tx")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates count error inside tx", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnError(errDB)
		mock.ExpectRollback()

		err := repo.Reorder(context.Background(), "user-123", []string{"tpl-1"})

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates update error inside tx", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectBegin()
		mock.ExpectQuery("SELECT COUNT").
			WithArgs("user-123").
			WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectExec("UPDATE stage_templates").
			WithArgs("tpl-1", "user-123", 0, pgxmock.AnyArg()).
			WillReturnError(errDB)
		mock.ExpectRollback()

		err := repo.Reorder(context.Background(), "user-123", []string{"tpl-1"})

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStageTemplateRepository_Delete(t *testing.T) {
	t.Run("deletes template successfully", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectExec("DELETE FROM stage_templates").
			WithArgs("tpl-1", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		err := repo.Delete(context.Background(), "user-123", "tpl-1")

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps foreign key violation to ErrStageTemplateInUse", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectExec("DELETE FROM stage_templates").
			WithArgs("tpl-1", "user-123").
			WillReturnError(&pgconn.PgError{Code: "23503"})

		err := repo.Delete(context.Background(), "user-123", "tpl-1")

		assert.ErrorIs(t, err, model.ErrStageTemplateInUse)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns not found when zero rows affected", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectExec("DELETE FROM stage_templates").
			WithArgs("nope", "user-123").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		err := repo.Delete(context.Background(), "user-123", "nope")

		assert.ErrorIs(t, err, model.ErrStageTemplateNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates other db error", func(t *testing.T) {
		repo, mock := newStageTemplateRepo(t)

		mock.ExpectExec("DELETE FROM stage_templates").
			WithArgs("tpl-1", "user-123").
			WillReturnError(errDB)

		err := repo.Delete(context.Background(), "user-123", "tpl-1")

		assert.ErrorIs(t, err, errDB)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
