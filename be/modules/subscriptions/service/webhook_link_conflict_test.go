package service

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The second subscription in these tests: a user who already pays for one pays
// for another, so the row would have to choose which of the two it names.
const (
	secondSubscriptionID = "SuBsCr1PT10nJobberPro02"
	secondActivationID   = "evt-activated-second-0001"
)

// secondActivationBody rewrites a recorded event into the activation of a
// *different* subscription on the same FastSpring account — what a second paid
// checkout actually delivers.
func secondActivationBody(t *testing.T) []byte {
	t.Helper()
	return transformFirstEvent(t, loadFixture(t, "subscription_updated.json"), func(event map[string]any) {
		event["id"] = secondActivationID
		event["type"] = "subscription.activated"
		data, ok := event["data"].(map[string]any)
		require.True(t, ok)
		data["id"] = secondSubscriptionID
		data["subscription"] = secondSubscriptionID
	})
}

// TestHandleWebhookRefusesASecondLiveSubscription covers the invariant from the
// side the order-tag guard cannot see.
//
// A user can start two checkouts while they are still free — the
// already-subscribed check passes both times, because neither has paid yet — and
// then pay for both. The second activation resolves through the provider
// *account ID*, which no order tag is involved in, so nothing before the write
// has an opinion about it. Applying it would overwrite the row's single
// external_subscription_id and leave the first subscription billing at
// FastSpring with nothing in Jobber able to cancel it.
func TestHandleWebhookRefusesASecondLiveSubscription(t *testing.T) {
	body := secondActivationBody(t)

	t.Run("the account hop cannot repoint a row at a second subscription", func(t *testing.T) {
		repo := newRecordingRepo(activeProSubscription())
		svc := newTestService(repo.MockSubscriptionRepository)
		svc.repo = repo

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.Empty(t, repo.upserts, "the subscription the user actually pays for must keep its row")
		assert.Empty(t, repo.claims, "a refused event claims nothing, so a resend can still land later")
		assert.Empty(t, result.Failed, "no redelivery can make a second subscription fit one row")
		require.Len(t, result.Skipped, 1)
		assert.ErrorIs(t, result.Skipped[0].Err, ErrSubscriptionLinkConflict)
		assert.NotErrorIs(t, result.Skipped[0].Err, errEventSuperseded,
			"a stranded live subscription is not routine event ordering")
		assert.NotErrorIs(t, result.Skipped[0].Err, ErrTaggedOwnerConflict,
			"no order tag was involved, so the diagnosis must not blame one")
		assert.Contains(t, result.Processed, secondActivationID)
	})

	t.Run("the refusal names the user and the event, never a stale subscription id", func(t *testing.T) {
		// The row was read before the write refused, so any identifier taken
		// from it could already describe state that has moved on.
		repo := newRecordingRepo(activeProSubscription())
		svc := newTestService(repo.MockSubscriptionRepository)
		svc.repo = repo

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		require.Len(t, result.Skipped, 1)
		message := result.Skipped[0].Err.Error()
		assert.Contains(t, message, testUserID)
		assert.Contains(t, message, secondSubscriptionID)
		assert.NotContains(t, message, fixtureSubscriptionID)
	})

	t.Run("the same subscription's own lifecycle events still apply", func(t *testing.T) {
		// The guard must refuse a *replacement*, not the ordinary stream of
		// events about the subscription the row already names.
		lifecycle := loadFixture(t, "subscription_updated.json")
		repo := newRecordingRepo(activeProSubscription())
		svc := newTestService(repo.MockSubscriptionRepository)
		svc.repo = repo

		result, err := svc.HandleWebhook(context.Background(), lifecycle, sign(t, lifecycle, testWebhookSecret))

		require.NoError(t, err)
		assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
		assert.Equal(t, PlanEnterprise, repo.lastUpsert(t).Plan)
	})

	t.Run("a repurchase after the provider ended the old subscription applies", func(t *testing.T) {
		cancelled := activeProSubscription()
		cancelled.Status = StatusCancelled
		cancelled.Plan = PlanFree
		repo := newRecordingRepo(cancelled)
		svc := newTestService(repo.MockSubscriptionRepository)
		svc.repo = repo

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
		written := repo.lastUpsert(t)
		require.NotNil(t, written.ExternalSubscriptionID)
		assert.Equal(t, secondSubscriptionID, *written.ExternalSubscriptionID)
	})
}

// TestHandleWebhookOrderTagConflictSurvivesAStalePrecheck is the reason the
// invariant lives in the write rather than only in resolveOwnerByOrderTag.
//
// That hop reads the row and then hands it to a separate write. Between the two,
// a concurrent activation can link a subscription the read never saw — so the
// precheck waves the event through and the write is the last thing standing
// between the user and a stranded, still-billing subscription.
func TestHandleWebhookOrderTagConflictSurvivesAStalePrecheck(t *testing.T) {
	body := loadFixture(t, taggedFixture)

	// What the write sees: a row already linked to another live subscription,
	// and no account link, so resolution has to go through the tag.
	linked := &model.Subscription{
		ID:                     "sub-row-1",
		UserID:                 testUserID,
		ExternalSubscriptionID: ptr(otherProviderSubscrID),
		Status:                 StatusActive,
		Plan:                   PlanPro,
	}
	repo := newRecordingRepo(linked)
	// What the precheck sees: the row as it was a moment earlier, with nothing
	// to protect.
	repo.GetByUserIDFunc = func(_ context.Context, userID string) (*model.Subscription, error) {
		return &model.Subscription{ID: "sub-row-1", UserID: userID, Status: StatusFree, Plan: PlanFree}, nil
	}
	svc, _ := newTaggedService(t, repo)

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, repo.upserts, "the stale read must not be what decides a paying user's row")
	assert.Empty(t, repo.claims)
	assert.Empty(t, result.Failed)
	require.Len(t, result.Skipped, 1)
	assert.ErrorIs(t, result.Skipped[0].Err, ErrSubscriptionLinkConflict)
	assert.Contains(t, result.Processed, taggedEventID)
}
