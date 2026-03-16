package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/andreypavlenko/jobber/modules/coverletters/ports"
	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	rbPorts "github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
)

// CoverLetterAIClient defines the AI operations for cover letter generation.
type CoverLetterAIClient interface {
	GenerateCoverLetter(ctx context.Context, companyName, recipientName, recipientTitle, jobDescription, resumeContext string) (*ai.CoverLetterContent, error)
}

// AIService handles AI-powered cover letter generation.
type AIService struct {
	repo         ports.CoverLetterRepository
	resumeRepo   rbPorts.ResumeBuilderRepository
	aiClient     CoverLetterAIClient
	limitChecker LimitChecker
}

// NewAIService creates a new cover letter AIService.
func NewAIService(repo ports.CoverLetterRepository, resumeRepo rbPorts.ResumeBuilderRepository, aiClient CoverLetterAIClient, limitChecker LimitChecker) *AIService {
	return &AIService{
		repo:         repo,
		resumeRepo:   resumeRepo,
		aiClient:     aiClient,
		limitChecker: limitChecker,
	}
}

// Generate generates cover letter content using AI.
func (s *AIService) Generate(ctx context.Context, userID, coverLetterID, jobDescription string) (*ai.CoverLetterContent, error) {
	if err := s.limitChecker.CheckLimit(ctx, userID, "ai"); err != nil {
		return nil, err
	}

	cl, err := s.repo.GetByID(ctx, coverLetterID)
	if err != nil {
		return nil, err
	}

	if cl.UserID != userID {
		return nil, model.ErrNotAuthorized
	}

	resumeContext := ""
	if cl.ResumeBuilderID != nil && *cl.ResumeBuilderID != "" {
		// VerifyOwnership failure is intentional — user may not own the linked resume.
		if err := s.resumeRepo.VerifyOwnership(ctx, userID, *cl.ResumeBuilderID); err == nil {
			resume, err := s.resumeRepo.GetFullResume(ctx, *cl.ResumeBuilderID)
			if err != nil {
				slog.Warn("failed to load linked resume for cover letter AI context",
					"cover_letter_id", coverLetterID,
					"resume_builder_id", *cl.ResumeBuilderID,
					"error", err,
				)
			} else {
				resumeContext = buildResumeContext(resume)
			}
		}
	}

	return s.aiClient.GenerateCoverLetter(ctx, cl.CompanyName, cl.RecipientName, cl.RecipientTitle, jobDescription, resumeContext)
}

func buildResumeContext(resume *rbModel.FullResumeDTO) string {
	var parts []string

	if resume.Contact != nil {
		if resume.Contact.FullName != "" {
			parts = append(parts, fmt.Sprintf("Name: %s", resume.Contact.FullName))
		}
		if resume.Contact.Email != "" {
			parts = append(parts, fmt.Sprintf("Email: %s", resume.Contact.Email))
		}
	}

	if resume.Summary != nil && resume.Summary.Content != "" {
		parts = append(parts, fmt.Sprintf("\nSummary: %s", resume.Summary.Content))
	}

	if len(resume.Experiences) > 0 {
		parts = append(parts, "\nExperience:")
		for _, exp := range resume.Experiences {
			entry := fmt.Sprintf("- %s at %s", exp.Position, exp.Company)
			if exp.Description != "" {
				entry += ": " + exp.Description
			}
			parts = append(parts, entry)
		}
	}

	if len(resume.Skills) > 0 {
		skillNames := make([]string, 0, len(resume.Skills))
		for _, skill := range resume.Skills {
			skillNames = append(skillNames, skill.Name)
		}
		parts = append(parts, fmt.Sprintf("\nSkills: %s", strings.Join(skillNames, ", ")))
	}

	if len(resume.Educations) > 0 {
		parts = append(parts, "\nEducation:")
		for _, edu := range resume.Educations {
			parts = append(parts, fmt.Sprintf("- %s, %s - %s", edu.Degree, edu.FieldOfStudy, edu.Institution))
		}
	}

	return strings.Join(parts, "\n")
}
