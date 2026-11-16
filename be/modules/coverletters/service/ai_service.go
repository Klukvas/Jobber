package service

import (
	"context"
	"encoding/json"
	"errors"
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
	if strings.TrimSpace(jobDescription) == "" {
		return nil, model.ErrAIJobDescriptionRequired
	}

	if err := s.repo.VerifyOwnership(ctx, userID, coverLetterID); err != nil {
		return nil, err
	}

	cl, err := s.repo.GetByID(ctx, userID, coverLetterID)
	if err != nil {
		return nil, err
	}

	resumeContext, err := s.candidateSource(ctx, userID, cl)
	if err != nil {
		return nil, err
	}
	if err := s.limitChecker.CheckLimit(ctx, userID, "ai"); err != nil {
		return nil, err
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

func (s *AIService) candidateSource(ctx context.Context, userID string, cl *model.CoverLetter) (string, error) {
	if cl == nil || cl.ResumeBuilderID == nil || strings.TrimSpace(*cl.ResumeBuilderID) == "" {
		return "", model.ErrAICandidateSourceRequired
	}
	if err := s.resumeRepo.VerifyOwnership(ctx, userID, *cl.ResumeBuilderID); err != nil {
		if errors.Is(err, rbModel.ErrNotOwner) || errors.Is(err, rbModel.ErrResumeBuilderNotFound) {
			return "", model.ErrAICandidateSourceRequired
		}
		return "", fmt.Errorf("verify cover letter candidate source: %w", err)
	}
	resume, err := s.resumeRepo.GetFullResume(ctx, userID, *cl.ResumeBuilderID)
	if err != nil {
		if errors.Is(err, rbModel.ErrResumeBuilderNotFound) || errors.Is(err, rbModel.ErrNotOwner) {
			return "", model.ErrAICandidateSourceRequired
		}
		return "", fmt.Errorf("load cover letter candidate source: %w", err)
	}
	if resume == nil {
		return "", model.ErrAICandidateSourceRequired
	}
	resumeContext, err := buildResumeContext(resume)
	if err != nil {
		return "", err
	}
	if !ai.HasCoverLetterCandidateSource(resumeContext) {
		return "", model.ErrAICandidateSourceRequired
	}
	return resumeContext, nil
}

// buildResumeContext serializes the linked Builder Resume as structured JSON.
// The grounded generator reads complete typed source entries from this JSON.
func buildResumeContext(resume *rbModel.FullResumeDTO) (string, error) {
	data, err := json.Marshal(resume)
	if err != nil {
		return "", fmt.Errorf("serialize resume context: %w", err)
	}
	return string(data), nil
}
