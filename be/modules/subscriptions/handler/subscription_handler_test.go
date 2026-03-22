package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockSubscriptionRepository implements ports.SubscriptionRepository
type MockSubscriptionRepository struct {
	GetByUserIDFunc               func(ctx context.Context, userID string) (*model.Subscription, error)
	GetByPaddleSubscriptionIDFunc func(ctx context.Context, paddleSubID string) (*model.Subscription, error)
	UpsertFunc                    func(ctx context.Context, sub *model.Subscription) error
	CountUserJobsFunc             func(ctx context.Context, userID string) (int, error)
	CountUserResumesFunc          func(ctx context.Context, userID string) (int, error)
	CountUserApplicationsFunc     func(ctx context.Context, userID string) (int, error)
	CountUserAIRequestsFunc       func(ctx context.Context, userID string) (int, error)
	CountUserJobParsesFunc        func(ctx context.Context, userID string) (int, error)
	RecordAIUsageFunc             func(ctx context.Context, userID string) error
	RecordJobParseUsageFunc       func(ctx context.Context, userID string) error
	CountUserResumeBuildersFunc   func(ctx context.Context, userID string) (int, error)
	CountUserCoverLettersFunc     func(ctx context.Context, userID string) (int, error)
	GetAllCountsFunc              func(ctx context.Context, userID string) (int, int, int, int, int, int, int, error)
	WebhookEventExistsFunc        func(ctx context.Context, eventID string) (bool, error)
	RecordWebhookEventFunc        func(ctx context.Context, eventID, eventType string) error
	TryClaimWebhookEventFunc      func(ctx context.Context, eventID, eventType string) (bool, error)
}

func (m *MockSubscriptionRepository) GetByUserID(ctx context.Context, userID string) (*model.Subscription, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return nil, model.ErrSubscriptionNotFound
}

func (m *MockSubscriptionRepository) GetByPaddleSubscriptionID(ctx context.Context, paddleSubID string) (*model.Subscription, error) {
	if m.GetByPaddleSubscriptionIDFunc != nil {
		return m.GetByPaddleSubscriptionIDFunc(ctx, paddleSubID)
	}
	return nil, model.ErrSubscriptionNotFound
}

func (m *MockSubscriptionRepository) Upsert(ctx context.Context, sub *model.Subscription) error {
	if m.UpsertFunc != nil {
		return m.UpsertFunc(ctx, sub)
	}
	return nil
}

func (m *MockSubscriptionRepository) CountUserJobs(ctx context.Context, userID string) (int, error) {
	if m.CountUserJobsFunc != nil {
		return m.CountUserJobsFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) CountUserResumes(ctx context.Context, userID string) (int, error) {
	if m.CountUserResumesFunc != nil {
		return m.CountUserResumesFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) CountUserApplications(ctx context.Context, userID string) (int, error) {
	if m.CountUserApplicationsFunc != nil {
		return m.CountUserApplicationsFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) CountUserAIRequestsThisMonth(ctx context.Context, userID string) (int, error) {
	if m.CountUserAIRequestsFunc != nil {
		return m.CountUserAIRequestsFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) CountUserJobParsesThisMonth(ctx context.Context, userID string) (int, error) {
	if m.CountUserJobParsesFunc != nil {
		return m.CountUserJobParsesFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) RecordAIUsage(ctx context.Context, userID string) error {
	if m.RecordAIUsageFunc != nil {
		return m.RecordAIUsageFunc(ctx, userID)
	}
	return nil
}

func (m *MockSubscriptionRepository) RecordJobParseUsage(ctx context.Context, userID string) error {
	if m.RecordJobParseUsageFunc != nil {
		return m.RecordJobParseUsageFunc(ctx, userID)
	}
	return nil
}

func (m *MockSubscriptionRepository) CountUserResumeBuilders(ctx context.Context, userID string) (int, error) {
	if m.CountUserResumeBuildersFunc != nil {
		return m.CountUserResumeBuildersFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) CountUserCoverLetters(ctx context.Context, userID string) (int, error) {
	if m.CountUserCoverLettersFunc != nil {
		return m.CountUserCoverLettersFunc(ctx, userID)
	}
	return 0, nil
}

func (m *MockSubscriptionRepository) GetAllCounts(ctx context.Context, userID string) (int, int, int, int, int, int, int, error) {
	if m.GetAllCountsFunc != nil {
		return m.GetAllCountsFunc(ctx, userID)
	}
	return 0, 0, 0, 0, 0, 0, 0, nil
}

func (m *MockSubscriptionRepository) WebhookEventExists(ctx context.Context, eventID string) (bool, error) {
	if m.WebhookEventExistsFunc != nil {
		return m.WebhookEventExistsFunc(ctx, eventID)
	}
	return false, nil
}

func (m *MockSubscriptionRepository) RecordWebhookEvent(ctx context.Context, eventID, eventType string) error {
	if m.RecordWebhookEventFunc != nil {
		return m.RecordWebhookEventFunc(ctx, eventID, eventType)
	}
	return nil
}

func (m *MockSubscriptionRepository) TryClaimWebhookEvent(ctx context.Context, eventID, eventType string) (bool, error) {
	if m.TryClaimWebhookEventFunc != nil {
		return m.TryClaimWebhookEventFunc(ctx, eventID, eventType)
	}
	return true, nil
}

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

func newTestService(repo *MockSubscriptionRepository) *service.SubscriptionService {
	return service.NewSubscriptionService(
		repo,
		"test-webhook-secret",
		"test-paddle-api-key",
		"price_pro_123",
		"price_ent_456",
		"client-token-xyz",
		"sandbox",
	)
}

func newTestSubscriptionHandler(repo *MockSubscriptionRepository) *SubscriptionHandler {
	svc := newTestService(repo)
	logger := zap.NewNop()
	return NewSubscriptionHandler(svc, logger)
}

func newTestWebhookHandler(repo *MockSubscriptionRepository) *WebhookHandler {
	svc := newTestService(repo)
	logger := zap.NewNop()
	return NewWebhookHandler(svc, logger)
}

// --- SubscriptionHandler Tests ---

func TestSubscriptionHandler_GetSubscription(t *testing.T) {
	userID := "user-123"

	t.Run("returns subscription successfully", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(ctx context.Context, uid string) (*model.Subscription, error) {
				return &model.Subscription{
					ID:     "sub-1",
					UserID: uid,
					Status: "active",
					Plan:   "pro",
				}, nil
			},
			GetAllCountsFunc: func(ctx context.Context, uid string) (int, int, int, int, int, int, int, error) {
				return 3, 1, 2, 5, 3, 1, 1, nil
			},
		}

		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.GET("/subscription", mockAuthMiddleware(userID), handler.GetSubscription)

		req, _ := http.NewRequest(http.MethodGet, "/subscription", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.SubscriptionDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "pro", response.Plan)
		assert.Equal(t, "active", response.Status)
		assert.Equal(t, 3, response.Usage.Jobs)
	})

	t.Run("auto-creates free subscription when not found", func(t *testing.T) {
		callCount := 0
		mockRepo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(ctx context.Context, uid string) (*model.Subscription, error) {
				callCount++
				if callCount == 1 {
					return nil, model.ErrSubscriptionNotFound
				}
				return &model.Subscription{
					ID:     "sub-1",
					UserID: uid,
					Status: "free",
					Plan:   "free",
				}, nil
			},
			UpsertFunc: func(ctx context.Context, sub *model.Subscription) error {
				return nil
			},
			GetAllCountsFunc: func(ctx context.Context, uid string) (int, int, int, int, int, int, int, error) {
				return 0, 0, 0, 0, 0, 0, 0, nil
			},
		}

		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.GET("/subscription", mockAuthMiddleware(userID), handler.GetSubscription)

		req, _ := http.NewRequest(http.MethodGet, "/subscription", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.SubscriptionDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "free", response.Plan)
	})

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.GET("/subscription", handler.GetSubscription)

		req, _ := http.NewRequest(http.MethodGet, "/subscription", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 500 when ensure free subscription fails", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(ctx context.Context, uid string) (*model.Subscription, error) {
				return nil, model.ErrSubscriptionNotFound
			},
			UpsertFunc: func(ctx context.Context, sub *model.Subscription) error {
				return assert.AnError
			},
		}

		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.GET("/subscription", mockAuthMiddleware(userID), handler.GetSubscription)

		req, _ := http.NewRequest(http.MethodGet, "/subscription", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("returns 500 when get subscription after ensure fails", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(ctx context.Context, uid string) (*model.Subscription, error) {
				return nil, model.ErrSubscriptionNotFound
			},
			UpsertFunc: func(ctx context.Context, sub *model.Subscription) error {
				return nil
			},
		}

		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.GET("/subscription", mockAuthMiddleware(userID), handler.GetSubscription)

		req, _ := http.NewRequest(http.MethodGet, "/subscription", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// GetByUserID returns ErrSubscriptionNotFound again after ensure -> 500
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("returns 500 for non-not-found error", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(ctx context.Context, uid string) (*model.Subscription, error) {
				return nil, assert.AnError
			},
		}

		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.GET("/subscription", mockAuthMiddleware(userID), handler.GetSubscription)

		req, _ := http.NewRequest(http.MethodGet, "/subscription", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSubscriptionHandler_GetCheckoutConfig(t *testing.T) {
	t.Run("returns checkout config", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.GET("/subscription/checkout-config", handler.GetCheckoutConfig)

		req, _ := http.NewRequest(http.MethodGet, "/subscription/checkout-config", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response model.CheckoutConfigDTO
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "client-token-xyz", response.ClientToken)
		assert.Equal(t, "sandbox", response.Environment)
		assert.Equal(t, "price_pro_123", response.Prices["pro"])
		assert.Equal(t, "price_ent_456", response.Prices["enterprise"])
	})
}

func TestSubscriptionHandler_CreatePortalSession(t *testing.T) {
	userID := "user-123"

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/portal", handler.CreatePortalSession)

		req, _ := http.NewRequest(http.MethodPost, "/subscription/portal", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		// GetByUserID returns subscription not found -> portal session creation fails
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/portal", mockAuthMiddleware(userID), handler.CreatePortalSession)

		req, _ := http.NewRequest(http.MethodPost, "/subscription/portal", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSubscriptionHandler_ChangePlan(t *testing.T) {
	userID := "user-123"

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/change-plan", handler.ChangePlan)

		body := `{"plan":"pro"}`
		req, _ := http.NewRequest(http.MethodPost, "/subscription/change-plan", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/change-plan", mockAuthMiddleware(userID), handler.ChangePlan)

		body := `invalid json`
		req, _ := http.NewRequest(http.MethodPost, "/subscription/change-plan", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for missing plan", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/change-plan", mockAuthMiddleware(userID), handler.ChangePlan)

		body := `{}`
		req, _ := http.NewRequest(http.MethodPost, "/subscription/change-plan", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for invalid plan name", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/change-plan", mockAuthMiddleware(userID), handler.ChangePlan)

		body := `{"plan":"invalid"}`
		req, _ := http.NewRequest(http.MethodPost, "/subscription/change-plan", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for free plan", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/change-plan", mockAuthMiddleware(userID), handler.ChangePlan)

		body := `{"plan":"free"}`
		req, _ := http.NewRequest(http.MethodPost, "/subscription/change-plan", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 500 when service fails for valid plan", func(t *testing.T) {
		// Service will fail because GetByUserID returns not found
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/change-plan", mockAuthMiddleware(userID), handler.ChangePlan)

		body := `{"plan":"pro"}`
		req, _ := http.NewRequest(http.MethodPost, "/subscription/change-plan", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSubscriptionHandler_CancelSubscription(t *testing.T) {
	userID := "user-123"

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/cancel", handler.CancelSubscription)

		req, _ := http.NewRequest(http.MethodPost, "/subscription/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("returns 500 when service fails", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/cancel", mockAuthMiddleware(userID), handler.CancelSubscription)

		req, _ := http.NewRequest(http.MethodPost, "/subscription/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestSubscriptionHandler_RegisterRoutes(t *testing.T) {
	mockRepo := &MockSubscriptionRepository{
		GetByUserIDFunc: func(ctx context.Context, uid string) (*model.Subscription, error) {
			return &model.Subscription{UserID: uid, Status: "free", Plan: "free"}, nil
		},
		GetAllCountsFunc: func(ctx context.Context, uid string) (int, int, int, int, int, int, int, error) {
			return 0, 0, 0, 0, 0, 0, 0, nil
		},
	}
	handler := newTestSubscriptionHandler(mockRepo)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"), true)

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/subscription"},
		{http.MethodGet, "/api/v1/subscription/checkout-config"},
		{http.MethodPost, "/api/v1/subscription/portal"},
		{http.MethodPost, "/api/v1/subscription/change-plan"},
		{http.MethodPost, "/api/v1/subscription/cancel"},
	}

	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			var body *bytes.Buffer
			if route.method == http.MethodPost {
				body = bytes.NewBufferString(`{"plan":"pro"}`)
			} else {
				body = bytes.NewBuffer(nil)
			}
			req, _ := http.NewRequest(route.method, route.path, body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.NotEqual(t, http.StatusNotFound, w.Code, "Route %s %s should be registered", route.method, route.path)
		})
	}
}

func TestSubscriptionHandler_RegisterRoutes_PaymentsDisabled(t *testing.T) {
	mockRepo := &MockSubscriptionRepository{
		GetByUserIDFunc: func(ctx context.Context, uid string) (*model.Subscription, error) {
			return &model.Subscription{UserID: uid, Status: "free", Plan: "free"}, nil
		},
		GetAllCountsFunc: func(ctx context.Context, uid string) (int, int, int, int, int, int, int, error) {
			return 0, 0, 0, 0, 0, 0, 0, nil
		},
	}
	handler := newTestSubscriptionHandler(mockRepo)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware("user-123"), false)

	t.Run("GET /subscription is registered", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/subscription", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	// When payments disabled, checkout/portal/change-plan/cancel should NOT be registered
	disabledRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/subscription/checkout-config"},
		{http.MethodPost, "/api/v1/subscription/portal"},
		{http.MethodPost, "/api/v1/subscription/change-plan"},
		{http.MethodPost, "/api/v1/subscription/cancel"},
	}

	for _, route := range disabledRoutes {
		t.Run(route.method+" "+route.path+" is NOT registered", func(t *testing.T) {
			body := bytes.NewBufferString(`{"plan":"pro"}`)
			req, _ := http.NewRequest(route.method, route.path, body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusNotFound, w.Code, "Route %s %s should NOT be registered when payments disabled", route.method, route.path)
		})
	}
}

// --- WebhookHandler Tests ---

func TestWebhookHandler_HandlePaddleWebhook(t *testing.T) {
	t.Run("returns 400 for empty body", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestWebhookHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/webhooks/paddle", handler.HandlePaddleWebhook)

		req, _ := http.NewRequest(http.MethodPost, "/webhooks/paddle", bytes.NewBufferString(""))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Empty body will fail signature verification
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for invalid signature", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestWebhookHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/webhooks/paddle", handler.HandlePaddleWebhook)

		body := `{"event_type":"subscription.created","data":{}}`
		req, _ := http.NewRequest(http.MethodPost, "/webhooks/paddle", bytes.NewBufferString(body))
		req.Header.Set("Paddle-Signature", "invalid-signature")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for missing signature", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestWebhookHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/webhooks/paddle", handler.HandlePaddleWebhook)

		body := `{"event_type":"subscription.created","data":{}}`
		req, _ := http.NewRequest(http.MethodPost, "/webhooks/paddle", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestWebhookHandler_RegisterRoutes(t *testing.T) {
	mockRepo := &MockSubscriptionRepository{}
	handler := newTestWebhookHandler(mockRepo)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1)

	t.Run("POST /api/v1/webhooks/paddle is registered", func(t *testing.T) {
		body := `{}`
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/webhooks/paddle", bytes.NewBufferString(body))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.NotEqual(t, http.StatusNotFound, w.Code, "Route should be registered")
	})
}
