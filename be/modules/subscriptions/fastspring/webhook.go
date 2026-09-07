package fastspring

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SignatureHeader is the header carrying the webhook payload signature.
// FastSpring may send it in any casing; net/http canonicalises header lookups.
//
// https://developer.fastspring.com/docs/message-security
const SignatureHeader = "X-FS-Signature"

// Webhook event types Jobber acts on.
//
// https://developer.fastspring.com/docs/webhooks-overview
const (
	EventSubscriptionActivated       = "subscription.activated"
	EventSubscriptionUpdated         = "subscription.updated"
	EventSubscriptionChargeCompleted = "subscription.charge.completed"
	EventSubscriptionChargeFailed    = "subscription.charge.failed"
	EventSubscriptionCanceled        = "subscription.canceled"
	EventSubscriptionUncanceled      = "subscription.uncanceled"
	EventSubscriptionDeactivated     = "subscription.deactivated"
	EventSubscriptionPaused          = "subscription.paused"
	EventSubscriptionResumed         = "subscription.resumed"
	EventSubscriptionPaymentOverdue  = "subscription.payment.overdue"
	EventOrderCompleted              = "order.completed"
)

// FastSpring subscription states.
//
// `canceled` means "cancellation scheduled": the subscription keeps working
// until its deactivation date. Only `deactivated` ends access.
const (
	StateActive      = "active"
	StateTrial       = "trial"
	StateOverdue     = "overdue"
	StateCanceled    = "canceled"
	StateDeactivated = "deactivated"
	StatePaused      = "paused"
)

var (
	// ErrSignatureMissing means the request carried no signature header.
	ErrSignatureMissing = errors.New("fastspring: missing webhook signature")
	// ErrSignatureInvalid means the signature did not match the payload.
	ErrSignatureInvalid = errors.New("fastspring: webhook signature mismatch")
	// ErrSecretMissing means no HMAC secret is configured, so no payload can be
	// trusted.
	ErrSecretMissing = errors.New("fastspring: webhook secret is not configured")
)

// VerifySignature checks the X-FS-Signature header against the raw request body.
//
// FastSpring computes base64(HMAC-SHA256(rawBody, secret)) using standard
// base64. The digests are compared decoded, so the comparison stays
// constant-time and a truncated signature cannot short-circuit it.
//
// https://developer.fastspring.com/docs/message-security
func VerifySignature(body []byte, signature, secret string) error {
	if secret == "" {
		return ErrSecretMissing
	}
	if strings.TrimSpace(signature) == "" {
		return ErrSignatureMissing
	}

	provided, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
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

// Event is one entry of a webhook payload. A single POST may batch many events.
//
// https://developer.fastspring.com/docs/webhooks-overview
type Event struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	// Live distinguishes real orders from test-mode orders.
	Live bool `json:"live"`
	// Created is a Unix timestamp in milliseconds.
	Created   int64           `json:"created"`
	Processed bool            `json:"processed"`
	Data      json.RawMessage `json:"data"`
}

// CreatedAt converts the millisecond timestamp to a time.Time.
func (e Event) CreatedAt() time.Time {
	return time.UnixMilli(e.Created).UTC()
}

type eventBatch struct {
	Events []Event `json:"events"`
}

// ParseEvents decodes a webhook body into its events. The body must already have
// passed VerifySignature.
func ParseEvents(body []byte) ([]Event, error) {
	var batch eventBatch
	if err := json.Unmarshal(body, &batch); err != nil {
		return nil, fmt.Errorf("fastspring: parse webhook body: %w", err)
	}
	return batch.Events, nil
}

// Subscription is the normalised view of a subscription lifecycle event, freed
// from FastSpring's per-event payload shapes.
type Subscription struct {
	ID          string
	AccountID   string
	ProductPath string
	State       string
	// Active is nil when the payload omitted the flag, which must not be
	// confused with an explicit false.
	Active *bool
	// ChangedAt is the provider's own timestamp for this state change. It is
	// what orders lifecycle events: the webhook envelope is re-stamped on a
	// manual resend, this is not.
	ChangedAt *time.Time
	Begin     *time.Time
	Next      *time.Time
	// Deactivation is the date access ends once a cancellation is scheduled.
	Deactivation *time.Time
}

// ParseSubscription extracts the normalised subscription from an event payload,
// handling both the flat subscription.* shape and the nested
// subscription.charge.* shape.
//
// https://developer.fastspring.com/reference/subscriptionactivated
// https://developer.fastspring.com/reference/subscription-charge-failed
func ParseSubscription(data []byte) (*Subscription, error) {
	// `subscription` is a string on flat payloads and an object on charge events,
	// so decode twice rather than fight one struct into both shapes.
	var flat struct {
		ID               string          `json:"id"`
		Subscription     json.RawMessage `json:"subscription"`
		State            string          `json:"state"`
		Active           *bool           `json:"active"`
		Account          json.RawMessage `json:"account"`
		Product          json.RawMessage `json:"product"`
		Changed          *int64          `json:"changed"`
		Begin            *int64          `json:"begin"`
		Next             *int64          `json:"next"`
		DeactivationDate *int64          `json:"deactivationDate"`
	}
	if err := json.Unmarshal(data, &flat); err != nil {
		return nil, fmt.Errorf("fastspring: parse subscription payload: %w", err)
	}

	// A charge event nests the whole subscription object under "subscription".
	if len(flat.Subscription) > 0 && flat.Subscription[0] == '{' {
		return ParseSubscription(flat.Subscription)
	}

	sub := &Subscription{
		ID:           flat.ID,
		State:        flat.State,
		Active:       flat.Active,
		AccountID:    decodeAccountID(flat.Account),
		ProductPath:  decodeProductPath(flat.Product),
		ChangedAt:    millisToTime(flat.Changed),
		Begin:        millisToTime(flat.Begin),
		Next:         millisToTime(flat.Next),
		Deactivation: millisToTime(flat.DeactivationDate),
	}
	if sub.ID == "" {
		var id string
		if err := json.Unmarshal(flat.Subscription, &id); err == nil {
			sub.ID = id
		}
	}
	if sub.ID == "" {
		return nil, errors.New("fastspring: subscription payload has no subscription ID")
	}
	return sub, nil
}

// decodeAccountID reads the account ID whether `account` is an object (most
// subscription events) or a bare ID string (charge events).
func decodeAccountID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var id string
	if err := json.Unmarshal(raw, &id); err == nil {
		return id
	}
	var object struct {
		ID      string `json:"id"`
		Account string `json:"account"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return ""
	}
	if object.ID != "" {
		return object.ID
	}
	return object.Account
}

// decodeProductPath reads the catalog product path whether `product` is an
// object or a bare path string.
func decodeProductPath(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var path string
	if err := json.Unmarshal(raw, &path); err == nil {
		return path
	}
	var object struct {
		Product string `json:"product"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return ""
	}
	return object.Product
}

func millisToTime(millis *int64) *time.Time {
	if millis == nil || *millis == 0 {
		return nil
	}
	t := time.UnixMilli(*millis).UTC()
	return &t
}
