package fastspring

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "s3cr3t-hmac-key"

func signature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"events":[{"id":"abc","type":"subscription.activated"}]}`)

	tests := []struct {
		name      string
		body      []byte
		signature string
		secret    string
		wantErr   error
	}{
		{
			name:      "matching signature",
			body:      body,
			signature: signature(body, testSecret),
			secret:    testSecret,
		},
		{
			name:      "signature computed with a different secret",
			body:      body,
			signature: signature(body, "other-secret"),
			secret:    testSecret,
			wantErr:   ErrSignatureInvalid,
		},
		{
			name:      "body altered after signing",
			body:      append(body, ' '),
			signature: signature(body, testSecret),
			secret:    testSecret,
			wantErr:   ErrSignatureInvalid,
		},
		{
			name:      "empty signature",
			body:      body,
			signature: "",
			secret:    testSecret,
			wantErr:   ErrSignatureMissing,
		},
		{
			name:      "whitespace-only signature",
			body:      body,
			signature: "   ",
			secret:    testSecret,
			wantErr:   ErrSignatureMissing,
		},
		{
			name:      "signature that is not base64",
			body:      body,
			signature: "@@@not-base64@@@",
			secret:    testSecret,
			wantErr:   ErrSignatureInvalid,
		},
		{
			name:      "truncated digest of the right encoding",
			body:      body,
			signature: base64.StdEncoding.EncodeToString([]byte("short")),
			secret:    testSecret,
			wantErr:   ErrSignatureInvalid,
		},
		{
			name:      "no secret configured",
			body:      body,
			signature: signature(body, testSecret),
			secret:    "",
			wantErr:   ErrSecretMissing,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifySignature(tc.body, tc.signature, tc.secret)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestVerifySignatureIgnoresSurroundingWhitespace(t *testing.T) {
	body := []byte(`{"events":[]}`)
	require.NoError(t, VerifySignature(body, " "+signature(body, testSecret)+"\n", testSecret))
}

func TestParseEvents(t *testing.T) {
	t.Run("reads a batch of events", func(t *testing.T) {
		body := []byte(`{"events":[
			{"id":"a","live":true,"processed":false,"type":"subscription.activated","created":1426560444800,"data":{}},
			{"id":"b","live":false,"processed":false,"type":"subscription.deactivated","created":1426560444900,"data":{}}
		]}`)

		events, err := ParseEvents(body)

		require.NoError(t, err)
		require.Len(t, events, 2)
		assert.Equal(t, "a", events[0].ID)
		assert.True(t, events[0].Live)
		assert.Equal(t, time.UnixMilli(1426560444800).UTC(), events[0].CreatedAt())
		assert.False(t, events[1].Live)
	})

	t.Run("an empty batch is not an error", func(t *testing.T) {
		events, err := ParseEvents([]byte(`{"events":[]}`))
		require.NoError(t, err)
		assert.Empty(t, events)
	})

	t.Run("malformed JSON is an error", func(t *testing.T) {
		_, err := ParseEvents([]byte(`{"events":`))
		require.Error(t, err)
	})
}

func TestParseSubscription(t *testing.T) {
	t.Run("flat subscription event with object account and product", func(t *testing.T) {
		data := []byte(`{
			"id":"sub-1","subscription":"sub-1","state":"active","active":true,
			"account":{"id":"acct-1","account":"acct-1","lookup":{"global":"g"}},
			"product":{"product":"jobber-pro"},
			"begin":1751328000000,"next":1782864000000,"deactivationDate":null
		}`)

		sub, err := ParseSubscription(data)

		require.NoError(t, err)
		assert.Equal(t, "sub-1", sub.ID)
		assert.Equal(t, "acct-1", sub.AccountID)
		assert.Equal(t, "jobber-pro", sub.ProductPath)
		assert.Equal(t, StateActive, sub.State)
		require.NotNil(t, sub.Active)
		assert.True(t, *sub.Active)
		require.NotNil(t, sub.Begin)
		assert.Equal(t, time.UnixMilli(1751328000000).UTC(), *sub.Begin)
		assert.Nil(t, sub.Deactivation)
	})

	t.Run("charge event nests the subscription and uses plain ID strings", func(t *testing.T) {
		data := []byte(`{
			"reason":"Payment declined",
			"account":{"id":"acct-1"},
			"subscription":{
				"id":"sub-1","subscription":"sub-1","state":"trial","active":true,
				"account":"acct-1","product":"jobber-pro",
				"begin":1749236450805,"next":1749340800000,"deactivationDate":1749945600000
			}
		}`)

		sub, err := ParseSubscription(data)

		require.NoError(t, err)
		assert.Equal(t, "sub-1", sub.ID)
		assert.Equal(t, "acct-1", sub.AccountID, "account is a bare ID string on charge events")
		assert.Equal(t, "jobber-pro", sub.ProductPath, "product is a bare path string on charge events")
		assert.Equal(t, StateTrial, sub.State)
		require.NotNil(t, sub.Deactivation)
	})

	t.Run("absent active flag stays nil rather than false", func(t *testing.T) {
		sub, err := ParseSubscription([]byte(`{"id":"sub-1","state":"active","product":"jobber-pro"}`))

		require.NoError(t, err)
		assert.Nil(t, sub.Active, "a missing flag must not read as an inactive subscription")
	})

	t.Run("falls back to the subscription field when id is absent", func(t *testing.T) {
		sub, err := ParseSubscription([]byte(`{"subscription":"sub-9","state":"active","product":"jobber-pro"}`))

		require.NoError(t, err)
		assert.Equal(t, "sub-9", sub.ID)
	})

	t.Run("a payload with no subscription ID is an error", func(t *testing.T) {
		_, err := ParseSubscription([]byte(`{"state":"active","product":"jobber-pro"}`))
		require.Error(t, err)
	})

	t.Run("malformed payload is an error", func(t *testing.T) {
		_, err := ParseSubscription([]byte(`not json`))
		require.Error(t, err)
	})
}

func TestParseSubscriptionReadsChangedTimestamp(t *testing.T) {
	t.Run("flat payload exposes the provider change moment", func(t *testing.T) {
		sub, err := ParseSubscription([]byte(
			`{"id":"sub-1","state":"active","product":"jobber-pro","changed":1751328000000}`))

		require.NoError(t, err)
		require.NotNil(t, sub.ChangedAt)
		assert.Equal(t, time.UnixMilli(1751328000000).UTC(), *sub.ChangedAt)
	})

	t.Run("charge payload reads changed from the nested subscription", func(t *testing.T) {
		sub, err := ParseSubscription([]byte(`{
			"reason":"Payment declined",
			"account":{"id":"acct-1"},
			"subscription":{"id":"sub-1","state":"active","product":"jobber-pro","changed":1751760000000}
		}`))

		require.NoError(t, err)
		require.NotNil(t, sub.ChangedAt)
		assert.Equal(t, time.UnixMilli(1751760000000).UTC(), *sub.ChangedAt)
	})

	t.Run("absent changed stays nil so the caller can fall back", func(t *testing.T) {
		sub, err := ParseSubscription([]byte(`{"id":"sub-1","state":"active","product":"jobber-pro"}`))

		require.NoError(t, err)
		assert.Nil(t, sub.ChangedAt)
	})
}

func TestPauseAndResumeEventTypes(t *testing.T) {
	// FastSpring's webhook list includes pause and resume; both must be named
	// here or the dashboard could be subscribed to events nothing acts on.
	assert.Equal(t, "subscription.paused", EventSubscriptionPaused)
	assert.Equal(t, "subscription.resumed", EventSubscriptionResumed)
}
