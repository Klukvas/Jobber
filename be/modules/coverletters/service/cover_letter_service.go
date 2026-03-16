package service

import (
	"context"
	"regexp"

	"github.com/andreypavlenko/jobber/modules/coverletters/model"
	"github.com/andreypavlenko/jobber/modules/coverletters/ports"
	rbService "github.com/andreypavlenko/jobber/modules/resumebuilder/service"
)

var colorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// LimitChecker checks subscription resource limits.
type LimitChecker interface {
	CheckLimit(ctx context.Context, userID, resource string) error
}

// CoverLetterService handles cover letter business logic.
type CoverLetterService struct {
	repo         ports.CoverLetterRepository
	limitChecker LimitChecker
}

// NewCoverLetterService creates a new CoverLetterService.
func NewCoverLetterService(repo ports.CoverLetterRepository, limitChecker LimitChecker) *CoverLetterService {
	return &CoverLetterService{
		repo:         repo,
		limitChecker: limitChecker,
	}
}

// Create creates a new cover letter.
func (s *CoverLetterService) Create(ctx context.Context, userID string, req *model.CreateCoverLetterRequest) (*model.CoverLetterDTO, error) {
	if err := s.limitChecker.CheckLimit(ctx, userID, "cover_letters"); err != nil {
		return nil, err
	}

	title := req.Title
	if title == "" {
		title = "Untitled Cover Letter"
	}
	template := req.Template
	if template == "" {
		template = "professional"
	}

	cl := &model.CoverLetter{
		UserID:          userID,
		ResumeBuilderID: req.ResumeBuilderID,
		JobID:           req.JobID,
		Title:           title,
		Template:        template,
		FontFamily:      rbService.DefaultFontFamily,
		FontSize:        rbService.DefaultFontSize,
		PrimaryColor:    rbService.DefaultAccentColor,
		Paragraphs:      []string{},
	}

	created, err := s.repo.Create(ctx, cl)
	if err != nil {
		return nil, err
	}

	return created.ToDTO(), nil
}

// List returns all cover letters for a user.
func (s *CoverLetterService) List(ctx context.Context, userID string) ([]*model.CoverLetterDTO, error) {
	letters, err := s.repo.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*model.CoverLetterDTO, 0, len(letters))
	for _, cl := range letters {
		dtos = append(dtos, cl.ToDTO())
	}

	return dtos, nil
}

// Get returns a single cover letter.
func (s *CoverLetterService) Get(ctx context.Context, userID, id string) (*model.CoverLetterDTO, error) {
	cl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if cl.UserID != userID {
		return nil, model.ErrNotAuthorized
	}

	return cl.ToDTO(), nil
}

// Update updates a cover letter.
func (s *CoverLetterService) Update(ctx context.Context, userID, id string, req *model.UpdateCoverLetterRequest) (*model.CoverLetterDTO, error) {
	cl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if cl.UserID != userID {
		return nil, model.ErrNotAuthorized
	}

	if req.Title != nil {
		cl.Title = *req.Title
	}
	if req.ResumeBuilderID != nil {
		cl.ResumeBuilderID = req.ResumeBuilderID
	}
	if req.JobID != nil {
		cl.JobID = req.JobID
	}
	if req.Template != nil {
		cl.Template = *req.Template
	}
	if req.RecipientName != nil {
		cl.RecipientName = *req.RecipientName
	}
	if req.RecipientTitle != nil {
		cl.RecipientTitle = *req.RecipientTitle
	}
	if req.CompanyName != nil {
		cl.CompanyName = *req.CompanyName
	}
	if req.CompanyAddress != nil {
		cl.CompanyAddress = *req.CompanyAddress
	}
	if req.Greeting != nil {
		cl.Greeting = *req.Greeting
	}
	if req.Paragraphs != nil {
		cl.Paragraphs = *req.Paragraphs
	}
	if req.Closing != nil {
		cl.Closing = *req.Closing
	}
	if req.FontFamily != nil {
		if !rbService.AllowedFonts[*req.FontFamily] {
			return nil, model.ErrInvalidFont
		}
		cl.FontFamily = *req.FontFamily
	}
	if req.FontSize != nil {
		if *req.FontSize < 8 || *req.FontSize > 18 {
			return nil, model.ErrInvalidFontSize
		}
		cl.FontSize = *req.FontSize
	}
	if req.PrimaryColor != nil {
		if !colorRegex.MatchString(*req.PrimaryColor) {
			return nil, model.ErrInvalidColor
		}
		cl.PrimaryColor = *req.PrimaryColor
	}

	updated, err := s.repo.Update(ctx, cl)
	if err != nil {
		return nil, err
	}

	return updated.ToDTO(), nil
}

// Duplicate creates a copy of an existing cover letter.
func (s *CoverLetterService) Duplicate(ctx context.Context, userID, id string) (*model.CoverLetterDTO, error) {
	if err := s.limitChecker.CheckLimit(ctx, userID, "cover_letters"); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if original.UserID != userID {
		return nil, model.ErrNotAuthorized
	}

	paragraphs := make([]string, len(original.Paragraphs))
	copy(paragraphs, original.Paragraphs)

	cl := &model.CoverLetter{
		UserID:          userID,
		ResumeBuilderID: original.ResumeBuilderID,
		JobID:           original.JobID,
		Title:           original.Title + " (Copy)",
		Template:        original.Template,
		RecipientName:   original.RecipientName,
		RecipientTitle:  original.RecipientTitle,
		CompanyName:     original.CompanyName,
		CompanyAddress:  original.CompanyAddress,
		Greeting:        original.Greeting,
		Paragraphs:      paragraphs,
		Closing:         original.Closing,
		FontFamily:      original.FontFamily,
		FontSize:        original.FontSize,
		PrimaryColor:    original.PrimaryColor,
	}

	created, err := s.repo.Create(ctx, cl)
	if err != nil {
		return nil, err
	}

	return created.ToDTO(), nil
}

// Delete deletes a cover letter.
func (s *CoverLetterService) Delete(ctx context.Context, userID, id string) error {
	cl, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if cl.UserID != userID {
		return model.ErrNotAuthorized
	}

	return s.repo.Delete(ctx, id)
}
