package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/modules/analytics/model"
	"github.com/andreypavlenko/jobber/modules/analytics/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAnalyticsRepository implements the repository interface for testing
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

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// mockAuthMiddleware sets a user_id in the context for testing
func mockAuthMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func TestAnalyticsHandler_GetOverview(t *testing.T) {
	userID := "user-123"

	t.Run("returns overview successfully", func(t *testing.T) {
		expectedOverview := &model.OverviewAnalytics{
			TotalApplications:      50,
			ActiveApplications:     30,
			ClosedApplications:     20,
			ResponseRate:           40.0,
			AvgDaysToFirstResponse: 4.5,
		}

		mockRepo := &MockAnalyticsRepository{
			GetOverviewFunc: func(ctx context.Context, uid string) (*model.OverviewAnalytics, error) {
				return expectedOverview, nil
			},
		}

		svc := service.NewAnalyticsService(mockRepo)
		handler := NewAnalyticsHandler(svc)

		router := setupTestRouter()
		router.GET("/analytics/overview", mockAuthMiddleware(userID), handler.GetOverview)

		req, _ := http.NewRequest(http.MethodGet, "/analytics/overview", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.OverviewAnalytics
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, expectedOverview.TotalApplications, response.TotalApplications)
		assert.Equal(t, expectedOverview.ResponseRate, response.ResponseRate)
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		mockRepo := &MockAnalyticsRepository{
			GetOverviewFunc: func(ctx context.Context, uid string) (*model.OverviewAnalytics, error) {
				return nil, errors.New("database error")
			},
		}

		svc := service.NewAnalyticsService(mockRepo)
		handler := NewAnalyticsHandler(svc)

		router := setupTestRouter()
		router.GET("/analytics/overview", mockAuthMiddleware(userID), handler.GetOverview)

		req, _ := http.NewRequest(http.MethodGet, "/analytics/overview", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAnalyticsHandler_GetFunnel(t *testing.T) {
	userID := "user-123"

	t.Run("returns funnel successfully", func(t *testing.T) {
		expectedFunnel := &model.FunnelAnalytics{
			Stages: []model.FunnelStage{
				{StageName: "Applied", StageOrder: 1, Count: 100, ConversionRate: 100.0, DropOffRate: 0.0},
				{StageName: "Phone Screen", StageOrder: 2, Count: 60, ConversionRate: 60.0, DropOffRate: 40.0},
			},
		}

		mockRepo := &MockAnalyticsRepository{
			GetFunnelFunc: func(ctx context.Context, uid string) (*model.FunnelAnalytics, error) {
				return expectedFunnel, nil
			},
		}

		svc := service.NewAnalyticsService(mockRepo)
		handler := NewAnalyticsHandler(svc)

		router := setupTestRouter()
		router.GET("/analytics/funnel", mockAuthMiddleware(userID), handler.GetFunnel)

		req, _ := http.NewRequest(http.MethodGet, "/analytics/funnel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.FunnelAnalytics
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Stages, 2)
		assert.Equal(t, "Applied", response.Stages[0].StageName)
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		mockRepo := &MockAnalyticsRepository{
			GetFunnelFunc: func(ctx context.Context, uid string) (*model.FunnelAnalytics, error) {
				return nil, errors.New("database error")
			},
		}

		svc := service.NewAnalyticsService(mockRepo)
		handler := NewAnalyticsHandler(svc)

		router := setupTestRouter()
		router.GET("/analytics/funnel", mockAuthMiddleware(userID), handler.GetFunnel)

		req, _ := http.NewRequest(http.MethodGet, "/analytics/funnel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAnalyticsHandler_GetStageTime(t *testing.T) {
	userID := "user-123"

	t.Run("returns stage time successfully", func(t *testing.T) {
		expectedStageTime := &model.StageTimeAnalytics{
			Stages: []model.StageTimeMetrics{
				{StageName: "Applied", StageOrder: 1, AvgDays: 5.0, MinDays: 2.0, MaxDays: 10.0, ApplicationsCount: 80},
			},
		}

		mockRepo := &MockAnalyticsRepository{
			GetStageTimeFunc: func(ctx context.Context, uid string) (*model.StageTimeAnalytics, error) {
				return expectedStageTime, nil
			},
		}

		svc := service.NewAnalyticsService(mockRepo)
		handler := NewAnalyticsHandler(svc)

		router := setupTestRouter()
		router.GET("/analytics/stages", mockAuthMiddleware(userID), handler.GetStageTime)

		req, _ := http.NewRequest(http.MethodGet, "/analytics/stages", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.StageTimeAnalytics
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Stages, 1)
		assert.Equal(t, 5.0, response.Stages[0].AvgDays)
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		mockRepo := &MockAnalyticsRepository{
			GetStageTimeFunc: func(ctx context.Context, uid string) (*model.StageTimeAnalytics, error) {
				return nil, errors.New("database error")
			},
		}

		svc := service.NewAnalyticsService(mockRepo)
		handler := NewAnalyticsHandler(svc)

		router := setupTestRouter()
		router.GET("/analytics/stages", mockAuthMiddleware(userID), handler.GetStageTime)

		req, _ := http.NewRequest(http.MethodGet, "/analytics/stages", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAnalyticsHandler_GetResumeEffectiveness(t *testing.T) {
	userID := "user-123"

	t.Run("returns resume effectiveness successfully", func(t *testing.T) {
		expectedResumes := &model.ResumeAnalytics{
			Resumes: []model.ResumeEffectiveness{
				{
					ResumeID:          "resume-1",
					ResumeTitle:       "Tech Resume",
					ApplicationsCount: 35,
					ResponsesCount:    20,
					InterviewsCount:   10,
					ResponseRate:      57.14,
				},
			},
		}

		mockRepo := &MockAnalyticsRepository{
			GetResumeEffectivenessFunc: func(ctx context.Context, uid string) (*model.ResumeAnalytics, error) {
				return expectedResumes, nil
			},
		}

		svc := service.NewAnalyticsService(mockRepo)
		handler := NewAnalyticsHandler(svc)

		router := setupTestRouter()
		router.GET("/analytics/resumes", mockAuthMiddleware(userID), handler.GetResumeEffectiveness)

		req, _ := http.NewRequest(http.MethodGet, "/analytics/resumes", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.ResumeAnalytics
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Resumes, 1)
		assert.Equal(t, "Tech Resume", response.Resumes[0].ResumeTitle)
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		mockRepo := &MockAnalyticsRepository{
			GetResumeEffectivenessFunc: func(ctx context.Context, uid string) (*model.ResumeAnalytics, error) {
				return nil, errors.New("database error")
			},
		}

		svc := service.NewAnalyticsService(mockRepo)
		handler := NewAnalyticsHandler(svc)

		router := setupTestRouter()
		router.GET("/analytics/resumes", mockAuthMiddleware(userID), handler.GetResumeEffectiveness)

		req, _ := http.NewRequest(http.MethodGet, "/analytics/resumes", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAnalyticsHandler_GetSourceAnalytics(t *testing.T) {
	userID := "user-123"

	t.Run("returns source analytics successfully", func(t *testing.T) {
		expectedSources := &model.SourceAnalytics{
			Sources: []model.SourceMetrics{
				{SourceName: "LinkedIn", ApplicationsCount: 45, ResponsesCount: 25, ConversionRate: 55.56},
				{SourceName: "Referral", ApplicationsCount: 20, ResponsesCount: 15, ConversionRate: 75.0},
			},
		}

		mockRepo := &MockAnalyticsRepository{
			GetSourceAnalyticsFunc: func(ctx context.Context, uid string) (*model.SourceAnalytics, error) {
				return expectedSources, nil
			},
		}

		svc := service.NewAnalyticsService(mockRepo)
		handler := NewAnalyticsHandler(svc)

		router := setupTestRouter()
		router.GET("/analytics/sources", mockAuthMiddleware(userID), handler.GetSourceAnalytics)

		req, _ := http.NewRequest(http.MethodGet, "/analytics/sources", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.SourceAnalytics
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response.Sources, 2)
		assert.Equal(t, "LinkedIn", response.Sources[0].SourceName)
		assert.Equal(t, 75.0, response.Sources[1].ConversionRate)
	})

	t.Run("returns 500 on service error", func(t *testing.T) {
		mockRepo := &MockAnalyticsRepository{
			GetSourceAnalyticsFunc: func(ctx context.Context, uid string) (*model.SourceAnalytics, error) {
				return nil, errors.New("database error")
			},
		}

		svc := service.NewAnalyticsService(mockRepo)
		handler := NewAnalyticsHandler(svc)

		router := setupTestRouter()
		router.GET("/analytics/sources", mockAuthMiddleware(userID), handler.GetSourceAnalytics)

		req, _ := http.NewRequest(http.MethodGet, "/analytics/sources", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAnalyticsHandler_RegisterRoutes(t *testing.T) {
	mockRepo := &MockAnalyticsRepository{
		GetOverviewFunc: func(ctx context.Context, uid string) (*model.OverviewAnalytics, error) {
			return &model.OverviewAnalytics{}, nil
		},
		GetFunnelFunc: func(ctx context.Context, uid string) (*model.FunnelAnalytics, error) {
			return &model.FunnelAnalytics{}, nil
		},
		GetStageTimeFunc: func(ctx context.Context, uid string) (*model.StageTimeAnalytics, error) {
			return &model.StageTimeAnalytics{}, nil
		},
		GetResumeEffectivenessFunc: func(ctx context.Context, uid string) (*model.ResumeAnalytics, error) {
			return &model.ResumeAnalytics{}, nil
		},
		GetSourceAnalyticsFunc: func(ctx context.Context, uid string) (*model.SourceAnalytics, error) {
			return &model.SourceAnalytics{}, nil
		},
	}

	svc := service.NewAnalyticsService(mockRepo)
	handler := NewAnalyticsHandler(svc)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"))

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/analytics/overview"},
		{http.MethodGet, "/api/v1/analytics/funnel"},
		{http.MethodGet, "/api/v1/analytics/stages"},
		{http.MethodGet, "/api/v1/analytics/resumes"},
		{http.MethodGet, "/api/v1/analytics/sources"},
	}

	for _, route := range routes {
		t.Run(route.path, func(t *testing.T) {
			req, _ := http.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code, "Expected 200 for %s %s", route.method, route.path)
		})
	}
}
