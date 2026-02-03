package service

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/analytics/model"
	"github.com/andreypavlenko/jobber/modules/analytics/ports"
)

type AnalyticsService struct {
	repo ports.AnalyticsRepository
}

func NewAnalyticsService(repo ports.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

// GetOverview returns high-level application statistics
func (s *AnalyticsService) GetOverview(ctx context.Context, userID string) (*model.OverviewAnalytics, error) {
	return s.repo.GetOverview(ctx, userID)
}

// GetFunnel returns stage-based funnel metrics
func (s *AnalyticsService) GetFunnel(ctx context.Context, userID string) (*model.FunnelAnalytics, error) {
	return s.repo.GetFunnel(ctx, userID)
}

// GetStageTime returns timing metrics per stage
func (s *AnalyticsService) GetStageTime(ctx context.Context, userID string) (*model.StageTimeAnalytics, error) {
	return s.repo.GetStageTime(ctx, userID)
}

// GetResumeEffectiveness returns effectiveness metrics per resume
func (s *AnalyticsService) GetResumeEffectiveness(ctx context.Context, userID string) (*model.ResumeAnalytics, error) {
	return s.repo.GetResumeEffectiveness(ctx, userID)
}

// GetSourceAnalytics returns metrics grouped by job source
func (s *AnalyticsService) GetSourceAnalytics(ctx context.Context, userID string) (*model.SourceAnalytics, error) {
	return s.repo.GetSourceAnalytics(ctx, userID)
}
