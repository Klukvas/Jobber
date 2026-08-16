package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Contact (1:1) ---

func TestSectionRepository_UpsertContact(t *testing.T) {
	t.Run("assigns id when empty and upserts", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_contacts").
			WithArgs(pgxmock.AnyArg(), "rb-1", "Jane", "j@x.com", "555", "NY", "site", "li", "gh", pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := newMockRepo(mock)
		c := &model.Contact{
			ResumeBuilderID: "rb-1", FullName: "Jane", Email: "j@x.com", Phone: "555",
			Location: "NY", Website: "site", LinkedIn: "li", GitHub: "gh",
		}
		require.NoError(t, repo.UpsertContact(context.Background(), c))
		assert.NotEmpty(t, c.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_contacts").
			WithArgs("c-1", "rb-1", "", "", "", "", "", "", "", pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		assert.Error(t, repo.UpsertContact(context.Background(), &model.Contact{ID: "c-1", ResumeBuilderID: "rb-1"}))
	})
}

func TestSectionRepository_GetContact(t *testing.T) {
	cols := []string{"id", "resume_builder_id", "full_name", "email", "phone", "location", "website", "linkedin", "github", "created_at", "updated_at"}

	t.Run("returns contact", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(cols).AddRow("c-1", "rb-1", "Jane", "j@x.com", "555", "NY", "site", "li", "gh", now, now)
		mock.ExpectQuery("FROM resume_contacts WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		c, err := repo.GetContact(context.Background(), "rb-1")
		require.NoError(t, err)
		assert.Equal(t, "Jane", c.FullName)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrSectionEntryNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("FROM resume_contacts WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnError(pgxNoRows())

		repo := newMockRepo(mock)
		c, err := repo.GetContact(context.Background(), "rb-1")
		assert.Nil(t, c)
		assert.ErrorIs(t, err, model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("propagates generic db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("FROM resume_contacts WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		_, err = repo.GetContact(context.Background(), "rb-1")
		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// --- Summary (1:1) ---

func TestSectionRepository_UpsertSummary(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("INSERT INTO resume_summaries").
		WithArgs(pgxmock.AnyArg(), "rb-1", "content", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	repo := newMockRepo(mock)
	s := &model.Summary{ResumeBuilderID: "rb-1", Content: "content"}
	require.NoError(t, repo.UpsertSummary(context.Background(), s))
	assert.NotEmpty(t, s.ID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSectionRepository_GetSummary(t *testing.T) {
	cols := []string{"id", "resume_builder_id", "content", "created_at", "updated_at"}

	t.Run("returns summary", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(cols).AddRow("s-1", "rb-1", "content", now, now)
		mock.ExpectQuery("FROM resume_summaries WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		s, err := repo.GetSummary(context.Background(), "rb-1")
		require.NoError(t, err)
		assert.Equal(t, "content", s.Content)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("maps no rows to ErrSectionEntryNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("FROM resume_summaries WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnError(pgxNoRows())

		repo := newMockRepo(mock)
		_, err = repo.GetSummary(context.Background(), "rb-1")
		assert.ErrorIs(t, err, model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// --- Experiences (representative CRUD covering the generic list/get/update/delete shape) ---

func TestSectionRepository_Experience_CRUD(t *testing.T) {
	expCols := []string{"id", "resume_builder_id", "company", "position", "location", "start_date", "end_date", "is_current", "description", "sort_order", "created_at", "updated_at"}

	t.Run("Create inserts and assigns id/timestamps", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_experiences").
			WithArgs(pgxmock.AnyArg(), "rb-1", "Acme", "Dev", "NY", "2020", "2022", false, "did things", 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := newMockRepo(mock)
		exp := &model.Experience{
			ResumeBuilderID: "rb-1", Company: "Acme", Position: "Dev", Location: "NY",
			StartDate: "2020", EndDate: "2022", IsCurrent: false, Description: "did things", SortOrder: 0,
		}
		require.NoError(t, repo.CreateExperience(context.Background(), exp))
		assert.NotEmpty(t, exp.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update returns ErrSectionEntryNotFound when no rows affected", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE resume_experiences").
			WithArgs("Acme", "Dev", "NY", "2020", "2022", false, "d", 0, "e-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := newMockRepo(mock)
		exp := &model.Experience{ID: "e-1", ResumeBuilderID: "rb-1", Company: "Acme", Position: "Dev", Location: "NY", StartDate: "2020", EndDate: "2022", Description: "d"}
		assert.ErrorIs(t, repo.UpdateExperience(context.Background(), exp), model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update succeeds when a row is affected", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE resume_experiences").
			WithArgs("Acme", "Dev", "NY", "2020", "2022", false, "d", 0, "e-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		repo := newMockRepo(mock)
		exp := &model.Experience{ID: "e-1", ResumeBuilderID: "rb-1", Company: "Acme", Position: "Dev", Location: "NY", StartDate: "2020", EndDate: "2022", Description: "d"}
		require.NoError(t, repo.UpdateExperience(context.Background(), exp))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE resume_experiences").
			WithArgs("", "", "", "", "", false, "", 0, "e-1", "rb-1").
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		exp := &model.Experience{ID: "e-1", ResumeBuilderID: "rb-1"}
		assert.Error(t, repo.UpdateExperience(context.Background(), exp))
	})

	t.Run("Delete returns ErrSectionEntryNotFound when no rows affected", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_experiences").
			WithArgs("e-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		repo := newMockRepo(mock)
		assert.ErrorIs(t, repo.DeleteExperience(context.Background(), "rb-1", "e-1"), model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete succeeds", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_experiences").
			WithArgs("e-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := newMockRepo(mock)
		require.NoError(t, repo.DeleteExperience(context.Background(), "rb-1", "e-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete propagates db error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_experiences").
			WithArgs("e-1", "rb-1").
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		assert.Error(t, repo.DeleteExperience(context.Background(), "rb-1", "e-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("List returns items ordered", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(expCols).
			AddRow("e-1", "rb-1", "Acme", "Dev", "NY", "2020", "2022", false, "d", 0, now, now).
			AddRow("e-2", "rb-1", "Beta", "Lead", "SF", "2022", "", true, "d2", 1, now, now)
		mock.ExpectQuery("FROM resume_experiences WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		items, err := repo.ListExperiences(context.Background(), "rb-1")
		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, "e-1", items[0].ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("List propagates query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("FROM resume_experiences WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		items, err := repo.ListExperiences(context.Background(), "rb-1")
		assert.Nil(t, items)
		require.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetByID returns item", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(expCols).AddRow("e-1", "rb-1", "Acme", "Dev", "NY", "2020", "2022", false, "d", 0, now, now)
		mock.ExpectQuery("FROM resume_experiences WHERE id").
			WithArgs("e-1", "rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		e, err := repo.GetExperienceByID(context.Background(), "rb-1", "e-1")
		require.NoError(t, err)
		assert.Equal(t, "Acme", e.Company)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetByID maps no rows to ErrSectionEntryNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("FROM resume_experiences WHERE id").
			WithArgs("e-1", "rb-1").
			WillReturnError(pgxNoRows())

		repo := newMockRepo(mock)
		_, err = repo.GetExperienceByID(context.Background(), "rb-1", "e-1")
		assert.ErrorIs(t, err, model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// --- Education ---

func TestSectionRepository_Education(t *testing.T) {
	eduCols := []string{"id", "resume_builder_id", "institution", "degree", "field_of_study", "start_date", "end_date", "is_current", "gpa", "description", "sort_order", "created_at", "updated_at"}

	t.Run("Create", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_educations").
			WithArgs(pgxmock.AnyArg(), "rb-1", "MIT", "BSc", "CS", "2016", "2020", false, "4.0", "d", 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := newMockRepo(mock)
		edu := &model.Education{ResumeBuilderID: "rb-1", Institution: "MIT", Degree: "BSc", FieldOfStudy: "CS", StartDate: "2016", EndDate: "2020", GPA: "4.0", Description: "d"}
		require.NoError(t, repo.CreateEducation(context.Background(), edu))
		assert.NotEmpty(t, edu.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update no rows -> not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE resume_educations").
			WithArgs("MIT", "BSc", "CS", "2016", "2020", false, "4.0", "d", 0, "ed-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := newMockRepo(mock)
		edu := &model.Education{ID: "ed-1", ResumeBuilderID: "rb-1", Institution: "MIT", Degree: "BSc", FieldOfStudy: "CS", StartDate: "2016", EndDate: "2020", GPA: "4.0", Description: "d"}
		assert.ErrorIs(t, repo.UpdateEducation(context.Background(), edu), model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete succeeds", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_educations").
			WithArgs("ed-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := newMockRepo(mock)
		require.NoError(t, repo.DeleteEducation(context.Background(), "rb-1", "ed-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("List", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(eduCols).AddRow("ed-1", "rb-1", "MIT", "BSc", "CS", "2016", "2020", false, "4.0", "d", 0, now, now)
		mock.ExpectQuery("FROM resume_educations WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		items, err := repo.ListEducations(context.Background(), "rb-1")
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetByID", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(eduCols).AddRow("ed-1", "rb-1", "MIT", "BSc", "CS", "2016", "2020", false, "4.0", "d", 0, now, now)
		mock.ExpectQuery("FROM resume_educations WHERE id").
			WithArgs("ed-1", "rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		e, err := repo.GetEducationByID(context.Background(), "rb-1", "ed-1")
		require.NoError(t, err)
		assert.Equal(t, "MIT", e.Institution)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// --- Skills ---

func TestSectionRepository_Skill(t *testing.T) {
	skillCols := []string{"id", "resume_builder_id", "name", "level", "sort_order", "created_at", "updated_at"}

	t.Run("Create", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_skills").
			WithArgs(pgxmock.AnyArg(), "rb-1", "Go", "expert", 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := newMockRepo(mock)
		s := &model.Skill{ResumeBuilderID: "rb-1", Name: "Go", Level: "expert"}
		require.NoError(t, repo.CreateSkill(context.Background(), s))
		assert.NotEmpty(t, s.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update no rows -> not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE resume_skills").
			WithArgs("Go", "expert", 0, "s-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := newMockRepo(mock)
		s := &model.Skill{ID: "s-1", ResumeBuilderID: "rb-1", Name: "Go", Level: "expert"}
		assert.ErrorIs(t, repo.UpdateSkill(context.Background(), s), model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_skills").
			WithArgs("s-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := newMockRepo(mock)
		require.NoError(t, repo.DeleteSkill(context.Background(), "rb-1", "s-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("List", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(skillCols).AddRow("s-1", "rb-1", "Go", "expert", 0, now, now)
		mock.ExpectQuery("FROM resume_skills WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		items, err := repo.ListSkills(context.Background(), "rb-1")
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetByID", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(skillCols).AddRow("s-1", "rb-1", "Go", "expert", 0, now, now)
		mock.ExpectQuery("FROM resume_skills WHERE id").
			WithArgs("s-1", "rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		s, err := repo.GetSkillByID(context.Background(), "rb-1", "s-1")
		require.NoError(t, err)
		assert.Equal(t, "Go", s.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// --- Languages ---

func TestSectionRepository_Language(t *testing.T) {
	langCols := []string{"id", "resume_builder_id", "name", "proficiency", "sort_order", "created_at", "updated_at"}

	t.Run("Create", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_languages").
			WithArgs(pgxmock.AnyArg(), "rb-1", "English", "native", 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := newMockRepo(mock)
		l := &model.Language{ResumeBuilderID: "rb-1", Name: "English", Proficiency: "native"}
		require.NoError(t, repo.CreateLanguage(context.Background(), l))
		assert.NotEmpty(t, l.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update no rows -> not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE resume_languages").
			WithArgs("English", "native", 0, "l-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := newMockRepo(mock)
		l := &model.Language{ID: "l-1", ResumeBuilderID: "rb-1", Name: "English", Proficiency: "native"}
		assert.ErrorIs(t, repo.UpdateLanguage(context.Background(), l), model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_languages").
			WithArgs("l-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := newMockRepo(mock)
		require.NoError(t, repo.DeleteLanguage(context.Background(), "rb-1", "l-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("List", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(langCols).AddRow("l-1", "rb-1", "English", "native", 0, now, now)
		mock.ExpectQuery("FROM resume_languages WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		items, err := repo.ListLanguages(context.Background(), "rb-1")
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetByID", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(langCols).AddRow("l-1", "rb-1", "English", "native", 0, now, now)
		mock.ExpectQuery("FROM resume_languages WHERE id").
			WithArgs("l-1", "rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		l, err := repo.GetLanguageByID(context.Background(), "rb-1", "l-1")
		require.NoError(t, err)
		assert.Equal(t, "English", l.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// --- Certifications ---

func TestSectionRepository_Certification(t *testing.T) {
	certCols := []string{"id", "resume_builder_id", "name", "issuer", "issue_date", "expiry_date", "url", "sort_order", "created_at", "updated_at"}

	t.Run("Create", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_certifications").
			WithArgs(pgxmock.AnyArg(), "rb-1", "AWS", "Amazon", "2021", "2024", "url", 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := newMockRepo(mock)
		c := &model.Certification{ResumeBuilderID: "rb-1", Name: "AWS", Issuer: "Amazon", IssueDate: "2021", ExpiryDate: "2024", URL: "url"}
		require.NoError(t, repo.CreateCertification(context.Background(), c))
		assert.NotEmpty(t, c.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update no rows -> not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE resume_certifications").
			WithArgs("AWS", "Amazon", "2021", "2024", "url", 0, "c-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := newMockRepo(mock)
		c := &model.Certification{ID: "c-1", ResumeBuilderID: "rb-1", Name: "AWS", Issuer: "Amazon", IssueDate: "2021", ExpiryDate: "2024", URL: "url"}
		assert.ErrorIs(t, repo.UpdateCertification(context.Background(), c), model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_certifications").
			WithArgs("c-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := newMockRepo(mock)
		require.NoError(t, repo.DeleteCertification(context.Background(), "rb-1", "c-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("List", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(certCols).AddRow("c-1", "rb-1", "AWS", "Amazon", "2021", "2024", "url", 0, now, now)
		mock.ExpectQuery("FROM resume_certifications WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		items, err := repo.ListCertifications(context.Background(), "rb-1")
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetByID", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(certCols).AddRow("c-1", "rb-1", "AWS", "Amazon", "2021", "2024", "url", 0, now, now)
		mock.ExpectQuery("FROM resume_certifications WHERE id").
			WithArgs("c-1", "rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		c, err := repo.GetCertificationByID(context.Background(), "rb-1", "c-1")
		require.NoError(t, err)
		assert.Equal(t, "AWS", c.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// --- Projects ---

func TestSectionRepository_Project(t *testing.T) {
	projCols := []string{"id", "resume_builder_id", "name", "url", "start_date", "end_date", "description", "sort_order", "created_at", "updated_at"}

	t.Run("Create", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_projects").
			WithArgs(pgxmock.AnyArg(), "rb-1", "Proj", "url", "2021", "2022", "d", 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := newMockRepo(mock)
		p := &model.Project{ResumeBuilderID: "rb-1", Name: "Proj", URL: "url", StartDate: "2021", EndDate: "2022", Description: "d"}
		require.NoError(t, repo.CreateProject(context.Background(), p))
		assert.NotEmpty(t, p.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update no rows -> not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE resume_projects").
			WithArgs("Proj", "url", "2021", "2022", "d", 0, "p-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := newMockRepo(mock)
		p := &model.Project{ID: "p-1", ResumeBuilderID: "rb-1", Name: "Proj", URL: "url", StartDate: "2021", EndDate: "2022", Description: "d"}
		assert.ErrorIs(t, repo.UpdateProject(context.Background(), p), model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_projects").
			WithArgs("p-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := newMockRepo(mock)
		require.NoError(t, repo.DeleteProject(context.Background(), "rb-1", "p-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("List", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(projCols).AddRow("p-1", "rb-1", "Proj", "url", "2021", "2022", "d", 0, now, now)
		mock.ExpectQuery("FROM resume_projects WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		items, err := repo.ListProjects(context.Background(), "rb-1")
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetByID", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(projCols).AddRow("p-1", "rb-1", "Proj", "url", "2021", "2022", "d", 0, now, now)
		mock.ExpectQuery("FROM resume_projects WHERE id").
			WithArgs("p-1", "rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		p, err := repo.GetProjectByID(context.Background(), "rb-1", "p-1")
		require.NoError(t, err)
		assert.Equal(t, "Proj", p.Name)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// --- Volunteering ---

func TestSectionRepository_Volunteering(t *testing.T) {
	volCols := []string{"id", "resume_builder_id", "organization", "role", "start_date", "end_date", "description", "sort_order", "created_at", "updated_at"}

	t.Run("Create", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_volunteering").
			WithArgs(pgxmock.AnyArg(), "rb-1", "Org", "Helper", "2021", "2022", "d", 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := newMockRepo(mock)
		v := &model.Volunteering{ResumeBuilderID: "rb-1", Organization: "Org", Role: "Helper", StartDate: "2021", EndDate: "2022", Description: "d"}
		require.NoError(t, repo.CreateVolunteering(context.Background(), v))
		assert.NotEmpty(t, v.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update no rows -> not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE resume_volunteering").
			WithArgs("Org", "Helper", "2021", "2022", "d", 0, "v-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := newMockRepo(mock)
		v := &model.Volunteering{ID: "v-1", ResumeBuilderID: "rb-1", Organization: "Org", Role: "Helper", StartDate: "2021", EndDate: "2022", Description: "d"}
		assert.ErrorIs(t, repo.UpdateVolunteering(context.Background(), v), model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_volunteering").
			WithArgs("v-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := newMockRepo(mock)
		require.NoError(t, repo.DeleteVolunteering(context.Background(), "rb-1", "v-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("List", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(volCols).AddRow("v-1", "rb-1", "Org", "Helper", "2021", "2022", "d", 0, now, now)
		mock.ExpectQuery("FROM resume_volunteering WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		items, err := repo.ListVolunteering(context.Background(), "rb-1")
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetByID", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(volCols).AddRow("v-1", "rb-1", "Org", "Helper", "2021", "2022", "d", 0, now, now)
		mock.ExpectQuery("FROM resume_volunteering WHERE id").
			WithArgs("v-1", "rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		v, err := repo.GetVolunteeringByID(context.Background(), "rb-1", "v-1")
		require.NoError(t, err)
		assert.Equal(t, "Org", v.Organization)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// --- Custom Sections ---

func TestSectionRepository_CustomSection(t *testing.T) {
	csCols := []string{"id", "resume_builder_id", "title", "content", "sort_order", "created_at", "updated_at"}

	t.Run("Create", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_custom_sections").
			WithArgs(pgxmock.AnyArg(), "rb-1", "Awards", "body", 0, pgxmock.AnyArg(), pgxmock.AnyArg()).
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := newMockRepo(mock)
		cs := &model.CustomSection{ResumeBuilderID: "rb-1", Title: "Awards", Content: "body"}
		require.NoError(t, repo.CreateCustomSection(context.Background(), cs))
		assert.NotEmpty(t, cs.ID)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Update no rows -> not found", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("UPDATE resume_custom_sections").
			WithArgs("Awards", "body", 0, "cs-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		repo := newMockRepo(mock)
		cs := &model.CustomSection{ID: "cs-1", ResumeBuilderID: "rb-1", Title: "Awards", Content: "body"}
		assert.ErrorIs(t, repo.UpdateCustomSection(context.Background(), cs), model.ErrSectionEntryNotFound)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Delete", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("DELETE FROM resume_custom_sections").
			WithArgs("cs-1", "rb-1").
			WillReturnResult(pgxmock.NewResult("DELETE", 1))

		repo := newMockRepo(mock)
		require.NoError(t, repo.DeleteCustomSection(context.Background(), "rb-1", "cs-1"))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("List", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(csCols).AddRow("cs-1", "rb-1", "Awards", "body", 0, now, now)
		mock.ExpectQuery("FROM resume_custom_sections WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		items, err := repo.ListCustomSections(context.Background(), "rb-1")
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("GetByID", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		now := time.Now()
		rows := pgxmock.NewRows(csCols).AddRow("cs-1", "rb-1", "Awards", "body", 0, now, now)
		mock.ExpectQuery("FROM resume_custom_sections WHERE id").
			WithArgs("cs-1", "rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		cs, err := repo.GetCustomSectionByID(context.Background(), "rb-1", "cs-1")
		require.NoError(t, err)
		assert.Equal(t, "Awards", cs.Title)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

// --- Cross-section error branches ---
// These exercise the pgx.ErrNoRows -> ErrSectionEntryNotFound mapping and the
// generic-error propagation on the GetByID methods, plus the affected-rows and
// db-error branches on Delete, for the section types whose happy paths are
// already covered above.

func TestSectionRepository_GetByID_ErrorBranches(t *testing.T) {
	cases := []struct {
		name     string
		fragment string
		call     func(r *ResumeBuilderRepository) error
	}{
		{"Education", "FROM resume_educations WHERE id", func(r *ResumeBuilderRepository) error {
			_, err := r.GetEducationByID(context.Background(), "rb-1", "x")
			return err
		}},
		{"Skill", "FROM resume_skills WHERE id", func(r *ResumeBuilderRepository) error {
			_, err := r.GetSkillByID(context.Background(), "rb-1", "x")
			return err
		}},
		{"Language", "FROM resume_languages WHERE id", func(r *ResumeBuilderRepository) error {
			_, err := r.GetLanguageByID(context.Background(), "rb-1", "x")
			return err
		}},
		{"Certification", "FROM resume_certifications WHERE id", func(r *ResumeBuilderRepository) error {
			_, err := r.GetCertificationByID(context.Background(), "rb-1", "x")
			return err
		}},
		{"Project", "FROM resume_projects WHERE id", func(r *ResumeBuilderRepository) error {
			_, err := r.GetProjectByID(context.Background(), "rb-1", "x")
			return err
		}},
		{"Volunteering", "FROM resume_volunteering WHERE id", func(r *ResumeBuilderRepository) error {
			_, err := r.GetVolunteeringByID(context.Background(), "rb-1", "x")
			return err
		}},
		{"CustomSection", "FROM resume_custom_sections WHERE id", func(r *ResumeBuilderRepository) error {
			_, err := r.GetCustomSectionByID(context.Background(), "rb-1", "x")
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name+" maps no rows to ErrSectionEntryNotFound", func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			mock.ExpectQuery(tc.fragment).
				WithArgs("x", "rb-1").
				WillReturnError(pgxNoRows())

			repo := newMockRepo(mock)
			assert.ErrorIs(t, tc.call(repo), model.ErrSectionEntryNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})

		t.Run(tc.name+" propagates generic db error", func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			mock.ExpectQuery(tc.fragment).
				WithArgs("x", "rb-1").
				WillReturnError(errors.New("boom"))

			repo := newMockRepo(mock)
			err = tc.call(repo)
			require.Error(t, err)
			assert.NotErrorIs(t, err, model.ErrSectionEntryNotFound)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestSectionRepository_Delete_DBErrorBranches(t *testing.T) {
	cases := []struct {
		name     string
		fragment string
		call     func(r *ResumeBuilderRepository) error
	}{
		{"Education", "DELETE FROM resume_educations", func(r *ResumeBuilderRepository) error {
			return r.DeleteEducation(context.Background(), "rb-1", "x")
		}},
		{"Skill", "DELETE FROM resume_skills", func(r *ResumeBuilderRepository) error {
			return r.DeleteSkill(context.Background(), "rb-1", "x")
		}},
		{"Language", "DELETE FROM resume_languages", func(r *ResumeBuilderRepository) error {
			return r.DeleteLanguage(context.Background(), "rb-1", "x")
		}},
		{"Certification", "DELETE FROM resume_certifications", func(r *ResumeBuilderRepository) error {
			return r.DeleteCertification(context.Background(), "rb-1", "x")
		}},
		{"Project", "DELETE FROM resume_projects", func(r *ResumeBuilderRepository) error {
			return r.DeleteProject(context.Background(), "rb-1", "x")
		}},
		{"Volunteering", "DELETE FROM resume_volunteering", func(r *ResumeBuilderRepository) error {
			return r.DeleteVolunteering(context.Background(), "rb-1", "x")
		}},
		{"CustomSection", "DELETE FROM resume_custom_sections", func(r *ResumeBuilderRepository) error {
			return r.DeleteCustomSection(context.Background(), "rb-1", "x")
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			require.NoError(t, err)
			defer mock.Close()

			mock.ExpectExec(tc.fragment).
				WithArgs("x", "rb-1").
				WillReturnError(errors.New("boom"))

			repo := newMockRepo(mock)
			assert.Error(t, tc.call(repo))
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// --- Section Order ---

func TestSectionRepository_SectionOrder(t *testing.T) {
	t.Run("Upsert defaults empty column to main and iterates", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		// First order has an explicit column, second defaults to "main".
		mock.ExpectExec("INSERT INTO resume_section_orders").
			WithArgs("rb-1", "experience", 0, true, "sidebar").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))
		mock.ExpectExec("INSERT INTO resume_section_orders").
			WithArgs("rb-1", "education", 1, false, "main").
			WillReturnResult(pgxmock.NewResult("INSERT", 1))

		repo := newMockRepo(mock)
		orders := []*model.SectionOrder{
			{SectionKey: "experience", SortOrder: 0, IsVisible: true, Column: "sidebar"},
			{SectionKey: "education", SortOrder: 1, IsVisible: false, Column: ""},
		}
		require.NoError(t, repo.UpsertSectionOrder(context.Background(), "rb-1", orders))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Upsert propagates db error mid-loop", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectExec("INSERT INTO resume_section_orders").
			WithArgs("rb-1", "experience", 0, true, "main").
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		orders := []*model.SectionOrder{{SectionKey: "experience", SortOrder: 0, IsVisible: true}}
		assert.Error(t, repo.UpsertSectionOrder(context.Background(), "rb-1", orders))
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("List returns orders", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		cols := []string{"id", "resume_builder_id", "section_key", "sort_order", "is_visible", "column_placement"}
		rows := pgxmock.NewRows(cols).
			AddRow("o-1", "rb-1", "experience", 0, true, "main").
			AddRow("o-2", "rb-1", "education", 1, true, "sidebar")
		mock.ExpectQuery("FROM resume_section_orders WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnRows(rows)

		repo := newMockRepo(mock)
		items, err := repo.ListSectionOrders(context.Background(), "rb-1")
		require.NoError(t, err)
		require.Len(t, items, 2)
		assert.Equal(t, "experience", items[0].SectionKey)
		assert.Equal(t, "sidebar", items[1].Column)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("List propagates query error", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		require.NoError(t, err)
		defer mock.Close()

		mock.ExpectQuery("FROM resume_section_orders WHERE resume_builder_id").
			WithArgs("rb-1").
			WillReturnError(errors.New("boom"))

		repo := newMockRepo(mock)
		items, err := repo.ListSectionOrders(context.Background(), "rb-1")
		assert.Nil(t, items)
		require.Error(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}
