package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/creem"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	periodStartText = "2026-10-01T00:00:00Z"
	periodEndText   = "2026-11-01T00:00:00Z"
	updatedAtText   = "2026-10-05T12:00:00Z"
)

// subscriptionObject builds a Creem subscription object. Fields can be overridden
// or removed (nil value) per test.
func subscriptionObject(overrides map[string]any) map[string]any {
	object := map[string]any{
		"id":                        "sub_1",
		"object":                    "subscription",
		"mode":                      "test",
		"status":                    "active",
		"product":                   map[string]any{"id": testProProductID},
		"customer":                  map[string]any{"id": "cust_1"},
		"metadata":                  map[string]any{metadataUserIDKey: testUserID},
		"current_period_start_date": periodStartText,
		"current_period_end_date":   periodEndText,
		"updated_at":                updatedAtText,
	}
	for key, value := range overrides {
		if value == nil {
			delete(object, key)
			continue
		}
		object[key] = value
	}
	return object
}

// signed encodes an event and returns the body with its valid signature.
func signed(t *testing.T, eventID, eventType string, object map[string]any) ([]byte, string) {
	t.Helper()
	event := map[string]any{
		"id":         eventID,
		"eventType":  eventType,
		"created_at": time.Date(2026, 10, 5, 13, 0, 0, 0, time.UTC).UnixMilli(),
		"object":     object,
	}
	body, err := json.Marshal(event)
	require.NoError(t, err)
	return body, signBody(body, testWebhookSecret)
}

func signBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// ownedRepo returns a repository whose user row exists and records what
// ApplySubscriptionEvent was asked to write.
func ownedRepo(applied *[]*model.Subscription, outcome model.WebhookApplyOutcome) *MockSubscriptionRepository {
	return &MockSubscriptionRepository{
		GetByUserIDFunc: func(_ context.Context, userID string) (*model.Subscription, error) {
			return &model.Subscription{UserID: userID, Plan: PlanFree, Status: StatusFree}, nil
		},
		ApplySubscriptionEventFunc: func(_ context.Context, _, _ string, sub *model.Subscription) (model.WebhookApplyOutcome, error) {
			*applied = append(*applied, sub)
			return outcome, nil
		},
	}
}

func TestHandleWebhook_RejectsUntrustedDeliveries(t *testing.T) {
	body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))

	tests := []struct {
		name      string
		body      []byte
		signature string
		secret    string
		wantErr   error
	}{
		{name: "missing signature", body: body, signature: "", secret: testWebhookSecret, wantErr: creem.ErrSignatureMissing},
		{name: "wrong signature", body: body, signature: signBody(body, "another-secret"), secret: testWebhookSecret, wantErr: creem.ErrSignatureInvalid},
		{name: "signature that is not hex", body: body, signature: "not-hex!", secret: testWebhookSecret, wantErr: creem.ErrSignatureInvalid},
		{name: "signature of a different body", body: append([]byte(" "), body...), signature: signature, secret: testWebhookSecret, wantErr: creem.ErrSignatureInvalid},
		{name: "no secret configured", body: body, signature: signature, secret: "", wantErr: creem.ErrSecretMissing},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{
				GetByUserIDFunc: func(context.Context, string) (*model.Subscription, error) {
					t.Fatal("an untrusted delivery must not reach the database")
					return nil, nil
				},
			}
			cfg := testBillingConfig()
			cfg.WebhookSecret = tt.secret
			svc := NewSubscriptionService(repo, creem.NewClient(creem.Config{}), cfg)

			_, err := svc.HandleWebhook(context.Background(), tt.body, tt.signature)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.NotErrorIs(t, err, ErrEventFailed, "a rejection is not a retryable failure")
		})
	}

	t.Run("a signed body that is not JSON is rejected", func(t *testing.T) {
		garbage := []byte("not json")
		svc := newTestService(&MockSubscriptionRepository{})

		_, err := svc.HandleWebhook(context.Background(), garbage, signBody(garbage, testWebhookSecret))

		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrEventFailed)
	})
}

func TestHandleWebhook_MapsSubscriptionState(t *testing.T) {
	periodStart := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		eventType    string
		object       map[string]any
		wantStatus   string
		wantPlan     string
		wantPeriod   bool
		wantCancelAt *time.Time
	}{
		{
			name: "a paid subscription grants the purchased plan", eventType: creem.EventSubscriptionPaid,
			object: subscriptionObject(nil), wantStatus: StatusActive, wantPlan: PlanPro, wantPeriod: true,
		},
		{
			name: "the enterprise product grants enterprise", eventType: creem.EventSubscriptionPaid,
			object:     subscriptionObject(map[string]any{"product": map[string]any{"id": testEnterpriseProductID}}),
			wantStatus: StatusActive, wantPlan: PlanEnterprise, wantPeriod: true,
		},
		{
			name: "a product given as a bare ID is understood", eventType: creem.EventSubscriptionActive,
			object:     subscriptionObject(map[string]any{"product": testProProductID}),
			wantStatus: StatusActive, wantPlan: PlanPro, wantPeriod: true,
		},
		{
			name: "a trial is active", eventType: creem.EventSubscriptionTrialing,
			object:     subscriptionObject(map[string]any{"status": "trialing"}),
			wantStatus: StatusActive, wantPlan: PlanPro, wantPeriod: true,
		},
		{
			name: "a scheduled cancellation keeps access until the period ends", eventType: creem.EventSubscriptionScheduledCancel,
			object:     subscriptionObject(map[string]any{"status": "scheduled_cancel"}),
			wantStatus: StatusActive, wantPlan: PlanPro, wantPeriod: true, wantCancelAt: &periodEnd,
		},
		{
			name: "a failed payment is a dunning period, not a cancellation", eventType: creem.EventSubscriptionPastDue,
			object:     subscriptionObject(map[string]any{"status": "past_due"}),
			wantStatus: StatusPastDue, wantPlan: PlanPro, wantPeriod: true,
		},
		{
			name: "a pause keeps the purchase on the row", eventType: creem.EventSubscriptionPaused,
			object:     subscriptionObject(map[string]any{"status": "paused"}),
			wantStatus: StatusPaused, wantPlan: PlanPro, wantPeriod: true,
		},
		{
			name: "an unpaid subscription stops paid quotas but keeps the purchase", eventType: creem.EventSubscriptionUnpaid,
			object:     subscriptionObject(map[string]any{"status": "unpaid"}),
			wantStatus: StatusPaused, wantPlan: PlanPro, wantPeriod: true,
		},
		{
			name: "a cancelled subscription drops to free with no period", eventType: creem.EventSubscriptionCanceled,
			object:     subscriptionObject(map[string]any{"status": "canceled"}),
			wantStatus: StatusCancelled, wantPlan: PlanFree, wantPeriod: false,
		},
		{
			name: "cancelling never consults the product, so a catalog mistake cannot keep access alive", eventType: creem.EventSubscriptionCanceled,
			object:     subscriptionObject(map[string]any{"status": "canceled", "product": map[string]any{"id": "prod_unknown"}}),
			wantStatus: StatusCancelled, wantPlan: PlanFree, wantPeriod: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var applied []*model.Subscription
			svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

			body, signature := signed(t, "evt_1", tt.eventType, tt.object)
			result, err := svc.HandleWebhook(context.Background(), body, signature)

			require.NoError(t, err)
			assert.NoError(t, result.Skipped)
			require.Len(t, applied, 1)
			got := applied[0]
			assert.Equal(t, testUserID, got.UserID)
			assert.Equal(t, tt.wantStatus, got.Status)
			assert.Equal(t, tt.wantPlan, got.Plan)
			require.NotNil(t, got.ExternalSubscriptionID)
			assert.Equal(t, "sub_1", *got.ExternalSubscriptionID)
			require.NotNil(t, got.ExternalAccountID)
			assert.Equal(t, "cust_1", *got.ExternalAccountID)
			if tt.wantPeriod {
				require.NotNil(t, got.CurrentPeriodStart)
				require.NotNil(t, got.CurrentPeriodEnd)
				assert.True(t, got.CurrentPeriodStart.Equal(periodStart))
				assert.True(t, got.CurrentPeriodEnd.Equal(periodEnd))
			} else {
				assert.Nil(t, got.CurrentPeriodStart)
				assert.Nil(t, got.CurrentPeriodEnd)
			}
			if tt.wantCancelAt != nil {
				require.NotNil(t, got.CancelAt)
				assert.True(t, got.CancelAt.Equal(*tt.wantCancelAt))
			} else {
				assert.Nil(t, got.CancelAt, "only a scheduled cancellation carries cancel_at")
			}
		})
	}
}

func TestHandleWebhook_NeverGrantsAnUnknownProduct(t *testing.T) {
	var applied []*model.Subscription
	svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

	body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid,
		subscriptionObject(map[string]any{"product": map[string]any{"id": "prod_unknown"}}))
	_, err := svc.HandleWebhook(context.Background(), body, signature)

	require.ErrorIs(t, err, ErrEventFailed, "retried, so a corrected catalog can still grant the purchase")
	assert.Empty(t, applied)
}

func TestHandleWebhook_RejectsAnUnrecognisedStatus(t *testing.T) {
	var applied []*model.Subscription
	svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

	body, signature := signed(t, "evt_1", creem.EventSubscriptionUpdate,
		subscriptionObject(map[string]any{"status": "mystery"}))
	_, err := svc.HandleWebhook(context.Background(), body, signature)

	require.ErrorIs(t, err, ErrEventFailed)
	assert.Empty(t, applied)
}

func TestHandleWebhook_OrdersEventsByTheObjectsOwnTimestamp(t *testing.T) {
	t.Run("uses the object's updated_at, not the envelope", func(t *testing.T) {
		var applied []*model.Subscription
		svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		require.Len(t, applied, 1)
		require.NotNil(t, applied[0].LastEventAt)
		assert.True(t, applied[0].LastEventAt.Equal(time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)))
	})

	t.Run("falls back to the envelope when the object has no timestamp", func(t *testing.T) {
		var applied []*model.Subscription
		svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid,
			subscriptionObject(map[string]any{"updated_at": nil}))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		require.Len(t, applied, 1)
		require.NotNil(t, applied[0].LastEventAt)
		assert.True(t, applied[0].LastEventAt.Equal(time.Date(2026, 10, 5, 13, 0, 0, 0, time.UTC)))
	})

	t.Run("accepts epoch milliseconds as the timestamp", func(t *testing.T) {
		var applied []*model.Subscription
		svc := newTestService(ownedRepo(&applied, model.WebhookApplied))
		millis := time.Date(2026, 10, 5, 12, 30, 0, 0, time.UTC).UnixMilli()

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid,
			subscriptionObject(map[string]any{"updated_at": millis}))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		require.Len(t, applied, 1)
		assert.True(t, applied[0].LastEventAt.Equal(time.UnixMilli(millis).UTC()))
	})
}

func TestHandleWebhook_ResolvesTheOwner(t *testing.T) {
	linked := &model.Subscription{UserID: "user-by-link", Plan: PlanPro, Status: StatusActive}

	t.Run("the subscription ID already on a row wins over the metadata", func(t *testing.T) {
		var applied []*model.Subscription
		repo := ownedRepo(&applied, model.WebhookApplied)
		repo.GetByExternalSubscriptionIDFunc = func(_ context.Context, id string) (*model.Subscription, error) {
			assert.Equal(t, "sub_1", id)
			return linked, nil
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		require.Len(t, applied, 1)
		assert.Equal(t, "user-by-link", applied[0].UserID)
	})

	t.Run("the checkout metadata resolves a first purchase", func(t *testing.T) {
		var applied []*model.Subscription
		svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		require.Len(t, applied, 1)
		assert.Equal(t, testUserID, applied[0].UserID)
	})

	t.Run("the customer on a row resolves an event that arrives without metadata", func(t *testing.T) {
		var applied []*model.Subscription
		repo := ownedRepo(&applied, model.WebhookApplied)
		repo.GetByExternalAccountIDFunc = func(_ context.Context, id string) (*model.Subscription, error) {
			assert.Equal(t, "cust_1", id)
			return linked, nil
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid,
			subscriptionObject(map[string]any{"metadata": nil}))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		require.Len(t, applied, 1)
		assert.Equal(t, "user-by-link", applied[0].UserID)
	})

	t.Run("metadata that is not a UUID is ignored, never queried", func(t *testing.T) {
		var applied []*model.Subscription
		repo := ownedRepo(&applied, model.WebhookApplied)
		repo.GetByUserIDFunc = func(context.Context, string) (*model.Subscription, error) {
			t.Fatal("a malformed user ID must not reach the database")
			return nil, nil
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid,
			subscriptionObject(map[string]any{"metadata": map[string]any{metadataUserIDKey: "' OR 1=1 --"}}))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.ErrorIs(t, err, ErrEventFailed, "our own product that cannot be placed stays retryable")
		assert.Empty(t, applied)
	})

	t.Run("our product that cannot be placed is retried, not dropped", func(t *testing.T) {
		var applied []*model.Subscription
		svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid,
			subscriptionObject(map[string]any{"metadata": nil}))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.ErrorIs(t, err, ErrEventFailed)
		require.ErrorIs(t, err, model.ErrSubscriptionNotFound)
		assert.Empty(t, applied)
	})

	t.Run("another product on the same account is acknowledged and ignored", func(t *testing.T) {
		var applied []*model.Subscription
		svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid,
			subscriptionObject(map[string]any{"metadata": nil, "product": map[string]any{"id": "prod_other_app"}}))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.ErrorIs(t, result.Skipped, errForeignBillingEvent)
		assert.Empty(t, applied)
	})
}

func TestHandleWebhook_EnvironmentGuard(t *testing.T) {
	t.Run("a live event is refused by a test deployment", func(t *testing.T) {
		var applied []*model.Subscription
		svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid,
			subscriptionObject(map[string]any{"mode": "prod"}))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err, "acknowledged: a retry could never make it processable")
		assert.ErrorIs(t, result.Skipped, ErrEnvironmentMismatch)
		assert.Empty(t, applied)
	})

	t.Run("a test event is refused by a live deployment", func(t *testing.T) {
		var applied []*model.Subscription
		cfg := testBillingConfig()
		cfg.Environment = EnvironmentLive
		svc := NewSubscriptionService(ownedRepo(&applied, model.WebhookApplied), creem.NewClient(creem.Config{}), cfg)

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.ErrorIs(t, result.Skipped, ErrEnvironmentMismatch)
		assert.Empty(t, applied)
	})

	t.Run("the sandbox counts as test", func(t *testing.T) {
		var applied []*model.Subscription
		svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid,
			subscriptionObject(map[string]any{"mode": "sandbox"}))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.Len(t, applied, 1)
	})

	t.Run("an object that states no mode cannot be placed and is let through", func(t *testing.T) {
		var applied []*model.Subscription
		svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid,
			subscriptionObject(map[string]any{"mode": nil}))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.Len(t, applied, 1)
	})
}

func TestHandleWebhook_AtomicWriteOutcomes(t *testing.T) {
	tests := []struct {
		name        string
		outcome     model.WebhookApplyOutcome
		wantSkipped error
	}{
		{name: "applied", outcome: model.WebhookApplied},
		{name: "a redelivery is acknowledged", outcome: model.WebhookDuplicate, wantSkipped: errEventDuplicate},
		{name: "a stale event is acknowledged", outcome: model.WebhookSuperseded, wantSkipped: errEventSuperseded},
		{name: "a customer shared by two users is acknowledged and flagged", outcome: model.WebhookAccountConflict, wantSkipped: model.ErrBillingAccountTaken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var applied []*model.Subscription
			svc := newTestService(ownedRepo(&applied, tt.outcome))

			body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))
			result, err := svc.HandleWebhook(context.Background(), body, signature)

			require.NoError(t, err)
			assert.Equal(t, "evt_1", result.EventID)
			if tt.wantSkipped == nil {
				assert.NoError(t, result.Skipped)
				return
			}
			assert.ErrorIs(t, result.Skipped, tt.wantSkipped)
		})
	}

	t.Run("a second live subscription is retried, not lost", func(t *testing.T) {
		// The usual cause is a delayed cancellation of the old subscription. A
		// redelivery after that lands applies the new one by itself; acknowledging
		// would lose a paying user's plan for good.
		var applied []*model.Subscription
		svc := newTestService(ownedRepo(&applied, model.WebhookLinkConflict))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.ErrorIs(t, err, ErrEventFailed)
		require.ErrorIs(t, err, ErrSubscriptionLinkConflict)
		assert.Equal(t, "evt_1", result.EventID)
	})

	t.Run("a database failure is retried", func(t *testing.T) {
		repo := ownedRepo(new([]*model.Subscription), model.WebhookApplied)
		repo.ApplySubscriptionEventFunc = func(context.Context, string, string, *model.Subscription) (model.WebhookApplyOutcome, error) {
			return "", errors.New("connection reset")
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.ErrorIs(t, err, ErrEventFailed)
		assert.Equal(t, "evt_1", result.EventID)
	})

	t.Run("an actionable event with no ID cannot be de-duplicated and is retried", func(t *testing.T) {
		var applied []*model.Subscription
		svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

		body, signature := signed(t, "", creem.EventSubscriptionPaid, subscriptionObject(nil))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.ErrorIs(t, err, ErrEventFailed)
		assert.Empty(t, applied)
	})
}

func TestHandleWebhook_EventsThatAreNotActedOn(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		wantSkip  error
	}{
		{name: "expiry is not the end while Creem is still retrying the charge", eventType: "subscription.expired", wantSkip: errEventNotActionable},
		{name: "credits are unrelated", eventType: "credits.granted", wantSkip: errEventNotActionable},
		{name: "a refund is surfaced for a person", eventType: creem.EventRefundCreated, wantSkip: ErrRefundNeedsReview},
		{name: "a dispute is surfaced for a person", eventType: creem.EventDisputeCreated, wantSkip: ErrRefundNeedsReview},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSubscriptionRepository{
				ApplySubscriptionEventFunc: func(context.Context, string, string, *model.Subscription) (model.WebhookApplyOutcome, error) {
					t.Fatal("an event Jobber does not act on must not write")
					return "", nil
				},
			}
			svc := newTestService(repo)

			body, signature := signed(t, "evt_1", tt.eventType, subscriptionObject(nil))
			result, err := svc.HandleWebhook(context.Background(), body, signature)

			require.NoError(t, err)
			assert.ErrorIs(t, result.Skipped, tt.wantSkip)
		})
	}
}

func completedCheckout(overrides map[string]any) map[string]any {
	object := map[string]any{
		"id":       "ch_1",
		"object":   "checkout",
		"mode":     "test",
		"product":  map[string]any{"id": testProProductID},
		"customer": map[string]any{"id": "cust_1"},
		"metadata": map[string]any{metadataUserIDKey: testUserID},
	}
	for key, value := range overrides {
		if value == nil {
			delete(object, key)
			continue
		}
		object[key] = value
	}
	return object
}

func TestHandleWebhook_CheckoutCompletedLinksTheCustomer(t *testing.T) {
	t.Run("records the customer against the user and grants nothing", func(t *testing.T) {
		var linkedUser, linkedCustomer string
		repo := &MockSubscriptionRepository{
			LinkExternalAccountFunc: func(_ context.Context, userID, customerID string) error {
				linkedUser, linkedCustomer = userID, customerID
				return nil
			},
			ApplySubscriptionEventFunc: func(context.Context, string, string, *model.Subscription) (model.WebhookApplyOutcome, error) {
				t.Fatal("a completed checkout must not grant a plan")
				return "", nil
			},
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_c1", creem.EventCheckoutCompleted, completedCheckout(nil))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.NoError(t, result.Skipped)
		assert.Equal(t, testUserID, linkedUser)
		assert.Equal(t, "cust_1", linkedCustomer)
	})

	t.Run("a purchase of another product is ignored", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			LinkExternalAccountFunc: func(context.Context, string, string) error {
				t.Fatal("another product's checkout must not link a Jobber user")
				return nil
			},
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_c1", creem.EventCheckoutCompleted,
			completedCheckout(map[string]any{"product": map[string]any{"id": "prod_other_app"}}))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.ErrorIs(t, result.Skipped, errForeignBillingEvent)
	})

	t.Run("a checkout with no Jobber metadata is ignored", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			LinkExternalAccountFunc: func(context.Context, string, string) error {
				t.Fatal("a checkout nobody at Jobber started must not link anyone")
				return nil
			},
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_c1", creem.EventCheckoutCompleted,
			completedCheckout(map[string]any{"metadata": nil}))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.ErrorIs(t, result.Skipped, errForeignBillingEvent)
	})

	t.Run("a customer already owned by someone else is acknowledged and flagged", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			LinkExternalAccountFunc: func(context.Context, string, string) error {
				return model.ErrBillingAccountTaken
			},
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_c1", creem.EventCheckoutCompleted, completedCheckout(nil))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.ErrorIs(t, result.Skipped, model.ErrBillingAccountTaken)
	})

	t.Run("a database failure is retried", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			LinkExternalAccountFunc: func(context.Context, string, string) error { return errors.New("db down") },
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_c1", creem.EventCheckoutCompleted, completedCheckout(nil))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.ErrorIs(t, err, ErrEventFailed)
	})

	t.Run("a live checkout is refused by a test deployment", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			LinkExternalAccountFunc: func(context.Context, string, string) error {
				t.Fatal("the other environment must not link anyone")
				return nil
			},
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_c1", creem.EventCheckoutCompleted,
			completedCheckout(map[string]any{"mode": "prod"}))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.ErrorIs(t, result.Skipped, ErrEnvironmentMismatch)
	})
}

func TestSkipReport(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		wantSeverity SkipSeverity
	}{
		{name: "environment mismatch loses the event", err: ErrEnvironmentMismatch, wantSeverity: SkipLostEvent},
		{name: "customer shared by two users loses a paid purchase", err: model.ErrBillingAccountTaken, wantSeverity: SkipLostEvent},
		{name: "refund needs a person", err: ErrRefundNeedsReview, wantSeverity: SkipNeedsReview},
		{name: "a deleted user's purchase loses the event", err: model.ErrUserNotFound, wantSeverity: SkipLostEvent},
		{name: "a second subscription needs a person", err: errSecondSubscription, wantSeverity: SkipNeedsReview},
		{name: "nothing to end is routine", err: errNothingToEnd, wantSeverity: SkipRoutine},
		{name: "wrapped sentinel is still recognised", err: errors.Join(errors.New("ctx"), model.ErrBillingAccountTaken), wantSeverity: SkipLostEvent},
		{name: "duplicate is routine", err: errEventDuplicate, wantSeverity: SkipRoutine},
		{name: "superseded is routine", err: errEventSuperseded, wantSeverity: SkipRoutine},
		{name: "foreign product is routine", err: errForeignBillingEvent, wantSeverity: SkipRoutine},
		{name: "not actionable is routine", err: errEventNotActionable, wantSeverity: SkipRoutine},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, severity := SkipReport(tt.err)

			assert.NotEmpty(t, message)
			assert.Equal(t, tt.wantSeverity, severity)
		})
	}
}

func TestHandleWebhook_AnotherProductNeverReachesALinkedUser(t *testing.T) {
	// The Creem account may sell other things. A customer already linked here who
	// buys one of them must not have that purchase applied (or retried for a day)
	// as a Jobber subscription — through the customer hop or any other.
	linked := &model.Subscription{UserID: "user-by-link", Plan: PlanPro, Status: StatusActive}
	foreign := map[string]any{
		"metadata": nil,
		"product":  map[string]any{"id": "prod_other_app"},
	}

	for _, status := range []string{"active", "canceled"} {
		t.Run("status "+status, func(t *testing.T) {
			var applied []*model.Subscription
			repo := ownedRepo(&applied, model.WebhookApplied)
			repo.GetByExternalAccountIDFunc = func(context.Context, string) (*model.Subscription, error) {
				return linked, nil
			}
			svc := newTestService(repo)

			object := subscriptionObject(foreign)
			object["status"] = status
			body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, object)
			result, err := svc.HandleWebhook(context.Background(), body, signature)

			require.NoError(t, err, "acknowledged: no retry makes it ours")
			assert.ErrorIs(t, result.Skipped, errForeignBillingEvent)
			assert.Empty(t, applied, "nothing may be written into the linked user's row")
		})
	}
}

func TestHandleWebhook_OurCheckoutSurvivesAReconfiguredCatalog(t *testing.T) {
	// The metadata is written by our server, so an event carrying it is ours even
	// when the product ID is no longer configured. It must fail and be retried —
	// after the catalog is fixed the purchase still lands — and never be dropped.
	var applied []*model.Subscription
	svc := newTestService(ownedRepo(&applied, model.WebhookApplied))

	body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid,
		subscriptionObject(map[string]any{"product": map[string]any{"id": "prod_retired"}}))
	_, err := svc.HandleWebhook(context.Background(), body, signature)

	require.ErrorIs(t, err, ErrEventFailed)
	assert.Empty(t, applied)
}

func TestHandleWebhook_CancellationOfALinkedSubscriptionNeedsNoKnownProduct(t *testing.T) {
	// Ending access must work even if the product was retired from the catalog.
	var applied []*model.Subscription
	repo := ownedRepo(&applied, model.WebhookApplied)
	repo.GetByExternalSubscriptionIDFunc = func(context.Context, string) (*model.Subscription, error) {
		return &model.Subscription{UserID: testUserID, Plan: PlanPro, Status: StatusActive}, nil
	}
	svc := newTestService(repo)

	body, signature := signed(t, "evt_1", creem.EventSubscriptionCanceled,
		subscriptionObject(map[string]any{"status": "canceled", "metadata": nil, "product": map[string]any{"id": "prod_retired"}}))
	_, err := svc.HandleWebhook(context.Background(), body, signature)

	require.NoError(t, err)
	require.Len(t, applied, 1)
	assert.Equal(t, StatusCancelled, applied[0].Status)
	assert.Equal(t, PlanFree, applied[0].Plan)
}

func TestHandleWebhook_AnEventThatOmitsFieldsKeepsTheStoredOnes(t *testing.T) {
	storedStart := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	storedEnd := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC) // ahead of the events' updated_at (10-05)
	current := &model.Subscription{
		UserID: testUserID, Plan: PlanPro, Status: StatusActive,
		CurrentPeriodStart: &storedStart, CurrentPeriodEnd: &storedEnd,
		ExternalAccountID: strPtr("cust_stored"),
	}
	bare := func(overrides map[string]any) map[string]any {
		base := map[string]any{"current_period_start_date": nil, "current_period_end_date": nil, "customer": nil}
		for k, v := range overrides {
			base[k] = v
		}
		return base
	}
	appliedFor := func(t *testing.T, row *model.Subscription, eventType string, object map[string]any) *model.Subscription {
		t.Helper()
		var applied []*model.Subscription
		repo := ownedRepo(&applied, model.WebhookApplied)
		repo.GetByUserIDFunc = func(context.Context, string) (*model.Subscription, error) { return row, nil }

		body, signature := signed(t, "evt_1", eventType, subscriptionObject(object))
		_, err := newTestService(repo).HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		require.Len(t, applied, 1)
		return applied[0]
	}

	t.Run("a current stored period survives a partial payload", func(t *testing.T) {
		got := appliedFor(t, current, creem.EventSubscriptionUpdate, bare(nil))

		assert.True(t, got.CurrentPeriodStart.Equal(storedStart), "the renewal date must not be erased")
		assert.True(t, got.CurrentPeriodEnd.Equal(storedEnd))
	})

	t.Run("a customer the event does not carry is left to the write to keep", func(t *testing.T) {
		// Copying the value read a moment ago could undo a customer a concurrent
		// event just set; nil lets the SQL COALESCE keep whatever is stored.
		got := appliedFor(t, current, creem.EventSubscriptionUpdate, bare(nil))

		assert.Nil(t, got.ExternalAccountID)
	})

	t.Run("a customer the event carries is written", func(t *testing.T) {
		got := appliedFor(t, current, creem.EventSubscriptionUpdate,
			bare(map[string]any{"customer": map[string]any{"id": "cust_new"}}))

		require.NotNil(t, got.ExternalAccountID)
		assert.Equal(t, "cust_new", *got.ExternalAccountID)
	})

	t.Run("a scheduled cancellation still gets its end date", func(t *testing.T) {
		got := appliedFor(t, current, creem.EventSubscriptionScheduledCancel,
			bare(map[string]any{"status": "scheduled_cancel"}))

		require.NotNil(t, got.CancelAt, "an access end the UI cannot show is worse than a late one")
		assert.True(t, got.CancelAt.Equal(storedEnd))
	})

	t.Run("a stale stored period is not carried onto a later cancellation", func(t *testing.T) {
		// Renewals that carried no dates never advanced the stored end. Using it
		// would make cancel_at a date in the past and downgrade a paying user as
		// soon as the grace window ran out.
		staleEnd := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
		stale := *current
		stale.CurrentPeriodEnd = &staleEnd

		got := appliedFor(t, &stale, creem.EventSubscriptionScheduledCancel,
			bare(map[string]any{"status": "scheduled_cancel"}))

		assert.Nil(t, got.CancelAt)
		assert.Nil(t, got.CurrentPeriodEnd)
		assert.Equal(t, StatusActive, got.Status)
	})

	t.Run("a cancellation still clears the period", func(t *testing.T) {
		got := appliedFor(t, current, creem.EventSubscriptionCanceled, bare(map[string]any{"status": "canceled"}))

		assert.Nil(t, got.CurrentPeriodEnd)
		assert.Nil(t, got.CancelAt)
	})
}

func TestHandleWebhook_RefusedReplacementOfALiveSubscription(t *testing.T) {
	storedAt := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	liveRow := func() *model.Subscription {
		return &model.Subscription{
			UserID: testUserID, Plan: PlanPro, Status: StatusActive,
			ExternalSubscriptionID: strPtr("sub_live"), LastEventAt: &storedAt,
		}
	}
	conflictRepo := func(row *model.Subscription, outcome model.WebhookApplyOutcome) *MockSubscriptionRepository {
		var applied []*model.Subscription
		repo := ownedRepo(&applied, outcome)
		repo.GetByUserIDFunc = func(context.Context, string) (*model.Subscription, error) { return row, nil }
		return repo
	}
	at := func(updatedAt time.Time) map[string]any {
		return map[string]any{"id": "sub_new", "updated_at": updatedAt.Format(time.RFC3339)}
	}

	t.Run("a newer paid event is retried: the old subscription's cancel may just be late", func(t *testing.T) {
		svc := newTestService(conflictRepo(liveRow(), model.WebhookLinkConflict))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(at(storedAt.Add(time.Hour))))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.ErrorIs(t, err, ErrEventFailed)
		require.ErrorIs(t, err, ErrSubscriptionLinkConflict)
	})

	t.Run("the cancellation of a subscription the row does not track is acknowledged, not retried", func(t *testing.T) {
		// A replay of an old subscription's cancel, or the cancel of a second
		// subscription that was never linked: nothing here to end, and a redelivery
		// could only repeat the same answer for 24 hours.
		svc := newTestService(conflictRepo(liveRow(), model.WebhookLinkConflict))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionCanceled,
			subscriptionObject(map[string]any{"id": "sub_old", "status": "canceled", "updated_at": storedAt.Add(time.Hour).Format(time.RFC3339)}))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.ErrorIs(t, result.Skipped, errNothingToEnd)
	})

	t.Run("a paid event that is not newer than the row is acknowledged and flagged", func(t *testing.T) {
		// The older state can never win, so a redelivery cannot help; it is also
		// what a user paying for two subscriptions looks like.
		svc := newTestService(conflictRepo(liveRow(), model.WebhookLinkConflict))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(at(storedAt.Add(-time.Hour))))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.ErrorIs(t, result.Skipped, errSecondSubscription)
		message, severity := SkipReport(result.Skipped)
		assert.NotEmpty(t, message)
		assert.Equal(t, SkipNeedsReview, severity)
	})

	t.Run("a replacement that lost the ordering to the old subscription's cancel is flagged", func(t *testing.T) {
		// S1 was cancelled at T3 and S2's events (T2 < T3) were redelivered after
		// it. Ordering across two subscriptions is meaningless, but the purchase is
		// not applied, so it must not hide among the routine replays.
		row := liveRow()
		row.Status = StatusCancelled
		svc := newTestService(conflictRepo(row, model.WebhookSuperseded))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(at(storedAt.Add(-time.Hour))))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.ErrorIs(t, result.Skipped, errSecondSubscription)
		assert.ErrorIs(t, result.Skipped, errEventSuperseded)
	})

	t.Run("a routine replay of the linked subscription stays routine", func(t *testing.T) {
		row := liveRow()
		row.ExternalSubscriptionID = strPtr("sub_1")
		svc := newTestService(conflictRepo(row, model.WebhookSuperseded))

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.ErrorIs(t, result.Skipped, errEventSuperseded)
		assert.NotErrorIs(t, result.Skipped, errSecondSubscription)
	})
}

func TestHandleWebhook_AUserWithNoRowIsStillGranted(t *testing.T) {
	t.Run("the free row is created and the purchase lands", func(t *testing.T) {
		var applied []*model.Subscription
		var ensured string
		reads := 0
		repo := ownedRepo(&applied, model.WebhookApplied)
		repo.GetByUserIDFunc = func(_ context.Context, userID string) (*model.Subscription, error) {
			reads++
			if ensured == "" {
				return nil, model.ErrSubscriptionNotFound
			}
			return &model.Subscription{UserID: userID, Plan: PlanFree, Status: StatusFree}, nil
		}
		repo.EnsureFreeFunc = func(_ context.Context, userID string) error {
			ensured = userID
			return nil
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err, "a paying customer must not be retried into the ground over a missing row")
		assert.Equal(t, testUserID, ensured)
		assert.Equal(t, 2, reads)
		require.Len(t, applied, 1)
		assert.Equal(t, PlanPro, applied[0].Plan)
	})

	t.Run("a user who no longer exists is acknowledged and flagged", func(t *testing.T) {
		var applied []*model.Subscription
		repo := ownedRepo(&applied, model.WebhookApplied)
		repo.GetByUserIDFunc = func(context.Context, string) (*model.Subscription, error) {
			return nil, model.ErrSubscriptionNotFound
		}
		repo.EnsureFreeFunc = func(context.Context, string) error { return model.ErrUserNotFound }
		svc := newTestService(repo)

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err, "no retry brings a deleted account back")
		assert.ErrorIs(t, result.Skipped, model.ErrUserNotFound)
		assert.Empty(t, applied)
		_, severity := SkipReport(result.Skipped)
		assert.Equal(t, SkipLostEvent, severity)
	})

	t.Run("a checkout for a deleted user is acknowledged too", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			LinkExternalAccountFunc: func(context.Context, string, string) error { return model.ErrUserNotFound },
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_c1", creem.EventCheckoutCompleted, completedCheckout(nil))
		result, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.ErrorIs(t, result.Skipped, model.ErrUserNotFound)
	})

	t.Run("a failure creating the row is retried", func(t *testing.T) {
		repo := &MockSubscriptionRepository{
			EnsureFreeFunc: func(context.Context, string) error { return errors.New("db down") },
		}
		svc := newTestService(repo)

		body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, subscriptionObject(nil))
		_, err := svc.HandleWebhook(context.Background(), body, signature)

		require.ErrorIs(t, err, ErrEventFailed)
	})
}

func TestHandleWebhook_CheckoutCompletedWithoutACustomerIsAcknowledged(t *testing.T) {
	// There is nothing to link, and no retry will supply it.
	repo := &MockSubscriptionRepository{
		LinkExternalAccountFunc: func(context.Context, string, string) error {
			t.Fatal("no customer, nothing to link")
			return nil
		},
	}
	svc := newTestService(repo)

	body, signature := signed(t, "evt_c1", creem.EventCheckoutCompleted,
		completedCheckout(map[string]any{"customer": nil}))
	result, err := svc.HandleWebhook(context.Background(), body, signature)

	require.NoError(t, err)
	assert.ErrorIs(t, result.Skipped, errEventNotActionable)
}

func TestHandleWebhook_AMalformedObjectIsRejectedForRetryWithoutWriting(t *testing.T) {
	body, signature := signed(t, "evt_1", creem.EventSubscriptionPaid, nil)
	svc := newTestService(&MockSubscriptionRepository{
		ApplySubscriptionEventFunc: func(context.Context, string, string, *model.Subscription) (model.WebhookApplyOutcome, error) {
			t.Fatal("a payload with no subscription must not write")
			return "", nil
		},
	})

	_, err := svc.HandleWebhook(context.Background(), body, signature)

	require.ErrorIs(t, err, ErrEventFailed)
}
