package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/internal/platform/pdf"
	"github.com/andreypavlenko/jobber/internal/platform/storage"
	"github.com/andreypavlenko/jobber/modules/resumes/model"
	"github.com/andreypavlenko/jobber/modules/resumes/ports"
)

// autofillParserVersion stamps stored profiles with the prompt generation that
// produced them. Provenance metadata only — reads never filter by it, and
// bumping it must not invalidate existing profiles (users must not pay quota
// for our prompt improvements — docs/adr/0001-autofill-profile-economics.md).
const autofillParserVersion = 1

// ResumeParser extracts structured resume data from raw text.
type ResumeParser interface {
	ParseResumeText(ctx context.Context, text string) (*ai.ParsedResume, error)
}

// PlanChecker gates the paid extraction and accounts its AI quota.
type PlanChecker interface {
	RequirePaidPlan(ctx context.Context, userID string) error
	CheckLimit(ctx context.Context, userID, resource string) error
	RecordResumeAutofillUsage(ctx context.Context, userID string) error
}

// AutofillProfileService extracts and caches Autofill Profiles from Uploaded
// Resumes (docs/plans/autofill-uploaded-pdf.md).
type AutofillProfileService struct {
	resumeRepo  ports.ResumeRepository
	profileRepo ports.AutofillProfileRepository
	s3Client    *storage.S3Client
	parser      ResumeParser
	planChecker PlanChecker
	extractPDF  func([]byte) (string, error)
}

func NewAutofillProfileService(
	resumeRepo ports.ResumeRepository,
	profileRepo ports.AutofillProfileRepository,
	s3Client *storage.S3Client,
	parser ResumeParser,
	planChecker PlanChecker,
) *AutofillProfileService {
	return &AutofillProfileService{
		resumeRepo:  resumeRepo,
		profileRepo: profileRepo,
		s3Client:    s3Client,
		parser:      parser,
		planChecker: planChecker,
		extractPDF:  pdf.ExtractText,
	}
}

// GetProfile returns the Autofill Profile of an Uploaded Resume, extracting
// and caching it on first request.
func (s *AutofillProfileService) GetProfile(ctx context.Context, userID, resumeID string) (*model.AutofillProfileDTO, error) {
	// Owner check first — a foreign resume id must 404 before any plan or
	// quota state can leak.
	resume, err := s.resumeRepo.GetByID(ctx, userID, resumeID)
	if err != nil {
		return nil, err
	}

	// Cache hits are free and skip the paid gate: the paid act is the
	// extraction, not the use of its result (ADR-0001).
	cached, err := s.profileRepo.Get(ctx, userID, resumeID)
	if err != nil {
		// A read failure must NOT fall through to a fresh extraction: that
		// path would re-charge quota for an already-paid file and 403 a
		// downgraded user who owns a cached profile — both violate ADR-0001.
		// Only a true (nil, nil) miss proceeds.
		return nil, fmt.Errorf("failed to read autofill profile cache: %w", err)
	}
	if cached != nil {
		return mapParsedResume(cached), nil
	}

	if s.planChecker != nil {
		if err := s.planChecker.RequirePaidPlan(ctx, userID); err != nil {
			return nil, err
		}
		if err := s.planChecker.CheckLimit(ctx, userID, "ai"); err != nil {
			return nil, err
		}
	}

	pdfBytes, err := storage.DownloadResumeFile(ctx, s.s3Client, string(resume.StorageType), resume.StorageKey, resume.FileURL)
	if err != nil {
		if errors.Is(err, storage.ErrResumeFileMissing) {
			return nil, model.ErrResumeFileMissing
		}
		return nil, fmt.Errorf("failed to download resume file: %w", err)
	}

	// Scanned/image-only or non-PDF content yields no text: nothing to parse,
	// so no AI call is made, nothing cached, nothing charged. The cause stays
	// in logs — a pdf-library regression or an oversized file would otherwise
	// be indistinguishable from a user's bad scan behind the same 422.
	text, err := s.extractPDF(pdfBytes)
	if err != nil || strings.TrimSpace(text) == "" {
		log.Printf("[WARN] autofill text extraction failed for resume=%s: err=%v textLen=%d", resumeID, err, len(text))
		return nil, model.ErrResumeUnreadable
	}

	parsed, err := s.parser.ParseResumeText(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to extract autofill profile: %w", err)
	}

	// Usability threshold: autofill's core value is contact data. Without a
	// name or an email the extraction failed regardless of other sections —
	// not cached, not charged (the AI rate limiter bounds retry cost).
	if strings.TrimSpace(parsed.FullName) == "" && strings.TrimSpace(parsed.Email) == "" {
		return nil, model.ErrResumeUnreadable
	}

	// Cache before recording usage: if the write fails, the user retries
	// without having been charged for a profile they never received.
	inserted, err := s.profileRepo.Upsert(ctx, userID, resumeID, parsed, autofillParserVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to store autofill profile: %w", err)
	}
	// Only the request that actually created the row records usage: two
	// concurrent first extractions of the same file must charge the quota
	// once, not twice (ADR-0001: once per file).
	if inserted && s.planChecker != nil {
		if err := s.planChecker.RecordResumeAutofillUsage(ctx, userID); err != nil {
			log.Printf("[ERROR] failed to record autofill usage for user=%s: %v", userID, err)
		}
	}

	return mapParsedResume(parsed), nil
}

// mapParsedResume shapes the raw extraction like the Builder Resume profile
// subset the extension consumes, so both autofill sources look identical to it.
func mapParsedResume(p *ai.ParsedResume) *model.AutofillProfileDTO {
	dto := &model.AutofillProfileDTO{
		Contact: &model.AutofillContactDTO{
			FullName: p.FullName,
			Email:    p.Email,
			Phone:    p.Phone,
			Location: p.Location,
			Website:  p.Website,
			LinkedIn: p.LinkedIn,
			GitHub:   p.GitHub,
		},
		Experiences: make([]model.AutofillExperienceDTO, len(p.Experiences)),
		Educations:  make([]model.AutofillEducationDTO, len(p.Educations)),
		Skills:      make([]model.AutofillSkillDTO, len(p.Skills)),
	}

	if strings.TrimSpace(p.Summary) != "" {
		dto.Summary = &model.AutofillSummaryDTO{Content: p.Summary}
	}
	for i, exp := range p.Experiences {
		dto.Experiences[i] = model.AutofillExperienceDTO{
			Company:     exp.Company,
			Position:    exp.Position,
			Location:    exp.Location,
			StartDate:   exp.StartDate,
			EndDate:     exp.EndDate,
			IsCurrent:   exp.IsCurrent,
			Description: exp.Description,
		}
	}
	for i, edu := range p.Educations {
		dto.Educations[i] = model.AutofillEducationDTO{
			Institution:  edu.Institution,
			Degree:       edu.Degree,
			FieldOfStudy: edu.FieldOfStudy,
			StartDate:    edu.StartDate,
			EndDate:      edu.EndDate,
		}
	}
	for i, skill := range p.Skills {
		dto.Skills[i] = model.AutofillSkillDTO{Name: skill.Name, Level: skill.Level}
	}

	return dto
}

// CompositeInvalidator fans a resume-change invalidation out to every cache
// derived from the resume file (match-score results, autofill profiles), so
// ResumeService keeps its single CacheInvalidator seam.
type CompositeInvalidator struct {
	invalidators []CacheInvalidator
}

func NewCompositeInvalidator(invalidators ...CacheInvalidator) *CompositeInvalidator {
	return &CompositeInvalidator{invalidators: invalidators}
}

func (c *CompositeInvalidator) InvalidateByResume(ctx context.Context, resumeID string) error {
	var errs []error
	for _, inv := range c.invalidators {
		if inv == nil {
			continue
		}
		if err := inv.InvalidateByResume(ctx, resumeID); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
