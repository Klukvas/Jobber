package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/modules/analytics/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAnalyticsRepository is a mock implementation of the AnalyticsRepository interface
type MockAnalyticsRepository struct {
	GetOverviewFunc            func(ctx context.Context, userID string) (*model.OverviewAnalytics, error)
	GetFunnelFunc              func(ctx context.Context, userID string) (*model.FunnelAnalytics, error)
	GetStageTimeFunc           func(ctx context.Context, userID string) (*model.StageTimeAnalytics, error)
	GetResumeEffectivenessFunc func(ctx context.Context, userID string) (*model.ResumeAnalytics, error)
	GetSourceAnalyticsFunc     func(ctx context.Context, userID string) (*model.SourceAnalytics, error)
}

func (m *MockAnalyticsRepository) GetOverview(ctx context.Context, userID string) (*model.OverviewAnalytics, error) {
	if m.GetOverviewFunc != nil {
		return m.GetOverviewFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockAnalyticsRepository) GetFunnel(ctx context.Context, userID string) (*model.FunnelAnalytics, error) {
	if m.GetFunnelFunc != nil {
		return m.GetFunnelFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockAnalyticsRepository) GetStageTime(ctx context.Context, userID string) (*model.StageTimeAnalytics, error) {
	if m.GetStageTimeFunc != nil {
		return m.GetStageTimeFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockAnalyticsRepository) GetResumeEffectiveness(ctx context.Context, userID string) (*model.ResumeAnalytics, error) {
	if m.GetResumeEffectivenessFunc != nil {
		return m.GetResumeEffectivenessFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockAnalyticsRepository) GetSourceAnalytics(ctx context.Context, userID string) (*model.SourceAnalytics, error) {
	if m.GetSourceAnalyticsFunc != nil {
		return m.GetSourceAnalyticsFunc(ctx, userID)
	}
	return nil, nil
}

func TestAnalyticsService_GetOverview(t *testing.T) {
	userID := "user-123"

	t.Run("returns overview from repository", func(t *testing.T) {
		expectedOverview := &model.OverviewAnalytics{
			TotalApplications:      100,
			ActiveApplications:     60,
			ClosedApplications:     40,
			ResponseRate:           45.5,
			AvgDaysToFirstResponse: 5.2,
		}

		mockRepo := &MockAnalyticsRepository{
			GetOverviewFunc: func(ctx context.Context, uid string) (*model.OverviewAnalytics, error) {
				assert.Equal(t, userID, uid)
				return expectedOverview, nil
			},
		}

		service := NewAnalyticsService(mockRepo)
		result, err := service.GetOverview(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, expectedOverview, result)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockRepo := &MockAnalyticsRepository{
			GetOverviewFunc: func(ctx context.Context, uid string) (*model.OverviewAnalytics, error) {
				return nil, expectedError
			},
		}

		service := NewAnalyticsService(mockRepo)
		result, err := service.GetOverview(context.Background(), userID)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

func TestAnalyticsService_GetFunnel(t *testing.T) {
	userID := "user-123"

	t.Run("returns funnel from repository", func(t *testing.T) {
		expectedFunnel := &model.FunnelAnalytics{
			Stages: []model.FunnelStage{
				{StageName: "Applied", StageOrder: 1, Count: 100, ConversionRate: 100.0, DropOffRate: 0.0},
				{StageName: "Interview", StageOrder: 2, Count: 50, ConversionRate: 50.0, DropOffRate: 50.0},
			},
		}

		mockRepo := &MockAnalyticsRepository{
			GetFunnelFunc: func(ctx context.Context, uid string) (*model.FunnelAnalytics, error) {
				assert.Equal(t, userID, uid)
				return expectedFunnel, nil
			},
		}

		service := NewAnalyticsService(mockRepo)
		result, err := service.GetFunnel(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, expectedFunnel, result)
		assert.Len(t, result.Stages, 2)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockRepo := &MockAnalyticsRepository{
			GetFunnelFunc: func(ctx context.Context, uid string) (*model.FunnelAnalytics, error) {
				return nil, expectedError
			},
		}

		service := NewAnalyticsService(mockRepo)
		result, err := service.GetFunnel(context.Background(), userID)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

func TestAnalyticsService_GetStageTime(t *testing.T) {
	userID := "user-123"

	t.Run("returns stage time from repository", func(t *testing.T) {
		expectedStageTime := &model.StageTimeAnalytics{
			Stages: []model.StageTimeMetrics{
				{StageName: "Applied", StageOrder: 1, AvgDays: 3.5, MinDays: 1.0, MaxDays: 7.0, ApplicationsCount: 50},
				{StageName: "Interview", StageOrder: 2, AvgDays: 10.0, MinDays: 5.0, MaxDays: 21.0, ApplicationsCount: 30},
			},
		}

		mockRepo := &MockAnalyticsRepository{
			GetStageTimeFunc: func(ctx context.Context, uid string) (*model.StageTimeAnalytics, error) {
				assert.Equal(t, userID, uid)
				return expectedStageTime, nil
			},
		}

		service := NewAnalyticsService(mockRepo)
		result, err := service.GetStageTime(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, expectedStageTime, result)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockRepo := &MockAnalyticsRepository{
			GetStageTimeFunc: func(ctx context.Context, uid string) (*model.StageTimeAnalytics, error) {
				return nil, expectedError
			},
		}

		service := NewAnalyticsService(mockRepo)
		result, err := service.GetStageTime(context.Background(), userID)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

func TestAnalyticsService_GetResumeEffectiveness(t *testing.T) {
	userID := "user-123"

	t.Run("returns resume effectiveness from repository", func(t *testing.T) {
		expectedResumes := &model.ResumeAnalytics{
			Resumes: []model.ResumeEffectiveness{
				{
					ResumeID:          "resume-1",
					ResumeTitle:       "Software Engineer",
					ApplicationsCount: 25,
					ResponsesCount:    15,
					InterviewsCount:   8,
					ResponseRate:      60.0,
				},
			},
		}

		mockRepo := &MockAnalyticsRepository{
			GetResumeEffectivenessFunc: func(ctx context.Context, uid string) (*model.ResumeAnalytics, error) {
				assert.Equal(t, userID, uid)
				return expectedResumes, nil
			},
		}

		service := NewAnalyticsService(mockRepo)
		result, err := service.GetResumeEffectiveness(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, expectedResumes, result)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockRepo := &MockAnalyticsRepository{
			GetResumeEffectivenessFunc: func(ctx context.Context, uid string) (*model.ResumeAnalytics, error) {
				return nil, expectedError
			},
		}

		service := NewAnalyticsService(mockRepo)
		result, err := service.GetResumeEffectiveness(context.Background(), userID)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}

func TestAnalyticsService_GetSourceAnalytics(t *testing.T) {
	userID := "user-123"

	t.Run("returns source analytics from repository", func(t *testing.T) {
		expectedSources := &model.SourceAnalytics{
			Sources: []model.SourceMetrics{
				{SourceName: "LinkedIn", ApplicationsCount: 40, ResponsesCount: 20, ConversionRate: 50.0},
				{SourceName: "Indeed", ApplicationsCount: 30, ResponsesCount: 10, ConversionRate: 33.33},
			},
		}

		mockRepo := &MockAnalyticsRepository{
			GetSourceAnalyticsFunc: func(ctx context.Context, uid string) (*model.SourceAnalytics, error) {
				assert.Equal(t, userID, uid)
				return expectedSources, nil
			},
		}

		service := NewAnalyticsService(mockRepo)
		result, err := service.GetSourceAnalytics(context.Background(), userID)

		require.NoError(t, err)
		assert.Equal(t, expectedSources, result)
	})

	t.Run("returns error from repository", func(t *testing.T) {
		expectedError := errors.New("database error")

		mockRepo := &MockAnalyticsRepository{
			GetSourceAnalyticsFunc: func(ctx context.Context, uid string) (*model.SourceAnalytics, error) {
				return nil, expectedError
			},
		}

		service := NewAnalyticsService(mockRepo)
		result, err := service.GetSourceAnalytics(context.Background(), userID)

		assert.Nil(t, result)
		assert.Equal(t, expectedError, err)
	})
}
