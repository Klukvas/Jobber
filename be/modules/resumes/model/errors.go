package model

import "errors"

var (
	ErrResumeNotFound      = errors.New("resume not found")
	ErrResumeTitleRequired = errors.New("resume title is required")
	ErrResumeURLRequired   = errors.New("resume file URL is required")
	ErrResumeInUse         = errors.New("cannot delete resume: it is used in one or more applications")
	ErrInvalidFileURL      = errors.New("file URL is not allowed")
	ErrResumeUnreadable    = errors.New("could not read resume content")
	ErrResumeFileMissing   = errors.New("resume has no file attached")
	// ErrInvalidFileContent is returned when the uploaded object is not a PDF,
	// whatever its filename or declared content type claimed.
	ErrInvalidFileContent = errors.New("uploaded file is not a PDF")
	// ErrFileTooLarge is returned when the uploaded object exceeds the upload
	// size cap.
	ErrFileTooLarge = errors.New("uploaded file is too large")
	// ErrResumeAlreadyExists is returned when a resume row with the same id
	// already exists. It marks a repeated finalization of an upload that
	// already succeeded, which is answered idempotently rather than as a fault.
	ErrResumeAlreadyExists = errors.New("resume already exists")
	// ErrResumeLimitReached is returned when the insert itself refused because
	// the plan's resume allowance was already full at the moment it ran.
	//
	// Distinct from the subscription service's own limit error on purpose: this
	// one is only ever produced by the transaction that does the counting and
	// the writing together, which is the only place a *concurrent* finalize can
	// be caught. Both are answered to the client identically.
	ErrResumeLimitReached = errors.New("resume plan limit reached")
)

type ErrorCode string

const (
	CodeResumeNotFound      ErrorCode = "RESUME_NOT_FOUND"
	CodeResumeTitleRequired ErrorCode = "RESUME_TITLE_REQUIRED"
	CodeResumeURLRequired   ErrorCode = "RESUME_URL_REQUIRED"
	CodeResumeInUse         ErrorCode = "RESUME_IN_USE"
	CodeInvalidFileURL      ErrorCode = "INVALID_FILE_URL"
	CodeResumeUnreadable    ErrorCode = "RESUME_UNREADABLE"
	CodeResumeFileMissing   ErrorCode = "RESUME_FILE_MISSING"
	CodeInvalidFileContent  ErrorCode = "INVALID_FILE_CONTENT"
	CodeFileTooLarge        ErrorCode = "FILE_TOO_LARGE"
	CodePlanLimitReached    ErrorCode = "PLAN_LIMIT_REACHED"
	CodeInternalError       ErrorCode = "INTERNAL_ERROR"
)

func GetErrorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrResumeNotFound):
		return CodeResumeNotFound
	case errors.Is(err, ErrResumeTitleRequired):
		return CodeResumeTitleRequired
	case errors.Is(err, ErrResumeURLRequired):
		return CodeResumeURLRequired
	case errors.Is(err, ErrResumeInUse):
		return CodeResumeInUse
	case errors.Is(err, ErrInvalidFileURL):
		return CodeInvalidFileURL
	case errors.Is(err, ErrResumeUnreadable):
		return CodeResumeUnreadable
	case errors.Is(err, ErrResumeFileMissing):
		return CodeResumeFileMissing
	case errors.Is(err, ErrInvalidFileContent):
		return CodeInvalidFileContent
	case errors.Is(err, ErrFileTooLarge):
		return CodeFileTooLarge
	case errors.Is(err, ErrResumeLimitReached):
		return CodePlanLimitReached
	default:
		return CodeInternalError
	}
}

func GetErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrResumeNotFound):
		return "Resume not found"
	case errors.Is(err, ErrResumeTitleRequired):
		return "Resume title is required"
	case errors.Is(err, ErrResumeURLRequired):
		return "Resume file URL is required"
	case errors.Is(err, ErrResumeInUse):
		return "Cannot delete resume: it is used in one or more applications"
	case errors.Is(err, ErrInvalidFileURL):
		return "The provided file URL is not allowed"
	case errors.Is(err, ErrResumeUnreadable):
		return "Couldn't read this PDF. Try the Resume Builder instead."
	case errors.Is(err, ErrResumeFileMissing):
		return "This resume has no file attached"
	case errors.Is(err, ErrInvalidFileContent):
		return "That file isn't a PDF. Please upload a real PDF resume."
	case errors.Is(err, ErrFileTooLarge):
		return "That file is too large. PDFs must be 10 MB or smaller."
	case errors.Is(err, ErrResumeLimitReached):
		return "You have reached the limit for your current plan."
	default:
		return "Internal server error"
	}
}
