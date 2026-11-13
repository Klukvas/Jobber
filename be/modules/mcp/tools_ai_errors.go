package mcp

import (
	"errors"

	importModel "github.com/andreypavlenko/jobber/modules/jobimport/model"
	matchModel "github.com/andreypavlenko/jobber/modules/matchscore/model"
	rbModel "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
)

func aiCallError(err error) error {
	if err == nil {
		return nil
	}
	if message := aiFriendlyError(err); message != "" {
		return validationError(message)
	}
	return errAIProvider
}
func aiFriendlyError(err error) string {
	switch {
	case errors.Is(err, errAIUnavailable), errors.Is(err, rbModel.ErrAIUnavailable), errors.Is(err, rbModel.ErrProposalStoreUnavailable), errors.Is(err, importModel.ErrAINotConfigured):
		return "AI features are not available right now. Please try again later."
	case errors.Is(err, errPDFUnavailable):
		return "PDF export is not available. Please try again later."
	case errors.Is(err, errAIProvider):
		return "AI could not complete this request. Please try again later."
	case errors.Is(err, subModel.ErrPaidFeature):
		return "Upgrade to Pro or Enterprise to use AI assistant tools."
	case errors.Is(err, subModel.ErrCreditUnavailable), errors.Is(err, subModel.ErrLimitReached):
		return "Your plan's allowance has been reached or you have insufficient AI credits. Upgrade your plan to continue."
	case errors.Is(err, subModel.ErrAIBudgetReached), errors.Is(err, subModel.ErrAIOperationConflict):
		return "AI features are temporarily unavailable. Please try again later."
	case errors.Is(err, rbModel.ErrTailoringNotFound), errors.Is(err, rbModel.ErrResumeBuilderNotFound), errors.Is(err, rbModel.ErrNotOwner), errors.Is(err, resumeModel.ErrResumeNotFound):
		return "Job, resume or proposal not found in your account."
	case errors.Is(err, rbModel.ErrTailoringValidation):
		return "Select valid changes, provide a title and save a Job description."
	case errors.Is(err, rbModel.ErrProposalExpired), errors.Is(err, rbModel.ErrSourceChanged), errors.Is(err, rbModel.ErrJobChanged), errors.Is(err, rbModel.ErrProposalAlreadyApplied):
		return "The proposal expired, was applied, or its source changed. Generate a new proposal."
	case errors.Is(err, rbModel.ErrRecognitionRequired):
		return "Review and confirm the editable resume copy in the web app before adapting it."
	case errors.Is(err, matchModel.ErrJobDescriptionEmpty), errors.Is(err, matchModel.ErrResumeFileEmpty):
		return "Save a Job description and upload a resume file before checking the match."
	default:
		return ""
	}
}

func isAITool(name string) bool {
	switch name {
	case "parse_job_page", "tailor_resume", "apply_tailoring", "generate_cover_letter", "check_match", "get_resume_pdf_link":
		return true
	}
	return false
}
