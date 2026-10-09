package creem

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/circuitbreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type seenRequest struct {
	Method      string
	Path        string
	APIKey      string
	ContentType string
	Body        map[string]any
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient(Config{APIKey: "creem_key", BaseURL: server.URL})
}

func respond(t *testing.T, status int, body string, seen *seenRequest) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		if seen != nil {
			*seen = seenRequest{
				Method: r.Method, Path: r.URL.EscapedPath(),
				APIKey: r.Header.Get("x-api-key"), ContentType: r.Header.Get("Content-Type"),
			}
			if len(raw) > 0 {
				require.NoError(t, json.Unmarshal(raw, &seen.Body))
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func TestBaseURLFor(t *testing.T) {
	assert.Equal(t, LiveBaseURL, BaseURLFor(true))
	assert.Equal(t, TestBaseURL, BaseURLFor(false))
}

func TestNewClient_DefaultsToTheTestHost(t *testing.T) {
	assert.Equal(t, TestBaseURL, NewClient(Config{APIKey: "k"}).baseURL,
		"an unconfigured client must never talk to the live API")
}

func TestClient_NotConfigured(t *testing.T) {
	client := NewClient(Config{})

	assert.False(t, client.IsConfigured())
	_, err := client.CreateCheckout(context.Background(), CheckoutRequest{ProductID: "prod_1"})
	assert.ErrorIs(t, err, ErrNotConfigured)
	assert.ErrorIs(t, client.CancelSubscriptionAtPeriodEnd(context.Background(), "sub_1"), ErrNotConfigured)
	assert.ErrorIs(t, client.UpgradeSubscription(context.Background(), "sub_1", "prod_1"), ErrNotConfigured)
	_, err = client.CustomerPortalLink(context.Background(), "cust_1")
	assert.ErrorIs(t, err, ErrNotConfigured)
}

func TestClient_CreateCheckout(t *testing.T) {
	t.Run("sends the key and the request, returns the hosted URL", func(t *testing.T) {
		var seen seenRequest
		client := newTestClient(t, respond(t, http.StatusOK,
			`{"id":"ch_1","status":"pending","checkout_url":"https://www.creem.io/checkout/ch_1"}`, &seen))

		checkout, err := client.CreateCheckout(context.Background(), CheckoutRequest{
			ProductID:  "prod_1",
			SuccessURL: "https://app.test/settings",
			Customer:   &CheckoutCustomer{Email: "a@b.c", Name: "A B"},
			Metadata:   map[string]string{"jobber_user_id": "u1"},
		})

		require.NoError(t, err)
		assert.Equal(t, "ch_1", checkout.ID)
		assert.Equal(t, "https://www.creem.io/checkout/ch_1", checkout.CheckoutURL)
		assert.Equal(t, http.MethodPost, seen.Method)
		assert.Equal(t, "/v1/checkouts", seen.Path)
		assert.Equal(t, "creem_key", seen.APIKey)
		assert.Equal(t, "application/json", seen.ContentType)
		assert.Equal(t, "prod_1", seen.Body["product_id"])
		assert.Equal(t, "https://app.test/settings", seen.Body["success_url"])
		assert.Equal(t, map[string]any{"jobber_user_id": "u1"}, seen.Body["metadata"])
	})

	t.Run("omits optional fields that are empty", func(t *testing.T) {
		var seen seenRequest
		client := newTestClient(t, respond(t, http.StatusOK, `{"id":"ch_1","checkout_url":"https://x.test"}`, &seen))

		_, err := client.CreateCheckout(context.Background(), CheckoutRequest{ProductID: "prod_1"})

		require.NoError(t, err)
		assert.Equal(t, map[string]any{"product_id": "prod_1"}, seen.Body)
	})

	t.Run("a response with no URL is an error", func(t *testing.T) {
		client := newTestClient(t, respond(t, http.StatusOK, `{"id":"ch_1"}`, nil))

		_, err := client.CreateCheckout(context.Background(), CheckoutRequest{ProductID: "prod_1"})

		require.Error(t, err)
	})

	t.Run("a body that is not JSON is an error", func(t *testing.T) {
		client := newTestClient(t, respond(t, http.StatusOK, `<html>`, nil))

		_, err := client.CreateCheckout(context.Background(), CheckoutRequest{ProductID: "prod_1"})

		require.Error(t, err)
	})
}

func TestClient_SubscriptionManagement(t *testing.T) {
	t.Run("upgrade prorates immediately", func(t *testing.T) {
		var seen seenRequest
		client := newTestClient(t, respond(t, http.StatusOK, `{"id":"sub_1"}`, &seen))

		require.NoError(t, client.UpgradeSubscription(context.Background(), "sub_1", "prod_2"))

		assert.Equal(t, "/v1/subscriptions/sub_1/upgrade", seen.Path)
		assert.Equal(t, map[string]any{"product_id": "prod_2", "update_behavior": "proration-charge-immediately"}, seen.Body)
	})

	t.Run("cancel is scheduled for the end of the period", func(t *testing.T) {
		var seen seenRequest
		client := newTestClient(t, respond(t, http.StatusOK, `{"id":"sub_1"}`, &seen))

		require.NoError(t, client.CancelSubscriptionAtPeriodEnd(context.Background(), "sub_1"))

		assert.Equal(t, "/v1/subscriptions/sub_1/cancel", seen.Path)
		assert.Equal(t, map[string]any{"mode": "scheduled", "onExecute": "cancel"}, seen.Body)
	})

	t.Run("a subscription ID cannot escape its path segment", func(t *testing.T) {
		var seen seenRequest
		client := newTestClient(t, respond(t, http.StatusOK, `{}`, &seen))

		require.NoError(t, client.CancelSubscriptionAtPeriodEnd(context.Background(), "../customers/x"))

		assert.Equal(t, "/v1/subscriptions/..%2Fcustomers%2Fx/cancel", seen.Path)
	})

	t.Run("portal link", func(t *testing.T) {
		var seen seenRequest
		client := newTestClient(t, respond(t, http.StatusOK, `{"customer_portal_link":"https://creem.io/my-orders/login/abc"}`, &seen))

		link, err := client.CustomerPortalLink(context.Background(), "cust_1")

		require.NoError(t, err)
		assert.Equal(t, "https://creem.io/my-orders/login/abc", link)
		assert.Equal(t, "/v1/customers/billing", seen.Path)
		assert.Equal(t, map[string]any{"customer_id": "cust_1"}, seen.Body)
	})

	t.Run("a portal response with no link is an error", func(t *testing.T) {
		client := newTestClient(t, respond(t, http.StatusOK, `{}`, nil))

		_, err := client.CustomerPortalLink(context.Background(), "cust_1")

		require.Error(t, err)
	})
}

func TestClient_Errors(t *testing.T) {
	t.Run("a 4xx surfaces its status and a truncated body", func(t *testing.T) {
		client := newTestClient(t, respond(t, http.StatusNotFound, strings.Repeat("x", 2000), nil))

		err := client.CancelSubscriptionAtPeriodEnd(context.Background(), "sub_1")

		var apiErr *APIError
		require.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusNotFound, apiErr.StatusCode)
		assert.Less(t, len(apiErr.Body), 600, "error bodies are bounded before they reach a log")
	})

	t.Run("4xx responses never trip the breaker", func(t *testing.T) {
		var calls atomic.Int32
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusBadRequest)
		})

		for i := 0; i < breakerFailureThreshold+3; i++ {
			err := client.UpgradeSubscription(context.Background(), "sub_1", "prod_1")
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
		}

		assert.EqualValues(t, breakerFailureThreshold+3, calls.Load(), "every call must still reach the API")
	})

	t.Run("a caller that gives up cannot re-close a breaker that Creem tripped", func(t *testing.T) {
		var calls atomic.Int32
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusBadGateway)
		})
		for i := 0; i < breakerFailureThreshold; i++ {
			require.Error(t, client.UpgradeSubscription(context.Background(), "sub_1", "prod_1"))
		}
		cancelled, cancel := context.WithCancel(context.Background())
		cancel()

		err := client.UpgradeSubscription(cancelled, "sub_1", "prod_1")

		require.Error(t, err)
		assert.EqualValues(t, breakerFailureThreshold, calls.Load(), "the open breaker refuses before any request")
		assert.ErrorIs(t, client.UpgradeSubscription(context.Background(), "sub_1", "prod_1"), circuitbreaker.ErrCircuitOpen,
			"a cancelled call must leave the breaker open")
	})

	t.Run("a caller that gives up never counts against Creem", func(t *testing.T) {
		var calls atomic.Int32
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusOK)
		})
		cancelled, cancel := context.WithCancel(context.Background())
		cancel()

		for i := 0; i < breakerFailureThreshold+2; i++ {
			err := client.UpgradeSubscription(cancelled, "sub_1", "prod_1")
			require.ErrorIs(t, err, context.Canceled)
		}

		require.NoError(t, client.UpgradeSubscription(context.Background(), "sub_1", "prod_1"),
			"the breaker must still be closed after the caller's own cancellations")
		assert.EqualValues(t, 1, calls.Load())
	})

	t.Run("a timeout still counts against Creem, unlike a caller giving up", func(t *testing.T) {
		// Only context.Canceled is neutral. A deadline that expires while Creem
		// does not answer is exactly the failure the breaker exists to catch.
		var calls atomic.Int32
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			// The server only notices a vanished client once the body is read.
			_, _ = io.Copy(io.Discard, r.Body)
			select {
			case <-r.Context().Done():
			case <-release:
			}
		})

		for i := 0; i < breakerFailureThreshold+2; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			err := client.UpgradeSubscription(ctx, "sub_1", "prod_1")
			cancel()
			require.Error(t, err)
		}

		assert.EqualValues(t, breakerFailureThreshold, calls.Load(),
			"the breaker must open after the threshold of timeouts and refuse the rest")
		assert.ErrorIs(t, client.UpgradeSubscription(context.Background(), "sub_1", "prod_1"), circuitbreaker.ErrCircuitOpen)
	})

	t.Run("repeated server faults open the breaker instead of hammering Creem", func(t *testing.T) {
		var calls atomic.Int32
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusBadGateway)
		})

		for i := 0; i < breakerFailureThreshold+3; i++ {
			require.Error(t, client.UpgradeSubscription(context.Background(), "sub_1", "prod_1"))
		}

		assert.EqualValues(t, breakerFailureThreshold, calls.Load())
	})

	t.Run("a response larger than the cap is cut off", func(t *testing.T) {
		huge := `{"checkout_url":"https://x.test","pad":"` + strings.Repeat("a", maxResponseBytes) + `"}`
		client := newTestClient(t, respond(t, http.StatusOK, huge, nil))

		_, err := client.CreateCheckout(context.Background(), CheckoutRequest{ProductID: "prod_1"})

		require.Error(t, err, "a truncated body cannot decode")
	})
}
