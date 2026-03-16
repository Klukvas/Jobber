package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/pdf"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
)

// ResumeTextParser parses raw text into structured resume data.
type ResumeTextParser interface {
	ParseResumeText(ctx context.Context, text string) (*ai.ParsedResume, error)
}

// PDFTextExtractor extracts plain text from PDF bytes.
type PDFTextExtractor func(pdfBytes []byte) (string, error)

// ImportService handles resume import from text and PDF.
type ImportService struct {
	repo         ports.ResumeBuilderRepository
	aiClient     ResumeTextParser
	limitChecker LimitChecker
	extractPDF   PDFTextExtractor
}

// NewImportService creates a new ImportService.
func NewImportService(
	repo ports.ResumeBuilderRepository,
	aiClient *ai.AnthropicClient,
	limitChecker LimitChecker,
) *ImportService {
	return &ImportService{
		repo:         repo,
		aiClient:     aiClient,
		limitChecker: limitChecker,
		extractPDF:   pdf.ExtractText,
	}
}

// NewImportServiceWithDeps creates an ImportService with explicit dependencies (useful for testing).
func NewImportServiceWithDeps(
	repo ports.ResumeBuilderRepository,
	parser ResumeTextParser,
	limitChecker LimitChecker,
	pdfExtractor PDFTextExtractor,
) *ImportService {
	return &ImportService{
		repo:         repo,
		aiClient:     parser,
		limitChecker: limitChecker,
		extractPDF:   pdfExtractor,
	}
}

// ImportFromText parses resume text with AI and creates a new resume.
func (s *ImportService) ImportFromText(ctx context.Context, userID, text, title string) (*model.FullResumeDTO, error) {
	if s.limitChecker != nil {
		if err := s.limitChecker.CheckLimit(ctx, userID, "resume_builders"); err != nil {
			return nil, err
		}
	}

	parsed, err := s.aiClient.ParseResumeText(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resume text: %w", err)
	}

	return s.createFromParsed(ctx, userID, parsed, title)
}

// ImportFromPDF extracts text from a PDF and creates a new resume.
func (s *ImportService) ImportFromPDF(ctx context.Context, userID string, pdfBytes []byte, title string) (*model.FullResumeDTO, error) {
	if s.limitChecker != nil {
		if err := s.limitChecker.CheckLimit(ctx, userID, "resume_builders"); err != nil {
			return nil, err
		}
	}

	text, err := s.extractPDF(pdfBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to extract PDF text: %w", err)
	}

	parsed, err := s.aiClient.ParseResumeText(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resume text: %w", err)
	}

	return s.createFromParsed(ctx, userID, parsed, title)
}

func (s *ImportService) createFromParsed(ctx context.Context, userID string, parsed *ai.ParsedResume, title string) (*model.FullResumeDTO, error) {
	if title == "" {
		title = parsed.FullName
		if title == "" {
			title = "Imported Resume"
		}
	}
	title = strings.TrimSpace(title)

	var resumeID string

	txErr := s.repo.RunInTransaction(ctx, func(txRepo ports.ResumeBuilderRepository) error {
		rb := newDefaultResumeBuilder(userID, title, DefaultTemplateID)

		if err := txRepo.Create(ctx, rb); err != nil {
			return fmt.Errorf("failed to create resume: %w", err)
		}
		resumeID = rb.ID

		// Seed default section order
		orders := make([]*model.SectionOrder, len(defaultSectionOrder))
		for i, ds := range defaultSectionOrder {
			orders[i] = &model.SectionOrder{
				ResumeBuilderID: rb.ID,
				SectionKey:      ds.Key,
				SortOrder:       ds.Order,
				IsVisible:       true,
			}
		}
		if err := txRepo.UpsertSectionOrder(ctx, rb.ID, orders); err != nil {
			return fmt.Errorf("failed to seed section order: %w", err)
		}

		// Upsert contact
		contact := &model.Contact{
			ResumeBuilderID: rb.ID,
			FullName:        parsed.FullName,
			Email:           parsed.Email,
			Phone:           parsed.Phone,
			Location:        parsed.Location,
			Website:         parsed.Website,
			LinkedIn:        parsed.LinkedIn,
			GitHub:          parsed.GitHub,
		}
		if err := txRepo.UpsertContact(ctx, contact); err != nil {
			return fmt.Errorf("failed to create contact: %w", err)
		}

		// Upsert summary
		if parsed.Summary != "" {
			summary := &model.Summary{
				ResumeBuilderID: rb.ID,
				Content:         parsed.Summary,
			}
			if err := txRepo.UpsertSummary(ctx, summary); err != nil {
				return fmt.Errorf("failed to create summary: %w", err)
			}
		}

		// Add experiences
		for i, exp := range parsed.Experiences {
			e := &model.Experience{
				ResumeBuilderID: rb.ID,
				Company:         exp.Company,
				Position:        exp.Position,
				Location:        exp.Location,
				StartDate:       exp.StartDate,
				EndDate:         exp.EndDate,
				IsCurrent:       exp.IsCurrent,
				Description:     exp.Description,
				SortOrder:       i,
			}
			if err := txRepo.CreateExperience(ctx, e); err != nil {
				return fmt.Errorf("failed to add experience: %w", err)
			}
		}

		// Add educations
		for i, edu := range parsed.Educations {
			e := &model.Education{
				ResumeBuilderID: rb.ID,
				Institution:     edu.Institution,
				Degree:          edu.Degree,
				FieldOfStudy:    edu.FieldOfStudy,
				StartDate:       edu.StartDate,
				EndDate:         edu.EndDate,
				GPA:             edu.GPA,
				SortOrder:       i,
			}
			if err := txRepo.CreateEducation(ctx, e); err != nil {
				return fmt.Errorf("failed to add education: %w", err)
			}
		}

		// Add skills
		for i, skill := range parsed.Skills {
			sk := &model.Skill{
				ResumeBuilderID: rb.ID,
				Name:            skill.Name,
				Level:           skill.Level,
				SortOrder:       i,
			}
			if err := txRepo.CreateSkill(ctx, sk); err != nil {
				return fmt.Errorf("failed to add skill: %w", err)
			}
		}

		// Add languages
		for i, lang := range parsed.Languages {
			l := &model.Language{
				ResumeBuilderID: rb.ID,
				Name:            lang.Name,
				Proficiency:     lang.Proficiency,
				SortOrder:       i,
			}
			if err := txRepo.CreateLanguage(ctx, l); err != nil {
				return fmt.Errorf("failed to add language: %w", err)
			}
		}

		// Add certifications
		for i, cert := range parsed.Certifications {
			c := &model.Certification{
				ResumeBuilderID: rb.ID,
				Name:            cert.Name,
				Issuer:          cert.Issuer,
				IssueDate:       cert.IssueDate,
				SortOrder:       i,
			}
			if err := txRepo.CreateCertification(ctx, c); err != nil {
				return fmt.Errorf("failed to add certification: %w", err)
			}
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	// Return full resume (outside transaction — read from committed data)
	return s.repo.GetFullResume(ctx, resumeID)
}
