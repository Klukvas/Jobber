package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
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

	if err := s.repo.VerifyOwnership(ctx, userID, coverLetterID); err != nil {
		return nil, err
	}

	cl, err := s.repo.GetByID(ctx, userID, coverLetterID)
	if err != nil {
		return nil, err
	}

	resumeContext := ""
	if cl.ResumeBuilderID != nil && *cl.ResumeBuilderID != "" {
		// VerifyOwnership failure is intentional — user may not own the linked resume.
		if err := s.resumeRepo.VerifyOwnership(ctx, userID, *cl.ResumeBuilderID); err == nil {
			resume, err := s.resumeRepo.GetFullResume(ctx, userID, *cl.ResumeBuilderID)
			if err != nil {
				slog.Warn("failed to load linked resume for cover letter AI context",
					"cover_letter_id", coverLetterID,
					"resume_builder_id", *cl.ResumeBuilderID,
					"error", err,
				)
			} else if resumeContext, err = buildResumeContext(resume); err != nil {
				return nil, err
			}
		}
	}

	content, err := s.aiClient.GenerateCoverLetter(ctx, cl.CompanyName, cl.RecipientName, cl.RecipientTitle, jobDescription, resumeContext)
	if err != nil {
		return nil, err
	}
	// Record the billable AI call so it counts against the user's quota.
	if err := s.limitChecker.RecordAIUsage(ctx, userID); err != nil {
		slog.Warn("failed to record AI usage", "user_id", userID, "error", err)
	}
	return content, nil
}

// buildResumeContext serializes the linked Builder Resume as structured JSON.
// The grounded cover letter generator only accepts typed experience units as
// evidence and rejects free-form text, so a plain-text summary here makes every
// resume-linked generation fail.
func buildResumeContext(resume *rbModel.FullResumeDTO) (string, error) {
	data, err := json.Marshal(resume)
	if err != nil {
		return "", fmt.Errorf("serialize resume context: %w", err)
	}
	return string(data), nil
}
