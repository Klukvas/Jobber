package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Mocks for ImportService ---

type MockResumeTextParser struct {
	ParseResumeTextFunc func(ctx context.Context, text string) (*ai.ParsedResume, error)
}

func (m *MockResumeTextParser) ParseResumeText(ctx context.Context, text string) (*ai.ParsedResume, error) {
	if m.ParseResumeTextFunc != nil {
		return m.ParseResumeTextFunc(ctx, text)
	}
	return &ai.ParsedResume{}, nil
}

// --- Helpers ---

func newImportService(
	repo *MockResumeBuilderRepository,
	parser *MockResumeTextParser,
	limitChecker *MockLimitChecker,
	pdfExtractor PDFTextExtractor,
) *ImportService {
	if pdfExtractor == nil {
		pdfExtractor = func(_ []byte) (string, error) {
			return "extracted pdf text", nil
		}
	}
	return NewImportServiceWithDeps(repo, parser, limitChecker, pdfExtractor)
}

func sampleParsedResume() *ai.ParsedResume {
	return &ai.ParsedResume{
		FullName: "John Doe",
		Email:    "john@example.com",
		Phone:    "+1-555-0100",
		Location: "San Francisco, CA",
		Website:  "https://johndoe.dev",
		LinkedIn: "linkedin.com/in/johndoe",
		GitHub:   "github.com/johndoe",
		Summary:  "Experienced software engineer with 10 years of expertise.",
		Experiences: []ai.ParsedExperience{
			{
				Company:     "Acme Corp",
				Position:    "Senior Engineer",
				Location:    "SF",
				StartDate:   "2020-01",
				EndDate:     "",
				IsCurrent:   true,
				Description: "Led backend development",
			},
		},
		Educations: []ai.ParsedEducation{
			{
				Institution:  "MIT",
				Degree:       "BS",
				FieldOfStudy: "Computer Science",
				StartDate:    "2010-09",
				EndDate:      "2014-06",
				GPA:          "3.9",
			},
		},
		Skills: []ai.ParsedSkill{
			{Name: "Go", Level: "expert"},
			{Name: "Python", Level: "advanced"},
		},
		Languages: []ai.ParsedLanguage{
			{Name: "English", Proficiency: "native"},
		},
		Certifications: []ai.ParsedCertification{
			{Name: "AWS SA", Issuer: "Amazon", IssueDate: "2023-01"},
		},
	}
}

func sampleFullResumeDTO() *model.FullResumeDTO {
	return &model.FullResumeDTO{
		ResumeBuilderDTO: &model.ResumeBuilderDTO{
			ID:           "rb-new",
			Title:        "John Doe",
			TemplateID:   "00000000-0000-0000-0000-000000000001",
			FontFamily:   "Georgia",
			PrimaryColor: "#2563eb",
			Spacing:      100,
			MarginTop:    40,
			MarginBottom: 40,
			MarginLeft:   40,
			MarginRight:  40,
			LayoutMode:   "single",
			SidebarWidth: 35,
		},
		Contact: &model.ContactDTO{
			FullName: "John Doe",
			Email:    "john@example.com",
		},
		Summary: &model.SummaryDTO{
			Content: "Experienced software engineer with 10 years of expertise.",
		},
	}
}

// --- ImportFromText Tests ---

func TestImportFromText_Success(t *testing.T) {
	ctx := context.Background()
	parsed := sampleParsedResume()
	expectedDTO := sampleFullResumeDTO()

	var createdContact *model.Contact
	var createdSummary *model.Summary
	var experienceCount, educationCount, skillCount, languageCount, certCount int

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		UpsertSectionOrderFunc: func(_ context.Context, _ string, orders []*model.SectionOrder) error {
			assert.Len(t, orders, 10)
			return nil
		},
		UpsertContactFunc: func(_ context.Context, c *model.Contact) error {
			createdContact = c
			return nil
		},
		UpsertSummaryFunc: func(_ context.Context, s *model.Summary) error {
			createdSummary = s
			return nil
		},
		CreateExperienceFunc: func(_ context.Context, _ *model.Experience) error {
			experienceCount++
			return nil
		},
		CreateEducationFunc: func(_ context.Context, _ *model.Education) error {
			educationCount++
			return nil
		},
		CreateSkillFunc: func(_ context.Context, _ *model.Skill) error {
			skillCount++
			return nil
		},
		CreateLanguageFunc: func(_ context.Context, _ *model.Language) error {
			languageCount++
			return nil
		},
		CreateCertificationFunc: func(_ context.Context, _ *model.Certification) error {
			certCount++
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, id string) (*model.FullResumeDTO, error) {
			assert.Equal(t, "rb-new", id)
			return expectedDTO, nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, text string) (*ai.ParsedResume, error) {
			assert.Equal(t, "resume text content", text)
			return parsed, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	result, err := svc.ImportFromText(ctx, "user-1", "resume text content", "")
	require.NoError(t, err)
	assert.Equal(t, expectedDTO, result)

	// Verify contact was created with parsed data
	require.NotNil(t, createdContact)
	assert.Equal(t, "John Doe", createdContact.FullName)
	assert.Equal(t, "john@example.com", createdContact.Email)
	assert.Equal(t, "rb-new", createdContact.ResumeBuilderID)

	// Verify summary was created
	require.NotNil(t, createdSummary)
	assert.Equal(t, "Experienced software engineer with 10 years of expertise.", createdSummary.Content)

	// Verify all sections were created
	assert.Equal(t, 1, experienceCount)
	assert.Equal(t, 1, educationCount)
	assert.Equal(t, 2, skillCount)
	assert.Equal(t, 1, languageCount)
	assert.Equal(t, 1, certCount)
}

func TestImportFromText_UsesCustomTitle(t *testing.T) {
	ctx := context.Background()
	parsed := sampleParsedResume()

	var createdTitle string
	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			createdTitle = rb.Title
			rb.ID = "rb-new"
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return sampleFullResumeDTO(), nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return parsed, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	_, err := svc.ImportFromText(ctx, "user-1", "text", "  My Custom Title  ")
	require.NoError(t, err)
	assert.Equal(t, "My Custom Title", createdTitle)
}

func TestImportFromText_UsesFullNameAsTitleWhenEmpty(t *testing.T) {
	ctx := context.Background()

	var createdTitle string
	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			createdTitle = rb.Title
			rb.ID = "rb-new"
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return sampleFullResumeDTO(), nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{FullName: "Jane Smith"}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	_, err := svc.ImportFromText(ctx, "user-1", "text", "")
	require.NoError(t, err)
	assert.Equal(t, "Jane Smith", createdTitle)
}

func TestImportFromText_DefaultTitleWhenNoNameAndNoTitle(t *testing.T) {
	ctx := context.Background()

	var createdTitle string
	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			createdTitle = rb.Title
			rb.ID = "rb-new"
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return sampleFullResumeDTO(), nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{FullName: ""}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	_, err := svc.ImportFromText(ctx, "user-1", "text", "")
	require.NoError(t, err)
	assert.Equal(t, "Imported Resume", createdTitle)
}

func TestImportFromText_LimitExceeded(t *testing.T) {
	ctx := context.Background()

	svc := newImportService(
		&MockResumeBuilderRepository{},
		&MockResumeTextParser{},
		&MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		},
		nil,
	)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, subModel.ErrLimitReached)
}

func TestImportFromText_NilLimitChecker(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return sampleFullResumeDTO(), nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{}, nil
		},
	}

	svc := NewImportServiceWithDeps(repo, parser, nil, func(_ []byte) (string, error) {
		return "", nil
	})

	result, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestImportFromText_AIParsingFailure(t *testing.T) {
	ctx := context.Background()

	svc := newImportService(
		&MockResumeBuilderRepository{},
		&MockResumeTextParser{
			ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
				return nil, errors.New("anthropic API call failed")
			},
		},
		&MockLimitChecker{},
		nil,
	)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse resume text")
}

func TestImportFromText_RepoCreateFailure(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, _ *model.ResumeBuilder) error {
			return errors.New("db connection error")
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{FullName: "Test"}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create resume")
}

func TestImportFromText_SectionOrderFailure(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		UpsertSectionOrderFunc: func(_ context.Context, _ string, _ []*model.SectionOrder) error {
			return errors.New("section order error")
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{FullName: "Test"}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to seed section order")
}

func TestImportFromText_ContactUpsertFailure(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		UpsertContactFunc: func(_ context.Context, _ *model.Contact) error {
			return errors.New("contact error")
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{FullName: "Test"}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create contact")
}

func TestImportFromText_SummaryUpsertFailure(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		UpsertSummaryFunc: func(_ context.Context, _ *model.Summary) error {
			return errors.New("summary error")
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{Summary: "A summary"}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create summary")
}

func TestImportFromText_SkipsSummaryWhenEmpty(t *testing.T) {
	ctx := context.Background()

	summaryUpserted := false
	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		UpsertSummaryFunc: func(_ context.Context, _ *model.Summary) error {
			summaryUpserted = true
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return sampleFullResumeDTO(), nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{FullName: "Test", Summary: ""}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	_, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	require.NoError(t, err)
	assert.False(t, summaryUpserted, "should not upsert summary when empty")
}

func TestImportFromText_ExperienceCreateFailure(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		CreateExperienceFunc: func(_ context.Context, _ *model.Experience) error {
			return errors.New("experience error")
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{
				Experiences: []ai.ParsedExperience{{Company: "Acme"}},
			}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add experience")
}

func TestImportFromText_EducationCreateFailure(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		CreateEducationFunc: func(_ context.Context, _ *model.Education) error {
			return errors.New("education error")
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{
				Educations: []ai.ParsedEducation{{Institution: "MIT"}},
			}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add education")
}

func TestImportFromText_SkillCreateFailure(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		CreateSkillFunc: func(_ context.Context, _ *model.Skill) error {
			return errors.New("skill error")
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{
				Skills: []ai.ParsedSkill{{Name: "Go"}},
			}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add skill")
}

func TestImportFromText_LanguageCreateFailure(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		CreateLanguageFunc: func(_ context.Context, _ *model.Language) error {
			return errors.New("language error")
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{
				Languages: []ai.ParsedLanguage{{Name: "English"}},
			}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add language")
}

func TestImportFromText_CertificationCreateFailure(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		CreateCertificationFunc: func(_ context.Context, _ *model.Certification) error {
			return errors.New("cert error")
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{
				Certifications: []ai.ParsedCertification{{Name: "AWS"}},
			}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add certification")
}

func TestImportFromText_GetFullResumeFailure(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return nil, errors.New("get full resume error")
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{FullName: "Test"}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	result, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get full resume error")
}

func TestImportFromText_SetsDefaultResumeProperties(t *testing.T) {
	ctx := context.Background()

	var createdRB *model.ResumeBuilder
	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			createdRB = rb
			rb.ID = "rb-new"
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return sampleFullResumeDTO(), nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{FullName: "Test"}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	_, err := svc.ImportFromText(ctx, "user-1", "text", "My Resume")
	require.NoError(t, err)

	require.NotNil(t, createdRB)
	assert.Equal(t, "user-1", createdRB.UserID)
	assert.Equal(t, "My Resume", createdRB.Title)
	assert.Equal(t, "00000000-0000-0000-0000-000000000001", createdRB.TemplateID)
	assert.Equal(t, "Georgia", createdRB.FontFamily)
	assert.Equal(t, "#2563eb", createdRB.PrimaryColor)
	assert.Equal(t, 100, createdRB.Spacing)
	assert.Equal(t, 40, createdRB.MarginTop)
	assert.Equal(t, 40, createdRB.MarginBottom)
	assert.Equal(t, 40, createdRB.MarginLeft)
	assert.Equal(t, 40, createdRB.MarginRight)
	assert.Equal(t, "single", createdRB.LayoutMode)
	assert.Equal(t, 35, createdRB.SidebarWidth)
}

func TestImportFromText_MapsAllExperienceFields(t *testing.T) {
	ctx := context.Background()

	var createdExp *model.Experience
	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		CreateExperienceFunc: func(_ context.Context, exp *model.Experience) error {
			createdExp = exp
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return sampleFullResumeDTO(), nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{
				Experiences: []ai.ParsedExperience{
					{
						Company:     "Acme Corp",
						Position:    "Senior Engineer",
						Location:    "SF",
						StartDate:   "2020-01",
						EndDate:     "",
						IsCurrent:   true,
						Description: "Led backend development",
					},
				},
			}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	_, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	require.NoError(t, err)

	require.NotNil(t, createdExp)
	assert.Equal(t, "rb-new", createdExp.ResumeBuilderID)
	assert.Equal(t, "Acme Corp", createdExp.Company)
	assert.Equal(t, "Senior Engineer", createdExp.Position)
	assert.Equal(t, "SF", createdExp.Location)
	assert.Equal(t, "2020-01", createdExp.StartDate)
	assert.Equal(t, "", createdExp.EndDate)
	assert.True(t, createdExp.IsCurrent)
	assert.Equal(t, "Led backend development", createdExp.Description)
	assert.Equal(t, 0, createdExp.SortOrder)
}

func TestImportFromText_MapsAllEducationFields(t *testing.T) {
	ctx := context.Background()

	var createdEdu *model.Education
	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		CreateEducationFunc: func(_ context.Context, edu *model.Education) error {
			createdEdu = edu
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return sampleFullResumeDTO(), nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{
				Educations: []ai.ParsedEducation{
					{
						Institution:  "MIT",
						Degree:       "BS",
						FieldOfStudy: "Computer Science",
						StartDate:    "2010-09",
						EndDate:      "2014-06",
						GPA:          "3.9",
					},
				},
			}, nil
		},
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

	_, err := svc.ImportFromText(ctx, "user-1", "text", "Title")
	require.NoError(t, err)

	require.NotNil(t, createdEdu)
	assert.Equal(t, "rb-new", createdEdu.ResumeBuilderID)
	assert.Equal(t, "MIT", createdEdu.Institution)
	assert.Equal(t, "BS", createdEdu.Degree)
	assert.Equal(t, "Computer Science", createdEdu.FieldOfStudy)
	assert.Equal(t, "2010-09", createdEdu.StartDate)
	assert.Equal(t, "2014-06", createdEdu.EndDate)
	assert.Equal(t, "3.9", createdEdu.GPA)
	assert.Equal(t, 0, createdEdu.SortOrder)
}

// --- ImportFromPDF Tests ---

func TestImportFromPDF_Success(t *testing.T) {
	ctx := context.Background()
	parsed := sampleParsedResume()
	expectedDTO := sampleFullResumeDTO()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return expectedDTO, nil
		},
	}

	var receivedText string
	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, text string) (*ai.ParsedResume, error) {
			receivedText = text
			return parsed, nil
		},
	}

	pdfExtractor := func(pdfBytes []byte) (string, error) {
		assert.Equal(t, []byte("fake-pdf-bytes"), pdfBytes)
		return "extracted resume text from PDF", nil
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, pdfExtractor)

	result, err := svc.ImportFromPDF(ctx, "user-1", []byte("fake-pdf-bytes"), "PDF Resume")
	require.NoError(t, err)
	assert.Equal(t, expectedDTO, result)
	assert.Equal(t, "extracted resume text from PDF", receivedText)
}

func TestImportFromPDF_LimitExceeded(t *testing.T) {
	ctx := context.Background()

	svc := newImportService(
		&MockResumeBuilderRepository{},
		&MockResumeTextParser{},
		&MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		},
		nil,
	)

	result, err := svc.ImportFromPDF(ctx, "user-1", []byte("pdf"), "")
	assert.Nil(t, result)
	assert.ErrorIs(t, err, subModel.ErrLimitReached)
}

func TestImportFromPDF_PDFExtractionFailure(t *testing.T) {
	ctx := context.Background()

	pdfExtractor := func(_ []byte) (string, error) {
		return "", errors.New("corrupted PDF")
	}

	svc := newImportService(
		&MockResumeBuilderRepository{},
		&MockResumeTextParser{},
		&MockLimitChecker{},
		pdfExtractor,
	)

	result, err := svc.ImportFromPDF(ctx, "user-1", []byte("bad-pdf"), "")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to extract PDF text")
}

func TestImportFromPDF_AIParsingFailure(t *testing.T) {
	ctx := context.Background()

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return nil, errors.New("AI parsing failed")
		},
	}

	pdfExtractor := func(_ []byte) (string, error) {
		return "some extracted text", nil
	}

	svc := newImportService(
		&MockResumeBuilderRepository{},
		parser,
		&MockLimitChecker{},
		pdfExtractor,
	)

	result, err := svc.ImportFromPDF(ctx, "user-1", []byte("pdf"), "")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse resume text")
}

func TestImportFromPDF_RepoCreateFailure(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, _ *model.ResumeBuilder) error {
			return errors.New("db error")
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{FullName: "Test"}, nil
		},
	}

	pdfExtractor := func(_ []byte) (string, error) {
		return "text", nil
	}

	svc := newImportService(repo, parser, &MockLimitChecker{}, pdfExtractor)

	result, err := svc.ImportFromPDF(ctx, "user-1", []byte("pdf"), "Title")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create resume")
}

func TestImportFromPDF_NilLimitChecker(t *testing.T) {
	ctx := context.Background()

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return sampleFullResumeDTO(), nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{}, nil
		},
	}

	pdfExtractor := func(_ []byte) (string, error) {
		return "text", nil
	}

	svc := NewImportServiceWithDeps(repo, parser, nil, pdfExtractor)

	result, err := svc.ImportFromPDF(ctx, "user-1", []byte("pdf"), "Title")
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestImportFromPDF_UsesCustomTitle(t *testing.T) {
	ctx := context.Background()

	var createdTitle string
	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			createdTitle = rb.Title
			rb.ID = "rb-new"
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return sampleFullResumeDTO(), nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return sampleParsedResume(), nil
		},
	}

	pdfExtractor := func(_ []byte) (string, error) { return "text", nil }

	svc := newImportService(repo, parser, &MockLimitChecker{}, pdfExtractor)

	_, err := svc.ImportFromPDF(ctx, "user-1", []byte("pdf"), "  My PDF Resume  ")
	require.NoError(t, err)
	assert.Equal(t, "My PDF Resume", createdTitle)
}

func TestImportFromPDF_LimitCheckPassesCorrectResource(t *testing.T) {
	ctx := context.Background()

	var checkedResource string
	limitChecker := &MockLimitChecker{
		CheckLimitFunc: func(_ context.Context, _ string, resource string) error {
			checkedResource = resource
			return nil
		},
	}

	repo := &MockResumeBuilderRepository{
		CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
			rb.ID = "rb-new"
			return nil
		},
		GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
			return sampleFullResumeDTO(), nil
		},
	}

	parser := &MockResumeTextParser{
		ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
			return &ai.ParsedResume{}, nil
		},
	}

	pdfExtractor := func(_ []byte) (string, error) { return "text", nil }

	svc := newImportService(repo, parser, limitChecker, pdfExtractor)

	_, err := svc.ImportFromPDF(ctx, "user-1", []byte("pdf"), "Title")
	require.NoError(t, err)
	assert.Equal(t, "resume_builders", checkedResource)
}

// --- Table-driven tests for createFromParsed title logic ---

func TestCreateFromParsed_TitleResolution(t *testing.T) {
	tests := []struct {
		name          string
		inputTitle    string
		parsedName    string
		expectedTitle string
	}{
		{
			name:          "uses provided title",
			inputTitle:    "Custom Title",
			parsedName:    "John Doe",
			expectedTitle: "Custom Title",
		},
		{
			name:          "trims whitespace from title",
			inputTitle:    "  Trimmed  ",
			parsedName:    "John Doe",
			expectedTitle: "Trimmed",
		},
		{
			name:          "falls back to full name when title empty",
			inputTitle:    "",
			parsedName:    "Jane Smith",
			expectedTitle: "Jane Smith",
		},
		{
			name:          "falls back to default when both empty",
			inputTitle:    "",
			parsedName:    "",
			expectedTitle: "Imported Resume",
		},
		{
			name:          "whitespace-only title trims to empty string",
			inputTitle:    "   ",
			parsedName:    "John Doe",
			expectedTitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var createdTitle string
			repo := &MockResumeBuilderRepository{
				CreateFunc: func(_ context.Context, rb *model.ResumeBuilder) error {
					createdTitle = rb.Title
					rb.ID = "rb-new"
					return nil
				},
				GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
					return sampleFullResumeDTO(), nil
				},
			}

			parser := &MockResumeTextParser{
				ParseResumeTextFunc: func(_ context.Context, _ string) (*ai.ParsedResume, error) {
					return &ai.ParsedResume{FullName: tt.parsedName}, nil
				},
			}

			svc := newImportService(repo, parser, &MockLimitChecker{}, nil)

			_, err := svc.ImportFromText(ctx, "user-1", "text", tt.inputTitle)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedTitle, createdTitle)
		})
	}
}
