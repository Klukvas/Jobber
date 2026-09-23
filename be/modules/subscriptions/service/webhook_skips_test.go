package service

import (
	"context"
	"errors"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleWebhookAcknowledgesAnAccountAlreadyLinkedElsewhere(t *testing.T) {
	// Two local users behind one provider billing account. The write refuses,
	// and the event must be acknowledged rather than retried: no redelivery can
	// untangle that, and with reconciliation running the retry is no longer
	// bounded by the provider's own give-up — it would come back every sweep
	// until it aged out of the window.
	body := loadFixture(t, taggedFixture)

	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	repo.ApplySubscriptionEventFunc = func(
		context.Context, string, string, *model.Subscription,
	) (model.WebhookApplyOutcome, error) {
		return model.WebhookAccountConflict, nil
	}
	svc, _ := newTaggedService(t, repo)

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, result.Failed, "a collision no retry can fix must not be queued for retry")
	assert.Contains(t, result.Processed, taggedEventID)

	require.Len(t, result.Skipped, 1)
	assert.ErrorIs(t, result.Skipped[0].Err, model.ErrBillingAccountTaken)

	_, needsAttention := SkipReport(result.Skipped[0].Err)
	assert.True(t, needsAttention, "somebody has to decide which user the purchase belongs to")
}

func TestSkipReportSeparatesRoutineFromNoteworthy(t *testing.T) {
	// Both channels — the webhook endpoint and the reconciliation sweep — ask
	// this one function how loudly to report a skip. If a sentinel is added
	// without an arm here it silently becomes routine on both.
	noteworthy := map[string]error{
		"environment mismatch":  ErrEnvironmentMismatch,
		"unproven order tag":    ErrUnprovenOrderTag,
		"tagged owner conflict": ErrTaggedOwnerConflict,
		"link conflict":         ErrSubscriptionLinkConflict,
		"account already taken": model.ErrBillingAccountTaken,
		"refund":                ErrRefundNeedsReview,
	}
	for name, err := range noteworthy {
		t.Run(name+" is worth attention", func(t *testing.T) {
			message, needsAttention := SkipReport(err)

			assert.True(t, needsAttention)
			assert.NotEmpty(t, message)
		})
	}

	routine := map[string]error{
		"duplicate":       errEventDuplicate,
		"superseded":      errEventSuperseded,
		"not actionable":  errEventNotActionable,
		"foreign account": errForeignBillingAccount,
	}
	for name, err := range routine {
		t.Run(name+" stays quiet", func(t *testing.T) {
			_, needsAttention := SkipReport(err)

			assert.False(t, needsAttention)
		})
	}

	t.Run("a wrapped sentinel is still recognised", func(t *testing.T) {
		// Every caller wraps with context before the error reaches here.
		_, needsAttention := SkipReport(errors.New("wrapped: " + ErrUnprovenOrderTag.Error()))
		assert.False(t, needsAttention, "a look-alike string is not the sentinel")

		_, needsAttention = SkipReport(errors.Join(ErrUnprovenOrderTag, errors.New("context")))
		assert.True(t, needsAttention)
	})
}
