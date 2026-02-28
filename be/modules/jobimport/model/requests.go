package model

import "errors"

// ParseJobRequest is the request payload for parsing a job page.
type ParseJobRequest struct {
	PageText string `json:"page_text" binding:"required,min=10,max=50000"`
	PageURL  string `json:"page_url" binding:"required,url"`
}

// ParseJobResponse is the response payload with parsed job data.
type ParseJobResponse struct {
	Title       string  `json:"title"`
	CompanyName *string `json:"company_name,omitempty"`
	Source      *string `json:"source,omitempty"`
	URL         *string `json:"url,omitempty"`
	Description *string `json:"description,omitempty"`
}

// Errors
var (
	ErrAINotConfigured = errors.New("AI parsing is not configured")
	ErrParsingFailed   = errors.New("failed to parse job page")
)

// ErrorCode represents a domain error code.
type ErrorCode string

const (
	CodeAINotConfigured ErrorCode = "AI_NOT_CONFIGURED"
	CodeParsingFailed   ErrorCode = "PARSING_FAILED"
	CodeValidationError ErrorCode = "VALIDATION_ERROR"
)

// GetErrorCode maps domain errors to error codes.
func GetErrorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrAINotConfigured):
		return CodeAINotConfigured
	case errors.Is(err, ErrParsingFailed):
		return CodeParsingFailed
	default:
		return "INTERNAL_ERROR"
	}
}

// GetErrorMessage returns a user-friendly error message.
func GetErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrAINotConfigured):
		return "AI job parsing is not available. Please contact support."
	case errors.Is(err, ErrParsingFailed):
		return "Failed to parse the job page. Please try again."
	default:
		return "An unexpected error occurred"
	}
}
