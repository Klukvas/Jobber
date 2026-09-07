// Package fastspring wraps the parts of the FastSpring API and webhook contract
// that Jobber depends on.
//
// API reference: https://developer.fastspring.com/reference/api-overview
// All endpoints live under https://api.fastspring.com and use HTTP Basic auth
// with an API username/password pair created in the FastSpring app.
package fastspring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/circuitbreaker"
)

const (
	// DefaultBaseURL is the single FastSpring API host. Test and live mode are a
	// property of the store and of each order, not of the endpoint.
	DefaultBaseURL = "https://api.fastspring.com"

	// userAgent identifies Jobber on every API call. The API overview lists
	// User-Agent as mandatory — an unidentified request may be rejected — and a
	// stable value is what lets a request be traced back to this integration.
	//
	// https://developer.fastspring.com/reference/api-overview
	userAgent = "Jobber-Billing/1.0 (+https://jobber-app.com)"

	requestTimeout = 15 * time.Second
	// maxResponseBytes caps every response read. FastSpring subscription
	// payloads are a few KB; anything larger is a bug or an attack.
	maxResponseBytes = 1 << 20

	breakerFailureThreshold = 3
	breakerOpenDuration     = 30 * time.Second
)

// ErrNotConfigured is returned when API credentials are missing. Callers must
// treat it as "billing is unavailable", never as "the request failed".
var ErrNotConfigured = errors.New("fastspring: API credentials are not configured")

// APIError reports a non-2xx response. The body is truncated and never contains
// request credentials.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("fastspring: API returned %d: %s", e.StatusCode, e.Body)
}

// OperationError reports a per-item failure inside a 200 response. FastSpring
// answers 200 for partially failed batch operations and marks each entry with
// `result: "error"`, so a 2xx status alone does not mean success.
type OperationError struct {
	Action string
	Detail string
}

func (e *OperationError) Error() string {
	return fmt.Sprintf("fastspring: operation %q failed: %s", e.Action, e.Detail)
}

// Client talks to the FastSpring API.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	breaker    *circuitbreaker.Breaker
}

// Config holds the credentials and endpoints a Client needs.
type Config struct {
	BaseURL  string
	Username string
	Password string
}

// NewClient creates a FastSpring API client. Missing credentials are allowed at
// construction time so the app can boot with billing switched off; every request
// then fails with ErrNotConfigured.
func NewClient(cfg Config) *Client {
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL:    baseURL,
		username:   cfg.Username,
		password:   cfg.Password,
		httpClient: &http.Client{Timeout: requestTimeout},
		breaker:    circuitbreaker.New("fastspring", breakerFailureThreshold, breakerOpenDuration),
	}
}

// IsConfigured reports whether API credentials are present.
func (c *Client) IsConfigured() bool {
	return c.username != "" && c.password != ""
}

// do performs an authenticated request and decodes a JSON response into out.
// The response body is always bounded and always closed.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	if !c.IsConfigured() {
		return ErrNotConfigured
	}

	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("fastspring: encode request: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return fmt.Errorf("fastspring: build request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	var raw []byte
	var status int
	breakerErr := c.breaker.Execute(func() error {
		resp, doErr := c.httpClient.Do(req)
		if doErr != nil {
			return doErr
		}
		defer resp.Body.Close()

		status = resp.StatusCode
		raw, doErr = io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		if doErr != nil {
			return fmt.Errorf("fastspring: read response: %w", doErr)
		}
		// Only server-side faults should trip the breaker. A 4xx is our bug or a
		// bad ID and will not heal by backing off.
		if status >= http.StatusInternalServerError {
			return &APIError{StatusCode: status, Body: truncate(raw)}
		}
		return nil
	})
	if breakerErr != nil {
		return fmt.Errorf("fastspring: %s %s: %w", method, path, breakerErr)
	}

	if status < 200 || status >= 300 {
		return &APIError{StatusCode: status, Body: truncate(raw)}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("fastspring: decode response: %w", err)
	}
	return nil
}

// CreateSession creates a server-side checkout session against a dashboard
// checkout path and returns the session id and the account it is bound to. The
// id is what the Store Builder Library opens in the popup; no checkout URL is
// read, because Jobber never navigates a buyer to a hosted page.
//
// https://developer.fastspring.com/reference/sessions-overview
// https://developer.fastspring.com/reference/createsession
func (c *Client) CreateSession(ctx context.Context, checkoutPath string, req SessionRequest) (*SessionResponse, error) {
	escaped, err := EscapeCheckoutPath(checkoutPath)
	if err != nil {
		return nil, err
	}

	var resp SessionResponse
	if err := c.do(ctx, http.MethodPost, "/v2/checkouts/"+escaped+"/sessions", req, &resp); err != nil {
		return nil, err
	}
	if resp.ID == "" {
		return nil, errors.New("fastspring: session response contained no session ID")
	}
	return &resp, nil
}

// ChangeSubscriptionProduct switches an existing subscription to another catalog
// product path, prorating the difference.
//
// https://developer.fastspring.com/reference/update-a-subscription
func (c *Client) ChangeSubscriptionProduct(ctx context.Context, subscriptionID, productPath string) error {
	req := updateSubscriptionsRequest{
		Subscriptions: []updateSubscriptionItem{{
			Subscription: subscriptionID,
			Product:      productPath,
			Quantity:     1,
			Prorate:      true,
		}},
	}
	var resp subscriptionActionResponse
	if err := c.do(ctx, http.MethodPost, "/subscriptions", req, &resp); err != nil {
		return err
	}
	return resp.firstError("subscription.update")
}

// CancelSubscription cancels a subscription. With atPeriodEnd the subscription
// stays usable until its deactivation date (billingPeriod=1, the API default);
// otherwise it is cancelled immediately (billingPeriod=0).
//
// https://developer.fastspring.com/reference/cancel-a-subscription
func (c *Client) CancelSubscription(ctx context.Context, subscriptionID string, atPeriodEnd bool) error {
	billingPeriod := "0"
	if atPeriodEnd {
		billingPeriod = "1"
	}
	path := "/subscriptions/" + url.PathEscape(subscriptionID) + "?billingPeriod=" + billingPeriod

	var resp subscriptionActionResponse
	if err := c.do(ctx, http.MethodDelete, path, nil, &resp); err != nil {
		return err
	}
	return resp.firstError("subscription.cancel")
}

// AuthenticateAccount returns a pre-authenticated Account Management Portal URL,
// valid for 24 hours.
//
// https://developer.fastspring.com/reference/retrieve-authenticated-account-management-url
func (c *Client) AuthenticateAccount(ctx context.Context, accountID string) (string, error) {
	var resp authenticateAccountResponse
	path := "/accounts/" + url.PathEscape(accountID) + "/authenticate"
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return "", err
	}
	if len(resp.Accounts) == 0 {
		return "", errors.New("fastspring: authenticate response contained no accounts")
	}
	entry := resp.Accounts[0]
	if !strings.EqualFold(entry.Result, "success") || entry.URL == "" {
		return "", &OperationError{Action: "account.authenticate.get", Detail: entry.errorDetail()}
	}
	return entry.URL, nil
}

// GetAccount retrieves a customer account, including its custom lookup key.
//
// https://developer.fastspring.com/reference/retrieve-an-account
func (c *Client) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	var account Account
	path := "/accounts/" + url.PathEscape(accountID)
	if err := c.do(ctx, http.MethodGet, path, nil, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

func truncate(body []byte) string {
	const maxErrorBody = 512
	if len(body) > maxErrorBody {
		return string(body[:maxErrorBody]) + "…"
	}
	return string(body)
}
