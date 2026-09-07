package fastspring

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/circuitbreaker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient(Config{BaseURL: server.URL, Username: "api-user", Password: "api-pass"})
}

// testCheckoutPath is a dashboard popup checkout path in the documented
// "<store-id>/<checkout-id>" shape.
const testCheckoutPath = "fluxlab/popup-jobber"

// proSessionRequest is a minimal valid current-Sessions-API request body.
func proSessionRequest() SessionRequest {
	return SessionRequest{
		Live: false,
		Cart: SessionCart{LineItems: []SessionLineItem{{ProductPath: "jobber-pro", Quantity: 1}}},
	}
}

func TestClientRequiresCredentials(t *testing.T) {
	client := NewClient(Config{BaseURL: "https://example.invalid"})

	assert.False(t, client.IsConfigured())

	t.Run("every call fails without touching the network", func(t *testing.T) {
		_, sessionErr := client.CreateSession(context.Background(), testCheckoutPath, proSessionRequest())
		require.ErrorIs(t, sessionErr, ErrNotConfigured)

		require.ErrorIs(t, client.CancelSubscription(context.Background(), "sub-1", true), ErrNotConfigured)
		require.ErrorIs(t, client.ChangeSubscriptionProduct(context.Background(), "sub-1", "jobber-pro"), ErrNotConfigured)

		_, authErr := client.AuthenticateAccount(context.Background(), "acct-1")
		require.ErrorIs(t, authErr, ErrNotConfigured)
	})
}

func TestClientSendsBasicAuth(t *testing.T) {
	var gotUser, gotPass string
	var ok bool
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, ok = r.BasicAuth()
		_, _ = w.Write([]byte(`{"id":"sess-1","customer":{"accountId":"acct-1"}}`))
	})

	_, err := client.CreateSession(context.Background(), testCheckoutPath, proSessionRequest())

	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "api-user", gotUser)
	assert.Equal(t, "api-pass", gotPass)
}

func TestClientErrorsDoNotLeakCredentials(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"account":"forbidden"}}`))
	})

	_, err := client.CreateSession(context.Background(), testCheckoutPath, proSessionRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
	assert.NotContains(t, err.Error(), "api-pass")
	assert.NotContains(t, err.Error(), "api-user")
}

func TestClientBoundsResponseReads(t *testing.T) {
	// A hostile or broken upstream must not be able to stream unbounded data
	// into memory; the read stops at maxResponseBytes and JSON decoding fails.
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		chunk := make([]byte, 64*1024)
		for i := range chunk {
			chunk[i] = 'a'
		}
		_, _ = w.Write([]byte(`{"id":"`))
		for range 32 { // 2 MB, twice the cap
			_, _ = w.Write(chunk)
		}
		_, _ = w.Write([]byte(`"}`))
	})

	_, err := client.CreateSession(context.Background(), testCheckoutPath, proSessionRequest())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestClientRespectsContextCancellation(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.CreateSession(ctx, testCheckoutPath, proSessionRequest())

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestClientCircuitBreakerOpensOnServerErrors(t *testing.T) {
	var calls int
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	})

	for range breakerFailureThreshold {
		require.Error(t, client.CancelSubscription(context.Background(), "sub-1", true))
	}
	callsBeforeTrip := calls

	err := client.CancelSubscription(context.Background(), "sub-1", true)

	require.ErrorIs(t, err, circuitbreaker.ErrCircuitOpen)
	assert.Equal(t, callsBeforeTrip, calls, "an open circuit must not reach the provider")
}

func TestClientClientErrorsDoNotTripBreaker(t *testing.T) {
	var calls int
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
	})

	for range breakerFailureThreshold + 2 {
		require.Error(t, client.CancelSubscription(context.Background(), "sub-1", true))
	}

	assert.Equal(t, breakerFailureThreshold+2, calls, "a 4xx will not heal by backing off")
}

func TestCancelSubscriptionBillingPeriod(t *testing.T) {
	tests := []struct {
		name        string
		atPeriodEnd bool
		wantQuery   string
	}{
		{name: "end of period", atPeriodEnd: true, wantQuery: "billingPeriod=1"},
		{name: "immediate", atPeriodEnd: false, wantQuery: "billingPeriod=0"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotQuery, gotPath, gotMethod string
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				gotQuery, gotPath, gotMethod = r.URL.RawQuery, r.URL.Path, r.Method
				_, _ = w.Write([]byte(`{"subscriptions":[{"subscription":"sub-1","action":"subscription.cancel","result":"success"}]}`))
			})

			require.NoError(t, client.CancelSubscription(context.Background(), "sub-1", tc.atPeriodEnd))

			assert.Equal(t, http.MethodDelete, gotMethod)
			assert.Equal(t, "/subscriptions/sub-1", gotPath)
			assert.Equal(t, tc.wantQuery, gotQuery)
		})
	}
}

func TestSubscriptionActionResponseErrors(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "success",
			body: `{"subscriptions":[{"subscription":"sub-1","action":"subscription.cancel","result":"success"}]}`,
		},
		{
			name:    "per-item error inside a 200",
			body:    `{"subscriptions":[{"subscription":"sub-1","action":"subscription.cancel","result":"error","error":{"subscription":"not found"}}]}`,
			wantErr: true,
		},
		{
			name:    "empty list means nothing happened",
			body:    `{"subscriptions":[]}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			})

			err := client.CancelSubscription(context.Background(), "sub-1", true)

			if tc.wantErr {
				require.Error(t, err)
				var opErr *OperationError
				assert.True(t, errors.As(err, &opErr), "want an OperationError, got %T", err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAuthenticateAccount(t *testing.T) {
	t.Run("returns the portal URL", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/accounts/acct-1/authenticate", r.URL.Path)
			_, _ = w.Write([]byte(`{"accounts":[{"account":"acct-1","result":"success","url":"https://store.onfastspring.com/account/a/b"}]}`))
		})

		url, err := client.AuthenticateAccount(context.Background(), "acct-1")

		require.NoError(t, err)
		assert.Equal(t, "https://store.onfastspring.com/account/a/b", url)
	})

	t.Run("rejects a success result with no URL", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"accounts":[{"account":"acct-1","result":"success"}]}`))
		})

		_, err := client.AuthenticateAccount(context.Background(), "acct-1")

		require.Error(t, err)
	})

	t.Run("rejects an empty account list", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"accounts":[]}`))
		})

		_, err := client.AuthenticateAccount(context.Background(), "acct-1")

		require.Error(t, err)
	})
}

func TestGetAccountReadsCustomLookupKey(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts/acct-1", r.URL.Path)
		_, _ = w.Write([]byte(`{"id":"acct-1","account":"acct-1","lookup":{"global":"g-1","custom":"jobber-abc"}}`))
	})

	account, err := client.GetAccount(context.Background(), "acct-1")

	require.NoError(t, err)
	assert.Equal(t, "acct-1", account.ID)
	assert.Equal(t, "jobber-abc", account.Lookup.Custom)
}

func TestCreateSessionRejectsResponseWithoutID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"customer":{"accountId":"acct-1"}}`))
	})

	_, err := client.CreateSession(context.Background(), testCheckoutPath, proSessionRequest())

	require.Error(t, err)
}

func TestClientSendsIdentifyingUserAgent(t *testing.T) {
	// The FastSpring API overview requires a User-Agent, and a stable
	// identifying value is what lets a request be traced back to Jobber.
	var seen []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("User-Agent"))
		_, _ = w.Write([]byte(`{"id":"sess-1","customer":{"accountId":"acct-1"},
			"accounts":[{"account":"acct-1","result":"success","url":"https://s.onfastspring.com/a"}],
			"subscriptions":[{"subscription":"sub-1","action":"subscription.cancel","result":"success"}]}`))
	})

	_, err := client.CreateSession(context.Background(), testCheckoutPath, proSessionRequest())
	require.NoError(t, err)
	require.NoError(t, client.CancelSubscription(context.Background(), "sub-1", true))
	require.NoError(t, client.ChangeSubscriptionProduct(context.Background(), "sub-1", "jobber-pro"))
	_, err = client.AuthenticateAccount(context.Background(), "acct-1")
	require.NoError(t, err)
	_, err = client.GetAccount(context.Background(), "acct-1")
	require.NoError(t, err)

	require.Len(t, seen, 5, "every API call must be covered")
	for _, agent := range seen {
		assert.Equal(t, userAgent, agent)
		assert.Contains(t, agent, "Jobber")
	}
}

func TestCreateSessionUsesCurrentSessionsAPI(t *testing.T) {
	var gotPath, gotMethod, gotContentType string
	var gotBody map[string]any
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{
			"id":"sess-1",
			"expires":"2026-07-01T00:00:00Z",
			"checkoutStatus":["READY_FOR_CHECKOUT"],
			"customer":{"accountId":"acct-1","externalAccountId":"jobber-abc"},
			"checkoutUrls":{"webcheckoutUrl":"https://jobber.test.onfastspring.com/checkout/sess-1"}
		}`))
	})

	session, err := client.CreateSession(context.Background(), testCheckoutPath, SessionRequest{
		Live:   true,
		Locale: "en",
		Customer: &SessionCustomer{
			ExternalAccountID: "jobber-abc",
			BillToContact:     &SessionContact{FirstName: "Test", LastName: "Buyer", Email: "buyer@example.com"},
		},
		OrderTags: map[string]string{"jobber_user_id": "user-1"},
		Cart:      SessionCart{LineItems: []SessionLineItem{{ProductPath: "jobber-pro", Quantity: 1}}},
	})

	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/v2/checkouts/"+testCheckoutPath+"/sessions", gotPath,
		"new integrations must use the current Sessions API, not legacy POST /sessions")
	assert.Equal(t, "application/json", gotContentType)

	assert.Equal(t, true, gotBody["live"], "live must be sent explicitly")
	assert.Equal(t, "en", gotBody["locale"], "locale is a 2-letter language code, not a regional tag")
	customer := gotBody["customer"].(map[string]any)
	assert.Equal(t, "jobber-abc", customer["externalAccountId"])
	billTo := customer["billToContact"].(map[string]any)
	assert.Equal(t, "buyer@example.com", billTo["email"])
	assert.NotContains(t, billTo, "country",
		"the documented contact object has no country; it is a top-level session field")
	assert.Equal(t, "user-1", gotBody["orderTags"].(map[string]any)["jobber_user_id"])
	lineItems := gotBody["cart"].(map[string]any)["lineItems"].([]any)
	require.Len(t, lineItems, 1)
	assert.Equal(t, "jobber-pro", lineItems[0].(map[string]any)["productPath"])
	assert.Equal(t, float64(1), lineItems[0].(map[string]any)["quantity"])

	assert.Equal(t, "acct-1", session.Customer.AccountID)
	assert.Equal(t, "sess-1", session.ID,
		"the session id is what the popup opens; nothing else from the body is needed")
	assert.Equal(t, []string{CheckoutStatusReady}, session.CheckoutStatus,
		"checkoutStatus is a list of statuses, not a single status")
	assert.True(t, session.IsReady())

	expires, ok := session.ExpiresAt()
	require.True(t, ok, "the documented expiry is ISO 8601, not a Unix timestamp")
	assert.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), expires)
}

func TestSessionResponseExpiresAt(t *testing.T) {
	tests := []struct {
		name    string
		expires SessionExpiry
		wantOK  bool
	}{
		{name: "ISO 8601 with zone", expires: "2026-07-01T00:00:00Z", wantOK: true},
		{name: "ISO 8601 with offset", expires: "2026-07-01T02:00:00+02:00", wantOK: true},
		{name: "absent", expires: ""},
		{name: "legacy millisecond timestamp", expires: "1751328000000"},
		{name: "null", expires: ""},
		{name: "garbage", expires: "tomorrow-ish"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := SessionResponse{Expires: tc.expires}.ExpiresAt()

			assert.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				assert.Equal(t, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), got)
			}
		})
	}
}

func TestEscapeCheckoutPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "store id and checkout id", path: "jobberstore/web-jobber-checkout", want: "jobberstore/web-jobber-checkout"},
		{name: "dots are allowed inside a segment", path: "store.v2/web-x", want: "store.v2/web-x"},
		{name: "surrounding whitespace is trimmed", path: "  jobberstore/web-x  ", want: "jobberstore/web-x"},

		// The endpoint takes exactly "<store-id>/<checkout-id>". Every other
		// arity is a misconfiguration, not something to repair silently.
		{name: "single segment", path: "jobber-checkout", wantErr: true},
		{name: "three segments", path: "jobberstore/web-x/extra", wantErr: true},
		{name: "four segments", path: "a/b/c/d", wantErr: true},
		{name: "leading slash", path: "/jobberstore/web-x", wantErr: true},
		{name: "trailing slash", path: "jobberstore/web-x/", wantErr: true},
		{name: "both surrounding slashes", path: "/jobberstore/web-x/", wantErr: true},
		{name: "empty", path: "", wantErr: true},
		{name: "only a slash", path: "/", wantErr: true},
		{name: "only slashes", path: "///", wantErr: true},
		{name: "empty first segment", path: "/web-x", wantErr: true},
		{name: "empty second segment", path: "jobberstore/", wantErr: true},

		// Per-segment whitelisting still rejects anything that could bend the
		// URL towards another endpoint.
		{name: "parent traversal", path: "jobberstore/..", wantErr: true},
		{name: "parent traversal with a target", path: "jobberstore/../accounts", wantErr: true},
		{name: "absolute escape", path: "jobberstore//../../accounts/acct-1", wantErr: true},
		{name: "current-directory segment", path: "./web-x", wantErr: true},
		{name: "query injection", path: "jobberstore/web-x?limit=1", wantErr: true},
		{name: "encoded slash", path: "jobberstore%2Faccounts/web-x", wantErr: true},
		{name: "fragment", path: "jobberstore/web-x#frag", wantErr: true},
		{name: "whitespace inside", path: "jobber store/web-x", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EscapeCheckoutPath(tc.path)

			if tc.wantErr {
				require.ErrorIs(t, err, ErrInvalidCheckoutPath)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestCreateSessionRejectsUnsafeCheckoutPathBeforeCallingProvider(t *testing.T) {
	// Validation happens before the request is built, so neither a wrong arity
	// nor a hostile segment ever reaches the network. The two-segment cases
	// matter most: they pass the arity check and are stopped by the per-segment
	// whitelist alone.
	tests := []struct {
		name string
		path string
	}{
		{name: "traversal with an extra segment", path: "jobberstore/../accounts"},
		{name: "traversal inside two segments", path: "jobberstore/.."},
		{name: "encoded slash inside two segments", path: "jobberstore/web%2F..%2Faccounts"},
		{name: "query inside two segments", path: "jobberstore/web-x?limit=1"},
		{name: "wrong arity", path: "jobberstore"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var called bool
			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				called = true
				_, _ = w.Write([]byte(`{"id":"sess-1"}`))
			})

			_, err := client.CreateSession(context.Background(), tc.path, proSessionRequest())

			require.ErrorIs(t, err, ErrInvalidCheckoutPath)
			assert.False(t, called, "a path that could reach another endpoint must never be sent")
		})
	}
}

func TestSessionIsReadyReadsTheStatusList(t *testing.T) {
	// `checkoutStatus` is an array of statuses (PRODUCTS_REQUIRED,
	// READY_FOR_CHECKOUT, CONCLUDED), so readiness is membership — never
	// equality against a single string.
	tests := []struct {
		name     string
		statuses []string
		want     bool
	}{
		{name: "ready", statuses: []string{CheckoutStatusReady}, want: true},
		{name: "ready alongside another status", statuses: []string{CheckoutStatusProductsRequired, CheckoutStatusReady}, want: true},
		{name: "lower case", statuses: []string{"ready_for_checkout"}, want: true},
		{name: "products required", statuses: []string{CheckoutStatusProductsRequired}},
		{name: "already concluded", statuses: []string{CheckoutStatusConcluded}},
		{name: "several statuses, none ready", statuses: []string{CheckoutStatusProductsRequired, CheckoutStatusConcluded}},
		{name: "empty list", statuses: []string{}},
		{name: "absent", statuses: nil},
		{name: "near miss", statuses: []string{"READY_FOR_CHECKOUT_SOON"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, SessionResponse{CheckoutStatus: tc.statuses}.IsReady())
		})
	}
}

func TestCreateSessionDecodesTheStatusList(t *testing.T) {
	t.Run("a session that is not ready decodes every status it reported", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"sess-1","checkoutStatus":["PRODUCTS_REQUIRED","CONCLUDED"],
				"customer":{"accountId":"acct-1"}}`))
		})

		session, err := client.CreateSession(context.Background(), testCheckoutPath, proSessionRequest())

		require.NoError(t, err)
		assert.Equal(t, []string{CheckoutStatusProductsRequired, CheckoutStatusConcluded}, session.CheckoutStatus)
		assert.False(t, session.IsReady())
		assert.Equal(t, "PRODUCTS_REQUIRED, CONCLUDED", session.CheckoutStatusString(),
			"an error message must name every status the provider reported")
	})

	t.Run("a session with no status at all is not ready", func(t *testing.T) {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"sess-1","customer":{"accountId":"acct-1"}}`))
		})

		session, err := client.CreateSession(context.Background(), testCheckoutPath, proSessionRequest())

		require.NoError(t, err)
		assert.False(t, session.IsReady())
		assert.Equal(t, "<none>", session.CheckoutStatusString())
	})
}
