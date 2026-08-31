package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pgxNoRows returns the sentinel pgx.ErrNoRows for mock error injection.
func pgxNoRows() error { return pgx.ErrNoRows }

// newMockRepo builds a ResumeBuilderRepository whose pool and querier are both
// backed by the pgxmock pool. Assigning the mock to q lets every q-based method
// run against the mock; assigning it to pool covers Begin/Acquire paths.
func newMockRepo(mock pgxmock.PgxPoolIface) *ResumeBuilderRepository {
	return &ResumeBuilderRepository{pool: mock, q: mock}
}

func resumeBuilderColumns() []string {
	return []string{
		"id", "user_id", "title", "template_id", "font_family", "primary_color", "text_color",
		"spacing", "margin_top", "margin_bottom", "margin_left", "margin_right",
		"layout_mode", "sidebar_width", "font_size", "skill_display", "created_at", "updated_at",
	}
}

func sampleResumeBuilder() *model.ResumeBuilder {
	return &model.ResumeBuilder{
		UserID: "user-1", Title: "My Resume", TemplateID: "modern",
		FontFamily: "Arial", PrimaryColor: "#000", TextColor: "#111",
		Spacing: 1, MarginTop: 10, MarginBottom: 10, MarginLeft: 10, MarginRight: 10,
		LayoutMode: "single", SidebarWidth: 30, FontSize: 12, SkillDisplay: "bar",
	}
}

func TestResumeBuilderRepository_Create(t *testing.T) {
	t.Run("inserts and assigns id + timestamps", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_builders").
			WithArgs(
				pgxmock.AnyArg(), "user-1", "My Resume", "modern", "Arial", "#000", "#111",
				1, 10, 10, 10, 10, "single", 30, 12, "bar",
				pgxmock.AnyArg(), pgxmock.AnyArg(),
			).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := newMockRepo(mock)
		rb := sampleResumeBuilder()
		require.NoError(t, repo.Create(context.Background(), rb))
		assert.NotEmpty(t, rb.ID)
		assert.False(t, rb.CreatedAt.IsZero())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_builders").
			WithArgs(
				pgxmock.AnyArg(), "user-1", "My Resume", "modern", "Arial", "#000", "#111",
				1, 10, 10, 10, 10, "single", 30, 12, "bar",
				pgxmock.AnyArg(), pgxmock.AnyArg(),
			).
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		assert.Error(t, repo.Create(context.Background(), sampleResumeBuilder()))
	})
}

func TestResumeBuilderRepository_GetByID(t *testing.T) {
	t.Run("returns resume builder", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(resumeBuilderColumns()).AddRow(
			"rb-1", "user-1", "My Resume", "modern", "Arial", "#000", "#111",
			1, 10, 10, 10, 10, "single", 30, 12, "bar", now, now,
		)
		mock.ExpectQuery("FROM resume_builders WHERE id").
			WithArgs("rb-1", "user-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		rb, err := repo.GetByID(context.Background(), "user-1", "rb-1")
		require.NoError(t, err)
		assert.Equal(t, "rb-1", rb.ID)
		assert.Equal(t, "My Resume", rb.Title)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrResumeBuilderNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("FROM resume_builders WHERE id").
			WithArgs("rb-1", "user-1").
			WillReturnError(pgxNoRows())

		repo := newMockRepo(mock)
		rb, err := repo.GetByID(context.Background(), "user-1", "rb-1")
		assert.Nil(t, rb)
		assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("FROM resume_builders WHERE id").
			WithArgs("rb-1", "user-1").
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		rb, err := repo.GetByID(context.Background(), "user-1", "rb-1")
		assert.Nil(t, rb)
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrResumeBuilderNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeBuilderRepository_List(t *testing.T) {
	t.Run("returns dtos", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		cols := []string{
			"id", "title", "template_id", "font_family", "primary_color", "text_color",
			"spacing", "margin_top", "margin_bottom", "margin_left", "margin_right",
			"layout_mode", "sidebar_width", "font_size", "skill_display", "created_at", "updated_at",
		}
		rows := pgxmock.NewRows(cols).
			AddRow("rb-1", "R1", "modern", "Arial", "#000", "#111", 1, 10, 10, 10, 10, "single", 30, 12, "bar", now, now).
			AddRow("rb-2", "R2", "classic", "Arial", "#000", "#111", 1, 10, 10, 10, 10, "single", 30, 12, "bar", now, now)

		mock.ExpectQuery("FROM resume_builders WHERE user_id").
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		items, err := repo.List(context.Background(), "user-1")
		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, "rb-1", items[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("FROM resume_builders WHERE user_id").
			WithArgs("user-1").
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		items, err := repo.List(context.Background(), "user-1")
		assert.Nil(t, items)
		require.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeBuilderRepository_Update(t *testing.T) {
	t.Run("updates and refreshes updated_at", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows([]string{"updated_at"}).AddRow(now)
		mock.ExpectQuery("UPDATE resume_builders").
			WithArgs(
				"My Resume", "modern", "Arial", "#000", "#111",
				1, 10, 10, 10, 10, "single", 30, 12, "bar", "rb-1", "user-1",
			).
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		rb := sampleResumeBuilder()
		rb.ID = "rb-1"
		require.NoError(t, repo.Update(context.Background(), rb))
		assert.Equal(t, now, rb.UpdatedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrResumeBuilderNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("UPDATE resume_builders").
			WithArgs(
				"My Resume", "modern", "Arial", "#000", "#111",
				1, 10, 10, 10, 10, "single", 30, 12, "bar", "rb-1", "user-1",
			).
			WillReturnError(pgxNoRows())

		repo := newMockRepo(mock)
		rb := sampleResumeBuilder()
		rb.ID = "rb-1"
		err = repo.Update(context.Background(), rb)
		assert.ErrorIs(t, err, model.ErrResumeBuilderNotFound)
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("UPDATE resume_builders").
			WithArgs(
				"My Resume", "modern", "Arial", "#000", "#111",
				1, 10, 10, 10, 10, "single", 30, 12, "bar", "rb-1", "user-1",
			).
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		rb := sampleResumeBuilder()
		rb.ID = "rb-1"
		err = repo.Update(context.Background(), rb)
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrResumeBuilderNotFound)
	})
}

func TestResumeBuilderRepository_Delete(t *testing.T) {
	t.Run("deletes", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_builders").
			WithArgs("rb-1", "user-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := newMockRepo(mock)
		require.NoError(t, repo.Delete(context.Background(), "user-1", "rb-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrResumeBuilderNotFound when no rows affected", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_builders").
			WithArgs("rb-1", "user-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := newMockRepo(mock)
		assert.ErrorIs(t, repo.Delete(context.Background(), "user-1", "rb-1"), model.ErrResumeBuilderNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_builders").
			WithArgs("rb-1", "user-1").
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		err = repo.Delete(context.Background(), "user-1", "rb-1")
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrResumeBuilderNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeBuilderRepository_VerifyOwnership(t *testing.T) {
	t.Run("passes when owner matches", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"user_id"}).AddRow("user-1")
		mock.ExpectQuery("SELECT user_id FROM resume_builders WHERE id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		require.NoError(t, repo.VerifyOwnership(context.Background(), "user-1", "rb-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrNotOwner when owner differs", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"user_id"}).AddRow("other-user")
		mock.ExpectQuery("SELECT user_id FROM resume_builders WHERE id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		assert.ErrorIs(t, repo.VerifyOwnership(context.Background(), "user-1", "rb-1"), model.ErrNotOwner)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrResumeBuilderNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT user_id FROM resume_builders WHERE id").
			WithArgs("rb-1").
			WillReturnError(pgxNoRows())

		repo := newMockRepo(mock)
		assert.ErrorIs(t, repo.VerifyOwnership(context.Background(), "user-1", "rb-1"), model.ErrResumeBuilderNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT user_id FROM resume_builders WHERE id").
			WithArgs("rb-1").
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		err = repo.VerifyOwnership(context.Background(), "user-1", "rb-1")
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrResumeBuilderNotFound)
		assert.NotErrorIs(t, err, model.ErrNotOwner)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeBuilderRepository_RunInTransaction(t *testing.T) {
	t.Run("commits when fn succeeds", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectBegin()
		mock.ExpectExec("DELETE FROM resume_builders").
			WithArgs("rb-1", "user-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))
		mock.ExpectCommit()

		repo := newMockRepo(mock)
		err = repo.RunInTransaction(context.Background(), func(txRepo ports.ResumeBuilderRepository) error {
			return txRepo.Delete(context.Background(), "user-1", "rb-1")
		})
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rolls back when fn returns error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectBegin()
		mock.ExpectRollback()

		repo := newMockRepo(mock)
		wantErr := errors.New("fn failed")
		err = repo.RunInTransaction(context.Background(), func(txRepo ports.ResumeBuilderRepository) error {
			return wantErr
		})
		assert.ErrorIs(t, err, wantErr)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns wrapped error when begin fails", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectBegin().WillReturnError(errors.New("cannot begin"))

		repo := newMockRepo(mock)
		err = repo.RunInTransaction(context.Background(), func(txRepo ports.ResumeBuilderRepository) error {
			return nil
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "begin transaction")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestResumeBuilderRepository_GetFullResume_AcquireError(t *testing.T) {
	// pgxmock's Acquire returns a "not implemented" error, so we can only
	// exercise the acquire-failure branch of GetFullResume here. The happy
	// path requires a real *pgxpool.Conn and is covered by integration tests.
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := newMockRepo(mock)
	dto, err := repo.GetFullResume(context.Background(), "user-1", "rb-1")
	assert.Nil(t, dto)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "acquire connection")
}
