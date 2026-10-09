package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/creem"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/andreypavlenko/jobber/modules/subscriptions/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newProviderBackedHandler wires the handler to a stub Creem API so the success
// paths — which the key-less default setup can never reach — are exercised end
// to end through the HTTP layer.
func newProviderBackedHandler(t *testing.T, repo *MockSubscriptionRepository, providerHandler http.HandlerFunc) *SubscriptionHandler {
	t.Helper()
	server := httptest.NewServer(providerHandler)
	t.Cleanup(server.Close)

	svc := service.NewSubscriptionService(
		repo,
		creem.NewClient(creem.Config{APIKey: "creem_key", BaseURL: server.URL}),
		testBillingConfig(),
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
	var seenBody map[string]any
	handler := newProviderBackedHandler(t, &MockSubscriptionRepository{}, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/checkouts", r.URL.Path)
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &seenBody)
		_, _ = w.Write([]byte(`{"id":"ch_1","status":"pending","checkout_url":"https://www.creem.io/test/checkout/ch_1"}`))
	})

	router := setupTestRouter()
	router.POST("/subscription/checkout-session", mockAuthMiddleware(testUserID), handler.CreateCheckoutSession)

	req, _ := http.NewRequest(http.MethodPost, "/subscription/checkout-session", bytes.NewBufferString(`{"plan":"pro"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var session model.CheckoutSessionDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &session))
	assert.Equal(t, "https://www.creem.io/test/checkout/ch_1", session.CheckoutURL)
	assert.Equal(t, map[string]any{"jobber_user_id": testUserID}, seenBody["metadata"],
		"the buyer is bound to the checkout server-side")
}

func TestSubscriptionHandler_ChangePlan_Success(t *testing.T) {
	var path string
	repo := activeSubscriptionRepo("sub_1", "cust_1")
	handler := newProviderBackedHandler(t, repo, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"id":"sub_1"}`))
	})

	router := setupTestRouter()
	router.POST("/subscription/change-plan", mockAuthMiddleware(testUserID), handler.ChangePlan)

	req, _ := http.NewRequest(http.MethodPost, "/subscription/change-plan", bytes.NewBufferString(`{"plan":"enterprise"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "/v1/subscriptions/sub_1/upgrade", path)
}

func TestSubscriptionHandler_CancelSubscription_Success(t *testing.T) {
	var body map[string]any
	repo := activeSubscriptionRepo("sub_1", "cust_1")
	handler := newProviderBackedHandler(t, repo, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"id":"sub_1","status":"scheduled_cancel"}`))
	})

	router := setupTestRouter()
	router.POST("/subscription/cancel", mockAuthMiddleware(testUserID), handler.CancelSubscription)

	req, _ := http.NewRequest(http.MethodPost, "/subscription/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, "scheduled", body["mode"], "a user-facing cancel keeps access until the period ends")
}

func TestSubscriptionHandler_CreatePortalSession_Success(t *testing.T) {
	repo := activeSubscriptionRepo("sub_1", "cust_1")
	handler := newProviderBackedHandler(t, repo, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"customer_portal_link":"https://creem.io/my-orders/login/abc"}`))
	})

	router := setupTestRouter()
	router.POST("/subscription/portal", mockAuthMiddleware(testUserID), handler.CreatePortalSession)

	req, _ := http.NewRequest(http.MethodPost, "/subscription/portal", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var portal model.PortalSessionDTO
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &portal))
	assert.Equal(t, "https://creem.io/my-orders/login/abc", portal.URL)
}

func TestWebhookHandler_OversizedBodyIsRejected(t *testing.T) {
	// A body over the 1 MB cap is refused outright, before any hashing, so an
	// unsigned flood of large payloads buys nothing.
	var wrote bool
	repo := &MockSubscriptionRepository{
		ApplySubscriptionEventFunc: func(context.Context, string, string, *model.Subscription) (model.WebhookApplyOutcome, error) {
			wrote = true
			return model.WebhookApplied, nil
		},
	}
	body := `{"id":"evt-1","eventType":"subscription.paid","object":{}}` + strings.Repeat(" ", 1<<20)

	w := postWebhook(newTestWebhookHandler(repo), body, signWebhook(body))

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	assert.False(t, wrote, "an oversized payload must not reach the database")
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func TestWebhookHandler_BodyReadFailure(t *testing.T) {
	handler := newTestWebhookHandler(&MockSubscriptionRepository{})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/webhooks/creem", handler.HandleCreemWebhook)

	req, err := http.NewRequest(http.MethodPost, "/webhooks/creem", failingReader{})
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
			UserID: testUserID, ExternalSubscriptionID: ptr("sub_1"),
			Status: "active", Plan: "pro",
		},
		"past_due": {
			UserID: testUserID, ExternalSubscriptionID: ptr("sub_1"),
			Status: "past_due", Plan: "pro",
		},
		"paused": {
			UserID: testUserID, ExternalSubscriptionID: ptr("sub_1"),
			Status: "paused", Plan: "pro",
		},
		"cancellation scheduled": {
			UserID: testUserID, ExternalSubscriptionID: ptr("sub_1"),
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
			router.POST("/subscription/checkout-session", mockAuthMiddleware(testUserID), handler.CreateCheckoutSession)

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
					UserID: testUserID, ExternalSubscriptionID: ptr("sub_old"),
					Status: "cancelled", Plan: "free",
				}, nil
			},
		}
		handler := newProviderBackedHandler(t, repo, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":"ch_2","checkout_url":"https://www.creem.io/test/checkout/ch_2"}`))
		})

		router := setupTestRouter()
		router.POST("/subscription/checkout-session", mockAuthMiddleware(testUserID), handler.CreateCheckoutSession)

		req, _ := http.NewRequest(http.MethodPost, "/subscription/checkout-session", bytes.NewBufferString(`{"plan":"pro"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	})
}

func TestSubscriptionHandler_CreateCheckoutSession_ProviderFailureIsAGenericError(t *testing.T) {
	handler := newProviderBackedHandler(t, &MockSubscriptionRepository{}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid api key creem_key"}`))
	})

	router := setupTestRouter()
	router.POST("/subscription/checkout-session", mockAuthMiddleware(testUserID), handler.CreateCheckoutSession)

	req, _ := http.NewRequest(http.MethodPost, "/subscription/checkout-session", bytes.NewBufferString(`{"plan":"pro"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "CHECKOUT_ERROR")
	assert.NotContains(t, w.Body.String(), "invalid api key", "provider detail stays in the logs")
}

func ptr[T any](v T) *T { return &v }
