package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/creem"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/google/uuid"
)

// WebhookResult describes what happened to one delivery, so the HTTP layer can
// log it without the service needing a logger.
type WebhookResult struct {
	EventID   string
	EventType string
	// Skipped is the reason the event was acknowledged without changing anything,
	// or nil when it was applied.
	Skipped error
}

// ErrEventFailed marks a delivery that must be retried: nothing was claimed and
// nothing was written, so the redelivery gets a clean run. The cause is wrapped.
var ErrEventFailed = errors.New("billing event failed and must be retried")

// ErrEnvironmentMismatch marks a test event received by a live deployment (or
// vice versa). It is acknowledged, never applied: retrying could not change the
// outcome, and applying it would cross the live/test boundary.
//
// It is exported because acknowledging it *loses* the event permanently — the
// only cause is a deployment pointed at the wrong billing environment — so the
// HTTP layer reports it louder than an ordinary skip.
var ErrEnvironmentMismatch = errors.New("event environment does not match the configured billing environment")

// ErrSubscriptionLinkConflict marks an event that would repoint a user's row at
// a provider subscription other than the still-billing one it already names.
//
// The row carries a single external_subscription_id. Replacing it while the old
// subscription is alive would leave that subscription billing at Creem with
// nothing in Jobber able to cancel it, so the write refuses — atomically, under
// the row lock, whichever hop resolved the owner.
//
// It is retried, not acknowledged, but only for an event that could still land:
// a non-cancellation that is newer than the row. The usual cause is a delayed
// cancellation of the old subscription, and a redelivery after that lands applies
// the new one by itself. Anything else is acknowledged (see errNothingToEnd and
// errSecondSubscription), because retrying cannot change its outcome. When the old
// subscription really is alive the retries run out after 24 hours and each one is
// logged at error, which is the signal that somebody may be paying twice.
var ErrSubscriptionLinkConflict = errors.New("event describes a provider subscription other than the live one linked to this user")

// ErrRefundNeedsReview marks a refund or dispute on this account.
//
// Nothing is applied, and that is a decision rather than an omission: a refund is
// not a cancellation — a partial refund leaves the subscription billing normally,
// and revoking access on one would take a paid plan away from someone who still
// has it. Ending a subscription because of a refund is a merchant decision, made
// in the dashboard, which then arrives here as the cancellation it really is.
//
// What it must not be is invisible. Money left the account, so the event is
// surfaced instead of being filed with the routine ones nobody reads. A downgrade
// refunds the unused time of the old plan, so expect one of these per downgrade.
var ErrRefundNeedsReview = errors.New("a refund or dispute was raised on this account and may need a subscription ended by hand")

var (
	// errEventNotActionable marks an event type Jobber does not act on.
	errEventNotActionable = errors.New("event type is not actionable")
	// errForeignBillingEvent marks an event for a product that is not Jobber's.
	// The Creem account may sell other products too, and their events reach this
	// endpoint as well. No retry could ever make one resolvable, so it is
	// acknowledged and dropped rather than redelivered until Creem gives up.
	errForeignBillingEvent = errors.New("event belongs to another product on this Creem account")
	// errEventSuperseded marks a replayed or out-of-order event that does not
	// describe a state newer than the one already applied.
	errEventSuperseded = errors.New("event is not newer than the applied subscription state")
	// errEventDuplicate marks a redelivery of an event that was already applied.
	errEventDuplicate = errors.New("event was already processed")
	// errNothingToEnd marks the cancellation of a subscription the row does not
	// track while it holds a different live one: a replay of an old
	// subscription's cancel, or the cancel of a second subscription that was
	// never linked. There is nothing in Jobber to end and retrying cannot change
	// that.
	errNothingToEnd = errors.New("cancellation is for a subscription this user's row does not track")
	// errSecondSubscription marks a paid event for a subscription other than the
	// live one on the row, that is not newer than the row. A redelivery cannot
	// help — the older state will never beat the newer one — so it is
	// acknowledged, but it is also the signature of a user paying for two
	// subscriptions, which a person should look at.
	errSecondSubscription = errors.New("event is for a second subscription that cannot replace the linked live one")
	// errEventMissingID marks an actionable event with no ID. It cannot be
	// de-duplicated, so it is treated as a malformed delivery and retried.
	errEventMissingID = errors.New("actionable event carried no event ID")
)

// HandleWebhook verifies and processes one Creem webhook delivery.
//
// The signature is checked against the raw body before anything is parsed or
// written, so an unsigned or forged request can never reach the database. A
// rejected signature or an unparseable body returns that error as is; a delivery
// that has to be retried returns ErrEventFailed.
//
// https://docs.creem.io/code/webhooks
func (s *SubscriptionService) HandleWebhook(ctx context.Context, body []byte, signature string) (WebhookResult, error) {
	if err := creem.VerifySignature(body, signature, s.cfg.WebhookSecret); err != nil {
		return WebhookResult{}, err
	}

	event, err := creem.ParseEvent(body)
	if err != nil {
		return WebhookResult{}, err
	}

	result := WebhookResult{EventID: event.ID, EventType: event.Type}
	skipped, err := s.processEvent(ctx, event)
	if err != nil {
		return result, fmt.Errorf("%w: %w", ErrEventFailed, err)
	}
	result.Skipped = skipped
	return result, nil
}

// processEvent routes one event to its handler. It returns the reason the event
// was acknowledged without changes (nil when it was applied), or an error when it
// has to be retried. De-duplication is not a step here: it happens inside the
// handler's single atomic write, together with the state change.
func (s *SubscriptionService) processEvent(ctx context.Context, event creem.Event) (skipped, failure error) {
	apply := s.handlerFor(event.Type)
	if apply == nil {
		return notActionableReason(event.Type), nil
	}
	// An actionable event without an ID cannot be claimed, so applying it would
	// give up idempotency entirely. Fail it instead: nothing is written and Creem
	// redelivers.
	if event.ID == "" {
		return nil, errEventMissingID
	}

	err := apply(ctx, event)
	switch {
	case err == nil:
		return nil, nil
	case isAcknowledgedSkip(err):
		return err, nil
	default:
		return nil, err
	}
}

// isAcknowledgedSkip reports whether an outcome means "done, stop redelivering":
// routine duplicates and replays, and the collisions no retry can resolve.
func isAcknowledgedSkip(err error) bool {
	return errors.Is(err, errEventDuplicate) ||
		errors.Is(err, errEventSuperseded) ||
		errors.Is(err, errForeignBillingEvent) ||
		errors.Is(err, errEventNotActionable) ||
		errors.Is(err, errNothingToEnd) ||
		errors.Is(err, errSecondSubscription) ||
		errors.Is(err, ErrEnvironmentMismatch) ||
		errors.Is(err, model.ErrBillingAccountTaken) ||
		errors.Is(err, model.ErrUserNotFound)
}

// notActionableReason says why an event is not acted on, so the one that is worth
// a person's attention is not filed with the routine ones.
func notActionableReason(eventType string) error {
	switch eventType {
	case creem.EventRefundCreated, creem.EventDisputeCreated:
		return ErrRefundNeedsReview
	default:
		return errEventNotActionable
	}
}

// SkipSeverity says how loudly an acknowledged-but-unapplied event is reported.
type SkipSeverity int

const (
	// SkipRoutine is noise by design: a duplicate, a replay, an event type Jobber
	// does not act on, another product's event.
	SkipRoutine SkipSeverity = iota
	// SkipNeedsReview did nothing wrong but a person should look: money moved and
	// no subscription did.
	SkipNeedsReview
	// SkipLostEvent was dropped for good and will not be redelivered, so nothing
	// but this report will ever say that a paying user may be on the wrong plan.
	SkipLostEvent
)

// SkipReport turns the reason an event was acknowledged without changes into the
// line to log for it and how loudly to log it.
func SkipReport(err error) (message string, severity SkipSeverity) {
	switch {
	case errors.Is(err, ErrEnvironmentMismatch):
		// This deployment is pointed at the wrong billing environment, and
		// acknowledging loses the event for good.
		return "Billing event dropped: environment mismatch", SkipLostEvent
	case errors.Is(err, model.ErrBillingAccountTaken):
		// Two local users behind one provider customer; only a person can decide
		// which one the purchase belongs to, and until then a paid purchase grants
		// nothing.
		return "Billing event dropped: provider customer is already linked to another user", SkipLostEvent
	case errors.Is(err, model.ErrUserNotFound):
		// A purchase for an account that no longer exists: money may have moved
		// and nothing can be granted.
		return "Billing event dropped: the Jobber user it names no longer exists", SkipLostEvent
	case errors.Is(err, errSecondSubscription):
		return "Billing event for a second subscription that cannot replace the linked live one", SkipNeedsReview
	case errors.Is(err, ErrRefundNeedsReview):
		return "Billing refund or dispute observed, no subscription changed by it", SkipNeedsReview
	default:
		return "Billing event acknowledged without changes", SkipRoutine
	}
}

type eventHandler func(ctx context.Context, event creem.Event) error

// handlerFor returns the handler for an event type, or nil when Jobber does not
// act on it.
//
// subscription.expired is deliberately absent: Creem's webhook reference keeps
// the status `active` while it retries the charge, and says the subscription is
// only over once a subscription.canceled follows. Its other pages describe expiry
// as revoking access, so this is a go-live checklist item (ADR-0003).
func (s *SubscriptionService) handlerFor(eventType string) eventHandler {
	switch eventType {
	case creem.EventSubscriptionActive,
		creem.EventSubscriptionPaid,
		creem.EventSubscriptionUpdate,
		creem.EventSubscriptionTrialing,
		creem.EventSubscriptionPaused,
		creem.EventSubscriptionScheduledCancel,
		creem.EventSubscriptionCanceled,
		creem.EventSubscriptionPastDue,
		creem.EventSubscriptionUnpaid:
		return s.applySubscriptionEvent
	case creem.EventCheckoutCompleted:
		return s.linkCompletedCheckout
	default:
		return nil
	}
}

// applySubscriptionEvent maps a subscription lifecycle event onto the local row.
func (s *SubscriptionService) applySubscriptionEvent(ctx context.Context, event creem.Event) error {
	incoming, err := creem.ParseSubscription(event.Object)
	if err != nil {
		return err
	}
	if err := s.checkEnvironment(incoming.HasMode, incoming.IsLive); err != nil {
		return err
	}

	existing, err := s.resolveOwner(ctx, incoming)
	if err != nil {
		return err
	}

	status, plan, err := s.entitlementFor(incoming)
	if err != nil {
		return err
	}

	eventAt := lifecycleTime(event, incoming)
	updated := buildSubscriptionUpdate(existing, incoming, status, plan, eventAt)

	// One statement claims the event and writes the state, so a failure can never
	// record the event as processed while losing what it carried.
	outcome, err := s.repo.ApplySubscriptionEvent(ctx, event.ID, event.Type, updated)
	if err != nil {
		return fmt.Errorf("failed to apply subscription event: %w", err)
	}
	return outcomeError(outcome, existing, incoming.ID, status, eventAt)
}

// entitlementFor decides the status and plan an event grants.
func (s *SubscriptionService) entitlementFor(incoming *creem.Subscription) (status, plan string, err error) {
	status, err = statusForSubscription(incoming)
	if err != nil {
		return "", "", err
	}
	if status == StatusCancelled {
		// Ending access consults no product, so a catalog mistake can never keep
		// someone subscribed.
		return status, PlanFree, nil
	}

	// An unrecognised product must never grant paid access. Failing here makes
	// Creem retry, so a mis-typed product ID can be corrected without losing the
	// customer's purchase.
	plan, err = s.planForProductID(incoming.ProductID)
	if err != nil {
		return "", "", err
	}
	return status, plan, nil
}

// buildSubscriptionUpdate assembles the row the atomic write applies.
//
// A payload that omits its period keeps the stored one, because some Creem
// samples carry no dates and writing them as nil would erase the renewal date
// the UI shows — but only while the stored end is still ahead of the event. A
// stored end that already passed is stale (renewals that carried no dates never
// advanced it), and carrying it forward would turn a later scheduled cancellation
// into a cancel date in the past. An ended subscription has no current period.
func buildSubscriptionUpdate(
	existing *model.Subscription, incoming *creem.Subscription, status, plan string, eventAt time.Time,
) *model.Subscription {
	periodStart, periodEnd := incoming.CurrentPeriodStart, incoming.CurrentPeriodEnd
	storedPeriodIsCurrent := existing.CurrentPeriodEnd != nil && existing.CurrentPeriodEnd.After(eventAt)
	if periodStart == nil && periodEnd == nil && storedPeriodIsCurrent {
		periodStart, periodEnd = existing.CurrentPeriodStart, existing.CurrentPeriodEnd
	}
	if status == StatusCancelled {
		periodStart, periodEnd = nil, nil
	}

	updated := &model.Subscription{
		UserID:                 existing.UserID,
		ExternalSubscriptionID: &incoming.ID,
		Status:                 status,
		Plan:                   plan,
		CurrentPeriodStart:     periodStart,
		CurrentPeriodEnd:       periodEnd,
		CancelAt:               pendingCancelAt(status, incoming, periodEnd),
		LastEventAt:            &eventAt,
	}
	// A customer is written only when the event carries one. Leaving it nil lets
	// the write keep whatever is stored, which — unlike copying the value read a
	// moment ago — cannot undo a customer a concurrent event just set.
	if incoming.CustomerID != "" {
		updated.ExternalAccountID = &incoming.CustomerID
	}
	return updated
}

// outcomeError translates what the atomic write did into the error (or nil) the
// pipeline classifies.
func outcomeError(
	outcome model.WebhookApplyOutcome, existing *model.Subscription, subscriptionID, status string, eventAt time.Time,
) error {
	switch outcome {
	case model.WebhookDuplicate:
		return errEventDuplicate
	case model.WebhookSuperseded:
		superseded := fmt.Errorf("%w (event changed at %s)", errEventSuperseded, eventAt.Format(time.RFC3339))
		if holdsOtherSubscription(existing, subscriptionID) && status != StatusCancelled {
			// Ordering across two subscriptions is meaningless, but this is the
			// shape of a purchase that lost to the other subscription's later
			// event. Say so instead of filing it with the routine replays.
			return fmt.Errorf("%w: %w", errSecondSubscription, superseded)
		}
		return superseded
	case model.WebhookAccountConflict:
		// Two local users behind one provider customer. Nothing was written; a
		// person has to decide which one the buyer meant.
		return fmt.Errorf("%w: user %q, event carries subscription %q",
			model.ErrBillingAccountTaken, existing.UserID, subscriptionID)
	case model.WebhookLinkConflict:
		return linkConflictError(existing, subscriptionID, status, eventAt)
	default:
		return nil
	}
}

// linkConflictError decides what a refused replacement of the live subscription
// means. Nothing was written and nothing was claimed, so the question is only
// whether a redelivery can ever do better.
//
// It can for a non-cancellation that is newer than the row: the usual cause is the
// old subscription's cancellation arriving late, and once it lands the same event
// applies. It cannot for a cancellation (there is nothing here to end — it is the
// replay of an old subscription's cancel, or the cancel of a second one) or for an
// event that is not newer than the row (the older state can never win).
//
// The reported subscription is deliberately not the one the row holds: that value
// was read before the write refused, so it could name state that has since moved
// on. What is certain is the user and the event.
func linkConflictError(existing *model.Subscription, subscriptionID, status string, eventAt time.Time) error {
	if status == StatusCancelled {
		return fmt.Errorf("%w: user %q, event carries %q", errNothingToEnd, existing.UserID, subscriptionID)
	}
	if existing.LastEventAt != nil && !eventAt.After(*existing.LastEventAt) {
		return fmt.Errorf("%w: user %q, event carries %q", errSecondSubscription, existing.UserID, subscriptionID)
	}
	return fmt.Errorf("%w: user %q, event carries %q", ErrSubscriptionLinkConflict, existing.UserID, subscriptionID)
}

func holdsOtherSubscription(sub *model.Subscription, subscriptionID string) bool {
	return sub.ExternalSubscriptionID != nil && *sub.ExternalSubscriptionID != "" &&
		*sub.ExternalSubscriptionID != subscriptionID
}

// linkCompletedCheckout records which Creem customer a finished checkout belongs
// to, using the user ID Jobber put in the checkout's metadata.
//
// It grants nothing — entitlement comes from the subscription events alone — but
// it gives later events a second way to find the row (the customer), for the case
// where one of them arrives without the metadata. It is idempotent, so it needs
// no event claim.
func (s *SubscriptionService) linkCompletedCheckout(ctx context.Context, event creem.Event) error {
	checkout, err := creem.ParseCompletedCheckout(event.Object)
	if err != nil {
		return err
	}
	if err := s.checkEnvironment(checkout.HasMode, checkout.IsLive); err != nil {
		return err
	}

	userID, hasUser := userIDFromMetadata(checkout.Metadata)
	if !hasUser || !s.isOwnProduct(checkout.ProductID) {
		return errForeignBillingEvent
	}
	if checkout.CustomerID == "" {
		return errEventNotActionable
	}

	return s.repo.LinkExternalAccount(ctx, userID, checkout.CustomerID)
}

// checkEnvironment refuses an object from the other billing mode. A payload with
// no mode at all cannot be placed, so it is let through rather than guessed at.
func (s *SubscriptionService) checkEnvironment(hasMode, isLive bool) error {
	if hasMode && isLive != s.cfg.IsLive() {
		return ErrEnvironmentMismatch
	}
	return nil
}

// isOwnProduct reports whether a product ID is one of Jobber's configured plans.
func (s *SubscriptionService) isOwnProduct(productID string) bool {
	_, err := s.planForProductID(productID)
	return err == nil
}

// lifecycleTime is the moment the provider says the subscription last changed,
// which is what orders lifecycle events against each other.
//
// The envelope's `created_at` is only a fallback. A manual resend arrives in a
// fresh envelope — new event ID, new timestamp — while still describing the
// original state, so ordering on the envelope would let stale state overwrite
// newer state. The two clocks are not comparable, so a row that has seen both
// kinds can misorder; the go-live checklist confirms `updated_at` is always set.
func lifecycleTime(event creem.Event, sub *creem.Subscription) time.Time {
	if sub.UpdatedAt != nil {
		return *sub.UpdatedAt
	}
	return event.CreatedAt()
}

// resolveOwner finds the local subscription row a provider event belongs to.
//
// The hops, in order: the provider subscription ID already on a row (exact, and
// what every event after the first resolves on), the user ID Jobber put in the
// checkout metadata, and the provider customer ID already on a row.
//
// The product is checked before the customer hop only. The metadata names a user
// that Jobber's own server wrote at checkout creation, so it stands for a
// purchase of ours even when the product IDs were reconfigured since (that case
// then fails and is retried, rather than being dropped). The customer hop
// identifies a *person*, not a purchase: without the check, another product sold
// to a customer who is already linked here would resolve to that user and be
// applied, or retried for a day, as if it were theirs.
//
// An event that identifies nobody is retryable: a payload for our own product that
// we failed to place is our bug to fix, and checkout.completed may still be on its
// way to supply the missing link.
func (s *SubscriptionService) resolveOwner(ctx context.Context, incoming *creem.Subscription) (*model.Subscription, error) {
	sub, err := s.repo.GetByExternalSubscriptionID(ctx, incoming.ID)
	if err == nil {
		return sub, nil
	}
	if !errors.Is(err, model.ErrSubscriptionNotFound) {
		return nil, fmt.Errorf("failed to look up subscription %q: %w", incoming.ID, err)
	}

	if userID, ok := userIDFromMetadata(incoming.Metadata); ok {
		return s.loadMetadataOwner(ctx, userID, incoming.ID)
	}

	if !s.isOwnProduct(incoming.ProductID) {
		return nil, errForeignBillingEvent
	}

	if incoming.CustomerID != "" {
		sub, err := s.repo.GetByExternalAccountID(ctx, incoming.CustomerID)
		if err == nil {
			return sub, nil
		}
		if !errors.Is(err, model.ErrSubscriptionNotFound) {
			return nil, fmt.Errorf("failed to look up customer of subscription %q: %w", incoming.ID, err)
		}
	}

	return nil, fmt.Errorf("cannot resolve subscription %q: %w", incoming.ID, model.ErrSubscriptionNotFound)
}

// loadMetadataOwner loads the row of the user a checkout named.
//
// A user with no subscriptions row yet (an account that predates the row being
// created at registration) is a real customer with a real payment, so the free
// row the write will upsert into is created rather than the purchase being
// retried until Creem gives up. A user who no longer exists comes back as
// model.ErrUserNotFound, which is acknowledged: no retry brings the account back.
func (s *SubscriptionService) loadMetadataOwner(ctx context.Context, userID, subscriptionID string) (*model.Subscription, error) {
	sub, err := s.repo.GetByUserID(ctx, userID)
	if errors.Is(err, model.ErrSubscriptionNotFound) {
		if ensureErr := s.repo.EnsureFree(ctx, userID); ensureErr != nil {
			return nil, fmt.Errorf("failed to prepare the row of the user named by subscription %q: %w", subscriptionID, ensureErr)
		}
		sub, err = s.repo.GetByUserID(ctx, userID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load the user named by subscription %q: %w", subscriptionID, err)
	}
	return sub, nil
}

// userIDFromMetadata reads the Jobber user ID out of checkout metadata. Anything
// that is not a well-formed UUID is treated as absent, so a malformed value can
// never reach a query.
func userIDFromMetadata(metadata map[string]string) (string, bool) {
	raw, ok := metadata[metadataUserIDKey]
	if !ok {
		return "", false
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", false
	}
	return parsed.String(), true
}

// statusForSubscription maps a Creem subscription status onto an internal one.
//
// `scheduled_cancel` keeps access: the subscriber cancelled but has paid through
// the end of the period. `unpaid` means collection failed for good, so it is
// treated like a pause — the purchased plan stays on the row, so recovery gives
// it back, but paid quotas stop. An unrecognised status fails the event rather
// than guessing.
func statusForSubscription(sub *creem.Subscription) (string, error) {
	switch sub.Status {
	case creem.StatusActive, creem.StatusTrialing, creem.StatusScheduledCancel:
		return StatusActive, nil
	case creem.StatusPastDue:
		return StatusPastDue, nil
	case creem.StatusPaused, creem.StatusUnpaid:
		return StatusPaused, nil
	case creem.StatusCanceled:
		return StatusCancelled, nil
	default:
		return "", fmt.Errorf("unrecognised subscription status %q", sub.Status)
	}
}

// pendingCancelAt returns the date access ends, but only while a cancellation is
// actually scheduled. Any other status clears a previously stored date. The
// period end is passed in already merged with the stored one, so a payload that
// omits it still yields a date instead of a cancellation the UI cannot show.
func pendingCancelAt(status string, sub *creem.Subscription, periodEnd *time.Time) *time.Time {
	if status == StatusCancelled || sub.Status != creem.StatusScheduledCancel {
		return nil
	}
	return periodEnd
}
