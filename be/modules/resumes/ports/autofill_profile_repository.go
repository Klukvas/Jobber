package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
)

// AutofillProfileRepository stores AI-extracted Autofill Profiles of Uploaded
// Resumes. The stored value is the raw ai.ParsedResume.
type AutofillProfileRepository interface {
	// Get returns the cached profile, or (nil, nil) when none exists yet.
	Get(ctx context.Context, userID, resumeID string) (*ai.ParsedResume, error)
	// Upsert stores the profile and reports whether a new row was created
	// (false = an existing row was overwritten by a concurrent extraction).
	Upsert(ctx context.Context, userID, resumeID string, profile *ai.ParsedResume, parserVersion int) (inserted bool, err error)
	// InvalidateByResume drops the profile when the resume file changes or the
	// resume is deleted (FK cascade is the safety net for the latter).
	InvalidateByResume(ctx context.Context, resumeID string) error
}
