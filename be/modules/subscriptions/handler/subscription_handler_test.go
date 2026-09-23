package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// MockSubscriptionRepository implements ports.SubscriptionRepository
type MockSubscriptionRepository struct {
	GetByUserIDFunc                 func(ctx context.Context, userID string) (*model.Subscription, error)
	GetByExternalSubscriptionIDFunc func(ctx context.Context, externalSubID string) (*model.Subscription, error)
	GetByExternalAccountIDFunc      func(ctx context.Context, externalAccountID string) (*model.Subscription, error)
	EnsureFreeFunc                  func(ctx context.Context, userID string) error
	LinkExternalAccountFunc         func(ctx context.Context, userID, externalAccountID string) error
	GetUserContactFunc              func(ctx context.Context, userID string) (*model.UserContact, error)
	CountUserJobsFunc               func(ctx context.Context, userID string) (int, error)
	CountUserResumesFunc            func(ctx context.Context, userID string) (int, error)
	CountUserAIRequestsFunc         func(ctx context.Context, userID string) (int, error)
	CountUserJobParsesFunc          func(ctx context.Context, userID string) (int, error)
	RecordAIUsageFunc               func(ctx context.Context, userID string) error
	RecordJobParseUsageFunc         func(ctx context.Context, userID string) error
	RecordResumeAutofillUsageFunc   func(ctx context.Context, userID string) error
	CountUserResumeBuildersFunc     func(ctx context.Context, userID string) (int, error)
	CountUserCoverLettersFunc       func(ctx context.Context, userID string) (int, error)
	GetAllCountsFunc                func(ctx context.Context, userID string) (int, int, int, int, int, int, error)
	ApplySubscriptionEventFunc      func(ctx context.Context, eventID, eventType string, sub *model.Subscription) (model.WebhookApplyOutcome, error)
}

func (m *MockSubscriptionRepository) GetByUserID(ctx context.Context, userID string) (*model.Subscription, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return nil, model.ErrSubscriptionNotFound
}

func (m *MockSubscriptionRepository) GetByExternalSubscriptionID(ctx context.Context, externalSubID string) (*model.Subscription, error) {
	if m.GetByExternalSubscriptionIDFunc != nil {
		return m.GetByExternalSubscriptionIDFunc(ctx, externalSubID)
	}
	return nil, model.ErrSubscriptionNotFound
}

func (m *MockSubscriptionRepository) GetByExternalAccountID(ctx context.Context, externalAccountID string) (*model.Subscription, error) {
	if m.GetByExternalAccountIDFunc != nil {
		return m.GetByExternalAccountIDFunc(ctx, externalAccountID)
	}
	return nil, model.ErrSubscriptionNotFound
}

func (m *MockSubscriptionRepository) EnsureFree(ctx context.Context, userID string) error {
	if m.EnsureFreeFunc != nil {
		return m.EnsureFreeFunc(ctx, userID)
	}
	return nil
}

func (m *MockSubscriptionRepository) LinkExternalAccount(ctx context.Context, userID, externalAccountID string) error {
	if m.LinkExternalAccountFunc != nil {
		return m.LinkExternalAccountFunc(ctx, userID, externalAccountID)
	}
	return nil
}

func (m *MockSubscriptionRepository) GetUserContact(ctx context.Context, userID string) (*model.UserContact, error) {
	if m.GetUserContactFunc != nil {
		return m.GetUserContactFunc(ctx, userID)
	}
	return &model.UserContact{Email: "buyer@example.com", Name: "Test Buyer", Locale: "en"}, nil
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

func (m *MockSubscriptionRepository) RecordResumeAutofillUsage(ctx context.Context, userID string) error {
	if m.RecordResumeAutofillUsageFunc != nil {
		return m.RecordResumeAutofillUsageFunc(ctx, userID)
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

func (m *MockSubscriptionRepository) GetAllCounts(ctx context.Context, userID string) (int, int, int, int, int, int, error) {
	if m.GetAllCountsFunc != nil {
		return m.GetAllCountsFunc(ctx, userID)
	}
	return 0, 0, 0, 0, 0, 0, nil
}

func (m *MockSubscriptionRepository) ApplySubscriptionEvent(
	ctx context.Context, eventID, eventType string, sub *model.Subscription,
) (model.WebhookApplyOutcome, error) {
	if m.ApplySubscriptionEventFunc != nil {
		return m.ApplySubscriptionEventFunc(ctx, eventID, eventType, sub)
	}
	return model.WebhookApplied, nil
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

const (
	// A real user ID is a UUID (`users.id` is a `uuid` column), and the checkout
	// path now depends on that: the order tag it writes is a MAC over the
	// canonical UUID, so a synthetic non-UUID id would fail the session it could
	// never make resolvable.
	testUserID         = "550e8400-e29b-41d4-a716-446655440000"
	testWebhookSecret  = "test-webhook-secret"
	testProPath        = "jobber-pro"
	testEnterprisePath = "jobber-enterprise"
	testCheckoutPath   = "fluxlab/popup-jobber"
)

func newTestService(repo *MockSubscriptionRepository) *service.SubscriptionService {
	return service.NewSubscriptionService(
		repo,
		// No API credentials: these handler tests exercise HTTP wiring, and an
		// unconfigured client fails fast instead of reaching the network.
		fastspring.NewClient(fastspring.Config{}),
		service.BillingConfig{
			WebhookSecret:         testWebhookSecret,
			CheckoutPath:          testCheckoutPath,
			Environment:           service.EnvironmentTest,
			ProProductPath:        testProPath,
			EnterpriseProductPath: testEnterprisePath,
		},
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

// newObservedWebhookHandler is newTestWebhookHandler with a logger whose
// entries can be read back, for the outcomes that are only visible in the log.
func newObservedWebhookHandler(repo *MockSubscriptionRepository) (*WebhookHandler, *observer.ObservedLogs) {
	core, logs := observer.New(zap.InfoLevel)
	return NewWebhookHandler(newTestService(repo), zap.New(core)), logs
}

// --- SubscriptionHandler Tests ---

func TestSubscriptionHandler_GetSubscription(t *testing.T) {
	userID := testUserID

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
			GetAllCountsFunc: func(ctx context.Context, uid string) (int, int, int, int, int, int, error) {
				return 3, 1, 5, 3, 1, 1, nil
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
			GetAllCountsFunc: func(ctx context.Context, uid string) (int, int, int, int, int, int, error) {
				return 0, 0, 0, 0, 0, 0, nil
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
			EnsureFreeFunc: func(ctx context.Context, uid string) error {
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
		assert.Equal(t, service.Provider, response.Provider)
		assert.Equal(t, service.EnvironmentTest, response.Environment)
		assert.Equal(t, []string{"pro", "enterprise"}, response.Plans)
		assert.NotContains(t, w.Body.String(), testWebhookSecret,
			"the public config must not expose any credential")
	})
}

func TestSubscriptionHandler_CreateCheckoutSession(t *testing.T) {
	userID := testUserID

	t.Run("returns 401 when not authenticated", func(t *testing.T) {
		handler := newTestSubscriptionHandler(&MockSubscriptionRepository{})

		router := setupTestRouter()
		router.POST("/subscription/checkout-session", handler.CreateCheckoutSession)

		req, _ := http.NewRequest(http.MethodPost, "/subscription/checkout-session", bytes.NewBufferString(`{"plan":"pro"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("rejects an unknown plan", func(t *testing.T) {
		handler := newTestSubscriptionHandler(&MockSubscriptionRepository{})

		router := setupTestRouter()
		router.POST("/subscription/checkout-session", mockAuthMiddleware(userID), handler.CreateCheckoutSession)

		req, _ := http.NewRequest(http.MethodPost, "/subscription/checkout-session", bytes.NewBufferString(`{"plan":"platinum"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("ignores a user_id supplied by the client", func(t *testing.T) {
		// The buyer must come from the auth context; anything in the body is
		// discarded by the DTO, which only carries a plan.
		var contactUserID string
		mockRepo := &MockSubscriptionRepository{
			GetUserContactFunc: func(_ context.Context, uid string) (*model.UserContact, error) {
				contactUserID = uid
				return &model.UserContact{Email: "buyer@example.com", Name: "Test Buyer", Locale: "en"}, nil
			},
		}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/checkout-session", mockAuthMiddleware(userID), handler.CreateCheckoutSession)

		body := `{"plan":"pro","user_id":"attacker-controlled","userID":"attacker-controlled"}`
		req, _ := http.NewRequest(http.MethodPost, "/subscription/checkout-session", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// The provider client has no credentials here, so the call fails after
		// the identity has already been resolved from the session.
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Equal(t, userID, contactUserID, "the buyer must come from the auth context")
	})
}

func TestSubscriptionHandler_CreatePortalSession(t *testing.T) {
	userID := testUserID

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

	t.Run("returns 404 when the user has no billing account", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/portal", mockAuthMiddleware(userID), handler.CreatePortalSession)

		req, _ := http.NewRequest(http.MethodPost, "/subscription/portal", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 500 when the provider call fails", func(t *testing.T) {
		accountID := "acct-external-1"
		mockRepo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(_ context.Context, uid string) (*model.Subscription, error) {
				return &model.Subscription{UserID: uid, ExternalAccountID: &accountID, Status: "active", Plan: "pro"}, nil
			},
		}
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
	userID := testUserID

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

	t.Run("returns 404 when there is no subscription to change", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/change-plan", mockAuthMiddleware(userID), handler.ChangePlan)

		body := bytes.NewBufferString(`{"plan":"pro"}`)
		req, _ := http.NewRequest(http.MethodPost, "/subscription/change-plan", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 500 when the provider call fails", func(t *testing.T) {
		externalID := "sub-external-1"
		mockRepo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(_ context.Context, uid string) (*model.Subscription, error) {
				return &model.Subscription{
					UserID:                 uid,
					ExternalSubscriptionID: &externalID,
					Status:                 "active",
					Plan:                   "pro",
				}, nil
			},
		}
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
	userID := testUserID

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

	t.Run("returns 404 when the user has no subscription to cancel", func(t *testing.T) {
		mockRepo := &MockSubscriptionRepository{}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/cancel", mockAuthMiddleware(userID), handler.CancelSubscription)

		req, _ := http.NewRequest(http.MethodPost, "/subscription/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("returns 500 when the provider call fails", func(t *testing.T) {
		// A linked subscription exists, so the handler reaches the provider —
		// which has no credentials and fails.
		externalID := "sub-external-1"
		mockRepo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(_ context.Context, uid string) (*model.Subscription, error) {
				return &model.Subscription{
					UserID:                 uid,
					ExternalSubscriptionID: &externalID,
					Status:                 "active",
					Plan:                   "pro",
				}, nil
			},
		}
		handler := newTestSubscriptionHandler(mockRepo)

		router := setupTestRouter()
		router.POST("/subscription/cancel", mockAuthMiddleware(userID), handler.CancelSubscription)

		req, _ := http.NewRequest(http.MethodPost, "/subscription/cancel", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.NotContains(t, w.Body.String(), "fastspring",
			"the client response must not expose provider internals")
	})
}

// registeredRoutes returns the "METHOD PATH" set actually mounted on a router,
// so registration is asserted directly instead of inferred from a status code.
func registeredRoutes(router *gin.Engine) map[string]bool {
	mounted := make(map[string]bool)
	for _, route := range router.Routes() {
		mounted[route.Method+" "+route.Path] = true
	}
	return mounted
}

func TestSubscriptionHandler_RegisterRoutes(t *testing.T) {
	mockRepo := &MockSubscriptionRepository{
		GetByUserIDFunc: func(ctx context.Context, uid string) (*model.Subscription, error) {
			return &model.Subscription{UserID: uid, Status: "free", Plan: "free"}, nil
		},
		GetAllCountsFunc: func(ctx context.Context, uid string) (int, int, int, int, int, int, error) {
			return 0, 0, 0, 0, 0, 0, nil
		},
	}
	handler := newTestSubscriptionHandler(mockRepo)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware(testUserID), true)

	mounted := registeredRoutes(router)
	for _, route := range []string{
		"GET /api/v1/subscription",
		"GET /api/v1/subscription/checkout-config",
		"POST /api/v1/subscription/checkout-session",
		"POST /api/v1/subscription/portal",
		"POST /api/v1/subscription/change-plan",
		"POST /api/v1/subscription/cancel",
	} {
		assert.True(t, mounted[route], "%s should be registered", route)
	}
}

func TestSubscriptionHandler_RegisterRoutes_PaymentsDisabled(t *testing.T) {
	mockRepo := &MockSubscriptionRepository{
		GetByUserIDFunc: func(ctx context.Context, uid string) (*model.Subscription, error) {
			return &model.Subscription{UserID: uid, Status: "free", Plan: "free"}, nil
		},
		GetAllCountsFunc: func(ctx context.Context, uid string) (int, int, int, int, int, int, error) {
			return 0, 0, 0, 0, 0, 0, nil
		},
	}
	handler := newTestSubscriptionHandler(mockRepo)

	router := setupTestRouter()
	v1 := router.Group("/api/v1")
	handler.RegisterRoutes(v1, mockAuthMiddleware(testUserID), false)

	t.Run("GET /subscription is registered", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/subscription", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	// With the checkout kill-switch on, purchase and plan-management routes are
	// not mounted at all — the read-only subscription endpoint stays available.
	mounted := registeredRoutes(router)
	for _, route := range []string{
		"GET /api/v1/subscription/checkout-config",
		"POST /api/v1/subscription/checkout-session",
		"POST /api/v1/subscription/portal",
		"POST /api/v1/subscription/change-plan",
		"POST /api/v1/subscription/cancel",
	} {
		assert.False(t, mounted[route], "%s must not be registered when payments are disabled", route)
	}
	assert.True(t, mounted["GET /api/v1/subscription"])
}

// --- WebhookHandler Tests ---

// signWebhook produces the header value FastSpring would send for a body.
func signWebhook(body string) string {
	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write([]byte(body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func postWebhook(handler *WebhookHandler, body, signature string) *httptest.ResponseRecorder {
	router := setupTestRouter()
	router.POST("/webhooks/fastspring", handler.HandleFastSpringWebhook)

	req, _ := http.NewRequest(http.MethodPost, "/webhooks/fastspring", bytes.NewBufferString(body))
	if signature != "" {
		req.Header.Set("X-FS-Signature", signature)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestWebhookHandler_RejectsUnverifiedPayloads(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		signature string
	}{
		{name: "empty body", body: "", signature: ""},
		{name: "missing signature", body: `{"events":[]}`, signature: ""},
		{name: "wrong signature", body: `{"events":[]}`, signature: "bm90LWEtc2lnbmF0dXJl"},
		{name: "signature for a different body", body: `{"events":[]}`, signature: signWebhook(`{"events":[{}]}`)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var wrote bool
			mockRepo := &MockSubscriptionRepository{
				ApplySubscriptionEventFunc: func(context.Context, string, string, *model.Subscription) (model.WebhookApplyOutcome, error) {
					wrote = true
					return model.WebhookApplied, nil
				},
			}

			w := postWebhook(newTestWebhookHandler(mockRepo), tc.body, tc.signature)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.False(t, wrote, "a rejected webhook must not touch the database")
		})
	}
}

func TestWebhookHandler_AcknowledgesProcessedBatch(t *testing.T) {
	// Two events Jobber does not act on: both are acknowledged with 200 so
	// FastSpring stops redelivering them.
	body := `{"events":[
		{"id":"evt-1","live":false,"processed":false,"type":"order.completed","created":1751328000000,"data":{}},
		{"id":"evt-2","live":false,"processed":false,"type":"account.updated","created":1751328000001,"data":{}}
	]}`

	w := postWebhook(newTestWebhookHandler(&MockSubscriptionRepository{}), body, signWebhook(body))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWebhookHandler_LogsPermanentDropsLouderThanOrdinarySkips(t *testing.T) {
	// All of these are acknowledged and none is retried, so the log is the only
	// place they differ — and they must differ. A routine skip is noise; a
	// mismatch means this deployment is reading the wrong billing environment,
	// and an unprovable order tag means someone tagged an order Jobber did not
	// create. Both lose the event for good.
	t.Run("an environment mismatch is a warning", func(t *testing.T) {
		// live:true against a test-mode deployment.
		body := `{"events":[
			{"id":"evt-live","live":true,"processed":false,"type":"subscription.activated","created":1751328000000,
			 "data":{"id":"sub-x","subscription":"sub-x","state":"active","active":true,
			         "account":{"id":"acct-x"},"product":{"product":"jobber-pro"}}}
		]}`
		handler, logs := newObservedWebhookHandler(&MockSubscriptionRepository{})

		w := postWebhook(handler, body, signWebhook(body))

		assert.Equal(t, http.StatusOK, w.Code, "a mismatched event is still acknowledged")
		warnings := logs.FilterLevelExact(zap.WarnLevel).All()
		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0].Message, "environment mismatch")
		assert.Equal(t, "evt-live", warnings[0].ContextMap()["event_id"])
	})

	t.Run("an order tag with no proof is a warning", func(t *testing.T) {
		// The forgery the proof exists to stop: a storefront buyer tagging their
		// own order with somebody else's Jobber user ID. It is acknowledged like
		// any other permanent drop, so the log is the only place it surfaces.
		body := `{"events":[
			{"id":"evt-forged","live":false,"processed":false,"type":"subscription.activated","created":1751328000000,
			 "data":{"id":"sub-x","subscription":"sub-x","state":"active","active":true,
			         "account":{"id":"acct-x"},"product":{"product":"jobber-pro"},
			         "tags":{"jobber_user_id":"550e8400-e29b-41d4-a716-446655440000"}}}
		]}`
		handler, logs := newObservedWebhookHandler(&MockSubscriptionRepository{})

		w := postWebhook(handler, body, signWebhook(body))

		assert.Equal(t, http.StatusOK, w.Code, "an unprovable claim is acknowledged, never retried")
		warnings := logs.FilterLevelExact(zap.WarnLevel).All()
		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0].Message, "cannot prove")
		assert.Equal(t, "evt-forged", warnings[0].ContextMap()["event_id"])
	})

	t.Run("a second live subscription for one user is a warning", func(t *testing.T) {
		// The atomic write refused to repoint the user at a second subscription
		// while the first still bills. Nobody is looking at the row, so this log
		// line is the only thing that says a subscriber may be paying twice with
		// only one of the two cancellable from Jobber.
		body := `{"events":[
			{"id":"evt-second-sub","live":false,"processed":false,"type":"subscription.activated","created":1751328000000,
			 "data":{"id":"sub-second","subscription":"sub-second","state":"active","active":true,
			         "account":{"id":"acct-x"},"product":{"product":"jobber-pro"}}}
		]}`
		mockRepo := &MockSubscriptionRepository{
			GetByExternalAccountIDFunc: func(_ context.Context, accountID string) (*model.Subscription, error) {
				return &model.Subscription{ID: "sub-row-1", UserID: "user-1", Status: "active", Plan: "pro"}, nil
			},
			ApplySubscriptionEventFunc: func(context.Context, string, string, *model.Subscription) (model.WebhookApplyOutcome, error) {
				return model.WebhookLinkConflict, nil
			},
		}
		handler, logs := newObservedWebhookHandler(mockRepo)

		w := postWebhook(handler, body, signWebhook(body))

		assert.Equal(t, http.StatusOK, w.Code, "a link conflict is acknowledged, never retried")
		warnings := logs.FilterLevelExact(zap.WarnLevel).All()
		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0].Message, "already linked to another live subscription")
		assert.Equal(t, "evt-second-sub", warnings[0].ContextMap()["event_id"])
	})

	t.Run("an event Jobber does not act on stays informational", func(t *testing.T) {
		body := `{"events":[
			{"id":"evt-ok","live":false,"processed":false,"type":"order.completed","created":1751328000000,"data":{}}
		]}`
		handler, logs := newObservedWebhookHandler(&MockSubscriptionRepository{})

		w := postWebhook(handler, body, signWebhook(body))

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, logs.FilterLevelExact(zap.WarnLevel).All(),
			"a routine skip must not raise the noise floor for the real misconfiguration")
		require.Len(t, logs.FilterLevelExact(zap.InfoLevel).All(), 1)
	})
}

func TestWebhookHandler_PartialBatchReturns202WithProcessedIDs(t *testing.T) {
	// evt-ok is acknowledged as non-actionable; evt-bad cannot be resolved to a
	// user, so it must be retried.
	body := `{"events":[
		{"id":"evt-ok","live":false,"processed":false,"type":"order.completed","created":1751328000000,"data":{}},
		{"id":"evt-bad","live":false,"processed":false,"type":"subscription.activated","created":1751328000001,
		 "data":{"id":"sub-x","subscription":"sub-x","state":"active","active":true,
		         "account":{"id":"acct-unknown"},"product":{"product":"jobber-pro"}}}
	]}`

	w := postWebhook(newTestWebhookHandler(&MockSubscriptionRepository{}), body, signWebhook(body))

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Equal(t, "evt-ok", w.Body.String(),
		"the 202 body lists only the processed event IDs, one per line")
}

func TestWebhookHandler_FullyFailedBatchAsksForRetry(t *testing.T) {
	body := `{"events":[
		{"id":"evt-bad","live":false,"processed":false,"type":"subscription.activated","created":1751328000000,
		 "data":{"id":"sub-x","subscription":"sub-x","state":"active","active":true,
		         "account":{"id":"acct-unknown"},"product":{"product":"jobber-pro"}}}
	]}`

	w := postWebhook(newTestWebhookHandler(&MockSubscriptionRepository{}), body, signWebhook(body))

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestWebhookHandler_MissingSecretAsksForRetry(t *testing.T) {
	// Our own misconfiguration must not be reported as a client error, or
	// FastSpring would stop retrying and the event would be lost.
	svc := service.NewSubscriptionService(&MockSubscriptionRepository{},
		fastspring.NewClient(fastspring.Config{}),
		service.BillingConfig{Environment: service.EnvironmentTest})
	handler := NewWebhookHandler(svc, zap.NewNop())

	w := postWebhook(handler, `{"events":[]}`, "c2lnbmF0dXJl")

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestWebhookHandler_RegisterRoutes(t *testing.T) {
	postToWebhook := func(t *testing.T, rateLimiter gin.HandlerFunc) int {
		t.Helper()
		handler := newTestWebhookHandler(&MockSubscriptionRepository{})
		router := setupTestRouter()
		handler.RegisterRoutes(router.Group("/api/v1"), rateLimiter)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/webhooks/fastspring", bytes.NewBufferString("{}"))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w.Code
	}

	t.Run("the route is registered", func(t *testing.T) {
		assert.NotEqual(t, http.StatusNotFound, postToWebhook(t, nil),
			"POST /api/v1/webhooks/fastspring should be registered")
	})

	t.Run("the rate limiter runs in front of it", func(t *testing.T) {
		// This is the only public route with no auth middleware, and verifying
		// the HMAC means reading and hashing the body first — so the limiter has
		// to be reached before the handler, not merely configured somewhere.
		refuse := func(c *gin.Context) { c.AbortWithStatus(http.StatusTooManyRequests) }

		assert.Equal(t, http.StatusTooManyRequests, postToWebhook(t, refuse))
	})
}
