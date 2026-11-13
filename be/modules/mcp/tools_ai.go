package mcp

import (
	"context"
	"errors"
	"strings"

	cvModel "github.com/andreypavlenko/jobber/modules/coverletters/model"
	importModel "github.com/andreypavlenko/jobber/modules/jobimport/model"
	matchModel "github.com/andreypavlenko/jobber/modules/matchscore/model"
	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
)

func (t *tools) parseJobPage(ctx context.Context, owner string, input ParseJobPageInput) (ParseJobPageOutput, error) {
	if err := validatePage(input); err != nil {
		return ParseJobPageOutput{}, err
	}
	if t.ai.Parser == nil {
		return ParseJobPageOutput{}, errAIUnavailable
	}
	ctx, finish, err := t.aiScope(ctx, owner, "parse_job_page", "job_parse", input)
	if err != nil {
		return ParseJobPageOutput{}, err
	}
	defer finish()
	if err := t.ai.Credits.CheckLimit(ctx, owner, "job_parses"); err != nil {
		return ParseJobPageOutput{}, err
	}
	draft, err := t.ai.Parser.ParseJobPage(ctx, owner, &importModel.ParseJobRequest{PageURL: input.PageURL, PageText: input.PageText})
	return ParseJobPageOutput{Draft: draft}, aiCallError(err)
}
func (t *tools) tailorResume(ctx context.Context, owner string, input TailorResumeInput) (TailorResumeOutput, error) {
	job, resume, err := canonicalPair(input.JobID, input.ResumeID)
	if err != nil {
		return TailorResumeOutput{}, err
	}
	locale := input.Locale
	if locale == "" {
		locale = "en"
	}
	if locale != "en" && locale != "ru" && locale != "uk" {
		return TailorResumeOutput{}, validationError("locale must be en, ru or uk.")
	}
	if t.ai.Tailoring == nil || t.ai.Builders == nil {
		return TailorResumeOutput{}, errAIUnavailable
	}
	if _, err := t.jobs.GetByID(ctx, owner, job); err != nil {
		return TailorResumeOutput{}, err
	}
	if _, err := t.ai.Builders.Get(ctx, owner, resume); err != nil {
		return TailorResumeOutput{}, err
	}
	ctx, finish, err := t.aiScope(ctx, owner, "tailor_resume", "tailor", TailorResumeInput{JobID: job, ResumeID: resume, Locale: locale})
	if err != nil {
		return TailorResumeOutput{}, err
	}
	defer finish()
	// The web workflow atomically reserves credits before invoking its provider.
	proposal, err := t.ai.Tailoring.Generate(ctx, owner, job, resume, locale)
	if err != nil {
		return TailorResumeOutput{}, aiCallError(err)
	}
	mapped, err := mapTailoringProposal(proposal)
	return TailorResumeOutput{Proposal: mapped}, err
}
func (t *tools) applyTailoring(ctx context.Context, owner string, input ApplyTailoringInput) (ApplyTailoringOutput, error) {
	canonical, err := validateApply(input)
	if err != nil {
		return ApplyTailoringOutput{}, err
	}
	if t.ai.Tailoring == nil {
		return ApplyTailoringOutput{}, errAIUnavailable
	}
	if _, err := t.jobs.GetByID(ctx, owner, canonical.JobID); err != nil {
		return ApplyTailoringOutput{}, err
	}
	result, replayed, err := t.ai.Tailoring.Apply(ctx, owner, canonical.JobID, canonical.ProposalID, &rbModel.ApplyTailoringRequest{Title: canonical.Title, AcceptedChangeIDs: canonical.AcceptedChangeIDs})
	if err != nil {
		return ApplyTailoringOutput{}, err
	}
	return ApplyTailoringOutput{ResumeID: result.ResumeBuilder.ID, JobID: result.JobID, Replayed: replayed}, nil
}
func (t *tools) generateCoverLetter(ctx context.Context, owner string, input GenerateCoverLetterInput) (GenerateCoverLetterOutput, error) {
	jobID, err := canonicalID(input.JobID, "job_id")
	if err != nil {
		return GenerateCoverLetterOutput{}, err
	}
	resumeID := ""
	if input.ResumeID != "" {
		resumeID, err = canonicalID(input.ResumeID, "resume_id")
		if err != nil {
			return GenerateCoverLetterOutput{}, err
		}
	}
	if t.ai.Letters == nil || t.ai.LetterAI == nil || (resumeID != "" && t.ai.Builders == nil) {
		return GenerateCoverLetterOutput{}, errAIUnavailable
	}
	job, err := t.jobs.GetByID(ctx, owner, jobID)
	if err != nil {
		return GenerateCoverLetterOutput{}, err
	}
	if job.Description == nil || strings.TrimSpace(*job.Description) == "" {
		return GenerateCoverLetterOutput{}, validationError("Save a Job description before generating a cover letter.")
	}
	if err := validateText(*job.Description, "job_description", 10000, true); err != nil {
		return GenerateCoverLetterOutput{}, err
	}
	if resumeID != "" {
		if _, err := t.ai.Builders.Get(ctx, owner, resumeID); err != nil {
			return GenerateCoverLetterOutput{}, err
		}
	}
	ctx, finish, err := t.aiScope(ctx, owner, "generate_cover_letter", "cover_letter", GenerateCoverLetterInput{JobID: jobID, ResumeID: resumeID})
	if err != nil {
		return GenerateCoverLetterOutput{}, err
	}
	defer finish()
	if err := t.ai.Credits.CheckLimit(ctx, owner, "ai"); err != nil {
		return GenerateCoverLetterOutput{}, err
	}
	letter, err := t.ai.Letters.Create(ctx, owner, &cvModel.CreateCoverLetterRequest{Title: "Cover letter", JobID: &jobID, ResumeBuilderID: optionalText(resumeID)})
	if err != nil {
		return GenerateCoverLetterOutput{}, err
	}
	return t.generateLetterContent(ctx, owner, letter.ID, *job.Description, job.CompanyName)
}
func (t *tools) generateLetterContent(ctx context.Context, owner, letterID, description string, company *string) (GenerateCoverLetterOutput, error) {
	// The web editor saves company context before generating, then saves the generated document.
	if company != nil {
		if _, err := t.ai.Letters.Update(ctx, owner, letterID, &cvModel.UpdateCoverLetterRequest{CompanyName: company}); err != nil {
			return GenerateCoverLetterOutput{}, err
		}
	}
	content, err := t.ai.LetterAI.Generate(ctx, owner, letterID, description)
	if err != nil {
		return GenerateCoverLetterOutput{}, aiCallError(err)
	}
	_, err = t.ai.Letters.Update(ctx, owner, letterID, &cvModel.UpdateCoverLetterRequest{Greeting: &content.Greeting, Paragraphs: &content.Paragraphs, Closing: &content.Closing})
	return GenerateCoverLetterOutput{CoverLetterID: letterID, Greeting: content.Greeting, Paragraphs: content.Paragraphs, Closing: content.Closing}, err
}
func (t *tools) checkMatch(ctx context.Context, owner string, input CheckMatchInput) (CheckMatchOutput, error) {
	job, resume, err := canonicalPair(input.JobID, input.ResumeID)
	if err != nil {
		return CheckMatchOutput{}, err
	}
	if t.ai.MatchCache == nil {
		return CheckMatchOutput{}, errAIUnavailable
	}
	request := &matchModel.MatchScoreRequest{JobID: job, ResumeID: resume}
	cached, err := t.ai.MatchCache.GetCachedMatch(ctx, owner, request)
	if err == nil {
		return CheckMatchOutput{Match: cached}, nil
	}
	if !errors.Is(err, matchModel.ErrCacheMiss) {
		return CheckMatchOutput{}, err
	}
	if t.ai.Matcher == nil {
		return CheckMatchOutput{}, errAIUnavailable
	}
	// GetCachedMatch verifies both owners even on a miss, before any credit scope.
	ctx, finish, err := t.aiScope(ctx, owner, "check_match", "match", CheckMatchInput{JobID: job, ResumeID: resume})
	if err != nil {
		return CheckMatchOutput{}, err
	}
	defer finish()
	if err := t.ai.Credits.CheckLimit(ctx, owner, "ai"); err != nil {
		return CheckMatchOutput{}, err
	}
	result, err := t.ai.Matcher.CheckMatch(ctx, owner, request)
	return CheckMatchOutput{Match: result}, aiCallError(err)
}
