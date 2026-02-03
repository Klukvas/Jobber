package ports

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/analytics/model"
)

// AnalyticsRepository defines the interface for analytics data access
type AnalyticsRepository interface {
	// GetOverview returns high-level application statistics
	GetOverview(ctx context.Context, userID string) (*model.OverviewAnalytics, error)

	// GetFunnel returns stage-based funnel metrics
	GetFunnel(ctx context.Context, userID string) (*model.FunnelAnalytics, error)

	// GetStageTime returns timing metrics per stage
	GetStageTime(ctx context.Context, userID string) (*model.StageTimeAnalytics, error)

	// GetResumeEffectiveness returns effectiveness metrics per resume
	GetResumeEffectiveness(ctx context.Context, userID string) (*model.ResumeAnalytics, error)

	// GetSourceAnalytics returns metrics grouped by job source
	GetSourceAnalytics(ctx context.Context, userID string) (*model.SourceAnalytics, error)
}
