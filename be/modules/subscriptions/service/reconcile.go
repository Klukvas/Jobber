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
// Reconciliation is the pull channel that closes it. See ADR-0002 for the whole
// design; the rules that constrain this file are below.
//
// https://developer.fastspring.com/reference/processed-and-unprocessed-webhook-events
const (
	// ReconcileWindowDays covers FastSpring's whole 7-day retry window with room
	// to spare, so an event only just given up on cannot fall between two sweeps.
	//
	// It is also what bounds a *permanently* failing event. Such an event is
	// never acknowledged — that is what makes a transient failure retryable —
	// so it is re-listed every sweep until it falls out of this window. Widening
	// it therefore widens how long a stuck event keeps reporting; the endpoint's
	// own hard ceiling is 30 days, beyond which it returns nothing at all.
	ReconcileWindowDays = 14

	// maxReconcileRounds bounds one sweep. Anything left over is picked up by
	// the next one — every event cleared here is acknowledged, so the backlog
	// shrinks — and the bound is what keeps a provider that never stops
	// answering from turning a sweep into an endless loop.
	maxReconcileRounds = 20
)

// ReconcileResult summarises one reconciliation sweep.
type ReconcileResult struct {
	WebhookResult
	// Listings counts the calls made to the events endpoint. It is not a count
	// of distinct pages: clearing events moves the provider's list, so the sweep
	// deliberately re-reads from the start (see ReconcileMissedEvents).
	Listings int
	// Unacknowledged lists events that were settled locally but could not be
	// reported back to FastSpring. They are harmless: the next sweep sees them
	// again and the event claim recognises them as duplicates, so they cost one
	// extra no-op rather than a second grant.
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
func (s *SubscriptionService) ReconcileMissedEvents(ctx context.Context) (ReconcileResult, error) {
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
	page := 1
	for round := 0; round < maxReconcileRounds; round++ {
		listed, err := s.billing.ListUnprocessedEvents(ctx, ReconcileWindowDays, page)
		if err != nil {
			return result, fmt.Errorf("failed to list unprocessed billing events: %w", err)
		}
		result.Listings++
		if len(listed.Events) == 0 {
			break
		}

		settledBefore := len(result.Processed)
		for _, event := range listed.Events {
			s.processEvent(ctx, event, &result.WebhookResult)
		}
		cleared := s.acknowledgeReconciled(ctx, result.Processed[settledBefore:], &result)

		// Acknowledging removes those events from the provider's unprocessed
		// list, so the list this loop is paging through has just shrunk under
		// it: what was page 2 is now page 1, and asking for page 2 next would
		// step straight over a page's worth. A round that cleared anything
		// therefore starts again from the beginning, and only a round that
		// cleared nothing — where the list cannot have moved — advances the
		// cursor past the events it could not settle.
		//
		// That is also what terminates the loop: every round either shrinks the
		// list or advances the cursor, and the round count bounds both.
		if cleared > 0 {
			page = 1
			continue
		}
		next, ok := nextReconcilePage(listed, page)
		if !ok {
			break
		}
		page = next
	}
	return result, nil
}

// acknowledgeReconciled reports the events this sweep settled back to
// FastSpring and returns how many the provider actually accepted.
//
// Only events the pipeline is done with are acknowledged — applied, duplicate,
// superseded or deliberately skipped — which is the same bar the webhook
// endpoint's 200/202 answers to. A failed event is left alone on purpose: being
// unacknowledged is exactly what makes the next sweep retry it.
//
// A failure to acknowledge is recorded rather than returned. The work already
// committed, and abandoning the rest of the round because one call failed would
// only widen the backlog.
func (s *SubscriptionService) acknowledgeReconciled(
	ctx context.Context, eventIDs []string, result *ReconcileResult,
) int {
	cleared := 0
	for index, eventID := range eventIDs {
		err := s.billing.MarkEventProcessed(ctx, eventID)
		if err == nil {
			cleared++
			continue
		}
		result.Unacknowledged = append(result.Unacknowledged, EventOutcome{EventID: eventID, Err: err})

		// The circuit is open or the client is unusable: every remaining call
		// would fail the same way and each one costs a request. Give up on the
		// round — but record what is being given up on, or the count the caller
		// logs would quietly claim those events were settled.
		if errors.Is(err, fastspring.ErrNotConfigured) || errors.Is(err, circuitbreaker.ErrCircuitOpen) {
			for _, skipped := range eventIDs[index+1:] {
				result.Unacknowledged = append(result.Unacknowledged, EventOutcome{EventID: skipped, Err: err})
			}
			return cleared
		}
	}
	return cleared
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
