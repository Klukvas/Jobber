package model

// WebhookApplyOutcome reports what a single atomic webhook apply did.
//
// The claim on the event ID and the subscription write happen in one
// transaction, so there is no window in which an event counts as processed
// while its state change was lost.
type WebhookApplyOutcome string

const (
	// WebhookApplied means the event was claimed and its state written.
	WebhookApplied WebhookApplyOutcome = "applied"
	// WebhookDuplicate means the event ID was already recorded, so this
	// delivery is a repeat and nothing was written.
	WebhookDuplicate WebhookApplyOutcome = "duplicate"
	// WebhookSuperseded means the event was claimed — it is processed and must
	// not be redelivered — but a newer provider event already stands on the row,
	// so its state was deliberately not applied.
	WebhookSuperseded WebhookApplyOutcome = "superseded"
	// WebhookLinkConflict means the event describes a *different* provider
	// subscription than the one this user's row is linked to, while that link is
	// still live. The row holds one external_subscription_id, so applying the
	// event would drop the identifier of a subscription that keeps billing —
	// leaving nothing in Jobber able to cancel it.
	//
	// Nothing was claimed and nothing was written: the whole transaction rolls
	// back, so once the stale link is genuinely ended a resent event can still
	// land. It is a separate outcome from WebhookSuperseded on purpose — a
	// superseded event is routine ordering, this one means a subscriber is at
	// risk of paying for two subscriptions.
	WebhookLinkConflict WebhookApplyOutcome = "link_conflict"
	// WebhookAccountConflict means the provider account this event carries is
	// already linked to a *different* local user, so writing it would break the
	// unique index that makes one account resolve to exactly one user.
	//
	// It is terminal rather than retryable, and that distinction is the whole
	// point of naming it. A retry cannot resolve two users behind one provider
	// customer — only a person can — and Creem would keep redelivering a failing
	// event for 24 hours, burying everything else in the log on the way.
	WebhookAccountConflict WebhookApplyOutcome = "account_conflict"
)
