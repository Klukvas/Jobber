package service

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/resumebuilder/ports"
)

type resourceLimitProvider interface {
	ResourceLimit(context.Context, string, string) (int, error)
}

// Resolve the ceiling before taking a database connection. The transaction uses
// its own connection for the authoritative count, even with a one-connection pool.
func resumeBuilderCeiling(ctx context.Context, limits LimitChecker, userID string) (int, error) {
	if limits == nil {
		return -1, nil
	}
	if provider, ok := limits.(resourceLimitProvider); ok {
		return provider.ResourceLimit(ctx, userID, "resume_builders")
	}
	// Compatibility for callers that provide only the original checker contract.
	return -1, limits.CheckLimit(ctx, userID, "resume_builders")
}

func runResumeCreation(ctx context.Context, repo ports.ResumeBuilderRepository, limits LimitChecker, userID string, fn func(ports.ResumeBuilderRepository) error) error {
	limit, err := resumeBuilderCeiling(ctx, limits, userID)
	if err != nil {
		return err
	}
	if creator, ok := repo.(ports.ResumeCreationRepository); ok {
		return creator.RunResumeCreationTransaction(ctx, userID, limit, fn)
	}
	return repo.RunInTransaction(ctx, fn)
}
