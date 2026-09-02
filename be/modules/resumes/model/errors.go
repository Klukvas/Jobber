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
	default:
		return "Internal server error"
	}
}
