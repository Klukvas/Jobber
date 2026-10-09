package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/creem"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedRequest is what the stub Creem API saw.
type capturedRequest struct {
	Method string
	Path   string
	APIKey string
	Body   map[string]any
}

// stubCreemAPI answers every request with the given status and body and records
// the requests it receives.
func stubCreemAPI(t *testing.T, status int, response string) (*httptest.Server, *[]capturedRequest) {
	t.Helper()
	var requests []capturedRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		captured := capturedRequest{Method: r.Method, Path: r.URL.Path, APIKey: r.Header.Get("x-api-key")}
		if len(raw) > 0 {
			require.NoError(t, json.Unmarshal(raw, &captured.Body))
		}
		requests = append(requests, captured)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func subscriberRepo(sub *model.Subscription) *MockSubscriptionRepository {
	return &MockSubscriptionRepository{
		GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) { return sub, nil },
	}
}

func strPtr(s string) *string { return &s }

func TestGetCheckoutConfig(t *testing.T) {
	t.Run("advertises the provider, environment and configured plans only", func(t *testing.T) {
		config := newTestService(&MockSubscriptionRepository{}).GetCheckoutConfig()

		assert.Equal(t, "creem", config.Provider)
		assert.Equal(t, EnvironmentTest, config.Environment)
		assert.Equal(t, []string{PlanPro, PlanEnterprise}, config.Plans)
	})

	t.Run("a plan without a product is never advertised", func(t *testing.T) {
		cfg := testBillingConfig()
		cfg.EnterpriseProductID = ""
		svc := NewSubscriptionService(&MockSubscriptionRepository{}, creem.NewClient(creem.Config{}), cfg)

		assert.Equal(t, []string{PlanPro}, svc.GetCheckoutConfig().Plans)
	})

	t.Run("exposes no product IDs or credentials", func(t *testing.T) {
		raw, err := json.Marshal(newTestService(&MockSubscriptionRepository{}).GetCheckoutConfig())
		require.NoError(t, err)

		assert.NotContains(t, string(raw), testProProductID)
		assert.NotContains(t, string(raw), testWebhookSecret)
	})
}

func TestCreateCheckoutSession(t *testing.T) {
	const hostedURL = "https://www.creem.io/test/checkout/prod_pro/ch_1"
	response := `{"id":"ch_1","status":"pending","checkout_url":"` + hostedURL + `"}`

	t.Run("creates a hosted checkout bound to the authenticated user", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, response)
		var ensured string
		repo := &MockSubscriptionRepository{
			EnsureFreeFunc: func(_ context.Context, userID string) error {
				ensured = userID
				return nil
			},
		}
		svc := newTestServiceWithAPI(repo, server.URL)

		dto, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.NoError(t, err)
		assert.Equal(t, hostedURL, dto.CheckoutURL)
		assert.Equal(t, testUserID, ensured, "the row the webhook will write into must exist")

		require.Len(t, *requests, 1)
		got := (*requests)[0]
		assert.Equal(t, http.MethodPost, got.Method)
		assert.Equal(t, "/v1/checkouts", got.Path)
		assert.Equal(t, "test-key", got.APIKey)
		assert.Equal(t, testProProductID, got.Body["product_id"])
		assert.Equal(t, testSuccessURL, got.Body["success_url"])
		assert.Equal(t, map[string]any{metadataUserIDKey: testUserID}, got.Body["metadata"])
		assert.Equal(t, map[string]any{"email": "buyer@example.com", "name": "Test Buyer"}, got.Body["customer"])
	})

	t.Run("the enterprise plan buys the enterprise product", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, response)
		svc := newTestServiceWithAPI(&MockSubscriptionRepository{}, server.URL)

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanEnterprise)

		require.NoError(t, err)
		assert.Equal(t, testEnterpriseProductID, (*requests)[0].Body["product_id"])
	})

	t.Run("a plan that cannot be bought never reaches the API", func(t *testing.T) {
		for _, plan := range []string{"", "free", "platinum"} {
			server, requests := stubCreemAPI(t, http.StatusOK, response)
			svc := newTestServiceWithAPI(&MockSubscriptionRepository{}, server.URL)

			_, err := svc.CreateCheckoutSession(context.Background(), testUserID, plan)

			assert.ErrorIs(t, err, model.ErrUnknownPlan, "plan %q", plan)
			assert.Empty(t, *requests)
		}
	})

	t.Run("a plan with no configured product is refused", func(t *testing.T) {
		cfg := testBillingConfig()
		cfg.ProProductID = ""
		svc := NewSubscriptionService(&MockSubscriptionRepository{}, creem.NewClient(creem.Config{APIKey: "k"}), cfg)

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		assert.ErrorIs(t, err, model.ErrUnknownPlan)
	})

	t.Run("a user who already pays cannot start a second subscription", func(t *testing.T) {
		for _, status := range []string{StatusActive, StatusPastDue, StatusPaused} {
			server, requests := stubCreemAPI(t, http.StatusOK, response)
			repo := subscriberRepo(&model.Subscription{
				Plan: PlanPro, Status: status, ExternalSubscriptionID: strPtr("sub_1"),
			})
			svc := newTestServiceWithAPI(repo, server.URL)

			_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanEnterprise)

			assert.ErrorIs(t, err, model.ErrAlreadySubscribed, "status %q", status)
			assert.Empty(t, *requests, "status %q", status)
		}
	})

	t.Run("a scheduled cancellation still bills, so it is refused too", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, response)
		repo := subscriberRepo(&model.Subscription{
			Plan: PlanPro, Status: StatusActive, ExternalSubscriptionID: strPtr("sub_1"),
		})
		svc := newTestServiceWithAPI(repo, server.URL)

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		assert.ErrorIs(t, err, model.ErrAlreadySubscribed)
		assert.Empty(t, *requests)
	})

	t.Run("a cancelled subscriber can buy again", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, response)
		repo := subscriberRepo(&model.Subscription{
			Plan: PlanFree, Status: StatusCancelled, ExternalSubscriptionID: strPtr("sub_old"),
		})
		svc := newTestServiceWithAPI(repo, server.URL)

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.NoError(t, err)
		assert.Len(t, *requests, 1)
	})

	t.Run("an unreadable subscription fails the checkout instead of bypassing the guard", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, response)
		repo := &MockSubscriptionRepository{
			GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
				return nil, errors.New("db down")
			},
		}
		svc := newTestServiceWithAPI(repo, server.URL)

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		require.Error(t, err)
		assert.NotErrorIs(t, err, model.ErrAlreadySubscribed)
		assert.Empty(t, *requests)
	})

	t.Run("a checkout URL that is not absolute HTTPS is never handed to the browser", func(t *testing.T) {
		for _, unsafe := range []string{
			"javascript:alert(1)", "http://www.creem.io/checkout/ch_1", "//evil.example/x",
			"/relative/path", "data:text/html,hi", "https://",
			"https://www.creem.io@evil.example/x", "https://user:pw@www.creem.io/x",
		} {
			server, _ := stubCreemAPI(t, http.StatusOK, `{"id":"ch_1","checkout_url":"`+unsafe+`"}`)
			svc := newTestServiceWithAPI(&MockSubscriptionRepository{}, server.URL)

			dto, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

			require.Error(t, err, "url %q", unsafe)
			assert.Nil(t, dto)
		}
	})

	t.Run("an API failure surfaces as an error", func(t *testing.T) {
		server, _ := stubCreemAPI(t, http.StatusUnauthorized, `{"message":"invalid key"}`)
		svc := newTestServiceWithAPI(&MockSubscriptionRepository{}, server.URL)

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		var apiErr *creem.APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusUnauthorized, apiErr.StatusCode)
	})

	t.Run("billing without an API key reports itself as unavailable", func(t *testing.T) {
		svc := NewSubscriptionService(&MockSubscriptionRepository{}, creem.NewClient(creem.Config{}), testBillingConfig())

		_, err := svc.CreateCheckoutSession(context.Background(), testUserID, PlanPro)

		assert.ErrorIs(t, err, creem.ErrNotConfigured)
	})
}

func TestChangePlan(t *testing.T) {
	paying := &model.Subscription{Plan: PlanPro, Status: StatusActive, ExternalSubscriptionID: strPtr("sub_1")}

	t.Run("upgrades the subscription the user already pays for", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, `{"id":"sub_1"}`)
		svc := newTestServiceWithAPI(subscriberRepo(paying), server.URL)

		err := svc.ChangePlan(context.Background(), testUserID, PlanEnterprise)

		require.NoError(t, err)
		require.Len(t, *requests, 1)
		got := (*requests)[0]
		assert.Equal(t, "/v1/subscriptions/sub_1/upgrade", got.Path)
		assert.Equal(t, testEnterpriseProductID, got.Body["product_id"])
		assert.Equal(t, "proration-charge-immediately", got.Body["update_behavior"])
	})

	t.Run("the local plan is not touched, the webhook moves it", func(t *testing.T) {
		server, _ := stubCreemAPI(t, http.StatusOK, `{}`)
		repo := subscriberRepo(paying)
		repo.ApplySubscriptionEventFunc = func(context.Context, string, string, *model.Subscription) (model.WebhookApplyOutcome, error) {
			t.Fatal("changing plan must wait for the provider's webhook")
			return "", nil
		}

		require.NoError(t, newTestServiceWithAPI(repo, server.URL).ChangePlan(context.Background(), testUserID, PlanEnterprise))
	})

	t.Run("a free user has nothing to change", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, `{}`)
		svc := newTestServiceWithAPI(subscriberRepo(&model.Subscription{Plan: PlanFree, Status: StatusFree}), server.URL)

		err := svc.ChangePlan(context.Background(), testUserID, PlanPro)

		assert.ErrorIs(t, err, model.ErrNoActiveSubscription)
		assert.Empty(t, *requests)
	})

	t.Run("an unknown plan is refused before any lookup", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, `{}`)
		svc := newTestServiceWithAPI(subscriberRepo(paying), server.URL)

		err := svc.ChangePlan(context.Background(), testUserID, "platinum")

		assert.ErrorIs(t, err, model.ErrUnknownPlan)
		assert.Empty(t, *requests)
	})

	t.Run("a provider failure is reported", func(t *testing.T) {
		server, _ := stubCreemAPI(t, http.StatusBadRequest, `{"message":"nope"}`)
		svc := newTestServiceWithAPI(subscriberRepo(paying), server.URL)

		require.Error(t, svc.ChangePlan(context.Background(), testUserID, PlanEnterprise))
	})
}

func TestCancelSubscription(t *testing.T) {
	paying := &model.Subscription{Plan: PlanPro, Status: StatusActive, ExternalSubscriptionID: strPtr("sub_1")}

	t.Run("schedules the cancellation for the end of the period", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, `{"id":"sub_1","status":"scheduled_cancel"}`)
		svc := newTestServiceWithAPI(subscriberRepo(paying), server.URL)

		require.NoError(t, svc.CancelSubscription(context.Background(), testUserID))

		require.Len(t, *requests, 1)
		got := (*requests)[0]
		assert.Equal(t, "/v1/subscriptions/sub_1/cancel", got.Path)
		assert.Equal(t, "scheduled", got.Body["mode"], "an immediate cancel would end access the user paid for")
		assert.Equal(t, "cancel", got.Body["onExecute"])
	})

	t.Run("a user without a subscription has nothing to cancel", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, `{}`)
		svc := newTestServiceWithAPI(subscriberRepo(&model.Subscription{Plan: PlanFree}), server.URL)

		err := svc.CancelSubscription(context.Background(), testUserID)

		assert.ErrorIs(t, err, model.ErrNoActiveSubscription)
		assert.Empty(t, *requests)
	})
}

func TestCreatePortalSession(t *testing.T) {
	customer := &model.Subscription{ExternalAccountID: strPtr("cust_1")}

	t.Run("returns the customer portal link", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, `{"customer_portal_link":"https://creem.io/my-orders/login/abc"}`)
		svc := newTestServiceWithAPI(subscriberRepo(customer), server.URL)

		link, err := svc.CreatePortalSession(context.Background(), testUserID)

		require.NoError(t, err)
		assert.Equal(t, "https://creem.io/my-orders/login/abc", link)
		require.Len(t, *requests, 1)
		assert.Equal(t, "/v1/customers/billing", (*requests)[0].Path)
		assert.Equal(t, "cust_1", (*requests)[0].Body["customer_id"])
	})

	t.Run("a user with no Creem customer has no portal", func(t *testing.T) {
		server, requests := stubCreemAPI(t, http.StatusOK, `{}`)
		svc := newTestServiceWithAPI(subscriberRepo(&model.Subscription{}), server.URL)

		_, err := svc.CreatePortalSession(context.Background(), testUserID)

		assert.ErrorIs(t, err, model.ErrNoActiveSubscription)
		assert.Empty(t, *requests)
	})

	t.Run("a link that is not absolute HTTPS is refused", func(t *testing.T) {
		for _, unsafe := range []string{"javascript:alert(1)", "http://creem.io/x", "/relative", "https://creem.io@evil.example/x"} {
			server, _ := stubCreemAPI(t, http.StatusOK, `{"customer_portal_link":"`+unsafe+`"}`)
			svc := newTestServiceWithAPI(subscriberRepo(customer), server.URL)

			_, err := svc.CreatePortalSession(context.Background(), testUserID)

			require.Error(t, err, "link %q", unsafe)
		}
	})
}

func TestCheckoutSuccessURL(t *testing.T) {
	assert.Equal(t, "https://app.test/settings?subscription=success", CheckoutSuccessURL("https://app.test"))
	assert.Equal(t, "https://app.test/settings?subscription=success", CheckoutSuccessURL("https://app.test/"),
		"a trailing slash must not produce a double slash")
}
