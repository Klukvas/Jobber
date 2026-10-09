// Package creem wraps the parts of the Creem API and webhook contract that
// Jobber depends on.
//
// API reference: https://docs.creem.io/api-reference/introduction
// Requests authenticate with an `x-api-key` header. Test and live mode are
// separate hosts with separate keys.
package creem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/andreypavlenko/jobber/internal/platform/circuitbreaker"
)

const (
	// LiveBaseURL and TestBaseURL are the two Creem API hosts. A test key does not
	// work on the live host and vice versa.
	LiveBaseURL = "https://api.creem.io"
	TestBaseURL = "https://test-api.creem.io"

	requestTimeout = 15 * time.Second
	// maxResponseBytes caps every response read. Creem payloads are a few KB;
	// anything larger is a bug or an attack.
	maxResponseBytes = 1 << 20

	breakerFailureThreshold = 3
	breakerOpenDuration     = 30 * time.Second

	// updateBehaviorProrateNow charges the prorated difference immediately when a
	// subscription moves to another product.
	updateBehaviorProrateNow = "proration-charge-immediately"
)

// ErrNotConfigured is returned when the API key is missing. Callers must treat it
// as "billing is unavailable", never as "the request failed".
var ErrNotConfigured = errors.New("creem: API key is not configured")

// APIError reports a non-2xx response. The body is truncated and never contains
// request credentials.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("creem: API returned %d: %s", e.StatusCode, e.Body)
}

// Config holds the credentials and endpoint a Client needs.
type Config struct {
	APIKey string
	// BaseURL is LiveBaseURL or TestBaseURL; BaseURLFor picks one from the
	// billing environment.
	BaseURL string
}

// BaseURLFor returns the API host for the live or the test environment.
func BaseURLFor(isLive bool) string {
	if isLive {
		return LiveBaseURL
	}
	return TestBaseURL
}

// Client talks to the Creem API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	breaker    *circuitbreaker.Breaker
}

// NewClient creates a Creem API client. A missing key is allowed at construction
// time so the app can boot with billing switched off; every request then fails
// with ErrNotConfigured.
func NewClient(cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = TestBaseURL
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		httpClient: &http.Client{Timeout: requestTimeout},
		breaker:    circuitbreaker.New("creem", breakerFailureThreshold, breakerOpenDuration),
	}
}

// IsConfigured reports whether an API key is present.
func (c *Client) IsConfigured() bool { return c.apiKey != "" }

// do performs an authenticated request and decodes a JSON response into out.
// The response body is always bounded and always closed.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	if !c.IsConfigured() {
		return ErrNotConfigured
	}

	req, cancel, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer cancel()

	status, raw, err := c.send(req)
	if err != nil {
		return fmt.Errorf("creem: %s %s: %w", method, path, err)
	}

	if status < 200 || status >= 300 {
		return &APIError{StatusCode: status, Body: truncate(raw)}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("creem: decode response: %w", err)
	}
	return nil
}

// newRequest builds an authenticated request bound to a timeout. The caller must
// call the returned cancel function.
func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, context.CancelFunc, error) {
	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("creem: encode request: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}

	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("creem: build request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, cancel, nil
}

// send performs the request through the circuit breaker and returns the status
// and the bounded body.
//
// Only Creem's own faults trip the breaker: a 5xx or a transport failure. A 4xx
// is our bug or a bad ID and will not heal by backing off, and a caller that
// gave up (context.Canceled) says nothing about Creem's health.
func (c *Client) send(req *http.Request) (status int, raw []byte, err error) {
	var callerGaveUp error
	breakerErr := c.breaker.Execute(func() error {
		resp, doErr := c.httpClient.Do(req)
		if doErr != nil {
			if errors.Is(doErr, context.Canceled) {
				callerGaveUp = doErr
				return nil
			}
			return doErr
		}
		defer resp.Body.Close()

		status = resp.StatusCode
		raw, doErr = io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		if doErr != nil {
			return fmt.Errorf("read response: %w", doErr)
		}
		if status >= http.StatusInternalServerError {
			return &APIError{StatusCode: status, Body: truncate(raw)}
		}
		return nil
	})
	if breakerErr != nil {
		return 0, nil, breakerErr
	}
	if callerGaveUp != nil {
		return 0, nil, callerGaveUp
	}
	return status, raw, nil
}

// CheckoutRequest is the body of POST /v1/checkouts.
//
// https://docs.creem.io/api-reference/endpoint/create-checkout
type CheckoutRequest struct {
	ProductID  string            `json:"product_id"`
	SuccessURL string            `json:"success_url,omitempty"`
	Customer   *CheckoutCustomer `json:"customer,omitempty"`
	// Metadata is stored on the checkout and echoed back on its webhook events.
	// It is written with the API key, so a buyer cannot add to it.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// CheckoutCustomer pre-fills the buyer on the hosted checkout page.
type CheckoutCustomer struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// HostedCheckout is the part of the create-checkout response Jobber reads.
type HostedCheckout struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	CheckoutURL string `json:"checkout_url"`
}

// CreateCheckout creates a hosted checkout and returns the URL to send the
// buyer to.
func (c *Client) CreateCheckout(ctx context.Context, req CheckoutRequest) (*HostedCheckout, error) {
	var resp HostedCheckout
	if err := c.do(ctx, http.MethodPost, "/v1/checkouts", req, &resp); err != nil {
		return nil, err
	}
	if resp.CheckoutURL == "" {
		return nil, errors.New("creem: checkout response contained no checkout_url")
	}
	return &resp, nil
}

// UpgradeSubscription moves a subscription to another product, prorating the
// difference. Creem uses the same call for moving down as for moving up.
//
// https://docs.creem.io/api-reference/endpoint/upgrade-subscription
func (c *Client) UpgradeSubscription(ctx context.Context, subscriptionID, productID string) error {
	body := map[string]string{
		"product_id":      productID,
		"update_behavior": updateBehaviorProrateNow,
	}
	return c.do(ctx, http.MethodPost, "/v1/subscriptions/"+url.PathEscape(subscriptionID)+"/upgrade", body, nil)
}

// CancelSubscriptionAtPeriodEnd schedules the cancellation: the subscriber keeps
// access until the current billing period ends. An immediate cancellation is
// deliberately not exposed, since no user-facing path uses it.
//
// https://docs.creem.io/api-reference/endpoint/cancel-subscription
func (c *Client) CancelSubscriptionAtPeriodEnd(ctx context.Context, subscriptionID string) error {
	body := map[string]string{"mode": "scheduled", "onExecute": "cancel"}
	return c.do(ctx, http.MethodPost, "/v1/subscriptions/"+url.PathEscape(subscriptionID)+"/cancel", body, nil)
}

// CustomerPortalLink returns a login link to the Creem customer portal, where the
// buyer manages the payment method and sees invoices.
//
// https://docs.creem.io/features/customer-portal
func (c *Client) CustomerPortalLink(ctx context.Context, customerID string) (string, error) {
	var resp struct {
		Link string `json:"customer_portal_link"`
	}
	body := map[string]string{"customer_id": customerID}
	if err := c.do(ctx, http.MethodPost, "/v1/customers/billing", body, &resp); err != nil {
		return "", err
	}
	if resp.Link == "" {
		return "", errors.New("creem: portal response contained no customer_portal_link")
	}
	return resp.Link, nil
}

func truncate(body []byte) string {
	const maxErrorBody = 512
	if len(body) > maxErrorBody {
		return string(body[:maxErrorBody]) + "…"
	}
	return string(body)
}
