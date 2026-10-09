package creem

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SignatureHeader carries the hex HMAC-SHA256 of the raw request body.
//
// https://docs.creem.io/code/webhooks
const SignatureHeader = "creem-signature"

// Webhook event types Jobber acts on or has to recognise.
const (
	EventCheckoutCompleted           = "checkout.completed"
	EventSubscriptionActive          = "subscription.active"
	EventSubscriptionPaid            = "subscription.paid"
	EventSubscriptionUpdate          = "subscription.update"
	EventSubscriptionTrialing        = "subscription.trialing"
	EventSubscriptionPaused          = "subscription.paused"
	EventSubscriptionScheduledCancel = "subscription.scheduled_cancel"
	EventSubscriptionCanceled        = "subscription.canceled"
	EventSubscriptionPastDue         = "subscription.past_due"
	EventSubscriptionUnpaid          = "subscription.unpaid"
	// EventRefundCreated and EventDisputeCreated are subscribed to so that money
	// leaving the account is visible at all, not because anything is applied: a
	// partial refund leaves the subscription billing.
	EventRefundCreated  = "refund.created"
	EventDisputeCreated = "dispute.created"
)

// Creem subscription statuses.
//
// https://docs.creem.io/api-reference/endpoint/get-subscription
const (
	StatusActive          = "active"
	StatusTrialing        = "trialing"
	StatusPaused          = "paused"
	StatusScheduledCancel = "scheduled_cancel"
	StatusPastDue         = "past_due"
	StatusUnpaid          = "unpaid"
	StatusCanceled        = "canceled"
)

// modeProd is the `mode` Creem stamps on objects created in live mode. Test mode
// objects say "test" or "sandbox".
const modeProd = "prod"

var (
	// ErrSignatureMissing means the request carried no signature header.
	ErrSignatureMissing = errors.New("creem: missing webhook signature")
	// ErrSignatureInvalid means the signature did not match the payload.
	ErrSignatureInvalid = errors.New("creem: webhook signature mismatch")
	// ErrSecretMissing means no HMAC secret is configured, so no payload can be
	// trusted.
	ErrSecretMissing = errors.New("creem: webhook secret is not configured")
)

// VerifySignature checks the creem-signature header against the raw request body.
//
// Creem sends the lowercase hex of HMAC-SHA256(rawBody, secret). The digests are
// compared decoded and in constant time, so a truncated or malformed signature
// cannot short-circuit the check.
func VerifySignature(body []byte, signature, secret string) error {
	if secret == "" {
		return ErrSecretMissing
	}
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return ErrSignatureMissing
	}

	provided, err := hex.DecodeString(signature)
	if err != nil {
		return ErrSignatureInvalid
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	if !hmac.Equal(provided, mac.Sum(nil)) {
		return ErrSignatureInvalid
	}
	return nil
}

// Event is one webhook delivery. Unlike a batched provider, Creem posts a single
// event per request.
type Event struct {
	ID   string `json:"id"`
	Type string `json:"eventType"`
	// Created is a Unix timestamp in milliseconds.
	Created int64           `json:"created_at"`
	Object  json.RawMessage `json:"object"`
}

// CreatedAt converts the millisecond timestamp to a time.Time.
func (e Event) CreatedAt() time.Time {
	return time.UnixMilli(e.Created).UTC()
}

// ParseEvent decodes a webhook body. The body must already have passed
// VerifySignature.
func ParseEvent(body []byte) (Event, error) {
	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		return Event{}, fmt.Errorf("creem: parse webhook body: %w", err)
	}
	return event, nil
}

// Subscription is the normalised view of a subscription.* event object.
type Subscription struct {
	ID         string
	CustomerID string
	ProductID  string
	Status     string
	// IsLive is false for test-mode objects. HasMode is false when the payload
	// carried no mode at all, which must not be read as "test".
	IsLive  bool
	HasMode bool
	// Metadata is the checkout metadata echoed onto the subscription. Only string
	// values are kept.
	Metadata           map[string]string
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time
	// UpdatedAt is the provider's own timestamp for this state, which is what
	// orders lifecycle events: a manual resend arrives under a new event ID but
	// describes the original state.
	UpdatedAt *time.Time
}

// CompletedCheckout is the normalised view of a checkout.completed event object.
type CompletedCheckout struct {
	CustomerID string
	ProductID  string
	IsLive     bool
	HasMode    bool
	Metadata   map[string]string
}

type rawSubscription struct {
	ID                 string          `json:"id"`
	Mode               string          `json:"mode"`
	Status             string          `json:"status"`
	Product            json.RawMessage `json:"product"`
	Customer           json.RawMessage `json:"customer"`
	Items              []rawItem       `json:"items"`
	Metadata           json.RawMessage `json:"metadata"`
	CurrentPeriodStart flexibleTime    `json:"current_period_start_date"`
	CurrentPeriodEnd   flexibleTime    `json:"current_period_end_date"`
	UpdatedAt          flexibleTime    `json:"updated_at"`
}

type rawItem struct {
	ProductID string `json:"product_id"`
}

type rawCheckout struct {
	Mode     string          `json:"mode"`
	Product  json.RawMessage `json:"product"`
	Customer json.RawMessage `json:"customer"`
	Metadata json.RawMessage `json:"metadata"`
}

// ParseSubscription extracts the normalised subscription from an event object.
func ParseSubscription(object []byte) (*Subscription, error) {
	var raw rawSubscription
	if err := json.Unmarshal(object, &raw); err != nil {
		return nil, fmt.Errorf("creem: parse subscription object: %w", err)
	}
	if raw.ID == "" {
		return nil, errors.New("creem: subscription object has no ID")
	}

	productID := decodeID(raw.Product)
	if productID == "" && len(raw.Items) > 0 {
		productID = raw.Items[0].ProductID
	}

	return &Subscription{
		ID:                 raw.ID,
		CustomerID:         decodeID(raw.Customer),
		ProductID:          productID,
		Status:             raw.Status,
		IsLive:             raw.Mode == modeProd,
		HasMode:            raw.Mode != "",
		Metadata:           decodeMetadata(raw.Metadata),
		CurrentPeriodStart: raw.CurrentPeriodStart.ptr(),
		CurrentPeriodEnd:   raw.CurrentPeriodEnd.ptr(),
		UpdatedAt:          raw.UpdatedAt.ptr(),
	}, nil
}

// ParseCompletedCheckout extracts the normalised checkout from an event object.
func ParseCompletedCheckout(object []byte) (*CompletedCheckout, error) {
	var raw rawCheckout
	if err := json.Unmarshal(object, &raw); err != nil {
		return nil, fmt.Errorf("creem: parse checkout object: %w", err)
	}
	return &CompletedCheckout{
		CustomerID: decodeID(raw.Customer),
		ProductID:  decodeID(raw.Product),
		IsLive:     raw.Mode == modeProd,
		HasMode:    raw.Mode != "",
		Metadata:   decodeMetadata(raw.Metadata),
	}, nil
}

// decodeID reads the ID whether the field is a bare ID string or an expanded
// object, which Creem uses interchangeably across events.
func decodeID(raw json.RawMessage) string {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return ""
	}
	var id string
	if err := json.Unmarshal(raw, &id); err == nil {
		return id
	}
	var object struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return ""
	}
	return object.ID
}

// decodeMetadata reads the metadata object leniently: a value of an unexpected
// type costs that one entry rather than failing an otherwise valid event.
func decodeMetadata(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil
	}
	metadata := make(map[string]string, len(object))
	for key, value := range object {
		var text string
		if err := json.Unmarshal(value, &text); err == nil {
			metadata[key] = text
		}
	}
	return metadata
}

// flexibleTime accepts the two timestamp encodings Creem uses: an RFC 3339
// string, or a number of milliseconds since the epoch. A null, an empty value or
// anything unrecognised decodes to "absent" instead of failing the whole event.
type flexibleTime struct {
	value time.Time
	valid bool
}

func (t *flexibleTime) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}

	if unquoted, err := strconv.Unquote(trimmed); err == nil {
		parsed, parseErr := time.Parse(time.RFC3339Nano, unquoted)
		if parseErr != nil {
			return nil
		}
		t.value, t.valid = parsed.UTC(), true
		return nil
	}

	millis, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || millis <= 0 {
		return nil
	}
	t.value, t.valid = time.UnixMilli(millis).UTC(), true
	return nil
}

func (t flexibleTime) ptr() *time.Time {
	if !t.valid {
		return nil
	}
	value := t.value
	return &value
}
