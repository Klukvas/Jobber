package fastspring

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// The Events API is how a webhook delivery that never landed is recovered.
//
// FastSpring retries a failed delivery for up to 7 days (at most 12 attempts)
// and then marks it permanently failed — after which nothing is pushed again and
// the local row stays wrong until somebody notices. The documented remedy is a
// scheduled task that pulls the events still marked unprocessed and acknowledges
// them once they are applied, which is exactly what service.ReconcileMissedEvents
// does with these two calls.
//
// https://developer.fastspring.com/reference/processed-and-unprocessed-webhook-events
// https://developer.fastspring.com/reference/list-all-unprocessed-events
// https://developer.fastspring.com/reference/update-an-event
const (
	// MaxUnprocessedEventDays is the largest window the endpoint accepts.
	MaxUnprocessedEventDays = 30
)

// UnprocessedEventsPage is one page of GET /events/unprocessed.
//
// The events carry the same shape as a webhook delivery — id, type, live,
// created and data — so they run through the very same handler. What they do
// *not* carry is a signature: this response is authenticated by the API
// credentials and TLS of a call Jobber itself made, not by an HMAC over a body
// someone else posted. Every check that reads the payload's own contents,
// the order-tag proof above all, still applies unchanged.
type UnprocessedEventsPage struct {
	Events []Event `json:"events"`
	Page   int     `json:"page"`
	// More reports whether further pages exist. NextPage is null on the last
	// page, so it is a pointer rather than a zero that reads like page 0.
	More     bool `json:"more"`
	NextPage *int `json:"nextPage"`
}

// ListUnprocessedEvents returns one page of webhook events FastSpring still has
// no acknowledgement for, within the last `days` days.
//
// https://developer.fastspring.com/reference/list-all-unprocessed-events
func (c *Client) ListUnprocessedEvents(ctx context.Context, days, page int) (*UnprocessedEventsPage, error) {
	if days < 1 || days > MaxUnprocessedEventDays {
		return nil, fmt.Errorf("fastspring: unprocessed-events window must be 1..%d days, got %d",
			MaxUnprocessedEventDays, days)
	}

	query := url.Values{}
	query.Set("days", fmt.Sprint(days))
	if page > 1 {
		query.Set("page", fmt.Sprint(page))
	}

	var resp UnprocessedEventsPage
	if err := c.do(ctx, c.backgroundBreaker, http.MethodGet, "/events/unprocessed?"+query.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// MarkEventProcessed acknowledges an event pulled from the Events API, so the
// next sweep does not see it again.
//
// It is called only after the event has actually been applied or deliberately
// skipped — the same bar the webhook endpoint's 200/202 answers to. An event
// that failed is left unacknowledged on purpose: it is what the next sweep
// retries.
//
// https://developer.fastspring.com/reference/update-an-event
func (c *Client) MarkEventProcessed(ctx context.Context, eventID string) error {
	if eventID == "" {
		return fmt.Errorf("fastspring: cannot acknowledge an event with no ID")
	}
	body := struct {
		Processed bool `json:"processed"`
	}{Processed: true}
	return c.do(ctx, c.backgroundBreaker, http.MethodPost, "/events/"+url.PathEscape(eventID), body, nil)
}
