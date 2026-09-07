package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Identifiers used by the testdata fixtures.
const (
	fixtureSubscriptionID = "SuBsCr1PT10nJobberPro01"
	fixtureAccountID      = "acctJobberTestAcct001x"
)

// loadFixture reads a recorded FastSpring webhook body from testdata.
func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err, "fixture %s", name)
	return body
}

// sign produces the X-FS-Signature value FastSpring would send for a body.
func sign(t *testing.T, body []byte, secret string) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// recordingRepo is a mock wired for webhook tests: it resolves the fixture
// account to a known user and models the repository's atomic apply — one claim
// per event ID, an ordering guard on the stored state, and the write itself —
// capturing everything it applies. Later events in a batch therefore observe
// earlier writes, exactly as the single SQL statement makes them.
type recordingRepo struct {
	*MockSubscriptionRepository
	mu      sync.Mutex
	current *model.Subscription
	upserts []model.Subscription
	claims  []string
}

func newRecordingRepo(existing *model.Subscription) *recordingRepo {
	r := &recordingRepo{MockSubscriptionRepository: &MockSubscriptionRepository{}, current: existing}
	r.GetByExternalSubscriptionIDFunc = func(_ context.Context, id string) (*model.Subscription, error) {
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.current != nil && r.current.ExternalSubscriptionID != nil && *r.current.ExternalSubscriptionID == id {
			return copySubscription(r.current), nil
		}
		return nil, model.ErrSubscriptionNotFound
	}
	r.GetByExternalAccountIDFunc = func(_ context.Context, id string) (*model.Subscription, error) {
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.current != nil && r.current.ExternalAccountID != nil && *r.current.ExternalAccountID == id {
			return copySubscription(r.current), nil
		}
		return nil, model.ErrSubscriptionNotFound
	}
	r.ApplySubscriptionEventFunc = func(_ context.Context, eventID, _ string, sub *model.Subscription) (model.WebhookApplyOutcome, error) {
		r.mu.Lock()
		defer r.mu.Unlock()

		// The link guard the real write enforces under a row lock, before it
		// claims anything: the row holds one external_subscription_id, and it is
		// never replaced while the subscription it names is still billing.
		if replacesLiveLink(r.current, sub.ExternalSubscriptionID) {
			return model.WebhookLinkConflict, nil
		}

		for _, seen := range r.claims {
			if seen == eventID {
				return model.WebhookDuplicate, nil
			}
		}
		r.claims = append(r.claims, eventID)

		// The same ordering guard the SQL WHERE clause enforces, including its
		// strictness: only an event describing a *newer* change is applied. One
		// carrying the same `changed` is claimed and recorded as processed, but
		// must not rewrite the state that already stands.
		if r.current != nil && r.current.LastEventAt != nil &&
			(sub.LastEventAt == nil || !sub.LastEventAt.After(*r.current.LastEventAt)) {
			return model.WebhookSuperseded, nil
		}

		r.upserts = append(r.upserts, *sub)
		r.current = copySubscription(sub)
		return model.WebhookApplied, nil
	}
	return r
}

// replacesLiveLink mirrors the repository's link guard. It lives here rather
// than being imported because the service must not depend on the repository —
// the price is that the two are kept in step by the integration tests that run
// the real SQL.
func replacesLiveLink(stored *model.Subscription, incomingID *string) bool {
	if stored == nil || stored.ExternalSubscriptionID == nil || *stored.ExternalSubscriptionID == "" {
		return false
	}
	if incomingID != nil && *incomingID == *stored.ExternalSubscriptionID {
		return false
	}
	return stored.Status != StatusCancelled
}

func (r *recordingRepo) lastUpsert(t *testing.T) model.Subscription {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	require.NotEmpty(t, r.upserts, "expected a subscription write")
	return r.upserts[len(r.upserts)-1]
}

func copySubscription(s *model.Subscription) *model.Subscription {
	clone := *s
	return &clone
}

func ptr[T any](v T) *T { return &v }

// linkedFreeSubscription is a user who has started a checkout: the FastSpring
// account is linked but nothing is granted yet.
func linkedFreeSubscription() *model.Subscription {
	return &model.Subscription{
		ID:                "sub-row-1",
		UserID:            testUserID,
		ExternalAccountID: ptr(fixtureAccountID),
		Status:            StatusFree,
		Plan:              PlanFree,
	}
}

// activeProSubscription is a user already paying for the pro plan.
func activeProSubscription() *model.Subscription {
	return &model.Subscription{
		ID:                     "sub-row-1",
		UserID:                 testUserID,
		ExternalSubscriptionID: ptr(fixtureSubscriptionID),
		ExternalAccountID:      ptr(fixtureAccountID),
		Status:                 StatusActive,
		Plan:                   PlanPro,
	}
}

func TestHandleWebhookSignature(t *testing.T) {
	body := loadFixture(t, "batch_activated_order_completed.json")

	tests := []struct {
		name      string
		secret    string
		signature func() string
		wantErr   error
	}{
		{
			name:      "valid signature is accepted",
			secret:    testWebhookSecret,
			signature: func() string { return sign(t, body, testWebhookSecret) },
		},
		{
			name:      "signature from another secret is rejected",
			secret:    testWebhookSecret,
			signature: func() string { return sign(t, body, "attacker-secret") },
			wantErr:   fastspringSignatureInvalid,
		},
		{
			name:      "missing signature is rejected",
			secret:    testWebhookSecret,
			signature: func() string { return "" },
			wantErr:   fastspringSignatureMissing,
		},
		{
			name:      "non-base64 signature is rejected",
			secret:    testWebhookSecret,
			signature: func() string { return "not base64!!" },
			wantErr:   fastspringSignatureInvalid,
		},
		{
			name:      "empty secret rejects everything",
			secret:    "",
			signature: func() string { return sign(t, body, testWebhookSecret) },
			wantErr:   fastspringSecretMissing,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRecordingRepo(linkedFreeSubscription())
			svc := newTestService(repo.MockSubscriptionRepository)
			svc.repo = repo
			svc.cfg.WebhookSecret = tc.secret

			_, err := svc.HandleWebhook(context.Background(), body, tc.signature())

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				// A rejected payload must leave the database untouched.
				assert.Empty(t, repo.upserts, "rejected webhook must not write")
				assert.Empty(t, repo.claims, "rejected webhook must not claim events")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHandleWebhookLifecycle(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		existing   *model.Subscription
		wantStatus string
		wantPlan   string
		wantCancel bool
	}{
		{
			name:       "activation grants the purchased plan",
			fixture:    "batch_activated_order_completed.json",
			existing:   linkedFreeSubscription(),
			wantStatus: StatusActive,
			wantPlan:   PlanPro,
		},
		{
			name:       "update moves the subscriber to the new product",
			fixture:    "subscription_updated.json",
			existing:   activeProSubscription(),
			wantStatus: StatusActive,
			wantPlan:   PlanEnterprise,
		},
		{
			name:       "scheduled cancellation keeps access and records the end date",
			fixture:    "subscription_canceled.json",
			existing:   activeProSubscription(),
			wantStatus: StatusActive,
			wantPlan:   PlanPro,
			wantCancel: true,
		},
		{
			name:       "deactivation revokes access",
			fixture:    "subscription_deactivated.json",
			existing:   activeProSubscription(),
			wantStatus: StatusCancelled,
			wantPlan:   PlanFree,
		},
		{
			name:       "overdue payment keeps grace access",
			fixture:    "subscription_payment_overdue.json",
			existing:   activeProSubscription(),
			wantStatus: StatusPastDue,
			wantPlan:   PlanPro,
		},
		{
			name:       "failed rebill marks the subscription past due",
			fixture:    "subscription_charge_failed.json",
			existing:   activeProSubscription(),
			wantStatus: StatusPastDue,
			wantPlan:   PlanPro,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := loadFixture(t, tc.fixture)
			repo := newRecordingRepo(tc.existing)
			svc := newTestService(repo.MockSubscriptionRepository)
			svc.repo = repo

			result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

			require.NoError(t, err)
			assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)

			written := repo.lastUpsert(t)
			assert.Equal(t, testUserID, written.UserID)
			assert.Equal(t, tc.wantStatus, written.Status)
			assert.Equal(t, tc.wantPlan, written.Plan)
			require.NotNil(t, written.ExternalSubscriptionID)
			assert.Equal(t, fixtureSubscriptionID, *written.ExternalSubscriptionID)
			require.NotNil(t, written.LastEventAt)

			if tc.wantCancel {
				assert.NotNil(t, written.CancelAt, "a scheduled cancellation must record its end date")
			} else {
				assert.Nil(t, written.CancelAt)
			}

			if tc.wantStatus == StatusCancelled {
				// An ended subscription has no current period; leaving a future
				// renewal date behind would misreport the row.
				assert.Nil(t, written.CurrentPeriodEnd)
				assert.Nil(t, written.CurrentPeriodStart)
			} else {
				assert.NotNil(t, written.CurrentPeriodEnd, "an active subscription must record its next charge date")
			}
		})
	}
}

func TestHandleWebhookAcknowledgesNonActionableEvents(t *testing.T) {
	// order.completed rides along with subscription.activated; it is acknowledged
	// so FastSpring stops retrying, but entitlement comes from the subscription
	// event alone — exactly one write for the two-event batch.
	body := loadFixture(t, "batch_activated_order_completed.json")
	repo := newRecordingRepo(linkedFreeSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Equal(t, []string{"evt-activated-0001", "evt-order-completed-0001"}, result.Processed)
	assert.Len(t, repo.upserts, 1, "order.completed must not write a second time")
	require.Len(t, result.Skipped, 1)
	assert.Equal(t, "order.completed", result.Skipped[0].EventType)
}

func TestHandleWebhookDuplicateDelivery(t *testing.T) {
	body := loadFixture(t, "batch_duplicate_activated.json")
	repo := newRecordingRepo(linkedFreeSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.True(t, result.AllProcessed())
	assert.Len(t, result.Processed, 2, "both copies are acknowledged")
	assert.Len(t, repo.upserts, 1, "the repeated event must be applied only once")
}

func TestHandleWebhookRejectsForeignEnvironment(t *testing.T) {
	body := loadFixture(t, "subscription_activated_live.json")
	repo := newRecordingRepo(linkedFreeSubscription())

	t.Run("live event on a test deployment is acknowledged but not applied", func(t *testing.T) {
		svc := newTestService(repo.MockSubscriptionRepository)
		svc.repo = repo

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.True(t, result.AllProcessed())
		assert.Empty(t, repo.upserts, "a live event must not change test-mode data")
		assert.Empty(t, repo.claims, "a mismatched event must not consume an idempotency claim")
		require.Len(t, result.Skipped, 1)
		assert.ErrorIs(t, result.Skipped[0].Err, ErrEnvironmentMismatch)
	})

	t.Run("test event on a live deployment is acknowledged but not applied", func(t *testing.T) {
		testBody := loadFixture(t, "batch_activated_order_completed.json")
		liveRepo := newRecordingRepo(linkedFreeSubscription())
		svc := newTestService(liveRepo.MockSubscriptionRepository)
		svc.repo = liveRepo
		svc.cfg.Environment = EnvironmentLive

		result, err := svc.HandleWebhook(context.Background(), testBody, sign(t, testBody, testWebhookSecret))

		require.NoError(t, err)
		assert.True(t, result.AllProcessed())
		assert.Empty(t, liveRepo.upserts, "a test event must not change live data")
	})
}

func TestHandleWebhookUnknownProductGrantsNothing(t *testing.T) {
	body := loadFixture(t, "subscription_activated_unknown_product.json")
	repo := newRecordingRepo(linkedFreeSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, repo.upserts, "an unrecognised product must never grant paid access")
	require.Len(t, result.Failed, 1)
	assert.Contains(t, result.Failed[0].Err.Error(), "unrecognised product path")
	// The event never reached the atomic apply, so no claim exists to strand it:
	// the retry gets a clean run once the catalog is corrected.
	assert.Empty(t, repo.claims)
}

func TestHandleWebhookReplayCannotResurrectDeactivated(t *testing.T) {
	deactivated := loadFixture(t, "subscription_deactivated.json")
	repo := newRecordingRepo(activeProSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	_, err := svc.HandleWebhook(context.Background(), deactivated, sign(t, deactivated, testWebhookSecret))
	require.NoError(t, err)
	require.Equal(t, StatusCancelled, repo.lastUpsert(t).Status)

	// A manual resend gets a fresh event ID, so event-ID de-duplication cannot
	// stop it — the recorded event time must.
	cancelled := repo.lastUpsert(t)
	replayRepo := newRecordingRepo(&cancelled)
	replaySvc := newTestService(replayRepo.MockSubscriptionRepository)
	replaySvc.repo = replayRepo

	activated := replaceEventID(t, loadFixture(t, "batch_activated_order_completed.json"),
		"evt-activated-0001", "evt-activated-manual-resend")
	result, err := replaySvc.HandleWebhook(context.Background(), activated, sign(t, activated, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, replayRepo.upserts, "an older event must not reactivate a deactivated subscription")
	require.NotEmpty(t, result.Skipped)
	assert.ErrorIs(t, result.Skipped[0].Err, errEventSuperseded)
	assert.True(t, result.AllProcessed(), "a superseded replay is acknowledged, not retried forever")
}

func TestHandleWebhookPartialBatchFailure(t *testing.T) {
	// One batch, two events: the first cannot be resolved to a user, the second
	// can. Only the failed claim is released and only the succeeded ID is acked.
	body := mergeFixtures(t,
		loadFixture(t, "subscription_activated_unknown_product.json"),
		loadFixture(t, "subscription_canceled.json"),
	)
	repo := newRecordingRepo(activeProSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.False(t, result.AllProcessed())
	assert.Equal(t, []string{"evt-canceled-0001"}, result.Processed)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, "evt-activated-unknown-0001", result.Failed[0].EventID)
	assert.Equal(t, []string{"evt-canceled-0001"}, repo.claims,
		"only the applied event may hold a claim; the failed one must stay retryable")
	assert.Len(t, repo.upserts, 1, "the healthy event still applies")
}

func TestHandleWebhookUnknownAccountGrantsNothing(t *testing.T) {
	body := loadFixture(t, "batch_activated_order_completed.json")
	// No local row matches the fixture's account, and the account lookup fails
	// because the client has no credentials.
	repo := newRecordingRepo(nil)
	svc := NewSubscriptionService(repo, unconfiguredClient(), testBillingConfig())

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, repo.upserts, "an unresolvable purchase must not grant access to anyone")
	require.Len(t, result.Failed, 1)
	assert.Equal(t, "evt-activated-0001", result.Failed[0].EventID)
}

func TestHandleWebhookClaimFailureDoesNotApply(t *testing.T) {
	// The claim and the write are one statement, so a database failure rolls
	// both back: the event must come back as retryable, never as processed.
	body := loadFixture(t, "batch_activated_order_completed.json")
	repo := newRecordingRepo(linkedFreeSubscription())
	repo.ApplySubscriptionEventFunc = func(context.Context, string, string, *model.Subscription) (model.WebhookApplyOutcome, error) {
		return "", errors.New("database unavailable")
	}
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, repo.upserts)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, "evt-activated-0001", result.Failed[0].EventID)
	assert.NotContains(t, result.Processed, "evt-activated-0001",
		"a rolled-back apply must not be acknowledged")
}

func TestHandleWebhookMalformedBody(t *testing.T) {
	body := []byte(`{"events": [ this is not json ]}`)
	repo := newRecordingRepo(linkedFreeSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	_, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.Error(t, err)
	assert.Empty(t, repo.upserts)
	assert.Empty(t, repo.claims)
}

func TestStatusForEvent(t *testing.T) {
	tests := []struct {
		name       string
		eventType  string
		state      string
		active     *bool
		wantStatus string
		wantErr    bool
	}{
		{name: "active", eventType: eventUpdated, state: "active", active: ptr(true), wantStatus: StatusActive},
		{name: "trial counts as active", eventType: eventUpdated, state: "trial", active: ptr(true), wantStatus: StatusActive},
		{name: "scheduled cancellation keeps access", eventType: eventCanceled, state: "canceled", active: ptr(true), wantStatus: StatusActive},
		{name: "deactivated ends access", eventType: eventDeactivated, state: "deactivated", active: ptr(false), wantStatus: StatusCancelled},
		{name: "overdue state", eventType: eventUpdated, state: "overdue", active: ptr(true), wantStatus: StatusPastDue},
		{name: "paused", eventType: eventUpdated, state: "paused", active: ptr(true), wantStatus: StatusPaused},
		{name: "overdue notice overrides an active state", eventType: eventOverdue, state: "active", active: ptr(true), wantStatus: StatusPastDue},
		{
			name:       "pause suspends paid access even before the state catches up",
			eventType:  eventPaused,
			state:      "active",
			active:     ptr(true),
			wantStatus: StatusPaused,
		},
		{
			// The case the priority order exists for: FastSpring reports a
			// paused subscription as inactive. Reading that as a cancellation
			// would drop the row back to the free plan and lose the purchase
			// the buyer resumes into.
			name:       "pause reported as inactive is still a pause, not a cancellation",
			eventType:  eventPaused,
			state:      "paused",
			active:     ptr(false),
			wantStatus: StatusPaused,
		},
		{
			name:       "pause whose state has not caught up is still a pause",
			eventType:  eventPaused,
			state:      "active",
			active:     ptr(false),
			wantStatus: StatusPaused,
		},
		{
			name:       "pause of an already deactivated subscription stays cancelled",
			eventType:  eventPaused,
			state:      "deactivated",
			active:     ptr(false),
			wantStatus: StatusCancelled,
		},
		{name: "resume restores active access", eventType: eventResumed, state: "active", active: ptr(true), wantStatus: StatusActive},
		{
			// The mirror image of the pause case: the un-pause is explicit, so
			// an `active` flag still catching up must not cancel the row.
			name:       "resume whose active flag has not caught up still restores access",
			eventType:  eventResumed,
			state:      "active",
			active:     ptr(false),
			wantStatus: StatusActive,
		},
		{
			name:       "resume of a deactivated subscription cannot resurrect it",
			eventType:  eventResumed,
			state:      "deactivated",
			active:     ptr(false),
			wantStatus: StatusCancelled,
		},
		{
			// Deactivation is first in the priority order, so it wins even
			// against a payload that contradicts itself.
			name:       "deactivated state wins over a contradictory active flag",
			eventType:  eventUpdated,
			state:      "deactivated",
			active:     ptr(true),
			wantStatus: StatusCancelled,
		},
		{
			name:       "deactivated state wins over a dunning notice",
			eventType:  eventOverdue,
			state:      "deactivated",
			active:     ptr(false),
			wantStatus: StatusCancelled,
		},
		{name: "charge failure overrides an active state", eventType: eventChargeFailed, state: "active", active: ptr(true), wantStatus: StatusPastDue},
		{
			name:       "charge failure on an inactive subscription stays cancelled",
			eventType:  eventChargeFailed,
			state:      "active",
			active:     ptr(false),
			wantStatus: StatusCancelled,
		},
		{name: "unknown state is an error", eventType: eventUpdated, state: "warp-speed", active: ptr(true), wantErr: true},
		{
			name:       "absent active flag is not treated as false",
			eventType:  eventUpdated,
			state:      "active",
			active:     nil,
			wantStatus: StatusActive,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := statusForEvent(tc.eventType, newParsedSubscription(tc.state, tc.active))

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, got)
		})
	}
}

func TestPendingCancelAt(t *testing.T) {
	deactivation := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("set while a cancellation is scheduled", func(t *testing.T) {
		sub := newParsedSubscription("canceled", ptr(true))
		sub.Deactivation = &deactivation
		assert.Equal(t, &deactivation, pendingCancelAt(StatusActive, sub))
	})

	t.Run("cleared once the subscription is active again", func(t *testing.T) {
		sub := newParsedSubscription("active", ptr(true))
		sub.Deactivation = &deactivation
		assert.Nil(t, pendingCancelAt(StatusActive, sub))
	})

	t.Run("cleared once access has ended", func(t *testing.T) {
		sub := newParsedSubscription("deactivated", ptr(false))
		sub.Deactivation = &deactivation
		assert.Nil(t, pendingCancelAt(StatusCancelled, sub))
	})
}

func TestHandleWebhookOrdersOnProviderChangeTime(t *testing.T) {
	// A manual resend arrives in a brand-new envelope: fresh event ID, fresh
	// `created`. Only the payload's own `changed` still points at the original
	// state change, so ordering on the envelope would let a stale activation
	// undo a deactivation that already happened.
	deactivated := loadFixture(t, "subscription_deactivated.json")
	repo := newRecordingRepo(activeProSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	_, err := svc.HandleWebhook(context.Background(), deactivated, sign(t, deactivated, testWebhookSecret))
	require.NoError(t, err)
	require.Equal(t, StatusCancelled, repo.lastUpsert(t).Status)

	resend := transformFirstEvent(t, loadFixture(t, "batch_activated_order_completed.json"), func(event map[string]any) {
		event["id"] = "evt-activated-manual-resend"
		// Newer than the deactivation's envelope *and* newer than its `changed`.
		event["created"] = float64(1751999999000)
	})

	result, err := svc.HandleWebhook(context.Background(), resend, sign(t, resend, testWebhookSecret))

	require.NoError(t, err)
	assert.Len(t, repo.upserts, 1,
		"a resend stamped with a newer envelope must not revive the old state")
	assert.Equal(t, StatusCancelled, repo.lastUpsert(t).Status)
	require.NotEmpty(t, result.Skipped)
	assert.ErrorIs(t, result.Skipped[0].Err, errEventSuperseded)
	assert.True(t, result.AllProcessed(), "a superseded resend is acknowledged, not retried forever")
}

func TestHandleWebhookFallsBackToEnvelopeTimeWithoutChanged(t *testing.T) {
	// Charge payloads may omit `changed`; the envelope's creation time is then
	// the only ordering information available.
	body := transformFirstEvent(t, loadFixture(t, "subscription_charge_failed.json"), func(event map[string]any) {
		subscription := event["data"].(map[string]any)["subscription"].(map[string]any)
		delete(subscription, "changed")
	})
	repo := newRecordingRepo(activeProSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
	written := repo.lastUpsert(t)
	require.NotNil(t, written.LastEventAt)
	assert.Equal(t, time.UnixMilli(1751760000000).UTC(), *written.LastEventAt)
}

func TestHandleWebhookUsesChangedOverEnvelopeCreated(t *testing.T) {
	// The recorded event time must be the provider's change moment, not the
	// envelope's, so a later comparison sees the real lifecycle order.
	body := transformFirstEvent(t, loadFixture(t, "subscription_updated.json"), func(event map[string]any) {
		event["created"] = float64(1751999999000)
		event["data"].(map[string]any)["changed"] = float64(1751414400000)
	})
	repo := newRecordingRepo(activeProSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	_, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	written := repo.lastUpsert(t)
	require.NotNil(t, written.LastEventAt)
	assert.Equal(t, time.UnixMilli(1751414400000).UTC(), *written.LastEventAt)
}

// pausedEvent builds one of the pause/resume events FastSpring's webhook list
// defines but the recorded fixtures do not cover, leaving the payload's `active`
// flag as the fixture has it.
func pausedEvent(t *testing.T, eventID, eventType, state string, changed int64) []byte {
	t.Helper()
	return lifecycleEvent{id: eventID, kind: eventType, state: state, changed: changed}.body(t)
}

func TestHandleWebhookPauseAndResume(t *testing.T) {
	// FastSpring's webhook list includes subscription.paused and
	// subscription.resumed. A pause must stop paid entitlement even though the
	// subscription is not cancelled, and a resume must give it back.
	repo := newRecordingRepo(activeProSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	paused := pausedEvent(t, "evt-paused-0001", "subscription.paused", "paused", 1751500800000)
	result, err := svc.HandleWebhook(context.Background(), paused, sign(t, paused, testWebhookSecret))

	require.NoError(t, err)
	assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
	pausedRow := repo.lastUpsert(t)
	assert.Equal(t, StatusPaused, pausedRow.Status)
	assert.Equal(t, PlanPro, pausedRow.Plan, "the purchased plan is remembered, not revoked")

	// Paid limits must not apply while billing is suspended.
	pausedRepo := &MockSubscriptionRepository{
		GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
			return copySubscription(&pausedRow), nil
		},
	}
	plan, err := newTestService(pausedRepo).effectivePlan(context.Background(), testUserID)
	require.NoError(t, err)
	assert.Equal(t, PlanFree, plan, "a paused subscription must not keep paid quotas")
	assert.ErrorIs(t, newTestService(pausedRepo).RequirePaidPlan(context.Background(), testUserID), model.ErrPaidFeature)

	resumed := pausedEvent(t, "evt-resumed-0001", "subscription.resumed", "active", 1751587200000)
	result, err = svc.HandleWebhook(context.Background(), resumed, sign(t, resumed, testWebhookSecret))

	require.NoError(t, err)
	assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
	resumedRow := repo.lastUpsert(t)
	assert.Equal(t, StatusActive, resumedRow.Status)
	assert.Equal(t, PlanPro, resumedRow.Plan)

	resumedRepo := &MockSubscriptionRepository{
		GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
			return copySubscription(&resumedRow), nil
		},
	}
	plan, err = newTestService(resumedRepo).effectivePlan(context.Background(), testUserID)
	require.NoError(t, err)
	assert.Equal(t, PlanPro, plan, "a resume must give paid quotas back")
}

// lifecycleEvent rewrites the update fixture into an arbitrary subscription
// lifecycle event, covering payload combinations the recorded fixtures do not.
type lifecycleEvent struct {
	id      string
	kind    string
	state   string
	active  *bool
	changed int64
}

func (e lifecycleEvent) body(t *testing.T) []byte {
	t.Helper()
	return transformFirstEvent(t, loadFixture(t, "subscription_updated.json"), func(event map[string]any) {
		event["id"] = e.id
		event["type"] = e.kind
		event["created"] = float64(e.changed)
		data := event["data"].(map[string]any)
		data["state"] = e.state
		data["changed"] = float64(e.changed)
		if e.active != nil {
			data["active"] = *e.active
		}
		// The pro plan is what these events grant, keep or take away.
		data["product"] = map[string]any{"product": testProProductPath}
	})
}

// chargeCompletedEvent builds a subscription.charge.completed from the recorded
// charge fixture, keeping its nested `data.subscription` shape.
func chargeCompletedEvent(t *testing.T, eventID string, changed int64) []byte {
	t.Helper()
	return transformFirstEvent(t, loadFixture(t, "subscription_charge_failed.json"), func(event map[string]any) {
		event["id"] = eventID
		event["type"] = "subscription.charge.completed"
		event["created"] = float64(changed)
		subscription := event["data"].(map[string]any)["subscription"].(map[string]any)
		subscription["state"] = "active"
		subscription["active"] = true
		subscription["changed"] = float64(changed)
		subscription["product"] = map[string]any{"product": testProProductPath}
	})
}

func TestHandleWebhookSameChangedEventCannotFlipState(t *testing.T) {
	// `data.changed` is the provider's timestamp for a state change, so two
	// events bearing the same one describe the *same* change and must normalise
	// to the same state. The ordering guard is therefore strict: whichever
	// arrives first wins, and the second is acknowledged as superseded instead of
	// being replayed. A genuine later change always carries a later `changed`, so
	// nothing real is lost — but a resend, a reordered batch or a hand-crafted
	// replay can no longer flip entitlement in either direction.
	const changed = int64(1751414400000)

	tests := []struct {
		name       string
		first      lifecycleEvent
		replay     lifecycleEvent
		wantStatus string
		wantPlan   string
	}{
		{
			name:       "a same-instant deactivation cannot revoke an applied activation",
			first:      lifecycleEvent{id: "evt-activated-t0", kind: "subscription.activated", state: "active", active: ptr(true), changed: changed},
			replay:     lifecycleEvent{id: "evt-deactivated-t0", kind: "subscription.deactivated", state: "deactivated", active: ptr(false), changed: changed},
			wantStatus: StatusActive,
			wantPlan:   PlanPro,
		},
		{
			name:       "a same-instant activation cannot resurrect an applied deactivation",
			first:      lifecycleEvent{id: "evt-deactivated-t0", kind: "subscription.deactivated", state: "deactivated", active: ptr(false), changed: changed},
			replay:     lifecycleEvent{id: "evt-activated-t0", kind: "subscription.activated", state: "active", active: ptr(true), changed: changed},
			wantStatus: StatusCancelled,
			wantPlan:   PlanFree,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newRecordingRepo(activeProSubscription())
			svc := newTestService(repo.MockSubscriptionRepository)
			svc.repo = repo

			first := tc.first.body(t)
			_, err := svc.HandleWebhook(context.Background(), first, sign(t, first, testWebhookSecret))
			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, repo.lastUpsert(t).Status)

			replay := tc.replay.body(t)
			result, err := svc.HandleWebhook(context.Background(), replay, sign(t, replay, testWebhookSecret))

			require.NoError(t, err)
			assert.Len(t, repo.upserts, 1, "an event that is not strictly newer must not write")
			written := repo.lastUpsert(t)
			assert.Equal(t, tc.wantStatus, written.Status, "the applied state must stand")
			assert.Equal(t, tc.wantPlan, written.Plan)

			// A fresh event ID means the claim gate cannot catch it — the
			// ordering guard is what has to, and it reports the event as
			// processed so FastSpring stops redelivering it.
			require.Len(t, result.Skipped, 1)
			assert.Equal(t, tc.replay.id, result.Skipped[0].EventID)
			assert.ErrorIs(t, result.Skipped[0].Err, errEventSuperseded)
			assert.NotErrorIs(t, result.Skipped[0].Err, errEventDuplicate,
				"a fresh event ID is claimed, then rejected on ordering — not de-duplicated")
			assert.True(t, result.AllProcessed(), "a superseded replay is acknowledged, not retried forever")
		})
	}
}

func TestHandleWebhookSameChangedChargeAndUpdateAgree(t *testing.T) {
	// The realistic same-`changed` pair: a rebill emits both
	// subscription.charge.completed and subscription.updated for one change.
	// Both normalise to the same state, so applying only the first loses
	// nothing — which is exactly why the strict guard is safe.
	const changed = int64(1751500800000)

	apply := func(t *testing.T, bodies ...[]byte) model.Subscription {
		t.Helper()
		repo := newRecordingRepo(activeProSubscription())
		svc := newTestService(repo.MockSubscriptionRepository)
		svc.repo = repo

		for _, body := range bodies {
			result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))
			require.NoError(t, err)
			assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
		}
		assert.Len(t, repo.upserts, 1, "one change is one write, whichever event carried it")
		return repo.lastUpsert(t)
	}

	charge := chargeCompletedEvent(t, "evt-charge-completed-t0", changed)
	update := lifecycleEvent{id: "evt-updated-t0", kind: eventUpdated, state: "active", active: ptr(true), changed: changed}.body(t)

	chargeFirst := apply(t, charge, update)
	updateFirst := apply(t, update, charge)

	assert.Equal(t, StatusActive, chargeFirst.Status)
	assert.Equal(t, PlanPro, chargeFirst.Plan)
	assert.Equal(t, chargeFirst.Status, updateFirst.Status,
		"whichever of the pair lands first, the stored state is the same")
	assert.Equal(t, chargeFirst.Plan, updateFirst.Plan)
	require.NotNil(t, chargeFirst.LastEventAt)
	assert.Equal(t, *chargeFirst.LastEventAt, *updateFirst.LastEventAt)
}

func TestHandleWebhookPauseReportedAsInactiveKeepsThePurchase(t *testing.T) {
	// FastSpring reports a paused subscription as `active: false`. Treating that
	// flag as a cancellation would rewrite the row to the free plan, and the
	// resume would then have no purchase left to give back — so the explicit
	// pause/resume events outrank the flag.
	repo := newRecordingRepo(activeProSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	paused := lifecycleEvent{
		id: "evt-paused-inactive", kind: eventPaused, state: "paused", active: ptr(false), changed: 1751500800000,
	}.body(t)
	result, err := svc.HandleWebhook(context.Background(), paused, sign(t, paused, testWebhookSecret))

	require.NoError(t, err)
	assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
	pausedRow := repo.lastUpsert(t)
	assert.Equal(t, StatusPaused, pausedRow.Status, "an inactive flag on a pause is not a cancellation")
	assert.Equal(t, PlanPro, pausedRow.Plan, "the purchased plan must survive the pause")

	// Paid quotas still stop while billing is suspended.
	pausedRepo := &MockSubscriptionRepository{
		GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
			return copySubscription(&pausedRow), nil
		},
	}
	plan, err := newTestService(pausedRepo).effectivePlan(context.Background(), testUserID)
	require.NoError(t, err)
	assert.Equal(t, PlanFree, plan)

	// The resume arrives before the flag catches up, and must still restore access.
	resumed := lifecycleEvent{
		id: "evt-resumed-inactive", kind: eventResumed, state: "active", active: ptr(false), changed: 1751587200000,
	}.body(t)
	result, err = svc.HandleWebhook(context.Background(), resumed, sign(t, resumed, testWebhookSecret))

	require.NoError(t, err)
	assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
	resumedRow := repo.lastUpsert(t)
	assert.Equal(t, StatusActive, resumedRow.Status)
	assert.Equal(t, PlanPro, resumedRow.Plan, "a resume gives the purchase back")
}

func TestHandleWebhookDeactivationOutranksPauseAndResume(t *testing.T) {
	// Deactivation is first in the priority order: once the provider reports the
	// subscription as deactivated, no later pause or resume may revive it.
	deactivated := loadFixture(t, "subscription_deactivated.json")
	repo := newRecordingRepo(activeProSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	_, err := svc.HandleWebhook(context.Background(), deactivated, sign(t, deactivated, testWebhookSecret))
	require.NoError(t, err)
	require.Equal(t, StatusCancelled, repo.lastUpsert(t).Status)

	// A *newer* resume, so the ordering guard cannot be what stops it.
	resumed := lifecycleEvent{
		id: "evt-resumed-after-deactivation", kind: eventResumed, state: "deactivated", active: ptr(false),
		changed: 1799999999000,
	}.body(t)
	result, err := svc.HandleWebhook(context.Background(), resumed, sign(t, resumed, testWebhookSecret))

	require.NoError(t, err)
	assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
	written := repo.lastUpsert(t)
	assert.Equal(t, StatusCancelled, written.Status, "a deactivated subscription cannot be resumed into access")
	assert.Equal(t, PlanFree, written.Plan)
}

func TestHandleWebhookEventsWithoutAnID(t *testing.T) {
	// An event ID is what makes a delivery idempotent. Without one, an
	// actionable event is malformed and must be retried rather than applied
	// blind; and no event may ever be acknowledged as an empty ID, which
	// FastSpring would not recognise.
	t.Run("actionable event without an ID is retried, not applied", func(t *testing.T) {
		body := transformFirstEvent(t, loadFixture(t, "subscription_updated.json"), func(event map[string]any) {
			event["id"] = ""
		})
		repo := newRecordingRepo(activeProSubscription())
		svc := newTestService(repo.MockSubscriptionRepository)
		svc.repo = repo

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.Empty(t, repo.upserts, "an unidentifiable event must not touch entitlement")
		assert.Empty(t, repo.claims)
		require.Len(t, result.Failed, 1)
		assert.ErrorIs(t, result.Failed[0].Err, errEventMissingID)
		assert.Empty(t, result.Processed, "an empty ID must never be acknowledged")
		assert.False(t, result.AllProcessed())
	})

	t.Run("non-actionable event without an ID is acknowledged silently", func(t *testing.T) {
		body := []byte(`{"events":[
			{"id":"","live":false,"processed":false,"type":"order.completed","created":1751328000000,"data":{}},
			{"id":"evt-ok","live":false,"processed":false,"type":"account.updated","created":1751328000001,"data":{}}
		]}`)
		repo := newRecordingRepo(activeProSubscription())
		svc := newTestService(repo.MockSubscriptionRepository)
		svc.repo = repo

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.Equal(t, []string{"evt-ok"}, result.Processed,
			"an ID-less acknowledgement must not add an empty line to the 202 body")
		assert.True(t, result.AllProcessed())
		assert.Equal(t, 2, result.Total(), "both events were still handled")
	})

	t.Run("foreign-environment event without an ID is acknowledged silently", func(t *testing.T) {
		body := transformFirstEvent(t, loadFixture(t, "subscription_activated_live.json"), func(event map[string]any) {
			event["id"] = ""
		})
		repo := newRecordingRepo(linkedFreeSubscription())
		svc := newTestService(repo.MockSubscriptionRepository)
		svc.repo = repo

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.Empty(t, result.Processed)
		assert.Empty(t, repo.upserts)
		require.Len(t, result.Skipped, 1)
		assert.ErrorIs(t, result.Skipped[0].Err, ErrEnvironmentMismatch)
	})
}

func TestHandleWebhookDuplicateIsReportedAsSkipped(t *testing.T) {
	// A redelivery is acknowledged without a write, and the reason is surfaced
	// so the HTTP layer can log why nothing changed.
	body := loadFixture(t, "batch_duplicate_activated.json")
	repo := newRecordingRepo(linkedFreeSubscription())
	svc := newTestService(repo.MockSubscriptionRepository)
	svc.repo = repo

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	require.Len(t, result.Skipped, 1)
	assert.ErrorIs(t, result.Skipped[0].Err, errEventDuplicate)
	assert.Len(t, repo.upserts, 1)
}
