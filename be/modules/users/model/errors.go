package model

import "errors"

var (
	// ErrUserNotFound is returned when a user is not found
	ErrUserNotFound = errors.New("user not found")

	// ErrUserAlreadyExists is returned when a user with the same email already exists
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrInvalidCredentials is returned when credentials are invalid
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrInvalidEmail is returned when email format is invalid
	ErrInvalidEmail = errors.New("invalid email format")

	// ErrInvalidPassword is returned when password is invalid
	ErrInvalidPassword = errors.New("invalid password")
)

// ErrorCode represents a machine-readable error code
type ErrorCode string

const (
	CodeUserNotFound        ErrorCode = "USER_NOT_FOUND"
	CodeUserAlreadyExists   ErrorCode = "USER_ALREADY_EXISTS"
	CodeInvalidCredentials  ErrorCode = "INVALID_CREDENTIALS"
	CodeInvalidEmail        ErrorCode = "INVALID_EMAIL"
	CodeInvalidPassword     ErrorCode = "INVALID_PASSWORD"
	CodeInternalError       ErrorCode = "INTERNAL_ERROR"
	CodeUnauthorized        ErrorCode = "UNAUTHORIZED"
	CodeValidationError     ErrorCode = "VALIDATION_ERROR"
)

// GetErrorCode maps errors to error codes
func GetErrorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrUserNotFound):
		return CodeUserNotFound
	case errors.Is(err, ErrUserAlreadyExists):
		return CodeUserAlreadyExists
	case errors.Is(err, ErrInvalidCredentials):
		return CodeInvalidCredentials
	case errors.Is(err, ErrInvalidEmail):
		return CodeInvalidEmail
	case errors.Is(err, ErrInvalidPassword):
		return CodeInvalidPassword
	default:
		return CodeInternalError
	}
}

// GetErrorMessage returns a user-friendly error message
func GetErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrUserNotFound):
		return "User not found"
	case errors.Is(err, ErrUserAlreadyExists):
		return "User with this email already exists"
	case errors.Is(err, ErrInvalidCredentials):
		return "Invalid email or password"
	case errors.Is(err, ErrInvalidEmail):
		return "Invalid email format"
	case errors.Is(err, ErrInvalidPassword):
		return "Password must be at least 8 characters"
	default:
		return "Internal server error"
	}
}
