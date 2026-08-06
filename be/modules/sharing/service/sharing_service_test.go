package service

import (
	"context"
	"errors"
	"testing"
	"time"

	analyticsModel "github.com/andreypavlenko/jobber/modules/analytics/model"
	"github.com/andreypavlenko/jobber/modules/sharing/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSharingRepository implements ports.SharingRepository
type MockSharingRepository struct {
	CreateFunc     func(ctx context.Context, share *model.SharedStats) error
	GetByTokenFunc func(ctx context.Context, token string) (*model.SharedStats, error)
	ListByUserFunc func(ctx context.Context, userID string) ([]*model.SharedStats, error)
	DeleteFunc     func(ctx context.Context, userID, shareID string) error
}

func (m *MockSharingRepository) Create(ctx context.Context, share *model.SharedStats) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, share)
	}
	return nil
}

func (m *MockSharingRepository) GetByToken(ctx context.Context, token string) (*model.SharedStats, error) {
	if m.GetByTokenFunc != nil {
		return m.GetByTokenFunc(ctx, token)
	}
	return nil, nil
}

func (m *MockSharingRepository) ListByUser(ctx context.Context, userID string) ([]*model.SharedStats, error) {
	if m.ListByUserFunc != nil {
		return m.ListByUserFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockSharingRepository) Delete(ctx context.Context, userID, shareID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, userID, shareID)
	}
	return nil
}

// MockStatsProvider implements ports.StatsProvider
type MockStatsProvider struct {
	GetOverviewFunc func(ctx context.Context, userID string) (*analyticsModel.OverviewAnalytics, error)
	GetFunnelFunc   func(ctx context.Context, userID string) (*analyticsModel.FunnelAnalytics, error)
}

func (m *MockStatsProvider) GetOverview(ctx context.Context, userID string) (*analyticsModel.OverviewAnalytics, error) {
	if m.GetOverviewFunc != nil {
		return m.GetOverviewFunc(ctx, userID)
	}
	return &analyticsModel.OverviewAnalytics{}, nil
}

func (m *MockStatsProvider) GetFunnel(ctx context.Context, userID string) (*analyticsModel.FunnelAnalytics, error) {
	if m.GetFunnelFunc != nil {
		return m.GetFunnelFunc(ctx, userID)
	}
	return &analyticsModel.FunnelAnalytics{}, nil
}

func statsProviderFixture() *MockStatsProvider {
	return &MockStatsProvider{
		GetOverviewFunc: func(ctx context.Context, userID string) (*analyticsModel.OverviewAnalytics, error) {
			return &analyticsModel.OverviewAnalytics{
				TotalApplications:      127,
				ActiveApplications:     40,
				ClosedApplications:     87,
				ResponseRate:           18.5,
				AvgDaysToFirstResponse: 6.2,
			}, nil
		},
		GetFunnelFunc: func(ctx context.Context, userID string) (*analyticsModel.FunnelAnalytics, error) {
			return &analyticsModel.FunnelAnalytics{
				Stages: []analyticsModel.FunnelStage{
					{StageName: "Applied", StageOrder: 1, Count: 127, ConversionRate: 100, DropOffRate: 0},
					{StageName: "Interview", StageOrder: 2, Count: 20, ConversionRate: 15.75, DropOffRate: 84.25},
				},
			}, nil
		},
	}
}

func TestSharingService_Create(t *testing.T) {
	userID := "user-123"

	t.Run("freezes analytics into a snapshot", func(t *testing.T) {
		var created *model.SharedStats
		mockRepo := &MockSharingRepository{
			CreateFunc: func(ctx context.Context, share *model.SharedStats) error {
				created = share
				share.ID = "share-1"
				share.CreatedAt = time.Now().UTC()
				return nil
			},
		}

		svc := NewSharingService(mockRepo, statsProviderFixture())
		result, err := svc.Create(context.Background(), userID)

		require.NoError(t, err)
		require.NotNil(t, created)
		assert.Equal(t, "share-1", result.ID)
		assert.Equal(t, model.SnapshotSchemaVersion, result.Snapshot.SchemaVersion)
		assert.False(t, result.Snapshot.GeneratedAt.IsZero())
		assert.Equal(t, 127, result.Snapshot.Overview.TotalApplications)
		assert.Equal(t, 18.5, result.Snapshot.Overview.ResponseRate)
		require.Len(t, result.Snapshot.Funnel, 2)
		assert.Equal(t, "Interview", result.Snapshot.Funnel[1].StageName)
		assert.Equal(t, 15.75, result.Snapshot.Funnel[1].ConversionRate)
	})

	t.Run("generates unguessable unique tokens", func(t *testing.T) {
		mockRepo := &MockSharingRepository{}
		svc := NewSharingService(mockRepo, statsProviderFixture())

		first, err := svc.Create(context.Background(), userID)
		require.NoError(t, err)
		second, err := svc.Create(context.Background(), userID)
		require.NoError(t, err)

		// 32 random bytes base64url-encoded → 43 chars
		assert.Len(t, first.Token, 43)
		assert.NotEqual(t, first.Token, second.Token)
	})

	t.Run("propagates share limit error", func(t *testing.T) {
		mockRepo := &MockSharingRepository{
			CreateFunc: func(ctx context.Context, share *model.SharedStats) error {
				return model.ErrShareLimitReached
			},
		}

		svc := NewSharingService(mockRepo, statsProviderFixture())
		result, err := svc.Create(context.Background(), userID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrShareLimitReached)
	})

	t.Run("returns error when analytics fail", func(t *testing.T) {
		expectedError := errors.New("database error")
		stats := &MockStatsProvider{
			GetOverviewFunc: func(ctx context.Context, userID string) (*analyticsModel.OverviewAnalytics, error) {
				return nil, expectedError
			},
		}

		svc := NewSharingService(&MockSharingRepository{}, stats)
		result, err := svc.Create(context.Background(), userID)

		assert.Nil(t, result)
		assert.ErrorIs(t, err, expectedError)
	})
}

func TestSharingService_List(t *testing.T) {
	userID := "user-123"

	t.Run("returns shares as DTOs", func(t *testing.T) {
		mockRepo := &MockSharingRepository{
			ListByUserFunc: func(ctx context.Context, uid string) ([]*model.SharedStats, error) {
				assert.Equal(t, userID, uid)
				return []*model.SharedStats{
					{ID: "share-1", UserID: userID, Token: "token-1", CreatedAt: time.Now()},
					{ID: "share-2", UserID: userID, Token: "token-2", CreatedAt: time.Now()},
				}, nil
			},
		}

		svc := NewSharingService(mockRepo, statsProviderFixture())
		result, err := svc.List(context.Background(), userID)

		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Equal(t, "share-1", result[0].ID)
		assert.Equal(t, "token-2", result[1].Token)
	})

	t.Run("returns empty list", func(t *testing.T) {
		mockRepo := &MockSharingRepository{
			ListByUserFunc: func(ctx context.Context, uid string) ([]*model.SharedStats, error) {
				return nil, nil
			},
		}

		svc := NewSharingService(mockRepo, statsProviderFixture())
		result, err := svc.List(context.Background(), userID)

		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestSharingService_GetPublicByToken(t *testing.T) {
	t.Run("strips owner and id from public payload", func(t *testing.T) {
		mockRepo := &MockSharingRepository{
			GetByTokenFunc: func(ctx context.Context, token string) (*model.SharedStats, error) {
				return &model.SharedStats{
					ID:     "share-1",
					UserID: "user-123",
					Token:  token,
					Snapshot: model.StatsSnapshot{
						SchemaVersion: model.SnapshotSchemaVersion,
						Overview:      model.OverviewSnapshot{TotalApplications: 42},
					},
					CreatedAt: time.Now(),
				}, nil
			},
		}

		svc := NewSharingService(mockRepo, statsProviderFixture())
		result, err := svc.GetPublicByToken(context.Background(), "some-token")

		require.NoError(t, err)
		assert.Equal(t, 42, result.Snapshot.Overview.TotalApplications)
	})

	t.Run("returns not found error", func(t *testing.T) {
		mockRepo := &MockSharingRepository{
			GetByTokenFunc: func(ctx context.Context, token string) (*model.SharedStats, error) {
				return nil, model.ErrShareNotFound
			},
		}

		svc := NewSharingService(mockRepo, statsProviderFixture())
		result, err := svc.GetPublicByToken(context.Background(), "missing")

		assert.Nil(t, result)
		assert.ErrorIs(t, err, model.ErrShareNotFound)
	})
}

func TestSharingService_Delete(t *testing.T) {
	t.Run("delegates ownership-scoped delete", func(t *testing.T) {
		var gotUserID, gotShareID string
		mockRepo := &MockSharingRepository{
			DeleteFunc: func(ctx context.Context, userID, shareID string) error {
				gotUserID, gotShareID = userID, shareID
				return nil
			},
		}

		svc := NewSharingService(mockRepo, statsProviderFixture())
		err := svc.Delete(context.Background(), "user-123", "share-1")

		require.NoError(t, err)
		assert.Equal(t, "user-123", gotUserID)
		assert.Equal(t, "share-1", gotShareID)
	})

	t.Run("returns not found error", func(t *testing.T) {
		mockRepo := &MockSharingRepository{
			DeleteFunc: func(ctx context.Context, userID, shareID string) error {
				return model.ErrShareNotFound
			},
		}

		svc := NewSharingService(mockRepo, statsProviderFixture())
		err := svc.Delete(context.Background(), "user-123", "share-1")

		assert.ErrorIs(t, err, model.ErrShareNotFound)
	})
}
