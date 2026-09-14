package model

import "errors"

// ErrSupportUnavailable is returned when no support channel is configured for
// this deployment. The route stays registered either way — a POST to a route
// that does not exist answers 404 with the raw URL in the client's error, and
// a customer sending a support request deserves an explanation instead.
var ErrSupportUnavailable = errors.New("support channel is not configured")

// CreateSupportRequest is the payload for submitting a support ticket.
type CreateSupportRequest struct {
	Subject string `json:"subject" binding:"required,min=3,max=200"`
	Message string `json:"message" binding:"required,min=10,max=2000"`
	Page    string `json:"page"`
}

// Error codes for the support module.
type ErrorCode string

const (
	CodeValidationError    ErrorCode = "VALIDATION_ERROR"
	CodeInternalError      ErrorCode = "INTERNAL_ERROR"
	CodeTelegramError      ErrorCode = "TELEGRAM_ERROR"
	CodeSupportUnavailable ErrorCode = "SUPPORT_UNAVAILABLE"
)
