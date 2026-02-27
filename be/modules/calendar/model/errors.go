package model

import "errors"

var (
	ErrNotConnected     = errors.New("google calendar not connected")
	ErrTokenExpired     = errors.New("google calendar token expired")
	ErrInvalidTimeRange = errors.New("invalid time range")
	ErrInvalidState     = errors.New("invalid oauth state")
	ErrStageNotFound    = errors.New("application stage not found")
	ErrEventNotFound    = errors.New("calendar event not found for this stage")
	ErrEncryptionFailed = errors.New("token encryption failed")
	ErrDecryptionFailed = errors.New("token decryption failed")
	ErrCalendarAPI        = errors.New("google calendar API error")
	ErrEventAlreadyExists = errors.New("calendar event already exists for this stage")
)

// ErrorCode represents calendar error codes
type ErrorCode string

const (
	CodeNotConnected     ErrorCode = "CALENDAR_NOT_CONNECTED"
	CodeTokenExpired     ErrorCode = "CALENDAR_TOKEN_EXPIRED"
	CodeInvalidTimeRange ErrorCode = "INVALID_TIME_RANGE"
	CodeInvalidState     ErrorCode = "INVALID_OAUTH_STATE"
	CodeStageNotFound    ErrorCode = "STAGE_NOT_FOUND"
	CodeEventNotFound    ErrorCode = "CALENDAR_EVENT_NOT_FOUND"
	CodeCalendarAPI        ErrorCode = "CALENDAR_API_ERROR"
	CodeEventAlreadyExists ErrorCode = "CALENDAR_EVENT_ALREADY_EXISTS"
	CodeInternalError      ErrorCode = "INTERNAL_ERROR"
)

// GetErrorCode maps errors to error codes
func GetErrorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrNotConnected):
		return CodeNotConnected
	case errors.Is(err, ErrTokenExpired):
		return CodeTokenExpired
	case errors.Is(err, ErrInvalidTimeRange):
		return CodeInvalidTimeRange
	case errors.Is(err, ErrInvalidState):
		return CodeInvalidState
	case errors.Is(err, ErrStageNotFound):
		return CodeStageNotFound
	case errors.Is(err, ErrEventNotFound):
		return CodeEventNotFound
	case errors.Is(err, ErrCalendarAPI):
		return CodeCalendarAPI
	case errors.Is(err, ErrEventAlreadyExists):
		return CodeEventAlreadyExists
	default:
		return CodeInternalError
	}
}

// GetErrorMessage returns a user-friendly error message
func GetErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrNotConnected):
		return "Google Calendar is not connected. Please connect it in Settings."
	case errors.Is(err, ErrTokenExpired):
		return "Google Calendar token expired. Please reconnect in Settings."
	case errors.Is(err, ErrInvalidTimeRange):
		return "Invalid time range for the event"
	case errors.Is(err, ErrInvalidState):
		return "Invalid OAuth state. Please try again."
	case errors.Is(err, ErrStageNotFound):
		return "Application stage not found"
	case errors.Is(err, ErrEventNotFound):
		return "No calendar event found for this stage"
	case errors.Is(err, ErrCalendarAPI):
		return "Google Calendar API error. Please try again."
	case errors.Is(err, ErrEventAlreadyExists):
		return "This stage already has a calendar event"
	default:
		return "Internal server error"
	}
}
