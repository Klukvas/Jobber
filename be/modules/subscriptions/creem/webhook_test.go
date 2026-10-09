package creem

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "whsec_test"

func sign(body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	const body = `{"id":"evt_1","eventType":"subscription.paid"}`
	valid := sign(body, testSecret)

	tests := []struct {
		name      string
		body      string
		signature string
		secret    string
		wantErr   error
	}{
		{name: "valid", body: body, signature: valid, secret: testSecret},
		{name: "valid with surrounding whitespace", body: body, signature: " " + valid + "\n", secret: testSecret},
		{name: "upper-case hex is the same digest", body: body, signature: upper(valid), secret: testSecret},
		{name: "no secret configured", body: body, signature: valid, secret: "", wantErr: ErrSecretMissing},
		{name: "no signature", body: body, signature: "", secret: testSecret, wantErr: ErrSignatureMissing},
		{name: "blank signature", body: body, signature: "   ", secret: testSecret, wantErr: ErrSignatureMissing},
		{name: "not hex", body: body, signature: "zz" + valid[2:], secret: testSecret, wantErr: ErrSignatureInvalid},
		{name: "truncated", body: body, signature: valid[:32], secret: testSecret, wantErr: ErrSignatureInvalid},
		{name: "odd length", body: body, signature: valid[:63], secret: testSecret, wantErr: ErrSignatureInvalid},
		{name: "wrong secret", body: body, signature: sign(body, "other"), secret: testSecret, wantErr: ErrSignatureInvalid},
		{name: "body changed after signing", body: body + " ", signature: valid, secret: testSecret, wantErr: ErrSignatureInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifySignature([]byte(tt.body), tt.signature, tt.secret)

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func upper(s string) string {
	out := []byte(s)
	for i, c := range out {
		if c >= 'a' && c <= 'f' {
			out[i] = c - 32
		}
	}
	return string(out)
}

func TestParseEvent(t *testing.T) {
	t.Run("reads the envelope", func(t *testing.T) {
		event, err := ParseEvent([]byte(`{"id":"evt_1","eventType":"subscription.paid","created_at":1759665600000,"object":{"id":"sub_1"}}`))

		require.NoError(t, err)
		assert.Equal(t, "evt_1", event.ID)
		assert.Equal(t, EventSubscriptionPaid, event.Type)
		assert.True(t, event.CreatedAt().Equal(time.UnixMilli(1759665600000)))
		assert.JSONEq(t, `{"id":"sub_1"}`, string(event.Object))
	})

	t.Run("rejects a body that is not JSON", func(t *testing.T) {
		_, err := ParseEvent([]byte("nope"))

		require.Error(t, err)
	})
}

func TestParseSubscription(t *testing.T) {
	t.Run("reads an expanded object", func(t *testing.T) {
		sub, err := ParseSubscription([]byte(`{
			"id":"sub_1","mode":"prod","status":"scheduled_cancel",
			"product":{"id":"prod_1"},"customer":{"id":"cust_1","email":"a@b.c"},
			"metadata":{"jobber_user_id":"u1","count":3,"nested":{"a":1}},
			"current_period_start_date":"2026-10-01T00:00:00.000Z",
			"current_period_end_date":"2026-11-01T00:00:00Z",
			"updated_at":"2026-10-05T12:00:00.123456Z"}`))

		require.NoError(t, err)
		assert.Equal(t, "sub_1", sub.ID)
		assert.Equal(t, "prod_1", sub.ProductID)
		assert.Equal(t, "cust_1", sub.CustomerID)
		assert.Equal(t, StatusScheduledCancel, sub.Status)
		assert.True(t, sub.HasMode)
		assert.True(t, sub.IsLive)
		assert.Equal(t, map[string]string{"jobber_user_id": "u1"}, sub.Metadata,
			"a value of another type costs that entry only")
		require.NotNil(t, sub.CurrentPeriodStart)
		assert.True(t, sub.CurrentPeriodStart.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)))
		require.NotNil(t, sub.CurrentPeriodEnd)
		require.NotNil(t, sub.UpdatedAt)
		assert.Equal(t, 123456000, sub.UpdatedAt.Nanosecond())
	})

	t.Run("reads bare ID strings", func(t *testing.T) {
		sub, err := ParseSubscription([]byte(`{"id":"sub_1","product":"prod_1","customer":"cust_1"}`))

		require.NoError(t, err)
		assert.Equal(t, "prod_1", sub.ProductID)
		assert.Equal(t, "cust_1", sub.CustomerID)
	})

	t.Run("falls back to the item's product when the product is absent", func(t *testing.T) {
		sub, err := ParseSubscription([]byte(`{"id":"sub_1","items":[{"product_id":"prod_item","price_id":"p"}]}`))

		require.NoError(t, err)
		assert.Equal(t, "prod_item", sub.ProductID)
	})

	t.Run("test and sandbox objects are not live, a missing mode is unknown", func(t *testing.T) {
		for mode, wantLive := range map[string]bool{"prod": true, "test": false, "sandbox": false} {
			sub, err := ParseSubscription([]byte(`{"id":"sub_1","mode":"` + mode + `"}`))
			require.NoError(t, err)
			assert.True(t, sub.HasMode)
			assert.Equal(t, wantLive, sub.IsLive, mode)
		}

		sub, err := ParseSubscription([]byte(`{"id":"sub_1"}`))
		require.NoError(t, err)
		assert.False(t, sub.HasMode)
	})

	t.Run("timestamps in epoch milliseconds are understood", func(t *testing.T) {
		sub, err := ParseSubscription([]byte(`{"id":"sub_1","updated_at":1759665600000}`))

		require.NoError(t, err)
		require.NotNil(t, sub.UpdatedAt)
		assert.True(t, sub.UpdatedAt.Equal(time.UnixMilli(1759665600000)))
	})

	t.Run("an unusable timestamp is absent instead of failing the event", func(t *testing.T) {
		for _, value := range []string{`null`, `""`, `"yesterday"`, `0`, `-5`, `true`} {
			sub, err := ParseSubscription([]byte(`{"id":"sub_1","updated_at":` + value + `}`))

			require.NoError(t, err, value)
			assert.Nil(t, sub.UpdatedAt, value)
		}
	})

	t.Run("an object with no ID is rejected", func(t *testing.T) {
		_, err := ParseSubscription([]byte(`{"status":"active"}`))

		require.Error(t, err)
	})

	t.Run("a payload that is not an object is rejected", func(t *testing.T) {
		_, err := ParseSubscription([]byte(`"sub_1"`))

		require.Error(t, err)
	})
}

func TestParseCompletedCheckout(t *testing.T) {
	checkout, err := ParseCompletedCheckout([]byte(`{
		"id":"ch_1","mode":"test","product":{"id":"prod_1"},"customer":"cust_1",
		"subscription":{"id":"sub_1"},"metadata":{"jobber_user_id":"u1"}}`))

	require.NoError(t, err)
	assert.Equal(t, "prod_1", checkout.ProductID)
	assert.Equal(t, "cust_1", checkout.CustomerID)
	assert.True(t, checkout.HasMode)
	assert.False(t, checkout.IsLive)
	assert.Equal(t, "u1", checkout.Metadata["jobber_user_id"])
}
