package model

import "errors"

var (
	// ErrJobNotFound is returned when a job is not found
	ErrJobNotFound = errors.New("job not found")

	// ErrJobTitleRequired is returned when job title is empty
	ErrJobTitleRequired = errors.New("job title is required")

	// ErrInvalidJobStatus is returned when an invalid job status is provided
	ErrInvalidJobStatus = errors.New("invalid job status")

	// ErrCompanyNotFound is returned when a referenced company does not exist or does not belong to the user
	ErrCompanyNotFound = errors.New("company not found")

	// ErrStageTemplateNotFound is returned when a stage template is not found
	ErrStageTemplateNotFound = errors.New("stage template not found")

	// ErrStageTemplateInUse is returned when a stage template is referenced by job stages
	ErrStageTemplateInUse = errors.New("stage template is still in use by jobs")

	// ErrStageTemplateNameExists is returned when a stage template name collides
	// with an existing one for the same user (UNIQUE(user_id, name))
	ErrStageTemplateNameExists = errors.New("stage template name already exists")

	// ErrInvalidPhase is returned when a stage template phase is not one of the fixed phases
	ErrInvalidPhase = errors.New("invalid stage template phase")

	// ErrInvalidMoveTarget is returned when a move request has an unusable target
	ErrInvalidMoveTarget = errors.New("invalid move target")

	// ErrJobStageNotFound is returned when a job stage is not found
	ErrJobStageNotFound = errors.New("job stage not found")

	// ErrInvalidStatus is returned when a stage status value is invalid
	ErrInvalidStatus = errors.New("invalid status")

	// ErrStageNameRequired is returned when a stage template name is empty
	ErrStageNameRequired = errors.New("stage name is required")

	// ErrBothResumeTypesSet is returned when both resume kinds are supplied at once
	ErrBothResumeTypesSet = errors.New("only one of resume_id or resume_builder_id can be set")

	// ErrResumeNotFound is returned when the referenced resume/resume builder does not exist or does not belong to the user
	ErrResumeNotFound = errors.New("resume not found")
)

// ErrorCode represents error codes
type ErrorCode string

const (
	CodeJobNotFound             ErrorCode = "JOB_NOT_FOUND"
	CodeJobTitleRequired        ErrorCode = "JOB_TITLE_REQUIRED"
	CodeInvalidJobStatus        ErrorCode = "INVALID_JOB_STATUS"
	CodeCompanyNotFound         ErrorCode = "COMPANY_NOT_FOUND"
	CodeStageTemplateNotFound   ErrorCode = "STAGE_TEMPLATE_NOT_FOUND"
	CodeInvalidPhase            ErrorCode = "INVALID_PHASE"
	CodeInvalidMoveTarget       ErrorCode = "INVALID_MOVE_TARGET"
	CodeStageTemplateInUse      ErrorCode = "STAGE_TEMPLATE_IN_USE"
	CodeStageTemplateNameExists ErrorCode = "STAGE_TEMPLATE_NAME_EXISTS"
	CodeJobStageNotFound        ErrorCode = "JOB_STAGE_NOT_FOUND"
	CodeInvalidStatus           ErrorCode = "INVALID_STATUS"
	CodeStageNameRequired       ErrorCode = "STAGE_NAME_REQUIRED"
	CodeBothResumeTypesSet      ErrorCode = "BOTH_RESUME_TYPES_SET"
	CodeResumeNotFound          ErrorCode = "RESUME_NOT_FOUND"
	CodeInternalError           ErrorCode = "INTERNAL_ERROR"
)

// GetErrorCode maps errors to error codes
func GetErrorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrJobNotFound):
		return CodeJobNotFound
	case errors.Is(err, ErrJobTitleRequired):
		return CodeJobTitleRequired
	case errors.Is(err, ErrInvalidJobStatus):
		return CodeInvalidJobStatus
	case errors.Is(err, ErrCompanyNotFound):
		return CodeCompanyNotFound
	case errors.Is(err, ErrStageTemplateNotFound):
		return CodeStageTemplateNotFound
	case errors.Is(err, ErrStageTemplateInUse):
		return CodeStageTemplateInUse
	case errors.Is(err, ErrStageTemplateNameExists):
		return CodeStageTemplateNameExists
	case errors.Is(err, ErrInvalidPhase):
		return CodeInvalidPhase
	case errors.Is(err, ErrInvalidMoveTarget):
		return CodeInvalidMoveTarget
	case errors.Is(err, ErrJobStageNotFound):
		return CodeJobStageNotFound
	case errors.Is(err, ErrInvalidStatus):
		return CodeInvalidStatus
	case errors.Is(err, ErrStageNameRequired):
		return CodeStageNameRequired
	case errors.Is(err, ErrBothResumeTypesSet):
		return CodeBothResumeTypesSet
	case errors.Is(err, ErrResumeNotFound):
		return CodeResumeNotFound
	default:
		return CodeInternalError
	}
}

// GetErrorMessage returns a user-friendly error message
func GetErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrJobNotFound):
		return "Job not found"
	case errors.Is(err, ErrJobTitleRequired):
		return "Job title is required"
	case errors.Is(err, ErrInvalidJobStatus):
		return "Invalid job status"
	case errors.Is(err, ErrCompanyNotFound):
		return "Company not found"
	case errors.Is(err, ErrStageTemplateNotFound):
		return "Stage template not found"
	case errors.Is(err, ErrStageTemplateInUse):
		return "Stage template is still in use by jobs and cannot be deleted"
	case errors.Is(err, ErrStageTemplateNameExists):
		return "A stage with this name already exists"
	case errors.Is(err, ErrInvalidPhase):
		return "Phase must be one of: wishlist, applied, in_progress, offer, rejected"
	case errors.Is(err, ErrInvalidMoveTarget):
		return "Move target must be a stage (with stage_template_id) or a phase"
	case errors.Is(err, ErrJobStageNotFound):
		return "Job stage not found"
	case errors.Is(err, ErrInvalidStatus):
		return "Invalid status"
	case errors.Is(err, ErrStageNameRequired):
		return "Stage name is required"
	case errors.Is(err, ErrBothResumeTypesSet):
		return "Only one of resume_id or resume_builder_id can be set"
	case errors.Is(err, ErrResumeNotFound):
		return "Resume not found"
	default:
		return "Internal server error"
	}
}
