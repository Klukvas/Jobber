package model

import "errors"

// MatchScoreRequest is the request payload for checking resume-job match.
type MatchScoreRequest struct {
	JobID    string `json:"job_id" binding:"required"`
	ResumeID string `json:"resume_id" binding:"required"`
}

// MatchScoreCategory represents a scored category in the match analysis.
type MatchScoreCategory struct {
	Name    string `json:"name"`
	Score   int    `json:"score"`
	Details string `json:"details"`
}

// MatchScoreResponse is the response payload with match analysis results.
type MatchScoreResponse struct {
	OverallScore    int                  `json:"overall_score"`
	Categories      []MatchScoreCategory `json:"categories"`
	MissingKeywords []string             `json:"missing_keywords"`
	Strengths       []string             `json:"strengths"`
	Summary         string               `json:"summary"`
	FromCache       bool                 `json:"from_cache"`
}

// Errors
var (
	ErrAINotConfigured     = errors.New("AI matching is not configured")
	ErrJobDescriptionEmpty = errors.New("job description is empty")
	ErrResumeFileEmpty     = errors.New("resume has no file")
	ErrMatchFailed         = errors.New("failed to analyze match")
)

// ErrorCode represents a domain error code.
type ErrorCode string

const (
	CodeAINotConfigured     ErrorCode = "AI_NOT_CONFIGURED"
	CodeJobDescriptionEmpty ErrorCode = "JOB_DESCRIPTION_EMPTY"
	CodeResumeFileEmpty     ErrorCode = "RESUME_FILE_EMPTY"
	CodeMatchFailed         ErrorCode = "MATCH_FAILED"
	CodeValidationError     ErrorCode = "VALIDATION_ERROR"
	CodeJobNotFound         ErrorCode = "JOB_NOT_FOUND"
	CodeResumeNotFound      ErrorCode = "RESUME_NOT_FOUND"
)

// GetErrorCode maps domain errors to error codes.
func GetErrorCode(err error) ErrorCode {
	switch {
	case errors.Is(err, ErrAINotConfigured):
		return CodeAINotConfigured
	case errors.Is(err, ErrJobDescriptionEmpty):
		return CodeJobDescriptionEmpty
	case errors.Is(err, ErrResumeFileEmpty):
		return CodeResumeFileEmpty
	case errors.Is(err, ErrMatchFailed):
		return CodeMatchFailed
	default:
		return "INTERNAL_ERROR"
	}
}

// GetErrorMessage returns a user-friendly error message.
func GetErrorMessage(err error) string {
	switch {
	case errors.Is(err, ErrAINotConfigured):
		return "AI matching is not available"
	case errors.Is(err, ErrJobDescriptionEmpty):
		return "Job description is required for match analysis"
	case errors.Is(err, ErrResumeFileEmpty):
		return "Resume file is required for match analysis"
	case errors.Is(err, ErrMatchFailed):
		return "Failed to analyze match. Please try again."
	default:
		return "An unexpected error occurred"
	}
}
