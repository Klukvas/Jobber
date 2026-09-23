package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/andreypavlenko/jobber/internal/platform/circuitbreaker"
	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
)

// The webhook endpoint is a push channel, and a push channel loses things.
// FastSpring retries a failed delivery for up to 7 days and at most 12 attempts,
// then marks it permanently failed and never sends it again — so a deployment
// that was down, misconfigured or holding the wrong secret for longer than that
// window ends up with subscription rows that are simply wrong, with nothing left
// to correct them and nobody to notice.
//
// Reconciliation is the pull channel that closes it: the Events API still lists
// every event FastSpring has no acknowledgement for, and those events run
// through the exact same pipeline as a delivered one. FastSpring documents this
// as the intended recovery path.
//
// https://developer.fastspring.com/reference/processed-and-unprocessed-webhook-events
const (
	// DefaultReconcileDays covers FastSpring's whole 7-day retry window with
	// room to spare, so a sweep cannot miss an event that has only just been
	// given up on.
	DefaultReconcileDays = 14

	// maxReconcilePages bounds one sweep. Anything left over is picked up by the
	// next one — every event applied here is acknowledged, so the backlog only
	// shrinks — and the bound is what keeps a provider answering `more: true`
	// forever from turning a sweep into an endless loop.
	maxReconcilePages = 20
)

// ReconcileResult summarises one reconciliation sweep.
type ReconcileResult struct {
	WebhookResult
	// Pages is how many pages of the Events API the sweep read.
	Pages int
	// Unacknowledged lists events that were applied (or deliberately skipped)
	// locally but could not be reported back to FastSpring. They are harmless:
	// the next sweep sees them again and the event claim recognises them as
	// duplicates, so they cost one extra no-op rather than a second grant.
	Unacknowledged []EventOutcome
}

// ReconcileMissedEvents pulls the events FastSpring still has no
// acknowledgement for and applies them exactly as the webhook endpoint would.
//
// The two channels differ in one way only, and it is not a weakening: a webhook
// body is trusted because its HMAC proves FastSpring sent it, while this
// response is trusted because Jobber fetched it over TLS with its own API
// credentials. Everything that reads the *contents* of an event — the
// environment guard, the product allowlist, the order-tag proof, the link
// guard, the event claim — is the same code and runs unchanged.
//
// Applying is idempotent, so a sweep that overlaps a live delivery is safe: the
// claim recognises the event ID and reports a duplicate instead of granting
// anything twice.
func (s *SubscriptionService) ReconcileMissedEvents(ctx context.Context, days int) (ReconcileResult, error) {
	if days <= 0 {
		days = DefaultReconcileDays
	}
	if !s.billing.IsConfigured() {
		return ReconcileResult{}, fastspring.ErrNotConfigured
	}
	// Without the secret no order tag can be proven, so a first purchase would
	// be pulled in and then refused as unprovable — noisily, and for every sweep
	// from now on. Nothing to reconcile with is better than reconciling wrong.
	if s.cfg.WebhookSecret == "" {
		return ReconcileResult{}, fastspring.ErrSecretMissing
	}

	var result ReconcileResult
	for pageNumber := 1; pageNumber <= maxReconcilePages; {
		page, err := s.billing.ListUnprocessedEvents(ctx, days, pageNumber)
		if err != nil {
			return result, fmt.Errorf("failed to list unprocessed billing events: %w", err)
		}
		result.Pages++

		alreadyAcknowledged := len(result.Processed)
		for _, event := range page.Events {
			s.processEvent(ctx, event, &result.WebhookResult)
		}
		s.acknowledgeReconciled(ctx, result.Processed[alreadyAcknowledged:], &result)

		next, ok := nextReconcilePage(page, pageNumber)
		if !ok {
			break
		}
		pageNumber = next
	}
	return result, nil
}

// acknowledgeReconciled reports the events this sweep settled back to
// FastSpring, so the next sweep does not see them again.
//
// Only events the pipeline is done with are acknowledged — applied, duplicate,
// superseded or deliberately skipped — which is the same bar the webhook
// endpoint's 200/202 answers to. A failed event is left alone on purpose: being
// unacknowledged is exactly what makes the next sweep retry it.
//
// A failure to acknowledge is recorded, not returned. The work already
// committed, and refusing to acknowledge the rest of the page because one call
// failed would only widen the backlog.
func (s *SubscriptionService) acknowledgeReconciled(ctx context.Context, eventIDs []string, result *ReconcileResult) {
	for _, eventID := range eventIDs {
		if err := s.billing.MarkEventProcessed(ctx, eventID); err != nil {
			result.Unacknowledged = append(result.Unacknowledged, EventOutcome{EventID: eventID, Err: err})
			// The circuit is open or the API is down: the rest of this page will
			// fail the same way, and each attempt costs a request.
			if errors.Is(err, fastspring.ErrNotConfigured) || errors.Is(err, circuitbreaker.ErrCircuitOpen) {
				return
			}
		}
	}
}

// nextReconcilePage reports the page to read next, or false when the sweep is
// done. A provider that answers `more: true` without advancing the cursor ends
// the sweep rather than re-reading one page forever.
func nextReconcilePage(page *fastspring.UnprocessedEventsPage, current int) (int, bool) {
	if !page.More {
		return 0, false
	}
	if page.NextPage != nil {
		if *page.NextPage <= current {
			return 0, false
		}
		return *page.NextPage, true
	}
	return current + 1, true
}
