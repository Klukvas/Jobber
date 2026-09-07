package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// The order tags Jobber writes on every checkout session it creates. FastSpring
// echoes them back as `data.tags` on that order's subscription events, which is
// what lets a *first* purchase name its buyer: the provider mints the customer
// account during checkout, so the session answers with no `customer.accountId`
// to record and the account it then creates carries no `lookup.custom`.
//
// The two are written together and are only ever read together. The ID says
// which user the order belongs to; the proof is what makes that claim
// believable.
const (
	userIDTagKey    = "jobber_user_id"
	userProofTagKey = "jobber_user_proof"
)

// Why the ID alone is not enough.
//
// Order tags are not a server-only channel. The Store Builder Library exposes
// `fastspring.builder.tag()`, so anyone who loads the shared FluxLab storefront
// can attach whatever tags they like to their own order, and those tags reach
// the webhook payload exactly like Jobber's do. The webhook HMAC proves that
// *FastSpring* sent the event — it says nothing about who authored a tag inside
// it. Reading `jobber_user_id` on its own would therefore let a visitor buy a
// Jobber plan while naming somebody else's user ID.
//
// So the ID travels with a MAC over itself, minted server-side at session
// creation and verified when the event comes back. The MAC key is the
// FastSpring webhook secret, deliberately:
//
//   - it is already the root of trust for this whole path. Anyone able to forge
//     a proof would need the same secret that lets them forge an entire signed
//     event, so reusing it grants an attacker nothing they did not already have.
//   - it exists in exactly the deployments where the proof matters. The secret is
//     required whenever webhook ingestion is on, and with it absent no event is
//     accepted at all (VerifySignature refuses first), so there is no window
//     where a claim is read but unprovable by configuration.
//   - it needs no new environment variable, no new secret to leak, and no
//     migration.
//
// The cost is rotation: a proof minted before the secret changes will not verify
// after it, so a checkout in flight across a rotation loses its tag hop. That
// window is one checkout-session lifetime, it is the same window in which
// in-flight deliveries already fail their signature, and it fails loudly
// (ErrUnprovenOrderTag) rather than granting the wrong user anything.
//
// orderTagProofDomain separates this MAC from the webhook body signature in the
// direction that matters: FastSpring signs only its own event batches, and a
// JSON object is never a message beginning with this domain string, so no
// captured signature is also a valid proof.
//
// The reverse is not a property of the construction. A proof is a MAC under the
// same secret, so its owner can re-encode it as an X-FS-Signature and POST the
// one body it matches — the raw domain message. That request verifies and then
// dies in ParseEvents: the body is not an event batch, so nothing is ever read
// or written. See ADR-0002.
const (
	orderTagProofDomain  = "jobber.fastspring.order-tag"
	orderTagProofVersion = "v1"
)

// orderTags builds the tags for a checkout session: the buyer's user ID and the
// proof that Jobber's own server wrote it.
//
// Both sides of the MAC use the canonical spelling of the UUID, so a difference
// in case or spacing between the row and the payload can never turn a genuine
// proof into a mismatch.
//
// The two ways this returns nothing are not the same, which is why one is an
// error and the other is not:
//
//   - no secret — no proof can be minted, so no tag is sent. A user ID on the
//     wire that nothing can verify is worse than none: it looks like provenance
//     and is not. Nothing is lost either, because a deployment with no webhook
//     secret ingests no events for a tag to resolve.
//   - a user ID that is not a UUID — that is a broken caller, and the purchase it
//     is about to pay for could never be attributed to anyone. Failing the
//     checkout beats taking money for a grant that cannot land.
func orderTags(secret, userID string) (map[string]string, error) {
	canonical, ok := canonicalUserID(userID)
	if !ok {
		return nil, fmt.Errorf("cannot tag a checkout for %q: not a user ID", userID)
	}
	proof, ok := orderTagProof(secret, canonical)
	if !ok {
		return nil, nil
	}
	return map[string]string{
		userIDTagKey:    canonical,
		userProofTagKey: proof,
	}, nil
}

// orderTagProof mints the proof accompanying a user ID, or reports false when
// this deployment holds no secret to mint one with.
//
// The value carries its scheme version twice: inside the MAC message, so a
// future scheme can never be verified against this one's key derivation, and as
// a plain prefix, so a proof is recognisable in a payload without guessing.
func orderTagProof(secret, userID string) (string, bool) {
	if secret == "" || userID == "" {
		return "", false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	// Domain, version and value are separated by a byte that cannot occur in any
	// of them, so no two different inputs can produce the same MAC message.
	mac.Write([]byte(orderTagProofDomain))
	mac.Write([]byte{0})
	mac.Write([]byte(orderTagProofVersion))
	mac.Write([]byte{0})
	mac.Write([]byte(userID))
	return orderTagProofVersion + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), true
}

// orderTagProofIsValid reports whether proof is the one this deployment would
// mint for userID. The comparison is constant-time, and a missing secret or a
// missing proof is simply not valid — never an accidental pass.
func orderTagProofIsValid(secret, userID, proof string) bool {
	expected, ok := orderTagProof(secret, userID)
	if !ok {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(proof)) == 1
}

// provenOrderTagUserID reads the user a set of order tags claims, and returns it
// only if the claim carries a proof this deployment minted.
//
// Three outcomes, and the difference between the first two is the whole point:
//
//   - no user tag at all — the order claims nothing, so resolution falls through
//     to the account lookup key. This is every foreign order in the shared store.
//   - a claim that does not verify — malformed, unproven, or proving a different
//     user — is ErrUnprovenOrderTag. Nothing resolves it, and nothing may.
//   - a verified claim returns the user ID in canonical form.
//
// The proof is deliberately static per user rather than carrying a nonce or an
// expiry. It authorises exactly one thing — "this order belongs to user X" — and
// the only party who can ever obtain a proof for X is X, through an
// authenticated session-creation call. Replaying one's own proof to attribute
// one's own purchase to oneself is what the tag is *for*; what it cannot do is
// name anybody else. An expiry would buy nothing against that and would start
// rejecting legitimate late deliveries — a webhook retried for hours, or an
// activation that lands long after checkout.
//
// A refused claim never reveals the raw tag: the value is attacker-controlled and
// goes straight into logs.
func provenOrderTagUserID(secret string, tags map[string]string) (string, error) {
	claimed := strings.TrimSpace(tags[userIDTagKey])
	if claimed == "" {
		return "", nil
	}
	userID, ok := canonicalUserID(claimed)
	if !ok {
		return "", fmt.Errorf("%w: the user tag is not a user ID", ErrUnprovenOrderTag)
	}
	if !orderTagProofIsValid(secret, userID, tags[userProofTagKey]) {
		return "", fmt.Errorf("%w: user %q", ErrUnprovenOrderTag, userID)
	}
	return userID, nil
}

// canonicalUserID validates a local user ID and returns it in canonical form.
// Only a well-formed UUID is accepted, so nothing but a value that can address
// exactly one user row ever reaches a query — and canonicalising on both sides of
// the MAC means a re-cased UUID cannot fail to verify against a proof over a
// different spelling of itself.
func canonicalUserID(value string) (string, bool) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", false
	}
	return parsed.String(), true
}
