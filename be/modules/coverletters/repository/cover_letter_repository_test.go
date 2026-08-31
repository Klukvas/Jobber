package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleCoverLetter() *model.CoverLetter {
	rbID := "rb-1"
	jobID := "job-1"
	return &model.CoverLetter{
		UserID:          "user-1",
		ResumeBuilderID: &rbID,
		JobID:           &jobID,
		Title:           "My Letter",
		Template:        "modern",
		RecipientName:   "Jane",
		RecipientTitle:  "Manager",
		CompanyName:     "Acme",
		CompanyAddress:  "123 St",
		Greeting:        "Dear Jane,",
		Paragraphs:      []string{"p1", "p2"},
		Closing:         "Sincerely",
		FontFamily:      "Arial",
		FontSize:        12,
		PrimaryColor:    "#000",
	}
}

func TestCoverLetterRepository_Create(t *testing.T) {
	t.Run("creates cover letter and populates generated fields", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cl := sampleCoverLetter()
		now := time.Now()
		rows := pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("gen-id", now, now)

		mock.ExpectQuery("INSERT INTO cover_letters").
			WithArgs(
				cl.UserID, cl.ResumeBuilderID, cl.JobID, cl.Title, cl.Template,
				cl.RecipientName, cl.RecipientTitle, cl.CompanyName, cl.CompanyAddress,
				cl.Greeting, cl.Paragraphs, cl.Closing, cl.FontFamily, cl.FontSize, cl.PrimaryColor,
			).
			WillReturnRows(rows)

		repo := NewCoverLetterRepository(mock)
		got, err := repo.Create(context.Background(), cl)
		require.NoError(t, err)
		assert.Equal(t, "gen-id", got.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cl := sampleCoverLetter()
		mock.ExpectQuery("INSERT INTO cover_letters").
			WithArgs(
				cl.UserID, cl.ResumeBuilderID, cl.JobID, cl.Title, cl.Template,
				cl.RecipientName, cl.RecipientTitle, cl.CompanyName, cl.CompanyAddress,
				cl.Greeting, cl.Paragraphs, cl.Closing, cl.FontFamily, cl.FontSize, cl.PrimaryColor,
			).
			WillReturnError(errors.New("boom"))

		repo := NewCoverLetterRepository(mock)
		got, err := repo.Create(context.Background(), cl)
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create cover letter")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func coverLetterRowColumns() []string {
	return []string{
		"id", "user_id", "resume_builder_id", "job_id", "title", "template",
		"recipient_name", "recipient_title", "company_name", "company_address",
		"greeting", "paragraphs", "closing", "font_family", "font_size", "primary_color",
		"created_at", "updated_at",
	}
}

func TestCoverLetterRepository_GetByID(t *testing.T) {
	t.Run("returns cover letter", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rbID := "rb-1"
		jobID := "job-1"
		rows := pgxmock.NewRows(coverLetterRowColumns()).AddRow(
			"cl-1", "user-1", &rbID, &jobID, "My Letter", "modern",
			"Jane", "Manager", "Acme", "123 St",
			"Dear Jane,", []string{"p1"}, "Sincerely", "Arial", 12, "#000",
			now, now,
		)

		mock.ExpectQuery("SELECT id, user_id, resume_builder_id, job_id, title, template").
			WithArgs("cl-1", "user-1").
			WillReturnRows(rows)

		repo := NewCoverLetterRepository(mock)
		got, err := repo.GetByID(context.Background(), "user-1", "cl-1")
		require.NoError(t, err)
		assert.Equal(t, "cl-1", got.ID)
		assert.Equal(t, []string{"p1"}, got.Paragraphs)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrCoverLetterNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, resume_builder_id, job_id, title, template").
			WithArgs("cl-1", "user-1").
			WillReturnError(pgx.ErrNoRows)

		repo := NewCoverLetterRepository(mock)
		got, err := repo.GetByID(context.Background(), "user-1", "cl-1")
		assert.Nil(t, got)
		assert.ErrorIs(t, err, model.ErrCoverLetterNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, resume_builder_id, job_id, title, template").
			WithArgs("cl-1", "user-1").
			WillReturnError(errors.New("boom"))

		repo := NewCoverLetterRepository(mock)
		got, err := repo.GetByID(context.Background(), "user-1", "cl-1")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrCoverLetterNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCoverLetterRepository_List(t *testing.T) {
	t.Run("returns letters", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(coverLetterRowColumns()).
			AddRow("cl-1", "user-1", (*string)(nil), (*string)(nil), "L1", "modern",
				"", "", "", "", "", []string{"a"}, "", "Arial", 12, "#000", now, now).
			AddRow("cl-2", "user-1", (*string)(nil), (*string)(nil), "L2", "classic",
				"", "", "", "", "", []string{"b"}, "", "Arial", 11, "#111", now, now)

		mock.ExpectQuery("SELECT id, user_id, resume_builder_id, job_id, title, template").
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewCoverLetterRepository(mock)
		got, err := repo.List(context.Background(), "user-1")
		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, "cl-1", got[0].ID)
		assert.Equal(t, "cl-2", got[1].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("SELECT id, user_id, resume_builder_id, job_id, title, template").
			WithArgs("user-1").
			WillReturnError(errors.New("boom"))

		repo := NewCoverLetterRepository(mock)
		got, err := repo.List(context.Background(), "user-1")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to list cover letters")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps scan error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"id"}).AddRow("cl-1")
		mock.ExpectQuery("SELECT id, user_id, resume_builder_id, job_id, title, template").
			WithArgs("user-1").
			WillReturnRows(rows)

		repo := NewCoverLetterRepository(mock)
		got, err := repo.List(context.Background(), "user-1")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to scan cover letter")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCoverLetterRepository_Update(t *testing.T) {
	t.Run("updates and refreshes updated_at", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cl := sampleCoverLetter()
		cl.ID = "cl-1"
		now := time.Now()
		rows := pgxmock.NewRows([]string{"updated_at"}).AddRow(now)

		mock.ExpectQuery("UPDATE cover_letters SET").
			WithArgs(
				cl.Title, cl.ResumeBuilderID, cl.JobID, cl.Template, cl.RecipientName,
				cl.RecipientTitle, cl.CompanyName, cl.CompanyAddress,
				cl.Greeting, cl.Paragraphs, cl.Closing, cl.FontFamily, cl.FontSize, cl.PrimaryColor,
				cl.ID, cl.UserID,
			).
			WillReturnRows(rows)

		repo := NewCoverLetterRepository(mock)
		got, err := repo.Update(context.Background(), cl)
		require.NoError(t, err)
		assert.Equal(t, now, got.UpdatedAt)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrCoverLetterNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cl := sampleCoverLetter()
		cl.ID = "cl-1"
		mock.ExpectQuery("UPDATE cover_letters SET").
			WithArgs(
				cl.Title, cl.ResumeBuilderID, cl.JobID, cl.Template, cl.RecipientName,
				cl.RecipientTitle, cl.CompanyName, cl.CompanyAddress,
				cl.Greeting, cl.Paragraphs, cl.Closing, cl.FontFamily, cl.FontSize, cl.PrimaryColor,
				cl.ID, cl.UserID,
			).
			WillReturnError(pgx.ErrNoRows)

		repo := NewCoverLetterRepository(mock)
		got, err := repo.Update(context.Background(), cl)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, model.ErrCoverLetterNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cl := sampleCoverLetter()
		cl.ID = "cl-1"
		mock.ExpectQuery("UPDATE cover_letters SET").
			WithArgs(
				cl.Title, cl.ResumeBuilderID, cl.JobID, cl.Template, cl.RecipientName,
				cl.RecipientTitle, cl.CompanyName, cl.CompanyAddress,
				cl.Greeting, cl.Paragraphs, cl.Closing, cl.FontFamily, cl.FontSize, cl.PrimaryColor,
				cl.ID, cl.UserID,
			).
			WillReturnError(errors.New("boom"))

		repo := NewCoverLetterRepository(mock)
		got, err := repo.Update(context.Background(), cl)
		assert.Nil(t, got)
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrCoverLetterNotFound)
		assert.Contains(t, err.Error(), "failed to update cover letter")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCoverLetterRepository_Delete(t *testing.T) {
	t.Run("deletes cover letter", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM cover_letters").
			WithArgs("cl-1", "user-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := NewCoverLetterRepository(mock)
		require.NoError(t, repo.Delete(context.Background(), "user-1", "cl-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrCoverLetterNotFound when no rows affected", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM cover_letters").
			WithArgs("cl-1", "user-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := NewCoverLetterRepository(mock)
		err = repo.Delete(context.Background(), "user-1", "cl-1")
		assert.ErrorIs(t, err, model.ErrCoverLetterNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("wraps db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM cover_letters").
			WithArgs("cl-1", "user-1").
			WillReturnError(errors.New("boom"))

		repo := NewCoverLetterRepository(mock)
		err = repo.Delete(context.Background(), "user-1", "cl-1")
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrCoverLetterNotFound)
		assert.Contains(t, err.Error(), "failed to delete cover letter")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
