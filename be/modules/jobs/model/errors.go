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

	// ErrUnsupportedJobSite is returned when the job site is not supported for import
	ErrUnsupportedJobSite = errors.New("unsupported job site")

	// ErrFetchFailed is returned when fetching the job URL fails
	ErrFetchFailed = errors.New("failed to fetch job URL")

	// ErrParseFailed is returned when parsing the job page fails
	ErrParseFailed = errors.New("failed to parse job data")

	// ErrInvalidURL is returned when the provided URL is invalid
	ErrInvalidURL = errors.New("invalid URL")

	// ErrInvalidBoardColumn is returned when an invalid board column is provided
	ErrInvalidBoardColumn = errors.New("invalid board column")
)

// ErrorCode represents error codes
type ErrorCode string

const (
	CodeJobNotFound      ErrorCode = "JOB_NOT_FOUND"
	CodeJobTitleRequired ErrorCode = "JOB_TITLE_REQUIRED"
	CodeInvalidJobStatus ErrorCode = "INVALID_JOB_STATUS"
	CodeCompanyNotFound    ErrorCode = "COMPANY_NOT_FOUND"
	CodeUnsupportedSite    ErrorCode = "UNSUPPORTED_JOB_SITE"
	CodeFetchFailed        ErrorCode = "FETCH_FAILED"
	CodeParseFailed        ErrorCode = "PARSE_FAILED"
	CodeInvalidURL         ErrorCode = "INVALID_URL"
	CodeInvalidBoardColumn ErrorCode = "INVALID_BOARD_COLUMN"
	CodeInternalError      ErrorCode = "INTERNAL_ERROR"
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
	case errors.Is(err, ErrUnsupportedJobSite):
		return CodeUnsupportedSite
	case errors.Is(err, ErrFetchFailed):
		return CodeFetchFailed
	case errors.Is(err, ErrParseFailed):
		return CodeParseFailed
	case errors.Is(err, ErrInvalidURL):
		return CodeInvalidURL
	case errors.Is(err, ErrInvalidBoardColumn):
		return CodeInvalidBoardColumn
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
	case errors.Is(err, ErrUnsupportedJobSite):
		return "This job site is not supported. Supported sites: LinkedIn, Indeed, DOU"
	case errors.Is(err, ErrFetchFailed):
		return "Failed to fetch the job page. Please check the URL and try again"
	case errors.Is(err, ErrParseFailed):
		return "Failed to extract job data from the page"
	case errors.Is(err, ErrInvalidURL):
		return "Please provide a valid URL"
	case errors.Is(err, ErrInvalidBoardColumn):
		return "Invalid board column. Must be one of: wishlist, applied, interview, offer, rejected"
	default:
		return "Internal server error"
	}
}
