package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	rb "github.com/andreypavlenko/jobber/modules/resumebuilder/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"go.uber.org/zap"
)

type validationError string

func (e validationError) Error() string { return string(e) }

type requestIDKey struct{}

func (t *tools) toolError(ctx context.Context, tool string, err error) error {
	if message := aiFriendlyError(err); message != "" && isAITool(tool) {
		return errors.New(message)
	}
	var validation validationError
	var snapshot snapshotWriteError
	switch {
	case errors.As(err, &validation):
		return validation
	case errors.Is(err, jobModel.ErrJobNotFound):
		return errors.New("Job not found. Choose a Job from list_jobs.")
	case errors.Is(err, jobModel.ErrStageTemplateNotFound), errors.Is(err, jobModel.ErrInvalidMoveTarget):
		return errors.New("Stage not found. Use list_stages to choose one of your Stages.")
	case errors.Is(err, jobModel.ErrCompanyNotFound):
		return errors.New("Company not found in your account.")
	case errors.Is(err, jobModel.ErrJobTitleRequired):
		return errors.New("Please provide a Job title.")
	case errors.Is(err, rb.ErrResumeBuilderNotFound), errors.Is(err, rb.ErrNotOwner):
		return errors.New("Builder Resume or snapshot not found. Choose one from list_resumes or list_resume_snapshots.")
	case errors.Is(err, rb.ErrSectionEntryNotFound):
		return errors.New("Section entry not found in this Builder Resume. Use get_resume to choose an entry.")
	case errors.Is(err, rb.ErrUnsupportedSnapshotVersion):
		return errors.New("This snapshot uses an unsupported version and cannot be restored.")
	case errors.Is(err, rb.ErrInvalidSnapshotContent):
		return errors.New("This snapshot is incomplete or invalid and cannot be restored.")
	case errors.Is(err, subModel.ErrLimitReached):
		if t.resumes != nil && strings.Contains(tool, "resume") {
			return errors.New("Your plan's resume limit has been reached. Upgrade your plan to continue.")
		}
		return errors.New("Your plan's Jobs limit has been reached. Upgrade your plan to add more Jobs.")
	case errors.Is(err, subModel.ErrPaidFeature):
		return errors.New("Upgrade to Pro or Enterprise to use AI assistant tools.")
	case errors.As(err, &snapshot):
		t.logToolError(ctx, tool, snapshot.cause)
		return errors.New("Could not save an undo snapshot. Your resume was not changed. Please try again later.")
	default:
		t.logToolError(ctx, tool, err)
		return errors.New("Something went wrong. Please try again later.")
	}
}

func (t *tools) logToolError(ctx context.Context, tool string, err error) {
	// Error text can contain SQL parameters or private resume content.
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	t.logger.Error("MCP tool failed", zap.String("tool", tool), zap.String("request_id", requestID),
		zap.String("error_type", fmt.Sprintf("%T", err)), zap.Stack("stack"))
}
