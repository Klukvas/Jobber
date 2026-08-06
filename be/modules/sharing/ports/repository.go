package ports

import (
	"context"

	analyticsModel "github.com/andreypavlenko/jobber/modules/analytics/model"
	"github.com/andreypavlenko/jobber/modules/sharing/model"
)

// SharingRepository defines data access for shared stats snapshots.
type SharingRepository interface {
	// Create inserts the share while the user is under MaxActiveShares;
	// returns model.ErrShareLimitReached otherwise.
	Create(ctx context.Context, share *model.SharedStats) error

	GetByToken(ctx context.Context, token string) (*model.SharedStats, error)

	ListByUser(ctx context.Context, userID string) ([]*model.SharedStats, error)

	// Delete removes the share only when it belongs to userID;
	// returns model.ErrShareNotFound otherwise.
	Delete(ctx context.Context, userID, shareID string) error
}

// StatsProvider is the subset of analytics data a snapshot is built from.
// The analytics repository satisfies it structurally.
type StatsProvider interface {
	GetOverview(ctx context.Context, userID string) (*analyticsModel.OverviewAnalytics, error)
	GetFunnel(ctx context.Context, userID string) (*analyticsModel.FunnelAnalytics, error)
}
