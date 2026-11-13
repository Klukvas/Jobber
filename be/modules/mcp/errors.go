package mcp

import (
	"context"
	"errors"
	"fmt"

	jobModel "github.com/andreypavlenko/jobber/modules/jobs/model"
	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"go.uber.org/zap"
)

type validationError string

func (e validationError) Error() string { return string(e) }

type requestIDKey struct{}

func (t *tools) toolError(ctx context.Context, tool string, err error) error {
	var validation validationError
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
	case errors.Is(err, subModel.ErrLimitReached):
		return errors.New("Your plan's Jobs limit has been reached. Upgrade your plan to add more Jobs.")
	case errors.Is(err, subModel.ErrPaidFeature):
		return errors.New("Upgrade to Pro or Enterprise to use AI assistant tools.")
	default:
		// Error text can contain SQL parameters or user content. Preserve diagnostic
		// type and stack, never arguments, content, or token material.
		requestID, _ := ctx.Value(requestIDKey{}).(string)
		t.logger.Error("MCP tool failed", zap.String("tool", tool), zap.String("request_id", requestID),
			zap.String("error_type", fmt.Sprintf("%T", err)), zap.Stack("stack"))
		return errors.New("Something went wrong. Please try again later.")
	}
}
