package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	analyticsModel "github.com/andreypavlenko/jobber/modules/analytics/model"
	"github.com/andreypavlenko/jobber/modules/sharing/model"
	"github.com/andreypavlenko/jobber/modules/sharing/ports"
)

// tokenBytes gives 256 bits of entropy — share URLs must not be guessable.
const tokenBytes = 32

type SharingService struct {
	repo  ports.SharingRepository
	stats ports.StatsProvider
}

func NewSharingService(repo ports.SharingRepository, stats ports.StatsProvider) *SharingService {
	return &SharingService{repo: repo, stats: stats}
}

// Create freezes the user's current aggregates into a snapshot and stores it
// under a fresh unguessable token.
func (s *SharingService) Create(ctx context.Context, userID string) (*model.ShareDTO, error) {
	overview, err := s.stats.GetOverview(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load overview for share: %w", err)
	}
	funnel, err := s.stats.GetFunnel(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load funnel for share: %w", err)
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate share token: %w", err)
	}

	share := &model.SharedStats{
		UserID:   userID,
		Token:    token,
		Snapshot: buildSnapshot(overview, funnel, time.Now().UTC()),
	}
	if err := s.repo.Create(ctx, share); err != nil {
		return nil, err
	}
	return share.ToDTO(), nil
}

func (s *SharingService) List(ctx context.Context, userID string) ([]*model.ShareDTO, error) {
	shares, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	dtos := make([]*model.ShareDTO, len(shares))
	for i, share := range shares {
		dtos[i] = share.ToDTO()
	}
	return dtos, nil
}

// GetPublicByToken serves the unauthenticated share page.
func (s *SharingService) GetPublicByToken(ctx context.Context, token string) (*model.PublicShareDTO, error) {
	share, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return share.ToPublicDTO(), nil
}

func (s *SharingService) Delete(ctx context.Context, userID, shareID string) error {
	return s.repo.Delete(ctx, userID, shareID)
}

func buildSnapshot(overview *analyticsModel.OverviewAnalytics, funnel *analyticsModel.FunnelAnalytics, generatedAt time.Time) model.StatsSnapshot {
	stages := make([]model.FunnelStageSnapshot, len(funnel.Stages))
	for i, stage := range funnel.Stages {
		stages[i] = model.FunnelStageSnapshot{
			StageName:      stage.StageName,
			StageOrder:     stage.StageOrder,
			Count:          stage.Count,
			ConversionRate: stage.ConversionRate,
			DropOffRate:    stage.DropOffRate,
		}
	}
	return model.StatsSnapshot{
		SchemaVersion: model.SnapshotSchemaVersion,
		GeneratedAt:   generatedAt,
		Overview: model.OverviewSnapshot{
			TotalApplications:      overview.TotalApplications,
			ActiveApplications:     overview.ActiveApplications,
			ClosedApplications:     overview.ClosedApplications,
			RejectedApplications:   overview.RejectedApplications,
			ResponseRate:           overview.ResponseRate,
			AvgDaysToFirstResponse: overview.AvgDaysToFirstResponse,
		},
		Funnel: stages,
	}
}

func generateToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
