package model

import "errors"

var (
	// ErrCompanyNotFound is returned when a company is not found
	ErrCompanyNotFound = errors.New("company not found")

	// ErrCompanyNameRequired is returned when company name is empty
	ErrCompanyNameRequired = errors.New("company name is required")
)

// ErrorCode represents error codes
type ErrorCode string

const (
	CodeCompanyNotFound     ErrorCode = "COMPANY_NOT_FOUND"
	CodeCompanyNameRequired ErrorCode = "COMPANY_NAME_REQUIRED"
	CodeInternalError       ErrorCode = "INTERNAL_ERROR"
)

// GetErrorCode maps errors to error codes
func GetErrorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrCompanyNotFound):
		return CodeCompanyNotFound
	case errors.Is(err, ErrCompanyNameRequired):
		return CodeCompanyNameRequired
	default:
		return CodeInternalError
	}
}

// GetErrorMessage returns a user-friendly error message
func GetErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrCompanyNotFound):
		return "Company not found"
	case errors.Is(err, ErrCompanyNameRequired):
		return "Company name is required"
	default:
		return "Internal server error"
	}
}
