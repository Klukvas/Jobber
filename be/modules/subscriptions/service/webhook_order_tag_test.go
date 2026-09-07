package service

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Identifiers from the recorded first-purchase fixture. They follow the shape a
// real test-mode order produced: the account arrives as a bare ID string, the
// account itself is brand new, and the order carries the user ID as a tag —
// alongside the proof Jobber's server minted for it at session creation.
const (
	taggedFixture         = "subscription_activated_new_account_tagged.json"
	taggedEventID         = "evt-activated-new-account-0001"
	taggedAccountID       = "0_8wJT4fTyC_WpROHjJYQw"
	taggedSubscriptionID  = "SuBsCr1PT10nJobberEnt01"
	foreignProductPath    = "fluxradar-basic"
	otherProviderSubscrID = "SuBsCr1PT10nSomeoneElse"
	// attackerUserID is a second real Jobber user: someone who can start their
	// own checkout, and therefore learn their own valid proof.
	attackerUserID = "11111111-2222-4333-8444-555555555555"
)

// newTaggedService wires the service to an account API answering exactly what
// this store answers for an account FastSpring created during checkout: a
// lookup carrying only `global`, with no `custom` key at all. The counter shows
// whether resolution needed that call.
func newTaggedService(t *testing.T, repo *recordingRepo) (*SubscriptionService, *atomic.Int64) {
	t.Helper()
	var calls atomic.Int64
	svc := newAccountAPIService(t, repo, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		require.Equal(t, "/accounts/"+taggedAccountID, r.URL.Path)
		_, _ = fmt.Fprintf(w, `{"id":%q,"account":%q,"lookup":{"global":"gL0balLookupKeyTest01"}}`,
			taggedAccountID, taggedAccountID)
	})
	return svc, &calls
}

// answerUserFromCurrentRow makes GetByUserID read the same row the recording
// repo mutates, so a tag-resolved write is judged against the state that
// actually stands rather than a frozen copy.
func answerUserFromCurrentRow(repo *recordingRepo, userID string) {
	repo.GetByUserIDFunc = func(_ context.Context, id string) (*model.Subscription, error) {
		repo.mu.Lock()
		defer repo.mu.Unlock()
		if id != userID || repo.current == nil {
			return nil, model.ErrSubscriptionNotFound
		}
		return copySubscription(repo.current), nil
	}
}

// retaggedBody replaces the fixture order's tags, standing in for an order
// tagged by someone other than Jobber's own session-creation call.
func retaggedBody(t *testing.T, tags map[string]any) []byte {
	t.Helper()
	return transformFirstEvent(t, loadFixture(t, taggedFixture), func(event map[string]any) {
		data, ok := event["data"].(map[string]any)
		require.True(t, ok)
		if tags == nil {
			delete(data, "tags")
			return
		}
		data["tags"] = tags
	})
}

// freeRowForAnyUser answers every user with a free row, so nothing but the tag
// checks can decide the outcome of a test.
func freeRowForAnyUser(repo *recordingRepo) {
	repo.GetByUserIDFunc = func(_ context.Context, userID string) (*model.Subscription, error) {
		return &model.Subscription{ID: "sub-row-1", UserID: userID, Status: StatusFree, Plan: PlanFree}, nil
	}
}

func TestTaggedFixtureCarriesTheProofThisServerMints(t *testing.T) {
	// The fixture's proof is a literal, so it cannot silently drift with the
	// implementation: if the domain, the version or the key changes, this fails
	// and every tag test below is re-examined on purpose.
	body := loadFixture(t, taggedFixture)
	var got string
	transformFirstEvent(t, body, func(event map[string]any) {
		tags, ok := event["data"].(map[string]any)["tags"].(map[string]any)
		require.True(t, ok, "the fixture must carry order tags")
		got, ok = tags[userProofTagKey].(string)
		require.True(t, ok, "the fixture must carry the proof tag")
	})

	want, ok := orderTagProof(testWebhookSecret, testUserID)

	require.True(t, ok)
	assert.Equal(t, want, got)
}

func TestHandleWebhookResolvesFirstPurchaseByOrderTag(t *testing.T) {
	// The production failure this covers: FastSpring creates the buyer's account
	// *during* checkout, so the session answered with no `customer.accountId` to
	// record and the account it made carries no `lookup.custom`. Both local IDs
	// miss and the read-back has nothing to say — only the order tag, written
	// server-to-server with a proof no storefront visitor can mint, names the
	// buyer.
	body := loadFixture(t, taggedFixture)

	t.Run("the buyer is resolved and both provider IDs are recorded", func(t *testing.T) {
		repo := newRecordingRepo(nil)
		repo.GetByUserIDFunc = func(_ context.Context, userID string) (*model.Subscription, error) {
			if userID != testUserID {
				return nil, model.ErrSubscriptionNotFound
			}
			return &model.Subscription{ID: "sub-row-1", UserID: testUserID, Status: StatusFree, Plan: PlanFree}, nil
		}
		svc, accountCalls := newTaggedService(t, repo)

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
		assert.Contains(t, result.Processed, taggedEventID)

		written := repo.lastUpsert(t)
		assert.Equal(t, testUserID, written.UserID)
		assert.Equal(t, StatusActive, written.Status)
		assert.Equal(t, PlanEnterprise, written.Plan)
		require.NotNil(t, written.ExternalSubscriptionID)
		assert.Equal(t, taggedSubscriptionID, *written.ExternalSubscriptionID)
		require.NotNil(t, written.ExternalAccountID)
		assert.Equal(t, taggedAccountID, *written.ExternalAccountID,
			"the account must be stored so every later event resolves without a tag")
		assert.Zero(t, accountCalls.Load(),
			"the proven tag already names the buyer, so no account read-back is needed")
	})

	t.Run("resolution survives an account API that is down", func(t *testing.T) {
		// The tag is read off the payload, so a grant no longer depends on a
		// second endpoint being reachable at the moment the purchase lands.
		repo := newRecordingRepo(nil)
		freeRowForAnyUser(repo)
		svc := newAccountAPIService(t, repo, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
		assert.Equal(t, PlanEnterprise, repo.lastUpsert(t).Plan)
	})

	t.Run("the redelivery FastSpring sends is acknowledged, not applied twice", func(t *testing.T) {
		// The observed order was delivered twice, both times answered 200. The
		// second delivery must claim nothing new.
		repo := newRecordingRepo(nil)
		answerUserFromCurrentRow(repo, testUserID)
		repo.current = &model.Subscription{ID: "sub-row-1", UserID: testUserID, Status: StatusFree, Plan: PlanFree}
		svc, _ := newTaggedService(t, repo)
		signature := sign(t, body, testWebhookSecret)

		first, err := svc.HandleWebhook(context.Background(), body, signature)
		require.NoError(t, err)
		require.True(t, first.AllProcessed(), "failures: %v", first.Failed)

		second, err := svc.HandleWebhook(context.Background(), body, signature)

		require.NoError(t, err)
		assert.True(t, second.AllProcessed(), "a redelivery must be acknowledged, not retried")
		assert.Len(t, repo.upserts, 1, "the second delivery must not write again")
		require.Len(t, second.Skipped, 1)
		assert.ErrorIs(t, second.Skipped[0].Err, errEventDuplicate)
	})

	t.Run("a delayed redelivery still verifies: the proof does not expire", func(t *testing.T) {
		// FastSpring retries for hours. A proof carries no timestamp precisely so
		// that a late delivery of a legitimate purchase still resolves.
		repo := newRecordingRepo(nil)
		freeRowForAnyUser(repo)
		svc, _ := newTaggedService(t, repo)
		resend := transformFirstEvent(t, body, func(event map[string]any) {
			event["id"] = "evt-activated-new-account-resend"
			// A manual resend arrives in a fresh envelope, days later.
			event["created"] = float64(1757246400000 + 3*24*60*60*1000)
		})

		result, err := svc.HandleWebhook(context.Background(), resend, sign(t, resend, testWebhookSecret))

		require.NoError(t, err)
		assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
		assert.Equal(t, PlanEnterprise, repo.lastUpsert(t).Plan)
	})

	t.Run("a repurchase after cancellation resolves through the tag again", func(t *testing.T) {
		// The provider ended the old subscription, so the row holds nothing worth
		// protecting and a fresh checkout is the only way back to a paid plan.
		cancelled := &model.Subscription{
			ID:                     "sub-row-1",
			UserID:                 testUserID,
			ExternalSubscriptionID: ptr(otherProviderSubscrID),
			ExternalAccountID:      ptr(fixtureAccountID),
			Status:                 StatusCancelled,
			Plan:                   PlanFree,
		}
		repo := newRecordingRepo(cancelled)
		answerUserFromCurrentRow(repo, testUserID)
		svc, _ := newTaggedService(t, repo)

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		assert.True(t, result.AllProcessed(), "failures: %v", result.Failed)
		written := repo.lastUpsert(t)
		assert.Equal(t, StatusActive, written.Status)
		assert.Equal(t, PlanEnterprise, written.Plan)
		require.NotNil(t, written.ExternalSubscriptionID)
		assert.Equal(t, taggedSubscriptionID, *written.ExternalSubscriptionID)
	})
}

func TestHandleWebhookRefusesAnOrderTagItCannotProve(t *testing.T) {
	// The attack this closes. FastSpring's Store Builder Library exposes
	// `fastspring.builder.tag()`, so anyone can load the shared FluxLab
	// storefront, tag their own order with somebody else's Jobber user ID and buy
	// a Jobber plan. The webhook HMAC proves FastSpring sent the event — it says
	// nothing about who wrote the tag. Only the proof does, and none of these
	// orders has one that verifies.
	attackerProof, ok := orderTagProof(testWebhookSecret, attackerUserID)
	require.True(t, ok)
	victimProof, ok := orderTagProof(testWebhookSecret, testUserID)
	require.True(t, ok)

	cases := []struct {
		name string
		tags map[string]any
	}{
		{
			// Fable's scenario, exactly: a stranger's storefront purchase naming
			// the victim's UUID, with no proof at all.
			name: "a storefront buyer names a victim with no proof",
			tags: map[string]any{userIDTagKey: testUserID},
		},
		{
			// The attacker starts a real checkout of their own first, so they hold
			// one valid pair — for themselves. It does not travel to another user.
			name: "a valid proof for one user cannot carry another user's ID",
			tags: map[string]any{userIDTagKey: testUserID, userProofTagKey: attackerProof},
		},
		{
			name: "the proof is for the right user but one character is changed",
			tags: map[string]any{userIDTagKey: testUserID, userProofTagKey: flipLastRune(victimProof)},
		},
		{
			name: "the proof is truncated",
			tags: map[string]any{userIDTagKey: testUserID, userProofTagKey: victimProof[:len(victimProof)-4]},
		},
		{
			name: "the proof carries an unknown scheme version",
			tags: map[string]any{userIDTagKey: testUserID, userProofTagKey: "v2." + victimProof[3:]},
		},
		{
			name: "the proof is not even base64",
			tags: map[string]any{userIDTagKey: testUserID, userProofTagKey: "v1.$$$not-base64$$$"},
		},
		{
			name: "the proof is blank",
			tags: map[string]any{userIDTagKey: testUserID, userProofTagKey: ""},
		},
		{
			name: "the proof is not a string",
			tags: map[string]any{userIDTagKey: testUserID, userProofTagKey: 42},
		},
		{
			// A malformed ID never reaches a query, and a proof cannot rescue it:
			// there is no user for it to be a proof of.
			name: "the user tag is not a UUID",
			tags: map[string]any{userIDTagKey: "not-a-uuid", userProofTagKey: victimProof},
		},
		{
			name: "the user tag carries SQL rather than an identifier",
			tags: map[string]any{userIDTagKey: testUserID + "' OR '1'='1", userProofTagKey: victimProof},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := retaggedBody(t, tc.tags)
			repo := newRecordingRepo(nil)
			freeRowForAnyUser(repo)
			svc, accountCalls := newTaggedService(t, repo)

			result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

			require.NoError(t, err)
			assert.Empty(t, repo.upserts, "an unprovable tag must grant nothing")
			assert.Empty(t, repo.claims, "nothing was applied, so nothing may be claimed")
			assert.Empty(t, result.Failed, "the claim can never become provable, so retrying is pointless")
			require.Len(t, result.Skipped, 1)
			assert.ErrorIs(t, result.Skipped[0].Err, ErrUnprovenOrderTag)
			assert.NotErrorIs(t, result.Skipped[0].Err, errForeignBillingAccount,
				"a Jobber product with a forged tag is an anomaly, not another product's order")
			assert.Contains(t, result.Processed, taggedEventID)
			assert.Zero(t, accountCalls.Load(),
				"a distrusted payload must not be resolved by any other hop either")
		})
	}

	t.Run("the refusal never echoes the raw tag back into the logs", func(t *testing.T) {
		body := retaggedBody(t, map[string]any{userIDTagKey: "<script>alert(1)</script>"})
		repo := newRecordingRepo(nil)
		freeRowForAnyUser(repo)
		svc, _ := newTaggedService(t, repo)

		result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

		require.NoError(t, err)
		require.Len(t, result.Skipped, 1)
		assert.ErrorIs(t, result.Skipped[0].Err, ErrUnprovenOrderTag)
		assert.NotContains(t, result.Skipped[0].Err.Error(), "script")
	})
}

func TestHandleWebhookRefusesAProofFromAnotherSecret(t *testing.T) {
	// Rotation, and the reason it fails loudly rather than quietly. A proof
	// minted under the previous webhook secret cannot verify under the new one,
	// so a checkout in flight across a rotation loses its tag hop — but it is
	// reported, and it never resolves to the wrong user.
	proof, ok := orderTagProof("the-previous-webhook-secret", testUserID)
	require.True(t, ok)
	body := retaggedBody(t, map[string]any{userIDTagKey: testUserID, userProofTagKey: proof})
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc, _ := newTaggedService(t, repo)

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, repo.upserts)
	assert.Empty(t, repo.claims)
	require.Len(t, result.Skipped, 1)
	assert.ErrorIs(t, result.Skipped[0].Err, ErrUnprovenOrderTag)
}

func TestHandleWebhookUntaggedOrderStaysAForeignPurchase(t *testing.T) {
	// An order that claims nothing is not an anomaly — it is every other
	// product's order in the shared store. Resolution falls through to the
	// account lookup key, finds no Jobber key, and the event is acknowledged as
	// somebody else's rather than retried forever.
	cases := []struct {
		name string
		tags map[string]any
	}{
		{name: "no tags at all", tags: nil},
		{name: "tags present but empty", tags: map[string]any{}},
		{name: "unrelated tags only", tags: map[string]any{"campaign": "spring"}},
		{name: "the user tag is blank", tags: map[string]any{userIDTagKey: ""}},
		{name: "the user tag is whitespace", tags: map[string]any{userIDTagKey: "   "}},
		{
			// decodeTags keeps string values only, so a non-string ID is no claim.
			name: "the user tag is not even a string",
			tags: map[string]any{userIDTagKey: 42},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := retaggedBody(t, tc.tags)
			repo := newRecordingRepo(nil)
			freeRowForAnyUser(repo)
			svc, accountCalls := newTaggedService(t, repo)

			result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

			require.NoError(t, err)
			assert.Empty(t, repo.upserts, "an untagged order must grant nothing")
			assert.Empty(t, repo.claims)
			assert.Empty(t, result.Failed, "an unresolvable shared-store event must not be retried forever")
			require.Len(t, result.Skipped, 1)
			assert.ErrorIs(t, result.Skipped[0].Err, errForeignBillingAccount)
			assert.Contains(t, result.Processed, taggedEventID)
			assert.Equal(t, int64(1), accountCalls.Load(),
				"with no claim to check, the lookup key is still the last word")
		})
	}
}

func TestHandleWebhookForeignProductCannotClaimAJobberUser(t *testing.T) {
	// The FluxLab store sells more than Jobber. Even a *perfectly proven* Jobber
	// tag — replayed by its own owner onto a purchase of somebody else's product
	// — must not touch a Jobber subscription: the product gate is checked before
	// the tags are read at all.
	proof, ok := orderTagProof(testWebhookSecret, testUserID)
	require.True(t, ok)
	body := transformFirstEvent(t, loadFixture(t, taggedFixture), func(event map[string]any) {
		data, ok := event["data"].(map[string]any)
		require.True(t, ok)
		data["product"] = map[string]any{"product": foreignProductPath}
		data["tags"] = map[string]any{userIDTagKey: testUserID, userProofTagKey: proof}
	})
	repo := newRecordingRepo(nil)
	freeRowForAnyUser(repo)
	svc, _ := newTaggedService(t, repo)

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, repo.upserts, "another product's purchase must never grant a Jobber plan")
	assert.Empty(t, repo.claims)
	assert.Empty(t, result.Failed)
	require.Len(t, result.Skipped, 1)
	assert.ErrorIs(t, result.Skipped[0].Err, errForeignBillingAccount)
	assert.Contains(t, result.Processed, taggedEventID)
}

func TestHandleWebhookOrderTagRefusesAUserWhoAlreadyPays(t *testing.T) {
	// A proof says who an order belongs to; it does not say the order should be
	// applied. Its owner can replay it onto a storefront purchase Jobber never
	// brokered, so the guard that a paying user's row is never repointed has to
	// hold on the proven path too: honouring the event would overwrite the single
	// external_subscription_id and strand the subscription they actually pay for.
	body := loadFixture(t, taggedFixture)
	repo := newRecordingRepo(nil)
	repo.GetByUserIDFunc = func(_ context.Context, userID string) (*model.Subscription, error) {
		return &model.Subscription{
			ID:                     "sub-row-1",
			UserID:                 userID,
			ExternalSubscriptionID: ptr(otherProviderSubscrID),
			Status:                 StatusActive,
			Plan:                   PlanPro,
		}, nil
	}
	svc, _ := newTaggedService(t, repo)

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, repo.upserts, "a paying subscriber's row must not be repointed by a tag")
	assert.Empty(t, repo.claims)
	assert.Empty(t, result.Failed, "the contradiction is permanent, so retrying is pointless")
	require.Len(t, result.Skipped, 1)
	assert.ErrorIs(t, result.Skipped[0].Err, ErrTaggedOwnerConflict)
	assert.Contains(t, result.Processed, taggedEventID)
}

func TestHandleWebhookOrderTagWithNoUserRowIsRetried(t *testing.T) {
	// The tag is proven and names a Jobber product, so this is a broken link on
	// our side — a row still catching up, or one to repair — not another
	// product's customer. Retrying is what gives it a chance to land.
	body := loadFixture(t, taggedFixture)
	repo := newRecordingRepo(nil)
	svc, _ := newTaggedService(t, repo)

	result, err := svc.HandleWebhook(context.Background(), body, sign(t, body, testWebhookSecret))

	require.NoError(t, err)
	assert.Empty(t, repo.upserts)
	assert.Empty(t, repo.claims)
	require.Len(t, result.Failed, 1)
	assert.Equal(t, taggedEventID, result.Failed[0].EventID)
	assert.ErrorIs(t, result.Failed[0].Err, model.ErrSubscriptionNotFound)
	assert.NotErrorIs(t, result.Failed[0].Err, errForeignBillingAccount)
}
