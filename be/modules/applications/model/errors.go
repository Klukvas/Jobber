package model

import "errors"

var (
	ErrApplicationNotFound      = errors.New("application not found")
	ErrStageTemplateNotFound    = errors.New("stage template not found")
	ErrStageTemplateInUse       = errors.New("stage template is still in use by applications")
	ErrApplicationStageNotFound = errors.New("application stage not found")
	ErrInvalidStatus            = errors.New("invalid status")
	ErrStageNameRequired        = errors.New("stage name is required")
	ErrBothResumeTypesSet       = errors.New("only one of resume_id or resume_builder_id can be set")
)

type ErrorCode string

const (
	CodeApplicationNotFound      ErrorCode = "APPLICATION_NOT_FOUND"
	CodeStageTemplateNotFound    ErrorCode = "STAGE_TEMPLATE_NOT_FOUND"
	CodeStageTemplateInUse       ErrorCode = "STAGE_TEMPLATE_IN_USE"
	CodeApplicationStageNotFound ErrorCode = "APPLICATION_STAGE_NOT_FOUND"
	CodeInvalidStatus            ErrorCode = "INVALID_STATUS"
	CodeStageNameRequired        ErrorCode = "STAGE_NAME_REQUIRED"
	CodeBothResumeTypesSet       ErrorCode = "BOTH_RESUME_TYPES_SET"
	CodeInternalError            ErrorCode = "INTERNAL_ERROR"
)

func GetErrorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrApplicationNotFound):
		return CodeApplicationNotFound
	case errors.Is(err, ErrStageTemplateNotFound):
		return CodeStageTemplateNotFound
	case errors.Is(err, ErrStageTemplateInUse):
		return CodeStageTemplateInUse
	case errors.Is(err, ErrApplicationStageNotFound):
		return CodeApplicationStageNotFound
	case errors.Is(err, ErrInvalidStatus):
		return CodeInvalidStatus
	case errors.Is(err, ErrStageNameRequired):
		return CodeStageNameRequired
	case errors.Is(err, ErrBothResumeTypesSet):
		return CodeBothResumeTypesSet
	default:
		return CodeInternalError
	}
}

func GetErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrApplicationNotFound):
		return "Application not found"
	case errors.Is(err, ErrStageTemplateNotFound):
		return "Stage template not found"
	case errors.Is(err, ErrStageTemplateInUse):
		return "Stage template is still in use by applications and cannot be deleted"
	case errors.Is(err, ErrApplicationStageNotFound):
		return "Application stage not found"
	case errors.Is(err, ErrInvalidStatus):
		return "Invalid status"
	case errors.Is(err, ErrStageNameRequired):
		return "Stage name is required"
	case errors.Is(err, ErrBothResumeTypesSet):
		return "Only one of resume_id or resume_builder_id can be set"
	default:
		return "Internal server error"
	}
}
