package model

import "errors"

var (
	ErrResumeNotFound      = errors.New("resume not found")
	ErrResumeTitleRequired = errors.New("resume title is required")
	ErrResumeURLRequired   = errors.New("resume file URL is required")
	ErrResumeInUse         = errors.New("cannot delete resume: it is used in one or more applications")
)

type ErrorCode string

const (
	CodeResumeNotFound      ErrorCode = "RESUME_NOT_FOUND"
	CodeResumeTitleRequired ErrorCode = "RESUME_TITLE_REQUIRED"
	CodeResumeURLRequired   ErrorCode = "RESUME_URL_REQUIRED"
	CodeResumeInUse         ErrorCode = "RESUME_IN_USE"
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
	default:
		return "Internal server error"
	}
}
