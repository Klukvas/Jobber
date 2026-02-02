package model

import "errors"

var (
	ErrApplicationNotFound    = errors.New("application not found")
	ErrStageTemplateNotFound  = errors.New("stage template not found")
	ErrApplicationStageNotFound = errors.New("application stage not found")
	ErrInvalidStatus          = errors.New("invalid status")
	ErrStageNameRequired      = errors.New("stage name is required")
)

type ErrorCode string

const (
	CodeApplicationNotFound     ErrorCode = "APPLICATION_NOT_FOUND"
	CodeStageTemplateNotFound   ErrorCode = "STAGE_TEMPLATE_NOT_FOUND"
	CodeApplicationStageNotFound ErrorCode = "APPLICATION_STAGE_NOT_FOUND"
	CodeInvalidStatus           ErrorCode = "INVALID_STATUS"
	CodeStageNameRequired       ErrorCode = "STAGE_NAME_REQUIRED"
	CodeInternalError           ErrorCode = "INTERNAL_ERROR"
)

func GetErrorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrApplicationNotFound):
		return CodeApplicationNotFound
	case errors.Is(err, ErrStageTemplateNotFound):
		return CodeStageTemplateNotFound
	case errors.Is(err, ErrApplicationStageNotFound):
		return CodeApplicationStageNotFound
	case errors.Is(err, ErrInvalidStatus):
		return CodeInvalidStatus
	case errors.Is(err, ErrStageNameRequired):
		return CodeStageNameRequired
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
	case errors.Is(err, ErrApplicationStageNotFound):
		return "Application stage not found"
	case errors.Is(err, ErrInvalidStatus):
		return "Invalid status"
	case errors.Is(err, ErrStageNameRequired):
		return "Stage name is required"
	default:
		return "Internal server error"
	}
}
