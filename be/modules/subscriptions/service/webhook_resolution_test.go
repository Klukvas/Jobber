package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newAccountAPIService wires a service to a stub FastSpring API whose
// GET /accounts/{id} is whatever the test needs, standing in for the account
// read-back used when neither local ID resolves a webhook.
func newAccountAPIService(t *testing.T, repo *recordingRepo, accounts http.HandlerFunc) *SubscriptionService {
	t.Helper()
	server := httptest.NewServer(accounts)
	t.Cleanup(server.Close)

	client := fastspring.NewClient(fastspring.Config{
		BaseURL:  server.URL,
		Username: "api-user",
		Password: "api-pass",
	})
	return NewSubscriptionService(repo, client, testBillingConfig())
}

// newLookupService is the common case: the account read-back succeeds and
// carries the given custom lookup key.
func newLookupService(t *testing.T, repo *recordingRepo, lookupCustom string) *SubscriptionService {
	t.Helper()
	return newAccountAPIService(t, repo, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/accounts/"+fixtureAccountID, r.URL.Path)
		_, _ = fmt.Fprintf(w, `{"id":%q,"account":%q,"lookup":{"global":"gL0balLookupKeyTest01","custom":%q}}`,
			fixtureAccountID, fixtureAccountID, lookupCustom)
	})
}

func TestHandleWebhookResolvesOwnerByLookupKey(t *testing.T) {
	body := loadFixture(t, "batch_activated_order_completed.json")

	t.Run("falls back to the account lookup key when no local row matches", func(t *testing.T) {
		// The checkout-session link was lost: nothing matches by subscription or
		// account ID, so the only remaining path is the custom lookup key Jobber
		// wrote onto the FastSpring account at session creation.
		repo := newRecordingRepo(nil)
		repo.GetByUserIDFunc = func(_ context.Context, userID string) (*model.Subscription, error) {
			if userID == testUserID {
				return &model.Subscription{ID: "sub-row-1", UserID: testUserID, Status: StatusFree, Plan: PlanFree}, nil
			}
			return nil, model.ErrSubscriptionNotFound
		}
		svc := newLookupService(t, repo, accountLookupKey(testUserID))

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
		written := repo.lastUpsert(t)
		assert.Equal(t, testUserID, written.UserID)
		assert.Equal(t, StatusActive, written.Status)
		assert.Equal(t, PlanPro, written.Plan)
		require.NotNil(t, written.ExternalAccountID)
		assert.Equal(t, fixtureAccountID, *written.ExternalAccountID,
			"the event's account must be recorded so the next event resolves without the API")
	})

	// The FluxLab store sells more than Jobber, so another product's subscription
	// events land on this endpoint. They must be acknowledged and dropped — they
	// can never be resolved, and retrying them forever is the only alternative.
	foreignLookupKeys := []struct {
		name   string
		lookup string
	}{
		{"another product's own namespace", "fluxradar-7f3c9a11d4e94b2f8a6c5d0e1b2a3c4d"},
		{"no lookup key at all", ""},
		{"the prefix embedded, not leading", "acme-jobber-7f3c9a11d4e94b2f8a6c5d0e1b2a3c4d"},
		{"the Jobber prefix over a key that is not a UUID", "jobber-not-a-uuid"},
	}

	for _, tc := range foreignLookupKeys {
		t.Run("a foreign account is acknowledged and skipped: "+tc.name, func(t *testing.T) {
			repo := newRecordingRepo(nil)
			svc := newLookupService(t, repo, tc.lookup)

			result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

			require.NoError(t, err)
			assert.Empty(t, repo.upserts, "a key Jobber did not create must not resolve to any user")
			assert.Empty(t, repo.claims, "nothing was applied, so nothing may be claimed")
			assert.Empty(t, result.Failed, "a foreign product's event must never be queued for retry")
			require.True(t, result.AllProcessed())

			var skippedIDs []string
			for _, skipped := range result.Skipped {
				if skipped.EventType == "subscription.activated" {
					assert.ErrorIs(t, skipped.Err, errForeignBillingAccount)
					assert.NotErrorIs(t, skipped.Err, model.ErrSubscriptionNotFound,
						"someone else's customer is not a missing Jobber subscription")
					skippedIDs = append(skippedIDs, skipped.EventID)
				}
			}
			assert.Equal(t, []string{"evt-activated-0001"}, skippedIDs)
			assert.Contains(t, result.Processed, "evt-activated-0001",
				"the event must be reported back to FastSpring so it stops being redelivered")
		})
	}

	t.Run("a failed account read-back is retried, not acknowledged", func(t *testing.T) {
		// Transient: the API was unreachable, so the owner is still unknown. A
		// retry can resolve it, and dropping it here would lose a real purchase.
		repo := newRecordingRepo(nil)
		svc := newAccountAPIService(t, repo, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.Empty(t, repo.upserts)
		assert.Empty(t, repo.claims)
		require.Len(t, result.Failed, 1)
		assert.Equal(t, "evt-activated-0001", result.Failed[0].EventID)
		assert.NotErrorIs(t, result.Failed[0].Err, errForeignBillingAccount,
			"an unreachable API says nothing about who owns the account")
	})

	t.Run("a Jobber key with no user row is retried, not acknowledged", func(t *testing.T) {
		// The key is syntactically ours, so this is a broken link on our side —
		// a row still catching up, or one that needs repairing — not another
		// product's customer. Retrying is what gives it a chance to land.
		repo := newRecordingRepo(nil)
		repo.GetByUserIDFunc = func(context.Context, string) (*model.Subscription, error) {
			return nil, model.ErrSubscriptionNotFound
		}
		svc := newLookupService(t, repo, accountLookupKey(testUserID))

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.Empty(t, repo.upserts)
		assert.Empty(t, repo.claims)
		require.Len(t, result.Failed, 1)
		assert.ErrorIs(t, result.Failed[0].Err, model.ErrSubscriptionNotFound)
		assert.NotErrorIs(t, result.Failed[0].Err, errForeignBillingAccount)
	})
}

func TestHandleWebhookDatabaseOutageFailsEventForRetry(t *testing.T) {
	// A transient DB failure while resolving the owner must mark the event
	// failed (so FastSpring retries it), never acknowledge it away.
	body := loadFixture(t, "batch_activated_order_completed.json")
	repo := newRecordingRepo(linkedFreeSubscription())
	repo.GetByExternalSubscriptionIDFunc = func(context.Context, string) (*model.Subscription, error) {
		return nil, fmt.Errorf("connection refused")
	}
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, repo.upserts)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, "evt-activated-0001", result.Failed[0].EventID)
	assert.Empty(t, repo.claims,
		"a resolution failure must leave no claim, so the retry can be processed")
}

func TestHandleWebhookDeactivationIgnoresUnknownProduct(t *testing.T) {
	// Revocation must never consult the catalog (ADR-0002): a mis-typed product
	// path may block a grant, but it must not be able to keep access alive.
	body := bytes.ReplaceAll(loadFixture(t, "subscription_deactivated.json"),
		[]byte("jobber-pro"), []byte("product-missing-from-catalog"))
	repo := newRecordingRepo(activeProSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
	written := repo.lastUpsert(t)
	assert.Equal(t, StatusCancelled, written.Status)
	assert.Equal(t, PlanFree, written.Plan)
}

// transformFirstEvent rewrites the first event of a fixture body, standing in
// for payload variants FastSpring sends but the recorded fixtures do not cover.
func transformFirstEvent(t *testing.T, body []byte, mutate func(event map[string]any)) []byte {
	t.Helper()
	var batch struct {
		Events []map[string]any `json:"events"`
	}
	require.NoError(t, json.Unmarshal(body, &batch))
	require.NotEmpty(t, batch.Events)
	mutate(batch.Events[0])
	out, err := json.Marshal(batch)
	require.NoError(t, err)
	return out
}

func TestHandleWebhookUncanceledClearsScheduledCancellation(t *testing.T) {
	// The buyer cancels, then changes their mind before the deactivation date:
	// subscription.uncanceled arrives with the state back to active and must
	// clear the stored end-of-access date.
	body := transformFirstEvent(t, loadFixture(t, "subscription_canceled.json"), func(event map[string]any) {
		event["id"] = "evt-uncanceled-0001"
		event["type"] = "subscription.uncanceled"
		event["created"] = float64(1751501000000)
		data, ok := event["data"].(map[string]any)
		require.True(t, ok)
		data["state"] = "active"
		delete(data, "deactivationDate")
	})

	existing := activeProSubscription()
	cancelAt := time.Date(2027, 7, 3, 0, 0, 0, 0, time.UTC)
	existing.CancelAt = &cancelAt
	repo := newRecordingRepo(existing)
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
	written := repo.lastUpsert(t)
	assert.Equal(t, StatusActive, written.Status)
	assert.Equal(t, PlanPro, written.Plan)
	assert.Nil(t, written.CancelAt, "an uncancel must clear the scheduled end of access")
}

func TestHandleWebhookOutOfOrderBatchKeepsNewestState(t *testing.T) {
	// One batch delivers a deactivation first and a stale activation replay
	// (fresh event ID, older creation time) second. The stale event must be
	// acknowledged without reviving the subscription.
	body := mergeFixtures(t,
		loadFixture(t, "subscription_deactivated.json"),
		replaceEventID(t, loadFixture(t, "batch_activated_order_completed.json"),
			"evt-activated-0001", "evt-activated-out-of-order"),
	)

	// The repo applies each write to the row the lookup closures read from, so
	// the stale activation is judged against the deactivation that just landed.
	repo := newRecordingRepo(activeProSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.True(t, result.AllProcessed(), "a superseded replay is acknowledged, not retried: %v", result.Failed)
	assert.Equal(t, 3, result.Total(), "deactivation + stale activation + order.completed")
	assert.Len(t, repo.upserts, 1, "only the newest state may be written")
	assert.Equal(t, StatusCancelled, repo.lastUpsert(t).Status)

	var supersededIDs []string
	for _, skipped := range result.Skipped {
		if assert.Error(t, skipped.Err) && skipped.EventType == "subscription.activated" {
			assert.ErrorIs(t, skipped.Err, errEventSuperseded)
			supersededIDs = append(supersededIDs, skipped.EventID)
		}
	}
	assert.Equal(t, []string{"evt-activated-out-of-order"}, supersededIDs)
}
