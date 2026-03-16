package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
)

// ResumeAIClient defines the AI operations used by the resume builder.
type ResumeAIClient interface {
	SuggestBulletPoints(ctx context.Context, jobTitle, company, currentDescription string) (*ai.BulletSuggestions, error)
	SuggestSummary(ctx context.Context, name, jobTitle, experienceContext string) (string, error)
	ImproveText(ctx context.Context, text, instruction string) (string, error)
	AnalyzeATS(ctx context.Context, resumeContent, locale string) (*ai.ATSCheckResult, error)
}

// AIService handles AI-powered resume suggestions.
type AIService struct {
	repo         ports.ResumeBuilderRepository
	aiClient     ResumeAIClient
	limitChecker LimitChecker
}

// NewAIService creates a new AIService.
func NewAIService(repo ports.ResumeBuilderRepository, aiClient ResumeAIClient, limitChecker LimitChecker) *AIService {
	return &AIService{
		repo:         repo,
		aiClient:     aiClient,
		limitChecker: limitChecker,
	}
}

// SuggestBulletPoints generates bullet point suggestions for a work experience.
func (s *AIService) SuggestBulletPoints(ctx context.Context, userID, jobTitle, company, currentDescription string) (*ai.BulletSuggestions, error) {
	if err := s.limitChecker.CheckLimit(ctx, userID, "ai_requests"); err != nil {
		return nil, err
	}

	return s.aiClient.SuggestBulletPoints(ctx, jobTitle, company, currentDescription)
}

// SuggestSummary generates a professional summary based on resume context.
func (s *AIService) SuggestSummary(ctx context.Context, userID, resumeID string) (string, error) {
	if err := s.limitChecker.CheckLimit(ctx, userID, "ai_requests"); err != nil {
		return "", err
	}

	if err := s.repo.VerifyOwnership(ctx, userID, resumeID); err != nil {
		return "", err
	}

	// Get full resume to build context
	resume, err := s.repo.GetFullResume(ctx, resumeID)
	if err != nil {
		return "", fmt.Errorf("failed to get resume: %w", err)
	}

	// Build experience context from resume data
	var contextParts []string
	name := ""
	if resume.Contact != nil {
		name = resume.Contact.FullName
	}

	for _, exp := range resume.Experiences {
		entry := fmt.Sprintf("%s at %s", exp.Position, exp.Company)
		if exp.Description != "" {
			entry += ": " + exp.Description
		}
		contextParts = append(contextParts, entry)
	}

	jobTitle := ""
	if len(resume.Experiences) > 0 {
		jobTitle = resume.Experiences[0].Position
	}

	return s.aiClient.SuggestSummary(ctx, name, jobTitle, strings.Join(contextParts, "\n\n"))
}

// ImproveText improves a text snippet based on an instruction.
func (s *AIService) ImproveText(ctx context.Context, userID, text, instruction string) (string, error) {
	if err := s.limitChecker.CheckLimit(ctx, userID, "ai_requests"); err != nil {
		return "", err
	}

	return s.aiClient.ImproveText(ctx, text, instruction)
}

// ATSCheck analyzes a resume for ATS compatibility.
func (s *AIService) ATSCheck(ctx context.Context, userID, resumeID, locale string) (*ai.ATSCheckResult, error) {
	if err := s.limitChecker.CheckLimit(ctx, userID, "ai_requests"); err != nil {
		return nil, err
	}

	if err := s.repo.VerifyOwnership(ctx, userID, resumeID); err != nil {
		return nil, err
	}

	resume, err := s.repo.GetFullResume(ctx, resumeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resume: %w", err)
	}

	// Build a text representation of the resume for ATS analysis
	var parts []string

	if resume.Contact != nil {
		parts = append(parts, fmt.Sprintf("Name: %s", resume.Contact.FullName))
		if resume.Contact.Email != "" {
			parts = append(parts, fmt.Sprintf("Email: %s", resume.Contact.Email))
		}
		if resume.Contact.Phone != "" {
			parts = append(parts, fmt.Sprintf("Phone: %s", resume.Contact.Phone))
		}
		if resume.Contact.Location != "" {
			parts = append(parts, fmt.Sprintf("Location: %s", resume.Contact.Location))
		}
	}

	if resume.Summary != nil && resume.Summary.Content != "" {
		parts = append(parts, "\n--- SUMMARY ---")
		parts = append(parts, resume.Summary.Content)
	}

	if len(resume.Experiences) > 0 {
		parts = append(parts, "\n--- EXPERIENCE ---")
		for _, exp := range resume.Experiences {
			entry := fmt.Sprintf("%s at %s (%s)", exp.Position, exp.Company, exp.StartDate)
			if exp.Description != "" {
				entry += "\n" + exp.Description
			}
			parts = append(parts, entry)
		}
	}

	if len(resume.Educations) > 0 {
		parts = append(parts, "\n--- EDUCATION ---")
		for _, edu := range resume.Educations {
			parts = append(parts, fmt.Sprintf("%s, %s - %s", edu.Degree, edu.FieldOfStudy, edu.Institution))
		}
	}

	if len(resume.Skills) > 0 {
		parts = append(parts, "\n--- SKILLS ---")
		skillNames := make([]string, 0, len(resume.Skills))
		for _, skill := range resume.Skills {
			skillNames = append(skillNames, skill.Name)
		}
		parts = append(parts, strings.Join(skillNames, ", "))
	}

	if len(resume.Languages) > 0 {
		parts = append(parts, "\n--- LANGUAGES ---")
		for _, lang := range resume.Languages {
			parts = append(parts, fmt.Sprintf("%s (%s)", lang.Name, lang.Proficiency))
		}
	}

	if len(resume.Certifications) > 0 {
		parts = append(parts, "\n--- CERTIFICATIONS ---")
		for _, cert := range resume.Certifications {
			parts = append(parts, fmt.Sprintf("%s - %s", cert.Name, cert.Issuer))
		}
	}

	if len(resume.Projects) > 0 {
		parts = append(parts, "\n--- PROJECTS ---")
		for _, proj := range resume.Projects {
			entry := proj.Name
			if proj.Description != "" {
				entry += ": " + proj.Description
			}
			parts = append(parts, entry)
		}
	}

	resumeText := strings.Join(parts, "\n")
	const maxATSResumeTextLen = 50000
	if len(resumeText) > maxATSResumeTextLen {
		resumeText = resumeText[:maxATSResumeTextLen]
	}
	return s.aiClient.AnalyzeATS(ctx, resumeText, locale)
}
