package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	analyticsModel "github.com/andreypavlenko/jobber/modules/analytics/model"
	"github.com/andreypavlenko/jobber/modules/sharing/model"
	"github.com/andreypavlenko/jobber/modules/sharing/service"
	"github.com/gin-gonic/gin"
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
	return nil, model.ErrShareNotFound
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
type MockStatsProvider struct{}

func (m *MockStatsProvider) GetOverview(ctx context.Context, userID string) (*analyticsModel.OverviewAnalytics, error) {
	return &analyticsModel.OverviewAnalytics{
		TotalApplications:  127,
		ActiveApplications: 40,
		ClosedApplications: 87,
		ResponseRate:       18.5,
	}, nil
}

func (m *MockStatsProvider) GetFunnel(ctx context.Context, userID string) (*analyticsModel.FunnelAnalytics, error) {
	return &analyticsModel.FunnelAnalytics{
		Stages: []analyticsModel.FunnelStage{
			{StageName: "Applied", StageOrder: 1, Count: 127, ConversionRate: 100},
		},
	}, nil
}

const testFrontendURL = "https://jobber.example"

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func mockAuthMiddleware(userID string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	}
}

func newHandler(repo *MockSharingRepository) *SharingHandler {
	svc := service.NewSharingService(repo, &MockStatsProvider{})
	return NewSharingHandler(svc, testFrontendURL)
}

func sharedStatsFixture(token string) *model.SharedStats {
	return &model.SharedStats{
		ID:     "share-1",
		UserID: "user-123",
		Token:  token,
		Snapshot: model.StatsSnapshot{
			SchemaVersion: model.SnapshotSchemaVersion,
			GeneratedAt:   time.Now().UTC(),
			Overview: model.OverviewSnapshot{
				TotalApplications:  127,
				ActiveApplications: 40,
				ResponseRate:       18.5,
			},
		},
		CreatedAt: time.Now().UTC(),
	}
}

func TestSharingHandler_Create(t *testing.T) {
	userID := "user-123"

	t.Run("creates share", func(t *testing.T) {
		handler := newHandler(&MockSharingRepository{
			CreateFunc: func(ctx context.Context, share *model.SharedStats) error {
				return nil
			},
		})

		router := setupTestRouter()
		router.POST("/shares", mockAuthMiddleware(userID), handler.Create)

		req, _ := http.NewRequest(http.MethodPost, "/shares", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code)

		var response model.ShareDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		assert.NotEmpty(t, response.Token)
		assert.Equal(t, 127, response.Snapshot.Overview.TotalApplications)
	})

	t.Run("returns 409 when share limit reached", func(t *testing.T) {
		handler := newHandler(&MockSharingRepository{
			CreateFunc: func(ctx context.Context, share *model.SharedStats) error {
				return model.ErrShareLimitReached
			},
		})

		router := setupTestRouter()
		router.POST("/shares", mockAuthMiddleware(userID), handler.Create)

		req, _ := http.NewRequest(http.MethodPost, "/shares", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), string(model.CodeShareLimitReached))
	})
}

func TestSharingHandler_List(t *testing.T) {
	userID := "user-123"

	t.Run("lists own shares", func(t *testing.T) {
		handler := newHandler(&MockSharingRepository{
			ListByUserFunc: func(ctx context.Context, uid string) ([]*model.SharedStats, error) {
				assert.Equal(t, userID, uid)
				return []*model.SharedStats{sharedStatsFixture("token-1")}, nil
			},
		})

		router := setupTestRouter()
		router.GET("/shares", mockAuthMiddleware(userID), handler.List)

		req, _ := http.NewRequest(http.MethodGet, "/shares", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var response []model.ShareDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		require.Len(t, response, 1)
		assert.Equal(t, "token-1", response[0].Token)
	})
}

func TestSharingHandler_Delete(t *testing.T) {
	userID := "user-123"

	t.Run("deletes share", func(t *testing.T) {
		handler := newHandler(&MockSharingRepository{})

		router := setupTestRouter()
		router.DELETE("/shares/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/shares/share-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 404 for foreign or missing share", func(t *testing.T) {
		handler := newHandler(&MockSharingRepository{
			DeleteFunc: func(ctx context.Context, uid, shareID string) error {
				return model.ErrShareNotFound
			},
		})

		router := setupTestRouter()
		router.DELETE("/shares/:id", mockAuthMiddleware(userID), handler.Delete)

		req, _ := http.NewRequest(http.MethodDelete, "/shares/share-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), string(model.CodeShareNotFound))
	})
}

func TestSharingHandler_GetPublic(t *testing.T) {
	t.Run("serves snapshot without auth and without owner fields", func(t *testing.T) {
		handler := newHandler(&MockSharingRepository{
			GetByTokenFunc: func(ctx context.Context, token string) (*model.SharedStats, error) {
				return sharedStatsFixture(token), nil
			},
		})

		router := setupTestRouter()
		router.GET("/public/shares/:token", handler.GetPublic)

		req, _ := http.NewRequest(http.MethodGet, "/public/shares/some-token", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.NotContains(t, w.Body.String(), "user-123")
		assert.NotContains(t, w.Body.String(), "share-1")

		var response model.PublicShareDTO
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
		assert.Equal(t, 127, response.Snapshot.Overview.TotalApplications)
	})

	t.Run("returns 404 for unknown token", func(t *testing.T) {
		handler := newHandler(&MockSharingRepository{})

		router := setupTestRouter()
		router.GET("/public/shares/:token", handler.GetPublic)

		req, _ := http.NewRequest(http.MethodGet, "/public/shares/missing", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		assert.Contains(t, w.Body.String(), string(model.CodeShareNotFound))
	})
}

func TestSharingHandler_PreviewHTML(t *testing.T) {
	t.Run("serves OG meta for crawlers", func(t *testing.T) {
		handler := newHandler(&MockSharingRepository{
			GetByTokenFunc: func(ctx context.Context, token string) (*model.SharedStats, error) {
				return sharedStatsFixture(token), nil
			},
		})

		router := setupTestRouter()
		router.GET("/public/shares/:token/preview-html", handler.PreviewHTML)

		req, _ := http.NewRequest(http.MethodGet, "/public/shares/some-token/preview-html", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "noindex", w.Header().Get("X-Robots-Tag"))
		assert.Contains(t, w.Header().Get("Content-Type"), "text/html")

		body := w.Body.String()
		assert.Contains(t, body, "og:title")
		assert.Contains(t, body, "127 applications")
		assert.Contains(t, body, "18% response rate") // %.0f rounds half-to-even: 18.5 → 18
		assert.Contains(t, body, testFrontendURL+"/s/some-token")
		assert.Contains(t, body, testFrontendURL+"/og-image.png")
	})

	t.Run("returns noindex 404 page for unknown token", func(t *testing.T) {
		handler := newHandler(&MockSharingRepository{})

		router := setupTestRouter()
		router.GET("/public/shares/:token/preview-html", handler.PreviewHTML)

		req, _ := http.NewRequest(http.MethodGet, "/public/shares/missing/preview-html", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusNotFound, w.Code)
		assert.Equal(t, "noindex", w.Header().Get("X-Robots-Tag"))
		assert.Contains(t, w.Body.String(), "does not exist")
	})
}
