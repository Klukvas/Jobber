package service

import (
	"encoding/json"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/stretchr/testify/require"
)

// Aliases keep the table-driven tests readable without importing the provider
// package into every assertion.
const (
	eventUpdated      = fastspring.EventSubscriptionUpdated
	eventCanceled     = fastspring.EventSubscriptionCanceled
	eventDeactivated  = fastspring.EventSubscriptionDeactivated
	eventPaused       = fastspring.EventSubscriptionPaused
	eventResumed      = fastspring.EventSubscriptionResumed
	eventOverdue      = fastspring.EventSubscriptionPaymentOverdue
	eventChargeFailed = fastspring.EventSubscriptionChargeFailed
)

var (
	fastspringSignatureInvalid = fastspring.ErrSignatureInvalid
	fastspringSignatureMissing = fastspring.ErrSignatureMissing
	fastspringSecretMissing    = fastspring.ErrSecretMissing
)

func newParsedSubscription(state string, active *bool) *fastspring.Subscription {
	return &fastspring.Subscription{
		ID:          fixtureSubscriptionID,
		AccountID:   fixtureAccountID,
		ProductPath: testProProductPath,
		State:       state,
		Active:      active,
	}
}

// unconfiguredClient has no credentials, so every API call fails without
// leaking anything or reaching the network.
func unconfiguredClient() *fastspring.Client {
	return fastspring.NewClient(fastspring.Config{})
}

// replaceEventID rewrites one event ID in a fixture body, standing in for a
// manual resend (which FastSpring delivers under a fresh ID).
func replaceEventID(t *testing.T, body []byte, oldID, newID string) []byte {
	t.Helper()
	var batch struct {
		Events []map[string]any `json:"events"`
	}
	require.NoError(t, json.Unmarshal(body, &batch))
	for _, event := range batch.Events {
		if event["id"] == oldID {
			event["id"] = newID
		}
	}
	out, err := json.Marshal(batch)
	require.NoError(t, err)
	return out
}

// mergeFixtures concatenates the events of several fixture bodies into one
// batch, standing in for a single POST carrying multiple events.
func mergeFixtures(t *testing.T, bodies ...[]byte) []byte {
	t.Helper()
	merged := struct {
		Events []json.RawMessage `json:"events"`
	}{}
	for _, body := range bodies {
		var batch struct {
			Events []json.RawMessage `json:"events"`
		}
		require.NoError(t, json.Unmarshal(body, &batch))
		merged.Events = append(merged.Events, batch.Events...)
	}
	out, err := json.Marshal(merged)
	require.NoError(t, err)
	return out
}
