package model

// CreateSupportRequest is the payload for submitting a support ticket.
type CreateSupportRequest struct {
	Subject string `json:"subject" binding:"required,min=3,max=200"`
	Message string `json:"message" binding:"required,min=10,max=2000"`
	Page    string `json:"page"`
}

// Error codes for the support module.
type ErrorCode string

const (
	CodeValidationError ErrorCode = "VALIDATION_ERROR"
	CodeInternalError   ErrorCode = "INTERNAL_ERROR"
	CodeTelegramError   ErrorCode = "TELEGRAM_ERROR"
)
