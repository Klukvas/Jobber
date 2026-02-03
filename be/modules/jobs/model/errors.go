package model

import "errors"

var (
	// ErrJobNotFound is returned when a job is not found
	ErrJobNotFound = errors.New("job not found")

	// ErrJobTitleRequired is returned when job title is empty
	ErrJobTitleRequired = errors.New("job title is required")

	// ErrInvalidJobStatus is returned when an invalid job status is provided
	ErrInvalidJobStatus = errors.New("invalid job status")
)

// ErrorCode represents error codes
type ErrorCode string

const (
	CodeJobNotFound      ErrorCode = "JOB_NOT_FOUND"
	CodeJobTitleRequired ErrorCode = "JOB_TITLE_REQUIRED"
	CodeInvalidJobStatus ErrorCode = "INVALID_JOB_STATUS"
	CodeInternalError    ErrorCode = "INTERNAL_ERROR"
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
	default:
		return "Internal server error"
	}
}
