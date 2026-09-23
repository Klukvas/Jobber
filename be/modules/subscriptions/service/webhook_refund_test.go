package service

import (
	"context"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/fastspring"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleWebhookSurfacesARefundWithoutChangingAnything(t *testing.T) {
	// A refund is not a cancellation. The payload names an order rather than a
	// subscription, and a partial refund leaves the subscription billing
	// normally — so revoking access here would take a paid plan from someone who
	// still has one. Ending a subscription over a refund is a merchant decision
	// that arrives later as the deactivation it really is.
	//
	// What it must not do is disappear into the routine skips: money left the
	// account and nothing in the app moved, which is worth seeing before it
	// turns up in a payout.
	body := transformFirstEvent(t, loadFixture(t, taggedFixture), func(event map[string]any) {
		event["id"] = "evt-return-0001"
		event["type"] = fastspring.EventReturnCreated
	})

	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc, accountCalls := newTaggedService(t, repo)

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, repo.upserts, "a refund must not move a subscription on its own")
	assert.Empty(t, repo.claims, "nothing was applied, so nothing may be claimed")
	assert.Empty(t, result.Failed)
	assert.Contains(t, result.Processed, "evt-return-0001",
		"the event is understood, so FastSpring must stop redelivering it")

	require.Len(t, result.Skipped, 1)
	assert.ErrorIs(t, result.Skipped[0].Err, ErrRefundNeedsReview)
	assert.NotErrorIs(t, result.Skipped[0].Err, errEventNotActionable,
		"a refund is not one of the routine skips nobody reads")
	assert.Zero(t, accountCalls.Load())
}

func TestOrderCompletedStaysRoutine(t *testing.T) {
	// The counterpart: order.completed is subscribed to and deliberately does
	// nothing, because the subscription events already carry everything. It must
	// keep reporting as routine, or the distinction the refund relies on is
	// worth nothing.
	assert.ErrorIs(t, notActionableReason(fastspring.EventOrderCompleted), errEventNotActionable)
	assert.ErrorIs(t, notActionableReason(fastspring.EventReturnCreated), ErrRefundNeedsReview)
}
