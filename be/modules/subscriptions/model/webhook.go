package model

// WebhookApplyOutcome reports what a single atomic webhook apply did.
//
// The claim on the event ID and the subscription write happen in one statement,
// so there is no window in which an event counts as processed while its state
// change was lost.
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
)
