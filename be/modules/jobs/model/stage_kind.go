package model

import "strings"

// StageKind is what a pipeline column *means*, independent of its name and its
// position. The user owns the labels and the order; analytics reads the kind.
// That separation is the point: renaming "Technical Interview" to "Onsite
// loop", or dragging it one slot to the left, must not change what a metric
// counts.
type StageKind string

const (
	// StageKindSaved is a bookmarked role the user has not applied to yet.
	StageKindSaved StageKind = "saved"
	// StageKindApplied is "application sent, nothing back yet".
	StageKindApplied StageKind = "applied"
	// StageKindResponse is any first reply from the employer (screening call,
	// recruiter email, take-home task).
	StageKindResponse StageKind = "response"
	// StageKindInterview is an interview round.
	StageKindInterview StageKind = "interview"
	// StageKindOffer is an offer.
	StageKindOffer StageKind = "offer"
	// StageKindRejected is a rejection.
	StageKindRejected StageKind = "rejected"
	// StageKindOther carries no analytics meaning. It is the default for a
	// custom column, so a column the user invented never silently inflates a
	// response rate — they can classify it in the stage UI whenever they want.
	StageKindOther StageKind = "other"
)

// AllStageKinds lists every kind in application-lifecycle order. The HTTP
// layer's `oneof` binding tag mirrors it.
var AllStageKinds = []StageKind{
	StageKindSaved,
	StageKindApplied,
	StageKindResponse,
	StageKindInterview,
	StageKindOffer,
	StageKindRejected,
	StageKindOther,
}

// ResponseStageKinds are the kinds that mean "the employer came back": a reply,
// an interview, an offer or a rejection. Reaching a column of any of these
// kinds is what analytics counts as a response.
var ResponseStageKinds = []StageKind{
	StageKindResponse,
	StageKindInterview,
	StageKindOffer,
	StageKindRejected,
}

// AppliedStageKinds are the kinds that can only be reached once the
// application has been sent: the application itself, plus everything that
// implies it already happened. Landing in a column of one of these kinds is
// what stamps a card's applied_at.
//
// Position says nothing here. A pipeline whose owner dragged "Applied" to the
// front still means "applied", and a "Wishlist" column dragged to the middle
// still means "not applied yet" — which is exactly what the old
// order-based rule got wrong in both directions.
var AppliedStageKinds = []StageKind{
	StageKindApplied,
	StageKindResponse,
	StageKindInterview,
	StageKindOffer,
	StageKindRejected,
}

// IsValid reports whether k is one of the known kinds.
func (k StageKind) IsValid() bool {
	for _, known := range AllStageKinds {
		if k == known {
			return true
		}
	}
	return false
}

// IsResponse reports whether reaching a column of this kind means the employer
// responded. A rejection is a response — it is the employer coming back, just
// with bad news — and an offer or interview implies one happened earlier.
func (k StageKind) IsResponse() bool {
	for _, kind := range ResponseStageKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// ImpliesApplied reports whether a card that reached a column of this kind has
// necessarily applied to the role. Saved means it has not; other carries no
// meaning at all, so it answers no and leaves the decision to the caller.
func (k StageKind) ImpliesApplied() bool {
	for _, kind := range AppliedStageKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// ParseStageKind normalizes client input into a StageKind. An empty value means
// "not supplied" and yields StageKindOther, the default for a custom column.
func ParseStageKind(raw string) (StageKind, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return StageKindOther, nil
	}
	kind := StageKind(trimmed)
	if !kind.IsValid() {
		return "", ErrInvalidStageKind
	}
	return kind, nil
}
