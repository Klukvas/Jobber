package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// flipLastRune changes a proof by exactly one character, standing in for a
// near-miss forgery rather than obvious garbage.
func flipLastRune(proof string) string {
	replacement := "A"
	if strings.HasSuffix(proof, replacement) {
		replacement = "B"
	}
	return proof[:len(proof)-1] + replacement
}

func TestOrderTagsCarryTheUserAndItsProof(t *testing.T) {
	tags, err := orderTags(testWebhookSecret, testUserID)

	require.NoError(t, err)
	require.Len(t, tags, 2, "the ID and its proof are written together or not at all")
	assert.Equal(t, testUserID, tags[userIDTagKey])
	assert.True(t, orderTagProofIsValid(testWebhookSecret, testUserID, tags[userProofTagKey]),
		"the tag the checkout writes must be the one the webhook accepts")
}

func TestOrderTagsCanonicaliseTheUserID(t *testing.T) {
	// Both sides of the MAC use the canonical spelling, so a row that hands over
	// an upper-case UUID still produces a proof the webhook can verify against
	// the canonical form it parses out of the payload.
	tags, err := orderTags(testWebhookSecret, strings.ToUpper(testUserID))

	require.NoError(t, err)
	assert.Equal(t, testUserID, tags[userIDTagKey])
	assert.True(t, orderTagProofIsValid(testWebhookSecret, testUserID, tags[userProofTagKey]))
}

func TestOrderTagsAreOmittedWhenNothingCanProveThem(t *testing.T) {
	// No secret means no proof, and a user ID nothing can verify is worse than
	// none: it looks like provenance and is not. That is deliberate, so it is not
	// an error.
	tags, err := orderTags("", testUserID)

	require.NoError(t, err)
	assert.Nil(t, tags)
}

func TestOrderTagsRefuseAUserTheGrantCouldNeverReach(t *testing.T) {
	// A user ID that is not a UUID cannot be tagged, cannot be looked up, and
	// cannot be granted anything. Better to fail the checkout than to take money
	// for a purchase nothing could ever be attributed to.
	for _, userID := range []string{"", "not-a-uuid", accountLookupKey(testUserID)} {
		tags, err := orderTags(testWebhookSecret, userID)

		require.Error(t, err)
		assert.Nil(t, tags)
	}
}

func TestOrderTagProofBindsToExactlyOneUser(t *testing.T) {
	// The whole point: a proof is not a bearer token for the tag mechanism, it is
	// a statement about one user ID. Changing the ID invalidates it.
	proof, ok := orderTagProof(testWebhookSecret, testUserID)
	require.True(t, ok)

	assert.True(t, orderTagProofIsValid(testWebhookSecret, testUserID, proof))
	assert.False(t, orderTagProofIsValid(testWebhookSecret, attackerUserID, proof),
		"a proof one user legitimately holds must not carry another user's ID")

	other, ok := orderTagProof(testWebhookSecret, attackerUserID)
	require.True(t, ok)
	assert.NotEqual(t, proof, other)
}

func TestOrderTagProofIsStableAcrossCalls(t *testing.T) {
	// A checkout minted before a redelivery must still verify hours later: the
	// proof carries no nonce and no timestamp on purpose.
	first, ok := orderTagProof(testWebhookSecret, testUserID)
	require.True(t, ok)
	second, ok := orderTagProof(testWebhookSecret, testUserID)
	require.True(t, ok)

	assert.Equal(t, first, second)
}

func TestOrderTagProofIsDomainSeparated(t *testing.T) {
	proof, ok := orderTagProof(testWebhookSecret, testUserID)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(proof, orderTagProofVersion+"."),
		"the scheme version travels in the value, so a proof is recognisable without guessing")

	t.Run("it is not a bare MAC of the user ID", func(t *testing.T) {
		bare := hmac.New(sha256.New, []byte(testWebhookSecret))
		bare.Write([]byte(testUserID))

		assert.NotEqual(t, base64.RawURLEncoding.EncodeToString(bare.Sum(nil)),
			strings.TrimPrefix(proof, orderTagProofVersion+"."))
	})

	t.Run("a webhook body signature is not a proof", func(t *testing.T) {
		// The same key signs webhook bodies. The domain prefix is what keeps the
		// two uses from ever producing an interchangeable value.
		body := loadFixture(t, taggedFixture)

		assert.False(t, orderTagProofIsValid(testWebhookSecret, testUserID, sign(t, body, testWebhookSecret)))
	})
}

func TestOrderTagProofIsValidRefusesEverythingButTheRealThing(t *testing.T) {
	proof, ok := orderTagProof(testWebhookSecret, testUserID)
	require.True(t, ok)

	cases := []struct {
		name   string
		secret string
		proof  string
	}{
		{name: "missing", secret: testWebhookSecret},
		{name: "blank", secret: testWebhookSecret, proof: "   "},
		{name: "one character changed", secret: testWebhookSecret, proof: flipLastRune(proof)},
		{name: "truncated", secret: testWebhookSecret, proof: proof[:len(proof)-1]},
		{name: "padded", secret: testWebhookSecret, proof: proof + "="},
		{name: "surrounding whitespace", secret: testWebhookSecret, proof: " " + proof + " "},
		{name: "version stripped", secret: testWebhookSecret, proof: strings.TrimPrefix(proof, "v1.")},
		{name: "unknown version", secret: testWebhookSecret, proof: "v2." + strings.TrimPrefix(proof, "v1.")},
		{name: "standard base64 rather than url-safe", secret: testWebhookSecret,
			proof: "v1." + strings.NewReplacer("-", "+", "_", "/").Replace(strings.TrimPrefix(proof, "v1."))},
		// Rotation: a proof minted under the previous secret no longer verifies.
		{name: "another secret", secret: "the-previous-webhook-secret", proof: proof},
		// Fail closed: with no secret nothing verifies, including a real proof.
		{name: "no secret at all", proof: proof},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, orderTagProofIsValid(tc.secret, testUserID, tc.proof))
		})
	}

	t.Run("the genuine proof still passes", func(t *testing.T) {
		assert.True(t, orderTagProofIsValid(testWebhookSecret, testUserID, proof))
	})
}

func TestProvenOrderTagUserID(t *testing.T) {
	proof, ok := orderTagProof(testWebhookSecret, testUserID)
	require.True(t, ok)

	t.Run("a proven claim returns the user", func(t *testing.T) {
		userID, err := provenOrderTagUserID(testWebhookSecret,
			map[string]string{userIDTagKey: testUserID, userProofTagKey: proof})

		require.NoError(t, err)
		assert.Equal(t, testUserID, userID)
	})

	t.Run("a re-cased claim still verifies", func(t *testing.T) {
		userID, err := provenOrderTagUserID(testWebhookSecret,
			map[string]string{userIDTagKey: " " + strings.ToUpper(testUserID) + " ", userProofTagKey: proof})

		require.NoError(t, err)
		assert.Equal(t, testUserID, userID)
	})

	t.Run("no claim is not an error", func(t *testing.T) {
		noClaim := []map[string]string{
			nil,
			{},
			{"campaign": "spring"},
			{userIDTagKey: ""},
			{userIDTagKey: "  "},
			// A proof with nothing to prove claims nothing either.
			{userProofTagKey: proof},
		}
		for _, tags := range noClaim {
			userID, err := provenOrderTagUserID(testWebhookSecret, tags)

			require.NoError(t, err)
			assert.Empty(t, userID)
		}
	})

	t.Run("an unprovable claim is an error, never a fallthrough", func(t *testing.T) {
		unprovable := []map[string]string{
			{userIDTagKey: testUserID},
			{userIDTagKey: testUserID, userProofTagKey: ""},
			{userIDTagKey: testUserID, userProofTagKey: flipLastRune(proof)},
			{userIDTagKey: attackerUserID, userProofTagKey: proof},
			{userIDTagKey: "not-a-uuid", userProofTagKey: proof},
		}
		for _, tags := range unprovable {
			userID, err := provenOrderTagUserID(testWebhookSecret, tags)

			assert.ErrorIs(t, err, ErrUnprovenOrderTag)
			assert.Empty(t, userID)
		}
	})

	t.Run("without a secret even a real proof is refused", func(t *testing.T) {
		_, err := provenOrderTagUserID("",
			map[string]string{userIDTagKey: testUserID, userProofTagKey: proof})

		assert.ErrorIs(t, err, ErrUnprovenOrderTag)
	})
}

func TestCanonicalUserID(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "canonical UUID", value: testUserID, want: testUserID},
		{name: "surrounding whitespace", value: "  " + testUserID + "\n", want: testUserID},
		{name: "uppercase is normalised", value: "550E8400-E29B-41D4-A716-446655440000", want: testUserID},
		{name: "empty"},
		{name: "not a UUID", value: "not-a-uuid"},
		{name: "a lookup key rather than a UUID", value: accountLookupKey(testUserID)},
		{name: "SQL smuggled after a valid UUID", value: testUserID + "' OR '1'='1"},
		{name: "one hex digit short", value: "550e8400-e29b-41d4-a716-44665544000"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := canonicalUserID(tc.value)

			assert.Equal(t, tc.want != "", ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestProofReplayedAsABodySignatureIsRejectedByParsing pins the one crossover
// the domain separation does *not* close by construction, and the reason it is
// harmless.
//
// A proof is a MAC under the webhook secret, so its owner can re-encode it as an
// X-FS-Signature and POST the single body it matches: the raw domain message. The
// signature verifies — that half is real — and the request then dies in parsing,
// because that body is not a JSON event batch. ADR-0002 says exactly this; if it
// ever stops being true, the wording there is wrong.
func TestProofReplayedAsABodySignatureIsRejectedByParsing(t *testing.T) {
	// The message the proof is a MAC over, rebuilt byte for byte.
	body := []byte(orderTagProofDomain + "\x00" + orderTagProofVersion + "\x00" + testUserID)

	proof, ok := orderTagProof(testWebhookSecret, testUserID)
	require.True(t, ok)
	rawMAC, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(proof, orderTagProofVersion+"."))
	require.NoError(t, err)
	signature := base64.StdEncoding.EncodeToString(rawMAC)

	require.NoError(t, fastspring.VerifySignature(body, signature, testWebhookSecret),
		"the crossover is real at the MAC layer — the ADR must not claim otherwise")

	repo := newRecordingRepo(linkedFreeSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	_, err = svc.HandleWebhook(context.Background(), body, signature)

	require.Error(t, err, "a signed non-batch body must not be processed")
	assert.Empty(t, repo.upserts, "nothing may be written by a body that is not an event batch")
	assert.Empty(t, repo.claims)
}
