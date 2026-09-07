package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
)

// EventOutcome records what happened to one event in a webhook batch, so the
// HTTP layer can log it without the service needing a logger.
type EventOutcome struct {
	EventID   string
	EventType string
	Err       error
}

// WebhookResult summarises a processed batch.
//
// FastSpring acknowledges a batch with HTTP 200 (all events processed) or
// HTTP 202 plus the processed event IDs, one per line.
//
// https://developer.fastspring.com/docs/processed-and-unprocessed-webhook-events
type WebhookResult struct {
	// Processed lists every event FastSpring should stop redelivering: handled
	// successfully, an already-seen duplicate, or deliberately not actionable.
	Processed []string
	// Failed lists events that must be retried.
	Failed []EventOutcome
	// Skipped lists events that were acknowledged without touching the database.
	Skipped []EventOutcome
	// Received counts every event in the batch, including ones acknowledged
	// without an ID to report back.
	Received int
}

// AllProcessed reports whether every event in the batch can be acknowledged.
func (r WebhookResult) AllProcessed() bool { return len(r.Failed) == 0 }

// Total returns the number of events in the batch.
func (r WebhookResult) Total() int { return r.Received }

// acknowledge records an event as processed. An event with no ID cannot be
// named in the 202 body, so it is counted but never reported as an empty ID.
func (r *WebhookResult) acknowledge(eventID string) {
	if eventID != "" {
		r.Processed = append(r.Processed, eventID)
	}
}

// skip acknowledges an event that was deliberately not applied.
func (r *WebhookResult) skip(outcome EventOutcome) {
	r.Skipped = append(r.Skipped, outcome)
	r.acknowledge(outcome.EventID)
}

// ErrEnvironmentMismatch marks a test event received by a live deployment (or
// vice versa). It is acknowledged, never applied: retrying could not change the
// outcome, and applying it would cross the live/test boundary.
//
// It is exported because acknowledging it *loses* the event permanently — the
// only cause is a deployment pointed at the wrong billing environment — so the
// HTTP layer reports it louder than an ordinary skip.
var ErrEnvironmentMismatch = errors.New("event environment does not match the configured billing environment")

// ErrTaggedOwnerConflict marks an event whose order tag names a user who
// already holds a *different*, still-billing provider subscription.
//
// Jobber writes the tag only for a first checkout, and refuses to open one for a
// user who already pays — so a proven tag naming such a user did not come from
// that flow. Its owner can replay their own proof onto a storefront purchase
// Jobber never brokered, which is exactly what this refuses: the row carries a
// single external_subscription_id, and overwriting it would leave the
// subscription that user actually pays for billing at FastSpring with nothing in
// Jobber pointing at it. The event is acknowledged rather than retried — no
// redelivery could make it safe — and it is exported because a tag that
// contradicts the row is worth looking at, not routine.
var ErrTaggedOwnerConflict = errors.New("order tag names a user who already holds another provider subscription")

// ErrSubscriptionLinkConflict marks an event that would repoint a user's row at
// a provider subscription other than the still-billing one it already names.
//
// The row carries a single external_subscription_id. Replacing it while the old
// subscription is alive would leave that subscription billing at FastSpring with
// nothing in Jobber able to cancel it, so the write refuses — atomically, under
// the row lock, whichever hop resolved the owner. ErrTaggedOwnerConflict is the
// same invariant caught one step earlier and with a sharper diagnosis, on the
// one hop whose input a buyer can influence; this is the guard that covers the
// rest, including two checkouts a user started while free and then both paid.
//
// The event is acknowledged rather than retried — redelivery cannot make the
// second subscription fit a row that is already spoken for — and it is exported
// because it means a subscriber may be paying twice, which is worth waking
// someone up for.
var ErrSubscriptionLinkConflict = errors.New("event describes a provider subscription other than the live one linked to this user")

// ErrUnprovenOrderTag marks an event whose order tags name a Jobber user without
// the proof Jobber's own server mints alongside that name.
//
// Order tags are not a server-only channel: the Store Builder Library exposes
// `fastspring.builder.tag()`, so any visitor to the shared storefront can attach
// arbitrary tags to their own order, and the webhook HMAC proves only that
// FastSpring sent the event — never who authored a tag inside it. Without the
// proof, a bare user ID is a request, not an identifier, and honouring it would
// let a stranger's purchase claim someone else's account. See ordertag.go.
//
// The event is acknowledged rather than retried: redelivering the same
// unprovable claim would only ask the same question again. It is exported
// because on one of Jobber's own products this should never happen by accident —
// it is either a forged tag or a proof minted under a secret this deployment no
// longer holds, and both are worth waking someone up for.
var ErrUnprovenOrderTag = errors.New("order tag names a user without a valid Jobber proof")

var (
	// errEventNotActionable marks an event type Jobber subscribes to but does
	// not act on.
	errEventNotActionable = errors.New("event type is not actionable")
	// errForeignBillingAccount marks an event about a FastSpring account that is
	// not Jobber's. The FluxLab store is shared with the other products sold
	// from it, and their subscription lifecycle events reach this endpoint too.
	// Such an event makes no order-tag claim of its own and its account carries
	// no Jobber lookup key, and no retry could ever make one appear, so it is
	// acknowledged and dropped rather than redelivered forever.
	errForeignBillingAccount = errors.New("billing account belongs to another product in the shared store")
	// errEventSuperseded marks a replayed or out-of-order event that does not
	// describe a change newer than the state already applied. An event carrying
	// the same `data.changed` counts as superseded: it describes a change that
	// has already been accounted for, so re-applying it could only undo a
	// correct write.
	errEventSuperseded = errors.New("event is not newer than the applied subscription state")
	// errEventDuplicate marks a redelivery of an event that was already applied.
	errEventDuplicate = errors.New("event was already processed")
	// errEventMissingID marks an actionable event with no ID. It cannot be
	// de-duplicated, so it is treated as a malformed delivery and retried
	// instead of being applied blind.
	errEventMissingID = errors.New("actionable event carried no event ID")
)

// HandleWebhook verifies and processes a FastSpring webhook batch.
//
// The signature is checked against the raw body before anything is parsed or
// written, so an unsigned or forged request can never reach the database.
//
// https://developer.fastspring.com/docs/message-security
// https://developer.fastspring.com/docs/webhooks-overview
func (s *SubscriptionService) HandleWebhook(ctx context.Context, body []byte, signature string) (WebhookResult, error) {
	if err := fastspring.VerifySignature(body, signature, s.cfg.WebhookSecret); err != nil {
		return WebhookResult{}, err
	}

	events, err := fastspring.ParseEvents(body)
	if err != nil {
		return WebhookResult{}, err
	}

	var result WebhookResult
	for _, event := range events {
		s.processEvent(ctx, event, &result)
	}
	return result, nil
}

// processEvent runs one event through the environment guard and its handler,
// recording the outcome. De-duplication is not a step here: it happens inside
// the handler's single atomic write, together with the state change.
func (s *SubscriptionService) processEvent(ctx context.Context, event fastspring.Event, result *WebhookResult) {
	result.Received++
	outcome := EventOutcome{EventID: event.ID, EventType: event.Type}

	if event.Live != s.cfg.IsLive() {
		outcome.Err = ErrEnvironmentMismatch
		result.skip(outcome)
		return
	}

	handle := s.handlerFor(event.Type)
	if handle == nil {
		outcome.Err = errEventNotActionable
		result.skip(outcome)
		return
	}

	// An actionable event without an ID cannot be claimed, so applying it would
	// give up idempotency entirely. Fail it instead: nothing is written and the
	// provider redelivers.
	if event.ID == "" {
		outcome.Err = errEventMissingID
		result.Failed = append(result.Failed, outcome)
		return
	}

	err := handle(ctx, event)
	switch {
	case err == nil:
		result.acknowledge(event.ID)
	case errors.Is(err, errEventDuplicate):
		// Already applied under this ID; acknowledging stops the redelivery.
		outcome.Err = err
		result.skip(outcome)
	case errors.Is(err, errEventSuperseded):
		// The newer state already stands and the event is recorded as processed
		// by the same statement, so acknowledging stops pointless retries.
		outcome.Err = err
		result.skip(outcome)
	case errors.Is(err, ErrTaggedOwnerConflict), errors.Is(err, ErrUnprovenOrderTag),
		errors.Is(err, ErrSubscriptionLinkConflict):
		// The event either cannot be shown to be ours or contradicts the row it
		// names. A retry would only ask the same question again, so
		// acknowledging drops the event; the handler logs all three loudly.
		outcome.Err = err
		result.skip(outcome)
	case errors.Is(err, errForeignBillingAccount):
		// Another product's subscription in the shared store. Nothing in Jobber
		// will ever own it, so acknowledging is the only way to stop FastSpring
		// redelivering it for good.
		outcome.Err = err
		result.skip(outcome)
	default:
		// Nothing was claimed and nothing was written, so the retry gets a clean
		// run at the event.
		outcome.Err = err
		result.Failed = append(result.Failed, outcome)
	}
}

type eventHandler func(ctx context.Context, event fastspring.Event) error

// handlerFor returns the handler for an event type, or nil when Jobber does not
// act on it.
//
// order.completed is deliberately not actionable: entitlement is driven by the
// subscription lifecycle events, which carry the account ID used to resolve the
// buyer. Acting on the order too would duplicate the grant and race with
// subscription.activated.
func (s *SubscriptionService) handlerFor(eventType string) eventHandler {
	switch eventType {
	case fastspring.EventSubscriptionActivated,
		fastspring.EventSubscriptionUpdated,
		fastspring.EventSubscriptionUncanceled,
		fastspring.EventSubscriptionCanceled,
		fastspring.EventSubscriptionDeactivated,
		fastspring.EventSubscriptionPaused,
		fastspring.EventSubscriptionResumed,
		fastspring.EventSubscriptionChargeCompleted,
		fastspring.EventSubscriptionChargeFailed,
		fastspring.EventSubscriptionPaymentOverdue:
		return s.applySubscriptionEvent
	default:
		return nil
	}
}

// applySubscriptionEvent maps a subscription lifecycle event onto the local row.
func (s *SubscriptionService) applySubscriptionEvent(ctx context.Context, event fastspring.Event) error {
	incoming, err := fastspring.ParseSubscription(event.Data)
	if err != nil {
		return err
	}

	existing, err := s.resolveOwner(ctx, incoming)
	if err != nil {
		return err
	}

	status, err := statusForEvent(event.Type, incoming)
	if err != nil {
		return err
	}

	plan := PlanFree
	if status != StatusCancelled {
		// An unrecognised product must never grant paid access. Failing here
		// makes FastSpring retry, so a mis-typed product path can be corrected
		// without losing the customer's purchase.
		if plan, err = s.planForProductPath(incoming.ProductPath); err != nil {
			return err
		}
	}

	// FastSpring reports `begin` (subscription start) and `next` (next charge
	// date) rather than an explicit current period; `next` is what the UI shows
	// as the renewal date. An ended subscription has no current period at all.
	periodStart, periodEnd := incoming.Begin, incoming.Next
	if status == StatusCancelled {
		periodStart, periodEnd = nil, nil
	}

	eventAt := lifecycleTime(event, incoming)
	updated := &model.Subscription{
		UserID:                 existing.UserID,
		ExternalSubscriptionID: &incoming.ID,
		ExternalAccountID:      existing.ExternalAccountID,
		Status:                 status,
		Plan:                   plan,
		CurrentPeriodStart:     periodStart,
		CurrentPeriodEnd:       periodEnd,
		CancelAt:               pendingCancelAt(status, incoming),
		LastEventAt:            &eventAt,
	}
	if incoming.AccountID != "" {
		updated.ExternalAccountID = &incoming.AccountID
	}

	// One statement claims the event and writes the state, so a failure can
	// never record the event as processed while losing what it carried.
	outcome, err := s.repo.ApplySubscriptionEvent(ctx, event.ID, event.Type, updated)
	if err != nil {
		return fmt.Errorf("failed to apply subscription event: %w", err)
	}
	switch outcome {
	case model.WebhookDuplicate:
		return errEventDuplicate
	case model.WebhookSuperseded:
		return fmt.Errorf("%w (event changed at %s)", errEventSuperseded, eventAt.Format(time.RFC3339))
	case model.WebhookLinkConflict:
		// Deliberately not the subscription the row holds: that value was read
		// before the write refused, so reporting it could name state that has
		// since moved on. What is certain is the user and the event.
		return fmt.Errorf("%w: user %q, event carries %q",
			ErrSubscriptionLinkConflict, existing.UserID, incoming.ID)
	default:
		return nil
	}
}

// lifecycleTime is the moment the provider says the subscription changed, which
// is what orders lifecycle events against each other.
//
// The envelope's `created` is only a fallback for payloads that carry no
// `changed` (the charge events). A manual resend arrives in a fresh envelope —
// new event ID, new `created` — while still describing the original change, so
// ordering on the envelope would let stale state overwrite newer state.
func lifecycleTime(event fastspring.Event, sub *fastspring.Subscription) time.Time {
	if sub.ChangedAt != nil {
		return *sub.ChangedAt
	}
	return event.CreatedAt()
}

// resolveOwner finds the local subscription row a provider event belongs to.
//
// Three of the four hops read a value Jobber itself recorded: the provider
// subscription ID and account ID on the local row (written at checkout-session
// creation, then at activation), and the custom lookup key Jobber set on the
// FastSpring account.
//
// The fourth — the order tag — is the one hop whose value arrives *inside* the
// event, and the payload alone does not say who put it there. The signature
// proves FastSpring sent the body; it does not prove that Jobber, rather than a
// visitor using the storefront's own `fastspring.builder.tag()`, wrote the tag.
// So the tag is accepted only with the MAC that Jobber mints for it, which no
// unauthenticated buyer can produce. That check, not the position of the hop, is
// what makes it safe — see provenTaggedUserID and ordertag.go.
//
// The tag is tried before the account read-back because it is the hop that
// actually links a new buyer. FastSpring creates the customer account *during*
// checkout, so a first session answers with no `customer.accountId` to record,
// and the account it then creates carries only `lookup.global` — both local IDs
// miss on the very first lifecycle event, and the read-back has nothing to say.
// Resolving off the signed payload also keeps the grant independent of a second
// API call that could be down when the purchase lands.
func (s *SubscriptionService) resolveOwner(ctx context.Context, incoming *fastspring.Subscription) (*model.Subscription, error) {
	if incoming.ID != "" {
		sub, err := s.repo.GetByExternalSubscriptionID(ctx, incoming.ID)
		if err == nil {
			return sub, nil
		}
		if !errors.Is(err, model.ErrSubscriptionNotFound) {
			return nil, err
		}
	}

	if incoming.AccountID == "" {
		return nil, fmt.Errorf("cannot resolve subscription %q: %w", incoming.ID, model.ErrSubscriptionNotFound)
	}

	sub, err := s.repo.GetByExternalAccountID(ctx, incoming.AccountID)
	if err == nil {
		return sub, nil
	}
	if !errors.Is(err, model.ErrSubscriptionNotFound) {
		return nil, err
	}

	userID, err := s.provenTaggedUserID(incoming)
	if err != nil {
		return nil, err
	}
	if userID != "" {
		return s.resolveOwnerByOrderTag(ctx, incoming, userID)
	}

	return s.resolveOwnerByLookupKey(ctx, incoming.AccountID)
}

// provenTaggedUserID reads the owning user out of the event's own order tags,
// but only where the claim is both ours to act on and provably ours to begin
// with. Two gates, and the order matters:
//
//   - the event must be for one of Jobber's own catalog products. The FluxLab
//     store sells more than Jobber, and a purchase of someone else's product must
//     never claim a Jobber user however its order is tagged. Checked first, so a
//     foreign product's tags are not read at all.
//   - the claim must carry a proof that verifies against this deployment's
//     secret for exactly the user ID it names. Anyone can write the ID —
//     `fastspring.builder.tag()` is part of the storefront's own client library —
//     but only Jobber's server can write the MAC over it.
//
// An event that makes no claim (no user tag) returns "" and falls through to the
// account lookup key, which is where every foreign order in the shared store
// ends up. A claim that fails the proof does not fall through: it returns
// ErrUnprovenOrderTag, because an event for a Jobber product naming a Jobber user
// that Jobber cannot show it wrote is an integrity anomaly, not a routine miss.
func (s *SubscriptionService) provenTaggedUserID(incoming *fastspring.Subscription) (string, error) {
	if _, err := s.planForProductPath(incoming.ProductPath); err != nil {
		return "", nil
	}
	return provenOrderTagUserID(s.cfg.WebhookSecret, incoming.Tags)
}

// resolveOwnerByOrderTag loads the row a proven order tag names, refusing to
// hand the event to a user who is already paying for a different subscription.
//
// The tag is written once per checkout and a user holding a live subscription
// cannot start another one, so a row that already has one contradicts the tag.
// Applying the event anyway would overwrite the row's single
// external_subscription_id and strand the one that user actually pays for, so
// the conflict is reported instead — acknowledged, never applied.
//
// This guard survives the proof: a buyer can replay their own proof onto a
// storefront purchase Jobber never brokered, and this is what stops that
// purchase from repointing a subscription they are already paying for.
//
// It is a diagnosis, not the invariant. The invariant itself lives in the
// atomic write, which refuses the same replacement under a row lock for every
// hop — this read cannot, because the row it checks could change before the
// write lands. What the hop adds is the name of the suspicious input: an order
// tag contradicting the row it names is an integrity signal about the tag
// channel, and it is worth telling apart from a link conflict Jobber's own
// flows produced.
//
// A tag naming a user with no row at all stays retryable, exactly like a Jobber
// lookup key whose user row is missing: that is a link to repair, not somebody
// else's customer.
func (s *SubscriptionService) resolveOwnerByOrderTag(
	ctx context.Context, incoming *fastspring.Subscription, userID string,
) (*model.Subscription, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if holdsLiveProviderSubscription(sub) {
		return nil, fmt.Errorf("%w: user %q holds %q, event carries %q",
			ErrTaggedOwnerConflict, userID, *sub.ExternalSubscriptionID, incoming.ID)
	}
	return sub, nil
}

// resolveOwnerByLookupKey is the last hop: it reads back the custom lookup key
// FastSpring stored from the session's external account ID and decodes the user
// ID from it. For this store that key is absent — the observed account carries a
// `lookup` holding only `global` — so in practice the hop resolves nothing and
// exists for the day the field is populated.
//
// It is also the last word on whether a purchase is foreign. The store is shared
// with the other products sold from the FluxLab account, so their subscription
// events arrive here as well; an event that made no order-tag claim *and* whose
// account carries no Jobber lookup key is one of theirs, and is reported
// as errForeignBillingAccount — acknowledged rather than retried. The two
// failure modes around it stay retryable on purpose: a GetAccount error is
// transient, and a Jobber key whose user row is missing is a link to repair, not
// someone else's customer.
//
// https://developer.fastspring.com/reference/retrieve-an-account
func (s *SubscriptionService) resolveOwnerByLookupKey(ctx context.Context, accountID string) (*model.Subscription, error) {
	account, err := s.billing.GetAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to look up billing account %q: %w", accountID, err)
	}

	userID, ok := userIDFromLookupKey(account.Lookup.Custom)
	if !ok {
		return nil, fmt.Errorf("billing account %q carries no Jobber lookup key: %w", accountID, errForeignBillingAccount)
	}
	return s.repo.GetByUserID(ctx, userID)
}

// statusForEvent maps a provider state onto an internal status.
//
// The rules are ordered, and the order is the contract:
//  1. `deactivated` ends access, whatever the event says;
//  2. an explicit pause/resume beats the payload's `active` flag, because
//     FastSpring reports a paused subscription as inactive and a pause is not
//     a cancellation;
//  3. only then does a generic `active: false` cancel;
//  4. dunning notices downgrade an otherwise active subscription to past due.
func statusForEvent(eventType string, sub *fastspring.Subscription) (string, error) {
	// Deactivation wins over everything: a subscription the provider reports as
	// deactivated must never be revived by a late payment notice or a replay.
	if sub.State == fastspring.StateDeactivated {
		return StatusCancelled, nil
	}

	switch eventType {
	// A pause suspends billing, so paid access stops even if the payload's state
	// has not caught up yet. It is checked before `active` because FastSpring
	// reports a paused subscription as `active: false`, and reading that as a
	// cancellation would strip the remembered plan off the row and lose the
	// customer's purchase on resume. effectivePlan drops a paused paid plan back
	// to free limits, so a pause still costs the buyer their paid quotas.
	case fastspring.EventSubscriptionPaused:
		return StatusPaused, nil
	// The mirror image: a resume restores access even if the payload's `active`
	// flag is still catching up with the un-pause.
	case fastspring.EventSubscriptionResumed:
		return StatusActive, nil
	}

	// Any other event on a subscription the provider calls inactive is an
	// ending, not a dunning notice.
	if sub.Active != nil && !*sub.Active {
		return StatusCancelled, nil
	}

	// These events report a billing problem while the provider state is still
	// active, so the state alone would hide the dunning period.
	switch eventType {
	case fastspring.EventSubscriptionPaymentOverdue, fastspring.EventSubscriptionChargeFailed:
		return StatusPastDue, nil
	}

	switch sub.State {
	case fastspring.StateActive, fastspring.StateTrial, fastspring.StateCanceled:
		// `canceled` means a cancellation is scheduled; access runs to the
		// deactivation date.
		return StatusActive, nil
	case fastspring.StateOverdue:
		return StatusPastDue, nil
	case fastspring.StatePaused:
		return StatusPaused, nil
	default:
		return "", fmt.Errorf("unrecognised subscription state %q", sub.State)
	}
}

// pendingCancelAt returns the date access ends, but only while a cancellation is
// actually scheduled. Any other state clears a previously stored date.
func pendingCancelAt(status string, sub *fastspring.Subscription) *time.Time {
	if status == StatusCancelled || sub.State != fastspring.StateCanceled {
		return nil
	}
	return sub.Deactivation
}
