package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newProviderBackedHandler wires the handler to a stub FastSpring API so the
// success paths — which the credential-less default setup can never reach —
// are exercised end to end through the HTTP layer.
func newProviderBackedHandler(t *testing.T, repo *MockSubscriptionRepository, providerHandler http.HandlerFunc) *SubscriptionHandler {
	t.Helper()
	server := httptest.NewServer(providerHandler)
	t.Cleanup(server.Close)

	svc := service.NewSubscriptionService(
		repo,
		fastspring.NewClient(fastspring.Config{
			BaseURL:  server.URL,
			Username: "api-user",
			Password: "api-pass",
		}),
		service.BillingConfig{
			WebhookSecret:         testWebhookSecret,
			CheckoutPath:          testCheckoutPath,
			Environment:           service.EnvironmentTest,
			ProProductPath:        testProPath,
			EnterpriseProductPath: testEnterprisePath,
		},
	)
	return NewSubscriptionHandler(svc, zap.NewNop())
}

func activeSubscriptionRepo(externalSubID, externalAccountID string) *MockSubscriptionRepository {
	return &MockSubscriptionRepository{
		GetByUserIDFunc: func(_ context.Context, uid string) (*model.Subscription, error) {
			return &model.Subscription{
				UserID:                 uid,
				ExternalSubscriptionID: &externalSubID,
				ExternalAccountID:      &externalAccountID,
				Status:                 "active",
				Plan:                   "pro",
			}, nil
		},
	}
}

func TestSubscriptionHandler_CreateCheckoutSession_Success(t *testing.T) {
	userID := "user-123"
	var linkedAccount string
	repo := &MockSubscriptionRepository{
		LinkExternalAccountFunc: func(_ context.Context, _, accountID string) error {
			linkedAccount = accountID
			return nil
		},
	}
	handler := newProviderBackedHandler(t, repo, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/checkouts/"+testCheckoutPath+"/sessions", r.URL.Path)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"sess-1","expires":"2026-07-01T00:00:00Z","checkoutStatus":["READY_FOR_CHECKOUT"],
			"customer":{"accountId":"acct-1"},
			"checkoutUrls":{"webcheckoutUrl":"https://jobber.test.onfastspring.com/checkout/sess-1"}}`))
	})

	router := setupTestRouter()
	router.POST("/subscription/checkout-session", mockAuthMiddleware(userID), handler.CreateCheckoutSession)

	req, _ := http.NewRequest(http.MethodPost, "/subscription/checkout-session", bytes.NewBufferString(`{"plan":"pro"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var session model.CheckoutSessionDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &session))
	assert.Equal(t, "sess-1", session.SessionID)
	assert.NotContains(t, w.Body.String(), "onfastspring.com",
		"the endpoint returns a session id for the popup, never a URL to navigate to")
	assert.Equal(t, "acct-1", linkedAccount,
		"the provider account must be linked before the session reaches the browser")
}

func TestSubscriptionHandler_ChangePlan_Success(t *testing.T) {
	repo := activeSubscriptionRepo("sub-ext-1", "acct-ext-1")
	handler := newProviderBackedHandler(t, repo, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"subscriptions":[{"subscription":"sub-ext-1","action":"subscription.update","result":"success"}]}`))
	})

	router := setupTestRouter()
	router.POST("/subscription/change-plan", mockAuthMiddleware("user-123"), handler.ChangePlan)

	req, _ := http.NewRequest(http.MethodPost, "/subscription/change-plan", bytes.NewBufferString(`{"plan":"enterprise"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestSubscriptionHandler_CancelSubscription_Success(t *testing.T) {
	var query string
	repo := activeSubscriptionRepo("sub-ext-1", "acct-ext-1")
	handler := newProviderBackedHandler(t, repo, func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		_, _ = w.Write([]byte(`{"subscriptions":[{"subscription":"sub-ext-1","action":"subscription.cancel","result":"success"}]}`))
	})

	router := setupTestRouter()
	router.POST("/subscription/cancel", mockAuthMiddleware("user-123"), handler.CancelSubscription)

	req, _ := http.NewRequest(http.MethodPost, "/subscription/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "billingPeriod=1", query, "a user-facing cancel keeps access until the period ends")
}

func TestSubscriptionHandler_CreatePortalSession_Success(t *testing.T) {
	repo := activeSubscriptionRepo("sub-ext-1", "acct-ext-1")
	handler := newProviderBackedHandler(t, repo, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"accounts":[{"account":"acct-ext-1","result":"success","url":"https://store.onfastspring.com/account/a/b"}]}`))
	})

	router := setupTestRouter()
	router.POST("/subscription/portal", mockAuthMiddleware("user-123"), handler.CreatePortalSession)

	req, _ := http.NewRequest(http.MethodPost, "/subscription/portal", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var portal model.PortalSessionDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &portal))
	assert.Equal(t, "https://store.onfastspring.com/account/a/b#/subscriptions", portal.URL)
}

func TestWebhookHandler_PartialBatchListsEveryProcessedID(t *testing.T) {
	// Two acknowledged events and one that must be retried: the 202 body has to
	// carry both processed IDs, one per line, or FastSpring would redeliver them.
	body := `{"events":[
		{"id":"evt-a","live":false,"processed":false,"type":"order.completed","created":1751328000000,"data":{}},
		{"id":"evt-b","live":false,"processed":false,"type":"account.updated","created":1751328000001,"data":{}},
		{"id":"evt-bad","live":false,"processed":false,"type":"subscription.activated","created":1751328000002,
		 "data":{"id":"sub-x","subscription":"sub-x","state":"active","active":true,
		         "account":{"id":"acct-unknown"},"product":{"product":"jobber-pro"}}}
	]}`

	w := postWebhook(newTestWebhookHandler(&MockSubscriptionRepository{}), body, signWebhook(body))

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Equal(t, "evt-a\nevt-b", w.Body.String())
}

func TestWebhookHandler_OversizedBodyIsRejected(t *testing.T) {
	// A body over the 1 MB cap is read truncated, so the signature — computed
	// over the full payload — no longer matches and nothing is processed.
	var wrote bool
	repo := &MockSubscriptionRepository{
		ApplySubscriptionEventFunc: func(context.Context, string, string, *model.Subscription) (model.WebhookApplyOutcome, error) {
			wrote = true
			return model.WebhookApplied, nil
		},
	}
	body := `{"events":[]}` + strings.Repeat(" ", 1<<20)

	w := postWebhook(newTestWebhookHandler(repo), body, signWebhook(body))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, wrote, "an oversized payload must not reach the database")
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func TestWebhookHandler_BodyReadFailure(t *testing.T) {
	handler := newTestWebhookHandler(&MockSubscriptionRepository{})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/webhooks/fastspring", handler.HandleFastSpringWebhook)

	req, err := http.NewRequest(http.MethodPost, "/webhooks/fastspring", failingReader{})
	require.NoError(t, err)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubscriptionHandler_CreateCheckoutSession_ConflictForExistingSubscriber(t *testing.T) {
	// Defence in depth: the UI routes a subscriber to change-plan, but any other
	// client posting here would otherwise start a second, invisible subscription.
	cancelAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	subscribed := map[string]*model.Subscription{
		"active": {
			UserID: "user-123", ExternalSubscriptionID: ptr("sub-ext-1"),
			Status: "active", Plan: "pro",
		},
		"past_due": {
			UserID: "user-123", ExternalSubscriptionID: ptr("sub-ext-1"),
			Status: "past_due", Plan: "pro",
		},
		"paused": {
			UserID: "user-123", ExternalSubscriptionID: ptr("sub-ext-1"),
			Status: "paused", Plan: "pro",
		},
		"cancellation scheduled": {
			UserID: "user-123", ExternalSubscriptionID: ptr("sub-ext-1"),
			Status: "active", Plan: "pro", CancelAt: &cancelAt,
		},
	}

	for name, sub := range subscribed {
		t.Run(name+" gets 409", func(t *testing.T) {
			repo := &MockSubscriptionRepository{
				GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) { return sub, nil },
			}
			handler := newProviderBackedHandler(t, repo, func(http.ResponseWriter, *http.Request) {
				t.Fatal("provider must not be called for a user who already subscribes")
			})

			router := setupTestRouter()
			router.POST("/subscription/checkout-session", mockAuthMiddleware("user-123"), handler.CreateCheckoutSession)

			req, _ := http.NewRequest(http.MethodPost, "/subscription/checkout-session", bytes.NewBufferString(`{"plan":"enterprise"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
			assert.Contains(t, w.Body.String(), "ALREADY_SUBSCRIBED")
		})
	}

	t.Run("a cancelled subscriber may start a new checkout", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
				return &model.Subscription{
					UserID: "user-123", ExternalSubscriptionID: ptr("sub-ext-old"),
					Status: "cancelled", Plan: "free",
				}, nil
			},
		}
		handler := newProviderBackedHandler(t, repo, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"sess-2","checkoutStatus":["READY_FOR_CHECKOUT"],
				"customer":{"accountId":"acct-1"},
				"checkoutUrls":{"webcheckoutUrl":"https://jobber.test.onfastspring.com/checkout/sess-2"}}`))
		})

		router := setupTestRouter()
		router.POST("/subscription/checkout-session", mockAuthMiddleware("user-123"), handler.CreateCheckoutSession)

		req, _ := http.NewRequest(http.MethodPost, "/subscription/checkout-session", bytes.NewBufferString(`{"plan":"pro"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})
}

func ptr[T any](v T) *T { return &v }
