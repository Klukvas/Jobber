package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/matchscore/model"
)

// MatchScoreCacheRepository provides access to cached match score results.
type MatchScoreCacheRepository interface {
	// Get returns a cached result for the given user/job/resume triple, or nil if not found.
	Get(ctx context.Context, userID, jobID, resumeID string) (*model.MatchScoreResponse, error)
	// Upsert inserts or replaces the cached result.
	Upsert(ctx context.Context, userID, jobID, resumeID string, result *model.MatchScoreResponse) error
	// InvalidateByJob deletes all cached results for a given job.
	InvalidateByJob(ctx context.Context, jobID string) error
	// InvalidateByResume deletes all cached results for a given resume.
	InvalidateByResume(ctx context.Context, resumeID string) error
}
