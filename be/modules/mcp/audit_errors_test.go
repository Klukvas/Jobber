package mcp

import (
	"errors"
	"fmt"
	"testing"

	importModel "github.com/andreypavlenko/jobber/modules/jobimport/model"
	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	matchModel "github.com/andreypavlenko/jobber/modules/matchscore/model"
	rb "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	resumeModel "github.com/andreypavlenko/jobber/modules/resumes/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/require"
)

func TestAuditErrorCode(t *testing.T) {
	tests := []struct {
		name   string
		errors []error
		code   string
	}{
		{"not found", []error{jobModel.ErrJobNotFound, jobModel.ErrStageTemplateNotFound, jobModel.ErrInvalidMoveTarget,
			jobModel.ErrCompanyNotFound, rb.ErrResumeBuilderNotFound, rb.ErrNotOwner, rb.ErrSectionEntryNotFound,
			rb.ErrTailoringNotFound, resumeModel.ErrResumeNotFound}, "not_found"},
		{"validation", []error{validationError("private arguments"), jobModel.ErrJobTitleRequired, rb.ErrUnsupportedSnapshotVersion,
			rb.ErrInvalidSnapshotContent, rb.ErrTailoringValidation, rb.ErrProposalExpired, rb.ErrSourceChanged,
			rb.ErrJobChanged, rb.ErrProposalAlreadyApplied, rb.ErrRecognitionRequired, matchModel.ErrJobDescriptionEmpty,
			matchModel.ErrResumeFileEmpty}, "validation"},
		{"limits", []error{subModel.ErrLimitReached}, "limit_reached"},
		{"credits", []error{subModel.ErrCreditUnavailable}, "no_credits"},
		{"paid", []error{subModel.ErrPaidFeature}, "paid_feature"},
		{"AI unavailable", []error{errAIUnavailable, errAIProvider, rb.ErrAIUnavailable, rb.ErrProposalStoreUnavailable,
			importModel.ErrAINotConfigured, subModel.ErrAIBudgetReached, subModel.ErrAIOperationConflict}, "ai_unavailable"},
		{"PDF unavailable", []error{errPDFUnavailable}, "pdf_unavailable"},
		{"internal", []error{errors.New("SQL with PII"), snapshotWriteError{cause: errors.New("private resume")}}, "internal"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, err := range test.errors {
				require.Equal(t, test.code, auditErrorCode(fmt.Errorf("private wrapped context: %w", err)))
			}
		})
	}
	require.Empty(t, auditErrorCode(nil))
}

func TestAuditTranslatedAIErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{"unavailable", errAIUnavailable, "ai_unavailable"},
		{"provider", errors.New("private provider response"), "ai_unavailable"},
		{"credits", subModel.ErrCreditUnavailable, "no_credits"},
		{"paid", subModel.ErrPaidFeature, "paid_feature"},
		{"budget", subModel.ErrAIBudgetReached, "ai_unavailable"},
		{"missing proposal", rb.ErrTailoringNotFound, "not_found"},
		{"invalid selection", rb.ErrTailoringValidation, "validation"},
		{"source changed", rb.ErrSourceChanged, "validation"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.code, auditErrorCode(aiCallError(test.err)))
		})
	}
}
