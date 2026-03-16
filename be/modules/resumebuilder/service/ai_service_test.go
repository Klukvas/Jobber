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

// --- Mock AI Client ---

type MockResumeAIClient struct {
	SuggestBulletPointsFunc func(ctx context.Context, jobTitle, company, currentDescription string) (*ai.BulletSuggestions, error)
	SuggestSummaryFunc      func(ctx context.Context, name, jobTitle, experienceContext string) (string, error)
	ImproveTextFunc         func(ctx context.Context, text, instruction string) (string, error)
	AnalyzeATSFunc          func(ctx context.Context, resumeContent, locale string) (*ai.ATSCheckResult, error)
}

func (m *MockResumeAIClient) SuggestBulletPoints(ctx context.Context, jobTitle, company, currentDescription string) (*ai.BulletSuggestions, error) {
	if m.SuggestBulletPointsFunc != nil {
		return m.SuggestBulletPointsFunc(ctx, jobTitle, company, currentDescription)
	}
	return &ai.BulletSuggestions{}, nil
}

func (m *MockResumeAIClient) SuggestSummary(ctx context.Context, name, jobTitle, experienceContext string) (string, error) {
	if m.SuggestSummaryFunc != nil {
		return m.SuggestSummaryFunc(ctx, name, jobTitle, experienceContext)
	}
	return "", nil
}

func (m *MockResumeAIClient) ImproveText(ctx context.Context, text, instruction string) (string, error) {
	if m.ImproveTextFunc != nil {
		return m.ImproveTextFunc(ctx, text, instruction)
	}
	return "", nil
}

func (m *MockResumeAIClient) AnalyzeATS(ctx context.Context, resumeContent, locale string) (*ai.ATSCheckResult, error) {
	if m.AnalyzeATSFunc != nil {
		return m.AnalyzeATSFunc(ctx, resumeContent, locale)
	}
	return &ai.ATSCheckResult{}, nil
}

// --- Helpers ---

func newAIService(repo *MockResumeBuilderRepository, aiClient *MockResumeAIClient, limitChecker *MockLimitChecker) *AIService {
	return NewAIService(repo, aiClient, limitChecker)
}

func newFullResumeForAI() *model.FullResumeDTO {
	return &model.FullResumeDTO{
		ResumeBuilderDTO: &model.ResumeBuilderDTO{
			ID:    "rb-1",
			Title: "Test Resume",
		},
		Contact: &model.ContactDTO{
			FullName: "Jane Doe",
			Email:    "jane@example.com",
			Phone:    "+1234567890",
			Location: "San Francisco, CA",
		},
		Summary: &model.SummaryDTO{
			Content: "Experienced software engineer with 10 years of experience.",
		},
		Experiences: []*model.ExperienceDTO{
			{
				ID:          "exp-1",
				Company:     "Acme Corp",
				Position:    "Senior Engineer",
				StartDate:   "2020-01",
				Description: "Led backend team of 5 engineers.",
			},
			{
				ID:          "exp-2",
				Company:     "Startup Inc",
				Position:    "Software Engineer",
				StartDate:   "2018-01",
				Description: "Built microservices architecture.",
			},
		},
		Educations: []*model.EducationDTO{
			{
				ID:           "edu-1",
				Degree:       "B.S.",
				FieldOfStudy: "Computer Science",
				Institution:  "MIT",
			},
		},
		Skills: []*model.SkillDTO{
			{ID: "sk-1", Name: "Go"},
			{ID: "sk-2", Name: "TypeScript"},
		},
		Languages: []*model.LanguageDTO{
			{ID: "lang-1", Name: "English", Proficiency: "native"},
		},
		Certifications: []*model.CertificationDTO{
			{ID: "cert-1", Name: "AWS Solutions Architect", Issuer: "Amazon"},
		},
		Projects: []*model.ProjectDTO{
			{ID: "proj-1", Name: "OpenSource Tool", Description: "CLI for automated deployments"},
		},
	}
}

// --- SuggestBulletPoints Tests ---

func TestSuggestBulletPoints(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"

	t.Run("returns bullet suggestions on success", func(t *testing.T) {
		expected := &ai.BulletSuggestions{
			Bullets: []string{
				"Spearheaded migration to microservices, reducing latency by 40%",
				"Mentored 3 junior developers, improving team velocity by 25%",
			},
		}

		aiClient := &MockResumeAIClient{
			SuggestBulletPointsFunc: func(_ context.Context, jobTitle, company, desc string) (*ai.BulletSuggestions, error) {
				assert.Equal(t, "Senior Engineer", jobTitle)
				assert.Equal(t, "Acme Corp", company)
				assert.Equal(t, "Led a team", desc)
				return expected, nil
			},
		}

		svc := newAIService(&MockResumeBuilderRepository{}, aiClient, &MockLimitChecker{})
		result, err := svc.SuggestBulletPoints(ctx, userID, "Senior Engineer", "Acme Corp", "Led a team")

		require.NoError(t, err)
		assert.Equal(t, expected, result)
	})

	t.Run("returns error when limit exceeded", func(t *testing.T) {
		limitChecker := &MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		svc := newAIService(&MockResumeBuilderRepository{}, &MockResumeAIClient{}, limitChecker)
		result, err := svc.SuggestBulletPoints(ctx, userID, "Engineer", "Corp", "desc")

		assert.Nil(t, result)
		assert.ErrorIs(t, err, subModel.ErrLimitReached)
	})

	t.Run("returns error when AI client fails", func(t *testing.T) {
		aiErr := errors.New("anthropic API call failed")
		aiClient := &MockResumeAIClient{
			SuggestBulletPointsFunc: func(_ context.Context, _, _, _ string) (*ai.BulletSuggestions, error) {
				return nil, aiErr
			},
		}

		svc := newAIService(&MockResumeBuilderRepository{}, aiClient, &MockLimitChecker{})
		result, err := svc.SuggestBulletPoints(ctx, userID, "Engineer", "Corp", "desc")

		assert.Nil(t, result)
		assert.ErrorIs(t, err, aiErr)
	})

	t.Run("passes correct resource to limit checker", func(t *testing.T) {
		var capturedResource string
		limitChecker := &MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, resource string) error {
				capturedResource = resource
				return nil
			},
		}

		svc := newAIService(&MockResumeBuilderRepository{}, &MockResumeAIClient{}, limitChecker)
		_, _ = svc.SuggestBulletPoints(ctx, userID, "Engineer", "Corp", "desc")

		assert.Equal(t, "ai_requests", capturedResource)
	})
}

// --- SuggestSummary Tests ---

func TestSuggestSummary(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	resumeID := "rb-1"

	t.Run("returns summary on success with full resume data", func(t *testing.T) {
		fullResume := newFullResumeForAI()

		var capturedName, capturedJobTitle, capturedContext string
		aiClient := &MockResumeAIClient{
			SuggestSummaryFunc: func(_ context.Context, name, jobTitle, expContext string) (string, error) {
				capturedName = name
				capturedJobTitle = jobTitle
				capturedContext = expContext
				return "A seasoned software engineer with 10+ years of experience.", nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		result, err := svc.SuggestSummary(ctx, userID, resumeID)

		require.NoError(t, err)
		assert.Equal(t, "A seasoned software engineer with 10+ years of experience.", result)
		assert.Equal(t, "Jane Doe", capturedName)
		assert.Equal(t, "Senior Engineer", capturedJobTitle)
		assert.Contains(t, capturedContext, "Senior Engineer at Acme Corp")
		assert.Contains(t, capturedContext, "Led backend team of 5 engineers.")
		assert.Contains(t, capturedContext, "Software Engineer at Startup Inc")
	})

	t.Run("handles resume with no contact", func(t *testing.T) {
		fullResume := newFullResumeForAI()
		fullResume.Contact = nil

		var capturedName string
		aiClient := &MockResumeAIClient{
			SuggestSummaryFunc: func(_ context.Context, name, _, _ string) (string, error) {
				capturedName = name
				return "Summary", nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		_, err := svc.SuggestSummary(ctx, userID, resumeID)

		require.NoError(t, err)
		assert.Equal(t, "", capturedName)
	})

	t.Run("handles resume with no experiences", func(t *testing.T) {
		fullResume := newFullResumeForAI()
		fullResume.Experiences = nil

		var capturedJobTitle, capturedContext string
		aiClient := &MockResumeAIClient{
			SuggestSummaryFunc: func(_ context.Context, _, jobTitle, expContext string) (string, error) {
				capturedJobTitle = jobTitle
				capturedContext = expContext
				return "Summary", nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		_, err := svc.SuggestSummary(ctx, userID, resumeID)

		require.NoError(t, err)
		assert.Equal(t, "", capturedJobTitle)
		assert.Equal(t, "", capturedContext)
	})

	t.Run("handles experience with no description", func(t *testing.T) {
		fullResume := newFullResumeForAI()
		fullResume.Experiences = []*model.ExperienceDTO{
			{Position: "CTO", Company: "BigCo", Description: ""},
		}

		var capturedContext string
		aiClient := &MockResumeAIClient{
			SuggestSummaryFunc: func(_ context.Context, _, _, expContext string) (string, error) {
				capturedContext = expContext
				return "Summary", nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		_, err := svc.SuggestSummary(ctx, userID, resumeID)

		require.NoError(t, err)
		assert.Equal(t, "CTO at BigCo", capturedContext)
		assert.NotContains(t, capturedContext, ":")
	})

	t.Run("returns error when limit exceeded", func(t *testing.T) {
		limitChecker := &MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		svc := newAIService(&MockResumeBuilderRepository{}, &MockResumeAIClient{}, limitChecker)
		result, err := svc.SuggestSummary(ctx, userID, resumeID)

		assert.Equal(t, "", result)
		assert.ErrorIs(t, err, subModel.ErrLimitReached)
	})

	t.Run("returns error when ownership verification fails", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrNotOwner
			},
		}

		svc := newAIService(repo, &MockResumeAIClient{}, &MockLimitChecker{})
		result, err := svc.SuggestSummary(ctx, userID, resumeID)

		assert.Equal(t, "", result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("returns error when get full resume fails", func(t *testing.T) {
		repoErr := errors.New("db connection lost")
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return nil, repoErr
			},
		}

		svc := newAIService(repo, &MockResumeAIClient{}, &MockLimitChecker{})
		result, err := svc.SuggestSummary(ctx, userID, resumeID)

		assert.Equal(t, "", result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get resume")
		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("returns error when AI client fails", func(t *testing.T) {
		aiErr := errors.New("anthropic API call failed")
		aiClient := &MockResumeAIClient{
			SuggestSummaryFunc: func(_ context.Context, _, _, _ string) (string, error) {
				return "", aiErr
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return newFullResumeForAI(), nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		result, err := svc.SuggestSummary(ctx, userID, resumeID)

		assert.Equal(t, "", result)
		assert.ErrorIs(t, err, aiErr)
	})
}

// --- ImproveText Tests ---

func TestImproveText(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"

	t.Run("returns improved text on success", func(t *testing.T) {
		aiClient := &MockResumeAIClient{
			ImproveTextFunc: func(_ context.Context, text, instruction string) (string, error) {
				assert.Equal(t, "I did stuff at my job", text)
				assert.Equal(t, "Make it more professional", instruction)
				return "Delivered key business outcomes through strategic initiatives.", nil
			},
		}

		svc := newAIService(&MockResumeBuilderRepository{}, aiClient, &MockLimitChecker{})
		result, err := svc.ImproveText(ctx, userID, "I did stuff at my job", "Make it more professional")

		require.NoError(t, err)
		assert.Equal(t, "Delivered key business outcomes through strategic initiatives.", result)
	})

	t.Run("returns error when limit exceeded", func(t *testing.T) {
		limitChecker := &MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		svc := newAIService(&MockResumeBuilderRepository{}, &MockResumeAIClient{}, limitChecker)
		result, err := svc.ImproveText(ctx, userID, "text", "instruction")

		assert.Equal(t, "", result)
		assert.ErrorIs(t, err, subModel.ErrLimitReached)
	})

	t.Run("returns error when AI client fails", func(t *testing.T) {
		aiErr := errors.New("anthropic API call failed")
		aiClient := &MockResumeAIClient{
			ImproveTextFunc: func(_ context.Context, _, _ string) (string, error) {
				return "", aiErr
			},
		}

		svc := newAIService(&MockResumeBuilderRepository{}, aiClient, &MockLimitChecker{})
		result, err := svc.ImproveText(ctx, userID, "text", "instruction")

		assert.Equal(t, "", result)
		assert.ErrorIs(t, err, aiErr)
	})

	t.Run("passes correct resource to limit checker", func(t *testing.T) {
		var capturedResource string
		limitChecker := &MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, resource string) error {
				capturedResource = resource
				return nil
			},
		}

		svc := newAIService(&MockResumeBuilderRepository{}, &MockResumeAIClient{}, limitChecker)
		_, _ = svc.ImproveText(ctx, userID, "text", "instruction")

		assert.Equal(t, "ai_requests", capturedResource)
	})
}

// --- ATSCheck Tests ---

func TestATSCheck(t *testing.T) {
	ctx := context.Background()
	userID := "user-1"
	resumeID := "rb-1"

	t.Run("returns ATS result on success with full resume", func(t *testing.T) {
		fullResume := newFullResumeForAI()

		expected := &ai.ATSCheckResult{
			Score: 85,
			Issues: []ai.ATSIssue{
				{Severity: "warning", Description: "Missing keywords for target role"},
			},
			Suggestions: []string{"Add more quantified achievements"},
			Keywords:    []string{"Go", "TypeScript", "microservices"},
		}

		var capturedResumeText string
		aiClient := &MockResumeAIClient{
			AnalyzeATSFunc: func(_ context.Context, resumeContent, _ string) (*ai.ATSCheckResult, error) {
				capturedResumeText = resumeContent
				return expected, nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		result, err := svc.ATSCheck(ctx, userID, resumeID, "")

		require.NoError(t, err)
		assert.Equal(t, expected, result)

		// Verify resume text includes all sections
		assert.Contains(t, capturedResumeText, "Name: Jane Doe")
		assert.Contains(t, capturedResumeText, "Email: jane@example.com")
		assert.Contains(t, capturedResumeText, "Phone: +1234567890")
		assert.Contains(t, capturedResumeText, "Location: San Francisco, CA")
		assert.Contains(t, capturedResumeText, "--- SUMMARY ---")
		assert.Contains(t, capturedResumeText, "Experienced software engineer")
		assert.Contains(t, capturedResumeText, "--- EXPERIENCE ---")
		assert.Contains(t, capturedResumeText, "Senior Engineer at Acme Corp")
		assert.Contains(t, capturedResumeText, "--- EDUCATION ---")
		assert.Contains(t, capturedResumeText, "B.S., Computer Science - MIT")
		assert.Contains(t, capturedResumeText, "--- SKILLS ---")
		assert.Contains(t, capturedResumeText, "Go, TypeScript")
		assert.Contains(t, capturedResumeText, "--- LANGUAGES ---")
		assert.Contains(t, capturedResumeText, "English (native)")
		assert.Contains(t, capturedResumeText, "--- CERTIFICATIONS ---")
		assert.Contains(t, capturedResumeText, "AWS Solutions Architect - Amazon")
		assert.Contains(t, capturedResumeText, "--- PROJECTS ---")
		assert.Contains(t, capturedResumeText, "OpenSource Tool: CLI for automated deployments")
	})

	t.Run("handles resume with no contact", func(t *testing.T) {
		fullResume := newFullResumeForAI()
		fullResume.Contact = nil

		var capturedResumeText string
		aiClient := &MockResumeAIClient{
			AnalyzeATSFunc: func(_ context.Context, resumeContent, _ string) (*ai.ATSCheckResult, error) {
				capturedResumeText = resumeContent
				return &ai.ATSCheckResult{Score: 70}, nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		result, err := svc.ATSCheck(ctx, userID, resumeID, "")

		require.NoError(t, err)
		assert.Equal(t, 70, result.Score)
		assert.NotContains(t, capturedResumeText, "Name:")
	})

	t.Run("handles resume with no summary", func(t *testing.T) {
		fullResume := newFullResumeForAI()
		fullResume.Summary = nil

		var capturedResumeText string
		aiClient := &MockResumeAIClient{
			AnalyzeATSFunc: func(_ context.Context, resumeContent, _ string) (*ai.ATSCheckResult, error) {
				capturedResumeText = resumeContent
				return &ai.ATSCheckResult{Score: 60}, nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		_, err := svc.ATSCheck(ctx, userID, resumeID, "")

		require.NoError(t, err)
		assert.NotContains(t, capturedResumeText, "--- SUMMARY ---")
	})

	t.Run("handles resume with empty summary content", func(t *testing.T) {
		fullResume := newFullResumeForAI()
		fullResume.Summary = &model.SummaryDTO{Content: ""}

		var capturedResumeText string
		aiClient := &MockResumeAIClient{
			AnalyzeATSFunc: func(_ context.Context, resumeContent, _ string) (*ai.ATSCheckResult, error) {
				capturedResumeText = resumeContent
				return &ai.ATSCheckResult{Score: 60}, nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		_, err := svc.ATSCheck(ctx, userID, resumeID, "")

		require.NoError(t, err)
		assert.NotContains(t, capturedResumeText, "--- SUMMARY ---")
	})

	t.Run("handles resume with minimal data", func(t *testing.T) {
		fullResume := &model.FullResumeDTO{
			ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: "rb-1"},
		}

		var capturedResumeText string
		aiClient := &MockResumeAIClient{
			AnalyzeATSFunc: func(_ context.Context, resumeContent, _ string) (*ai.ATSCheckResult, error) {
				capturedResumeText = resumeContent
				return &ai.ATSCheckResult{Score: 20}, nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		result, err := svc.ATSCheck(ctx, userID, resumeID, "")

		require.NoError(t, err)
		assert.Equal(t, 20, result.Score)
		assert.Equal(t, "", capturedResumeText)
	})

	t.Run("includes experience description in text", func(t *testing.T) {
		fullResume := newFullResumeForAI()
		// Override with experience that has description
		fullResume.Experiences = []*model.ExperienceDTO{
			{Position: "Dev", Company: "Co", StartDate: "2022-01", Description: "Built APIs"},
		}
		// Remove other sections
		fullResume.Contact = nil
		fullResume.Summary = nil
		fullResume.Educations = nil
		fullResume.Skills = nil
		fullResume.Languages = nil
		fullResume.Certifications = nil
		fullResume.Projects = nil

		var capturedResumeText string
		aiClient := &MockResumeAIClient{
			AnalyzeATSFunc: func(_ context.Context, resumeContent, _ string) (*ai.ATSCheckResult, error) {
				capturedResumeText = resumeContent
				return &ai.ATSCheckResult{Score: 50}, nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		_, err := svc.ATSCheck(ctx, userID, resumeID, "")

		require.NoError(t, err)
		assert.Contains(t, capturedResumeText, "Dev at Co (2022-01)")
		assert.Contains(t, capturedResumeText, "Built APIs")
	})

	t.Run("handles project without description", func(t *testing.T) {
		fullResume := &model.FullResumeDTO{
			ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: "rb-1"},
			Projects: []*model.ProjectDTO{
				{Name: "MyProject", Description: ""},
			},
		}

		var capturedResumeText string
		aiClient := &MockResumeAIClient{
			AnalyzeATSFunc: func(_ context.Context, resumeContent, _ string) (*ai.ATSCheckResult, error) {
				capturedResumeText = resumeContent
				return &ai.ATSCheckResult{Score: 50}, nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		_, err := svc.ATSCheck(ctx, userID, resumeID, "")

		require.NoError(t, err)
		assert.Contains(t, capturedResumeText, "MyProject")
		assert.NotContains(t, capturedResumeText, "MyProject:")
	})

	t.Run("returns error when limit exceeded", func(t *testing.T) {
		limitChecker := &MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				return subModel.ErrLimitReached
			},
		}

		svc := newAIService(&MockResumeBuilderRepository{}, &MockResumeAIClient{}, limitChecker)
		result, err := svc.ATSCheck(ctx, userID, resumeID, "")

		assert.Nil(t, result)
		assert.ErrorIs(t, err, subModel.ErrLimitReached)
	})

	t.Run("returns error when ownership verification fails", func(t *testing.T) {
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				return model.ErrNotOwner
			},
		}

		svc := newAIService(repo, &MockResumeAIClient{}, &MockLimitChecker{})
		result, err := svc.ATSCheck(ctx, userID, resumeID, "")

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrNotOwner)
	})

	t.Run("returns error when get full resume fails", func(t *testing.T) {
		repoErr := errors.New("db connection lost")
		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return nil, repoErr
			},
		}

		svc := newAIService(repo, &MockResumeAIClient{}, &MockLimitChecker{})
		result, err := svc.ATSCheck(ctx, userID, resumeID, "")

		assert.Nil(t, result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get resume")
		assert.ErrorIs(t, err, repoErr)
	})

	t.Run("returns error when AI client fails", func(t *testing.T) {
		aiErr := errors.New("anthropic API call failed")
		aiClient := &MockResumeAIClient{
			AnalyzeATSFunc: func(_ context.Context, _, _ string) (*ai.ATSCheckResult, error) {
				return nil, aiErr
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return newFullResumeForAI(), nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		result, err := svc.ATSCheck(ctx, userID, resumeID, "")

		assert.Nil(t, result)
		assert.ErrorIs(t, err, aiErr)
	})

	t.Run("checks limit before ownership", func(t *testing.T) {
		callOrder := make([]string, 0, 2)

		limitChecker := &MockLimitChecker{
			CheckLimitFunc: func(_ context.Context, _, _ string) error {
				callOrder = append(callOrder, "limit")
				return subModel.ErrLimitReached
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error {
				callOrder = append(callOrder, "ownership")
				return nil
			},
		}

		svc := newAIService(repo, &MockResumeAIClient{}, limitChecker)
		_, _ = svc.ATSCheck(ctx, userID, resumeID, "")

		require.Len(t, callOrder, 1)
		assert.Equal(t, "limit", callOrder[0])
	})

	t.Run("contact with empty optional fields omits them", func(t *testing.T) {
		fullResume := &model.FullResumeDTO{
			ResumeBuilderDTO: &model.ResumeBuilderDTO{ID: "rb-1"},
			Contact: &model.ContactDTO{
				FullName: "John",
				Email:    "",
				Phone:    "",
				Location: "",
			},
		}

		var capturedResumeText string
		aiClient := &MockResumeAIClient{
			AnalyzeATSFunc: func(_ context.Context, resumeContent, _ string) (*ai.ATSCheckResult, error) {
				capturedResumeText = resumeContent
				return &ai.ATSCheckResult{Score: 40}, nil
			},
		}

		repo := &MockResumeBuilderRepository{
			VerifyOwnershipFunc: func(_ context.Context, _, _ string) error { return nil },
			GetFullResumeFunc: func(_ context.Context, _ string) (*model.FullResumeDTO, error) {
				return fullResume, nil
			},
		}

		svc := newAIService(repo, aiClient, &MockLimitChecker{})
		_, err := svc.ATSCheck(ctx, userID, resumeID, "")

		require.NoError(t, err)
		assert.Contains(t, capturedResumeText, "Name: John")
		assert.NotContains(t, capturedResumeText, "Email:")
		assert.NotContains(t, capturedResumeText, "Phone:")
		assert.NotContains(t, capturedResumeText, "Location:")
	})
}
