package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
)

// LimitChecker checks subscription resource limits.
type LimitChecker interface {
	CheckLimit(ctx context.Context, userID, resource string) error
}

// Default section keys and their initial order.
var defaultSectionOrder = []struct {
	Key   string
	Order int
}{
	{"contact", 0},
	{"summary", 1},
	{"experience", 2},
	{"education", 3},
	{"skills", 4},
	{"languages", 5},
	{"certifications", 6},
	{"projects", 7},
	{"volunteering", 8},
	{"custom", 9},
}

var colorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Resume builder defaults — single source of truth.
const (
	DefaultAccentColor  = "#2563eb"
	DefaultTextColor    = "#111827"
	DefaultTemplateID   = "00000000-0000-0000-0000-000000000001"
	DefaultFontFamily   = "Georgia"
	DefaultSpacing      = 100
	DefaultMargin       = 40
	DefaultLayoutMode   = "single"
	DefaultSidebarWidth = 35
	DefaultFontSize     = 12
)

// ValidTemplateIDs is the set of allowed template UUID values.
var ValidTemplateIDs = map[string]bool{
	"00000000-0000-0000-0000-000000000001": true,
	"00000000-0000-0000-0000-000000000002": true,
	"00000000-0000-0000-0000-000000000003": true,
	"00000000-0000-0000-0000-000000000004": true,
	"00000000-0000-0000-0000-000000000005": true,
	"00000000-0000-0000-0000-000000000006": true,
	"00000000-0000-0000-0000-000000000007": true,
	"00000000-0000-0000-0000-000000000008": true,
	"00000000-0000-0000-0000-000000000009": true,
	"00000000-0000-0000-0000-00000000000a": true,
	"00000000-0000-0000-0000-00000000000b": true,
	"00000000-0000-0000-0000-00000000000c": true,
}

// ValidSectionKeys is the set of allowed section key values.
var ValidSectionKeys = map[string]bool{
	"contact": true, "summary": true, "experience": true,
	"education": true, "skills": true, "languages": true,
	"certifications": true, "projects": true, "volunteering": true,
	"custom": true,
}

// ValidColumnValues is the set of allowed column placement values.
var ValidColumnValues = map[string]bool{
	"main": true, "sidebar": true,
}

// ValidSkillDisplayModes is the set of allowed skill display mode values.
// Empty string means "use template default".
var ValidSkillDisplayModes = map[string]bool{
	"":           true,
	"text-level": true,
	"pill":       true,
	"grid-level": true,
	"vertical":   true,
	"text-only":  true,
	"dots":       true,
	"bar":        true,
	"square":     true,
	"star":       true,
	"circle":     true,
	"segmented":  true,
	"bubble":     true,
}

// AllowedFonts is the set of valid font families for resume and cover letter design.
var AllowedFonts = map[string]bool{
	"Georgia": true, "Arial": true, "Times New Roman": true,
	"Roboto": true, "Open Sans": true, "Lato": true, "Montserrat": true,
	"Poppins": true, "Inter": true, "Merriweather": true, "PT Serif": true,
	"Source Sans Pro": true, "Nunito": true, "Raleway": true, "Playfair Display": true,
}

// ResumeBuilderService handles resume builder business logic.
type ResumeBuilderService struct {
	repo         ports.ResumeBuilderRepository
	limitChecker LimitChecker
}

// NewResumeBuilderService creates a new ResumeBuilderService.
func NewResumeBuilderService(repo ports.ResumeBuilderRepository, limitChecker LimitChecker) *ResumeBuilderService {
	return &ResumeBuilderService{
		repo:         repo,
		limitChecker: limitChecker,
	}
}

// newDefaultResumeBuilder creates a ResumeBuilder with default design settings.
func newDefaultResumeBuilder(userID, title, templateID string) *model.ResumeBuilder {
	return &model.ResumeBuilder{
		UserID:       userID,
		Title:        title,
		TemplateID:   templateID,
		FontFamily:   DefaultFontFamily,
		PrimaryColor: DefaultAccentColor,
		TextColor:    DefaultTextColor,
		Spacing:      DefaultSpacing,
		MarginTop:    DefaultMargin,
		MarginBottom: DefaultMargin,
		MarginLeft:   DefaultMargin,
		MarginRight:  DefaultMargin,
		LayoutMode:   DefaultLayoutMode,
		SidebarWidth: DefaultSidebarWidth,
		FontSize:     DefaultFontSize,
	}
}

// Create creates a new resume builder with default section ordering.
func (s *ResumeBuilderService) Create(ctx context.Context, userID string, req *model.CreateResumeBuilderRequest) (*model.ResumeBuilderDTO, error) {
	if s.limitChecker != nil {
		if err := s.limitChecker.CheckLimit(ctx, userID, "resume_builders"); err != nil {
			return nil, err
		}
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Untitled Resume"
	}

	templateID := req.TemplateID
	if templateID == "" {
		templateID = DefaultTemplateID
	}

	rb := newDefaultResumeBuilder(userID, title, templateID)

	txErr := s.repo.RunInTransaction(ctx, func(txRepo ports.ResumeBuilderRepository) error {
		if err := txRepo.Create(ctx, rb); err != nil {
			return err
		}

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
		return txRepo.UpsertSectionOrder(ctx, rb.ID, orders)
	})
	if txErr != nil {
		return nil, txErr
	}

	return rb.ToDTO(), nil
}

// List returns all resume builders for a user.
func (s *ResumeBuilderService) List(ctx context.Context, userID string) ([]*model.ResumeBuilderDTO, error) {
	return s.repo.List(ctx, userID)
}

// Get returns the full resume with all sections.
func (s *ResumeBuilderService) Get(ctx context.Context, userID, id string) (*model.FullResumeDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, id); err != nil {
		return nil, err
	}
	return s.repo.GetFullResume(ctx, id)
}

// Update updates resume builder metadata and design settings.
func (s *ResumeBuilderService) Update(ctx context.Context, userID, id string, req *model.UpdateResumeBuilderRequest) (*model.ResumeBuilderDTO, error) {
	if err := s.repo.VerifyOwnership(ctx, userID, id); err != nil {
		return nil, err
	}

	rb, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		rb.Title = strings.TrimSpace(*req.Title)
	}
	if req.TemplateID != nil {
		if !ValidTemplateIDs[*req.TemplateID] {
			return nil, model.ErrInvalidTemplate
		}
		rb.TemplateID = *req.TemplateID
	}
	if req.FontFamily != nil {
		if !AllowedFonts[*req.FontFamily] {
			return nil, model.ErrInvalidFont
		}
		rb.FontFamily = *req.FontFamily
	}
	if req.PrimaryColor != nil {
		if !colorRegex.MatchString(*req.PrimaryColor) {
			return nil, model.ErrInvalidColor
		}
		rb.PrimaryColor = *req.PrimaryColor
	}
	if req.TextColor != nil {
		if !colorRegex.MatchString(*req.TextColor) {
			return nil, model.ErrInvalidColor
		}
		rb.TextColor = *req.TextColor
	}
	if req.Spacing != nil {
		if *req.Spacing < 50 || *req.Spacing > 150 {
			return nil, model.ErrInvalidSpacing
		}
		rb.Spacing = *req.Spacing
	}
	if req.MarginTop != nil {
		if *req.MarginTop < 0 || *req.MarginTop > 200 {
			return nil, model.ErrInvalidMargin
		}
		rb.MarginTop = *req.MarginTop
	}
	if req.MarginBottom != nil {
		if *req.MarginBottom < 0 || *req.MarginBottom > 200 {
			return nil, model.ErrInvalidMargin
		}
		rb.MarginBottom = *req.MarginBottom
	}
	if req.MarginLeft != nil {
		if *req.MarginLeft < 0 || *req.MarginLeft > 200 {
			return nil, model.ErrInvalidMargin
		}
		rb.MarginLeft = *req.MarginLeft
	}
	if req.MarginRight != nil {
		if *req.MarginRight < 0 || *req.MarginRight > 200 {
			return nil, model.ErrInvalidMargin
		}
		rb.MarginRight = *req.MarginRight
	}
	if req.LayoutMode != nil {
		mode := *req.LayoutMode
		if mode != "single" && mode != "double-left" && mode != "double-right" && mode != "custom" {
			return nil, model.ErrInvalidLayoutMode
		}
		rb.LayoutMode = mode
	}
	if req.SidebarWidth != nil {
		if *req.SidebarWidth < 25 || *req.SidebarWidth > 50 {
			return nil, model.ErrInvalidSidebarWidth
		}
		rb.SidebarWidth = *req.SidebarWidth
	}
	if req.FontSize != nil {
		if *req.FontSize < 8 || *req.FontSize > 18 {
			return nil, model.ErrInvalidFontSize
		}
		rb.FontSize = *req.FontSize
	}
	if req.SkillDisplay != nil {
		if !ValidSkillDisplayModes[*req.SkillDisplay] {
			return nil, model.ErrInvalidSkillDisplay
		}
		rb.SkillDisplay = *req.SkillDisplay
	}

	if err := s.repo.Update(ctx, rb); err != nil {
		return nil, err
	}

	return rb.ToDTO(), nil
}

// Delete deletes a resume builder and all its sections (cascade).
func (s *ResumeBuilderService) Delete(ctx context.Context, userID, id string) error {
	if err := s.repo.VerifyOwnership(ctx, userID, id); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

// Duplicate deep-copies a resume builder with all sections inside a transaction.
func (s *ResumeBuilderService) Duplicate(ctx context.Context, userID, id string) (*model.ResumeBuilderDTO, error) {
	if s.limitChecker != nil {
		if err := s.limitChecker.CheckLimit(ctx, userID, "resume_builders"); err != nil {
			return nil, err
		}
	}

	if err := s.repo.VerifyOwnership(ctx, userID, id); err != nil {
		return nil, err
	}

	full, err := s.repo.GetFullResume(ctx, id)
	if err != nil {
		return nil, err
	}

	var result *model.ResumeBuilderDTO

	txErr := s.repo.RunInTransaction(ctx, func(txRepo ports.ResumeBuilderRepository) error {
		// Create new resume builder
		newRB := &model.ResumeBuilder{
			UserID:       userID,
			Title:        full.Title + " (Copy)",
			TemplateID:   full.TemplateID,
			FontFamily:   full.FontFamily,
			PrimaryColor: full.PrimaryColor,
			TextColor:    full.TextColor,
			Spacing:      full.Spacing,
			MarginTop:    full.MarginTop,
			MarginBottom: full.MarginBottom,
			MarginLeft:   full.MarginLeft,
			MarginRight:  full.MarginRight,
			LayoutMode:   full.LayoutMode,
			SidebarWidth: full.SidebarWidth,
			FontSize:     full.FontSize,
			SkillDisplay: full.SkillDisplay,
		}

		if err := txRepo.Create(ctx, newRB); err != nil {
			return err
		}

		// Copy contact
		if full.Contact != nil {
			if err := txRepo.UpsertContact(ctx, &model.Contact{
				ResumeBuilderID: newRB.ID,
				FullName:        full.Contact.FullName,
				Email:           full.Contact.Email,
				Phone:           full.Contact.Phone,
				Location:        full.Contact.Location,
				Website:         full.Contact.Website,
				LinkedIn:        full.Contact.LinkedIn,
				GitHub:          full.Contact.GitHub,
			}); err != nil {
				return fmt.Errorf("failed to copy contact: %w", err)
			}
		}

		// Copy summary
		if full.Summary != nil {
			if err := txRepo.UpsertSummary(ctx, &model.Summary{
				ResumeBuilderID: newRB.ID,
				Content:         full.Summary.Content,
			}); err != nil {
				return fmt.Errorf("failed to copy summary: %w", err)
			}
		}

		// Copy experiences
		for _, e := range full.Experiences {
			if err := txRepo.CreateExperience(ctx, &model.Experience{
				ResumeBuilderID: newRB.ID,
				Company:         e.Company,
				Position:        e.Position,
				Location:        e.Location,
				StartDate:       e.StartDate,
				EndDate:         e.EndDate,
				IsCurrent:       e.IsCurrent,
				Description:     e.Description,
				SortOrder:       e.SortOrder,
			}); err != nil {
				return fmt.Errorf("failed to copy experience: %w", err)
			}
		}

		// Copy educations
		for _, e := range full.Educations {
			if err := txRepo.CreateEducation(ctx, &model.Education{
				ResumeBuilderID: newRB.ID,
				Institution:     e.Institution,
				Degree:          e.Degree,
				FieldOfStudy:    e.FieldOfStudy,
				StartDate:       e.StartDate,
				EndDate:         e.EndDate,
				IsCurrent:       e.IsCurrent,
				GPA:             e.GPA,
				Description:     e.Description,
				SortOrder:       e.SortOrder,
			}); err != nil {
				return fmt.Errorf("failed to copy education: %w", err)
			}
		}

		// Copy skills
		for _, sk := range full.Skills {
			if err := txRepo.CreateSkill(ctx, &model.Skill{ResumeBuilderID: newRB.ID, Name: sk.Name, Level: sk.Level, SortOrder: sk.SortOrder}); err != nil {
				return fmt.Errorf("failed to copy skill: %w", err)
			}
		}

		// Copy languages
		for _, l := range full.Languages {
			if err := txRepo.CreateLanguage(ctx, &model.Language{ResumeBuilderID: newRB.ID, Name: l.Name, Proficiency: l.Proficiency, SortOrder: l.SortOrder}); err != nil {
				return fmt.Errorf("failed to copy language: %w", err)
			}
		}

		// Copy certifications
		for _, cert := range full.Certifications {
			if err := txRepo.CreateCertification(ctx, &model.Certification{
				ResumeBuilderID: newRB.ID, Name: cert.Name, Issuer: cert.Issuer,
				IssueDate: cert.IssueDate, ExpiryDate: cert.ExpiryDate, URL: cert.URL, SortOrder: cert.SortOrder,
			}); err != nil {
				return fmt.Errorf("failed to copy certification: %w", err)
			}
		}

		// Copy projects
		for _, p := range full.Projects {
			if err := txRepo.CreateProject(ctx, &model.Project{
				ResumeBuilderID: newRB.ID, Name: p.Name, URL: p.URL,
				StartDate: p.StartDate, EndDate: p.EndDate, Description: p.Description, SortOrder: p.SortOrder,
			}); err != nil {
				return fmt.Errorf("failed to copy project: %w", err)
			}
		}

		// Copy volunteering
		for _, v := range full.Volunteering {
			if err := txRepo.CreateVolunteering(ctx, &model.Volunteering{
				ResumeBuilderID: newRB.ID, Organization: v.Organization, Role: v.Role,
				StartDate: v.StartDate, EndDate: v.EndDate, Description: v.Description, SortOrder: v.SortOrder,
			}); err != nil {
				return fmt.Errorf("failed to copy volunteering: %w", err)
			}
		}

		// Copy custom sections
		for _, cs := range full.CustomSections {
			if err := txRepo.CreateCustomSection(ctx, &model.CustomSection{
				ResumeBuilderID: newRB.ID, Title: cs.Title, Content: cs.Content, SortOrder: cs.SortOrder,
			}); err != nil {
				return fmt.Errorf("failed to copy custom section: %w", err)
			}
		}

		// Copy section order
		if len(full.SectionOrder) > 0 {
			orders := make([]*model.SectionOrder, len(full.SectionOrder))
			for i, o := range full.SectionOrder {
				orders[i] = &model.SectionOrder{
					ResumeBuilderID: newRB.ID,
					SectionKey:      o.SectionKey,
					SortOrder:       o.SortOrder,
					IsVisible:       o.IsVisible,
					Column:          o.Column,
				}
			}
			if err := txRepo.UpsertSectionOrder(ctx, newRB.ID, orders); err != nil {
				return fmt.Errorf("failed to copy section order: %w", err)
			}
		}

		result = newRB.ToDTO()
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	return result, nil
}
