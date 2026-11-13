package mcp

import (
	"errors"

	importModel "github.com/andreypavlenko/jobber/modules/jobimport/model"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	matchModel "github.com/andreypavlenko/jobber/modules/matchscore/model"
	rb "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
)

func auditErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var validation validationError
	switch {
	case errors.Is(err, jobModel.ErrJobNotFound), errors.Is(err, jobModel.ErrStageTemplateNotFound),
		errors.Is(err, jobModel.ErrInvalidMoveTarget), errors.Is(err, jobModel.ErrCompanyNotFound),
		errors.Is(err, rb.ErrResumeBuilderNotFound), errors.Is(err, rb.ErrNotOwner),
		errors.Is(err, rb.ErrSectionEntryNotFound), errors.Is(err, rb.ErrTailoringNotFound),
		errors.Is(err, resumeModel.ErrResumeNotFound):
		return "not_found"
	case errors.Is(err, subModel.ErrLimitReached):
		return "limit_reached"
	case errors.Is(err, subModel.ErrCreditUnavailable):
		return "no_credits"
	case errors.Is(err, subModel.ErrPaidFeature):
		return "paid_feature"
	case errors.Is(err, errAIUnavailable), errors.Is(err, errAIProvider),
		errors.Is(err, rb.ErrAIUnavailable), errors.Is(err, rb.ErrProposalStoreUnavailable),
		errors.Is(err, importModel.ErrAINotConfigured), errors.Is(err, subModel.ErrAIBudgetReached),
		errors.Is(err, subModel.ErrAIOperationConflict):
		return "ai_unavailable"
	case errors.Is(err, errPDFUnavailable):
		return "pdf_unavailable"
	case errors.Is(err, jobModel.ErrJobTitleRequired), errors.Is(err, rb.ErrUnsupportedSnapshotVersion),
		errors.Is(err, rb.ErrInvalidSnapshotContent), errors.Is(err, rb.ErrTailoringValidation),
		errors.Is(err, rb.ErrProposalExpired), errors.Is(err, rb.ErrSourceChanged), errors.Is(err, rb.ErrJobChanged),
		errors.Is(err, rb.ErrProposalAlreadyApplied), errors.Is(err, rb.ErrRecognitionRequired),
		errors.Is(err, matchModel.ErrJobDescriptionEmpty), errors.Is(err, matchModel.ErrResumeFileEmpty):
		return "validation"
	case errors.As(err, &validation):
		return auditValidationCode(validation)
	default:
		return "internal"
	}
}

func auditValidationCode(err validationError) string {
	// aiCallError already translates known provider/domain failures into this
	// validation type. Compare only the existing fixed classification messages;
	// arbitrary validation text never becomes an audit field.
	for _, known := range []error{
		errAIUnavailable, errAIProvider, errPDFUnavailable, subModel.ErrCreditUnavailable,
		subModel.ErrPaidFeature, subModel.ErrAIBudgetReached, rb.ErrTailoringNotFound,
	} {
		if string(err) == aiFriendlyError(known) {
			return auditErrorCode(known)
		}
	}
	return "validation"
}
