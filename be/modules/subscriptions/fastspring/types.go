package fastspring

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Documented values of a session's `checkoutStatus` list. READY_FOR_CHECKOUT is
// the only one that lets a session be handed to a buyer: PRODUCTS_REQUIRED means
// the cart is incomplete and CONCLUDED means the session has already been used.
//
// https://developer.fastspring.com/reference/createsession
const (
	CheckoutStatusReady            = "READY_FOR_CHECKOUT"
	CheckoutStatusProductsRequired = "PRODUCTS_REQUIRED"
	CheckoutStatusConcluded        = "CONCLUDED"
)

// SessionRequest is the body of POST /v2/checkouts/{checkoutPath}/sessions.
//
// This is the current Sessions API. The legacy POST /sessions endpoint is not
// used: FastSpring documents the v2 endpoint as the one every new integration
// must build against.
//
// https://developer.fastspring.com/reference/sessions-overview
// https://developer.fastspring.com/reference/createsession
type SessionRequest struct {
	// Live selects the store mode the session belongs to. It is always sent
	// explicitly — never left to a default — so a test deployment cannot open a
	// live checkout, or the reverse.
	Live bool `json:"live"`
	// Locale is the FastSpring locale tag used to render the hosted checkout.
	Locale    string            `json:"locale,omitempty"`
	Customer  *SessionCustomer  `json:"customer,omitempty"`
	OrderTags map[string]string `json:"orderTags,omitempty"`
	Cart      SessionCart       `json:"cart"`
}

// SessionCustomer identifies the buyer. ExternalAccountID is merchant-owned, so
// it is the server-side link between a FastSpring account and a local user.
type SessionCustomer struct {
	ExternalAccountID string          `json:"externalAccountId,omitempty"`
	BillToContact     *SessionContact `json:"billToContact,omitempty"`
}

// SessionContact pre-fills the buyer details on the checkout.
//
// There is deliberately no country here: the documented contact object has none
// (only email, first/last name, company and phone). `country` is a *top-level*
// session field, so adding it to a contact would silently do nothing.
type SessionContact struct {
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Email     string `json:"email,omitempty"`
}

// SessionCart holds the line items the checkout opens with.
type SessionCart struct {
	LineItems []SessionLineItem `json:"lineItems"`
}

// SessionLineItem is one cart line: a catalog product path and a quantity.
type SessionLineItem struct {
	ProductPath string `json:"productPath"`
	Quantity    int    `json:"quantity,omitempty"`
}

// SessionExpiry is a session expiry exactly as the provider sent it.
//
// The current API documents an ISO 8601 instant. Decoding accepts any JSON
// scalar so that an unexpected shape costs only this display-only field instead
// of failing an otherwise valid checkout.
type SessionExpiry string

// UnmarshalJSON accepts a JSON string, number or null.
func (e *SessionExpiry) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "null" {
		*e = ""
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*e = SessionExpiry(text)
		return nil
	}
	*e = SessionExpiry(strings.Trim(raw, `"`))
	return nil
}

// SessionResponse is the 201 body of a created session.
//
// `checkoutUrls.webcheckoutUrl` is deliberately not decoded. It addresses the
// full-page hosted Web Checkout, which Jobber no longer uses: the session `id`
// is handed to the Store Builder Library popup instead, exactly as the Sessions
// API documents. Decoding a URL nothing may navigate to would only invite it
// back into the purchase flow.
type SessionResponse struct {
	ID string `json:"id"`
	// Expires is an ISO 8601 instant, not a Unix timestamp.
	Expires SessionExpiry `json:"expires"`
	// CheckoutStatus is a *list* of status strings, not a single status: the
	// current Sessions API types `checkoutStatus` as an array, so readiness is
	// a membership test.
	CheckoutStatus []string                `json:"checkoutStatus"`
	Customer       SessionCustomerResponse `json:"customer"`
}

// SessionCustomerResponse carries the FastSpring account the session is bound
// to, which is what later webhook events are resolved through.
type SessionCustomerResponse struct {
	AccountID         string `json:"accountId"`
	ExternalAccountID string `json:"externalAccountId"`
}

// IsReady reports whether the session can be handed to a buyer, which is true
// only while READY_FOR_CHECKOUT is among its statuses. An absent or empty list
// is not ready: a session that never claimed to be payable must not be opened.
func (r SessionResponse) IsReady() bool {
	return slices.ContainsFunc(r.CheckoutStatus, func(status string) bool {
		return strings.EqualFold(status, CheckoutStatusReady)
	})
}

// CheckoutStatusString renders the status list for an error message, so a
// refused session says which statuses the provider actually reported.
func (r SessionResponse) CheckoutStatusString() string {
	if len(r.CheckoutStatus) == 0 {
		return "<none>"
	}
	return strings.Join(r.CheckoutStatus, ", ")
}

// ExpiresAt parses the ISO 8601 expiry. The second return value is false when
// the field is absent or unparseable — expiry is display-only, so callers omit
// it rather than failing a working checkout.
func (r SessionResponse) ExpiresAt() (time.Time, bool) {
	expires := strings.TrimSpace(string(r.Expires))
	if expires == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, expires)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

// Account is the subset of GET /accounts/{account_id} that Jobber reads.
type Account struct {
	ID     string `json:"id"`
	Lookup struct {
		Global string `json:"global"`
		Custom string `json:"custom"`
	} `json:"lookup"`
}

type updateSubscriptionsRequest struct {
	Subscriptions []updateSubscriptionItem `json:"subscriptions"`
}

type updateSubscriptionItem struct {
	Subscription string `json:"subscription"`
	Product      string `json:"product"`
	Quantity     int    `json:"quantity"`
	Prorate      bool   `json:"prorate"`
}

// subscriptionActionResponse covers both POST /subscriptions and
// DELETE /subscriptions/{id}: each returns HTTP 200 with a per-subscription
// result that may still be an error.
type subscriptionActionResponse struct {
	Subscriptions []subscriptionActionResult `json:"subscriptions"`
}

type subscriptionActionResult struct {
	Subscription string          `json:"subscription"`
	Action       string          `json:"action"`
	Result       string          `json:"result"`
	Error        json.RawMessage `json:"error"`
}

// firstError reports the first non-success entry, or an error when the response
// carried no entries at all. An empty list means the call silently did nothing,
// which must not be reported as success.
func (r subscriptionActionResponse) firstError(action string) error {
	if len(r.Subscriptions) == 0 {
		return &OperationError{Action: action, Detail: "response contained no subscriptions"}
	}
	for _, entry := range r.Subscriptions {
		if !strings.EqualFold(entry.Result, "success") {
			return &OperationError{Action: action, Detail: entry.errorDetail()}
		}
	}
	return nil
}

func (r subscriptionActionResult) errorDetail() string {
	if len(r.Error) > 0 {
		return truncate(r.Error)
	}
	return fmt.Sprintf("result=%q", r.Result)
}

type authenticateAccountResponse struct {
	Accounts []authenticateAccountResult `json:"accounts"`
}

type authenticateAccountResult struct {
	Account string          `json:"account"`
	Result  string          `json:"result"`
	URL     string          `json:"url"`
	Expires string          `json:"expires"`
	Error   json.RawMessage `json:"error"`
}

func (r authenticateAccountResult) errorDetail() string {
	if len(r.Error) > 0 {
		return truncate(r.Error)
	}
	return fmt.Sprintf("result=%q", r.Result)
}
