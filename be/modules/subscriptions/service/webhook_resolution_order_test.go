package service

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sharedAccountUserID stands in for a *second* local user behind the same
// FastSpring account. FastSpring records an account per buyer contact, not per
// purchase, so two Jobber users who check out with one email can end up sharing
// one account ID.
const sharedAccountUserID = "22222222-3333-4444-8555-666666666666"

// withoutEventField drops one field from the first event's data object,
// standing in for a payload that arrives without it.
func withoutEventField(t *testing.T, body []byte, field string) []byte {
	t.Helper()
	return transformFirstEvent(t, body, func(event map[string]any) {
		data, ok := event["data"].(map[string]any)
		require.True(t, ok)
		delete(data, field)
	})
}

func TestResolutionPrefersTheProvenTagOverTheAccount(t *testing.T) {
	// The account hop is recorded per *buyer*, so it can name the wrong local
	// user: two Jobber accounts that check out with one email share a FastSpring
	// account, and whichever of them linked it first would then collect the
	// other's purchase. The order tag is the only evidence bound to this one
	// order — it names the user whose authenticated session opened the checkout
	// — so once its proof verifies it outranks the account.
	body := loadFixture(t, taggedFixture)

	repo := newRecordingRepo(&model.Subscription{
		ID:                "row-shared-account",
		UserID:            sharedAccountUserID,
		Status:            StatusFree,
		Plan:              PlanFree,
		ExternalAccountID: ptr(taggedAccountID),
	})
	repo.GetByUserIDFunc = func(_ context.Context, userID string) (*model.Subscription, error) {
		if userID != testUserID {
			return nil, model.ErrSubscriptionNotFound
		}
		return &model.Subscription{ID: "row-buyer", UserID: testUserID, Status: StatusFree, Plan: PlanFree}, nil
	}
	svc, accountCalls := newTaggedService(t, repo)

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)

	written := repo.lastUpsert(t)
	assert.Equal(t, testUserID, written.UserID,
		"the purchase belongs to the user the proven tag names, not to whoever linked the shared account first")
	assert.Equal(t, PlanEnterprise, written.Plan)
	assert.Zero(t, accountCalls.Load(), "a proven tag resolves without any account read-back")
}

func TestResolutionReadsTheProvenTagWhenTheEventCarriesNoAccount(t *testing.T) {
	// The tag hop used to sit *below* a guard that gave up on any event without
	// an account ID, which made the one hop designed to link a first purchase
	// unreachable for exactly those payloads — and left them retrying until
	// FastSpring gave up.
	body := withoutEventField(t, loadFixture(t, taggedFixture), "account")

	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc, accountCalls := newTaggedService(t, repo)

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, result.Failed, "a provable claim must not be left to retry")
	assert.Contains(t, result.Processed, taggedEventID)

	written := repo.lastUpsert(t)
	assert.Equal(t, testUserID, written.UserID)
	assert.Equal(t, PlanEnterprise, written.Plan)
	assert.Nil(t, written.ExternalAccountID, "there was no account in the payload to record")
	assert.Zero(t, accountCalls.Load(), "an absent account ID is nothing to read back")
}

func TestResolutionRetriesAnEventThatIdentifiesNobody(t *testing.T) {
	// No account and no tag: nothing identifies the event. That is a payload
	// shape we failed to read, not somebody else's customer, so it must stay
	// retryable rather than being acknowledged away like a foreign account.
	body := withoutEventField(t, loadFixture(t, taggedFixture), "account")
	body = withoutEventField(t, body, "tags")

	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc, accountCalls := newTaggedService(t, repo)

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, repo.upserts, "an unidentifiable event must grant nothing")
	require.Len(t, result.Failed, 1)
	assert.ErrorIs(t, result.Failed[0].Err, model.ErrSubscriptionNotFound)
	assert.NotErrorIs(t, result.Failed[0].Err, errForeignBillingAccount,
		"a payload we could not read is not a foreign purchase")
	assert.Zero(t, accountCalls.Load())
}
