//go:build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ApplySubscriptionEvent is one hand-written statement whose correctness lives
// entirely in PostgreSQL: a CTE that claims an event and writes entitlement
// together, an ON CONFLICT target, a WHERE guard re-evaluated under concurrency,
// and a COALESCE. A mock can only replay whatever behaviour the test author
// already believed; these tests run the real statement against a real server.
//
// Run them with:
//
//	TEST_DATABASE_URL='postgres://jobber:jobber@localhost:5434/jobber?sslmode=disable' \
//	  go test -race -tags integration -run Integration ./modules/subscriptions/repository/
//
// Without TEST_DATABASE_URL every test here skips.

// testSchemaDDL mirrors the production shape of everything the statement
// touches: migration 000013 with the 000045 renames applied, the webhook_events
// claim table from 000033, and the minimal users table the foreign key needs.
const testSchemaDDL = `
CREATE TABLE users (
	id    UUID PRIMARY KEY,
	email TEXT NOT NULL
);

CREATE TABLE subscriptions (
	id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id                  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	external_subscription_id TEXT UNIQUE,
	external_account_id      TEXT,
	status                   TEXT NOT NULL DEFAULT 'free',
	plan                     TEXT NOT NULL DEFAULT 'free',
	current_period_start     TIMESTAMPTZ,
	current_period_end       TIMESTAMPTZ,
	cancel_at                TIMESTAMPTZ,
	last_event_at            TIMESTAMPTZ,
	created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The ON CONFLICT (user_id) arbiter the apply statement depends on.
CREATE UNIQUE INDEX idx_subscriptions_user_id ON subscriptions (user_id);

CREATE UNIQUE INDEX idx_subscriptions_external_account_id
	ON subscriptions (external_account_id)
	WHERE external_account_id IS NOT NULL;

CREATE TABLE webhook_events (
	event_id     TEXT PRIMARY KEY,
	event_type   TEXT NOT NULL,
	processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

// newIntegrationRepo brings up an isolated PostgreSQL schema for one test and
// returns a repository wired to it.
//
// Isolation is the whole point. The pool's search_path names *only* the
// throwaway schema, so every unqualified statement — the repository's included —
// resolves inside it and cannot read or write the developer database whose
// server it borrows. search_path is a connection runtime parameter, so it
// applies to this pool alone and never leaks into another. The schema name is
// unique per process and per call, so parallel runs and leftovers from a killed
// run cannot collide.
func newIntegrationRepo(t *testing.T) (*SubscriptionRepository, *pgxpool.Pool) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set — skipping live-PostgreSQL test")
	}

	schema := uniqueSchemaName()
	quoted := pgx.Identifier{schema}.Sanitize()

	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err, "TEST_DATABASE_URL must be a valid PostgreSQL DSN")
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	// Two racing deliveries need two connections; the rest is headroom.
	cfg.MaxConns = 8

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err, "could not connect to TEST_DATABASE_URL")

	_, err = pool.Exec(ctx, "CREATE SCHEMA "+quoted)
	if err != nil {
		pool.Close()
		require.NoError(t, err, "could not create the test schema")
	}

	t.Cleanup(func() {
		// Dropped by exact name, so CASCADE can only reach objects this test
		// created. A failure here is reported rather than swallowed: a leftover
		// schema is the one way this test can pollute the database it borrowed.
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dropCancel()
		if _, err := pool.Exec(dropCtx, "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Errorf("failed to drop test schema %s: %v", schema, err)
		}
		pool.Close()
	})

	_, err = pool.Exec(ctx, testSchemaDDL)
	require.NoError(t, err, "could not create the test tables")

	return NewSubscriptionRepository(pool), pool
}

// uniqueSchemaName builds a name no concurrent run can collide with, prefixed so
// a leftover is recognisably a test artefact. Comfortably inside PostgreSQL's
// 63-byte identifier limit.
func uniqueSchemaName() string {
	return fmt.Sprintf("jobber_it_%d_%d", os.Getpid(), time.Now().UnixNano())
}

func seedUser(t *testing.T, pool *pgxpool.Pool, userID string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email) VALUES ($1, $2)`, userID, userID+"@example.test")
	require.NoError(t, err)
}

// eventState builds the subscription state a single webhook event carries.
// changedAt is the provider's `data.changed` — the state-change time the
// ordering guard compares, not a delivery timestamp.
func eventState(userID, subID string, account *string, status, plan string, changedAt time.Time) *model.Subscription {
	return &model.Subscription{
		UserID:                 userID,
		ExternalSubscriptionID: &subID,
		ExternalAccountID:      account,
		Status:                 status,
		Plan:                   plan,
		LastEventAt:            &changedAt,
	}
}

func recordedEventIDs(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT event_id FROM webhook_events`)
	require.NoError(t, err)
	defer rows.Close()
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	return ids
}

func TestIntegrationApplySubscriptionEventOrdering(t *testing.T) {
	repo, pool := newIntegrationRepo(t)
	ctx := context.Background()

	const (
		userID    = "11111111-1111-4111-8111-111111111111"
		subID     = "SuBsCr1PT10nJobberPro01"
		accountID = "acctJobberIntegration01"
	)
	seedUser(t, pool, userID)
	account := accountID

	var (
		earlier = time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
		applied = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
		later   = time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
		latest  = time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	)

	t.Run("a first event is applied", func(t *testing.T) {
		outcome, err := repo.ApplySubscriptionEvent(ctx, "evt-activated", "subscription.activated",
			eventState(userID, subID, &account, "active", "pro", applied))

		require.NoError(t, err)
		assert.Equal(t, model.WebhookApplied, outcome)

		sub, err := repo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "active", sub.Status)
		assert.Equal(t, "pro", sub.Plan)
		require.NotNil(t, sub.ExternalAccountID)
		assert.Equal(t, accountID, *sub.ExternalAccountID)
		require.NotNil(t, sub.LastEventAt)
		assert.True(t, sub.LastEventAt.Equal(applied))
	})

	t.Run("the same event ID again is a duplicate and writes nothing", func(t *testing.T) {
		// Deliberately carries *newer*, destructive state: if the claim did not
		// stop it, the ordering guard would happily let it through and the row
		// would end up cancelled from a mere retry.
		outcome, err := repo.ApplySubscriptionEvent(ctx, "evt-activated", "subscription.activated",
			eventState(userID, subID, &account, "cancelled", "free", latest))

		require.NoError(t, err)
		assert.Equal(t, model.WebhookDuplicate, outcome)

		sub, err := repo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "active", sub.Status, "an automatic retry must not rewrite the row")
		assert.Equal(t, "pro", sub.Plan)
		assert.True(t, sub.LastEventAt.Equal(applied))
	})

	t.Run("an event with the same changed time is superseded", func(t *testing.T) {
		// The realistic pair: a charge and the subscription.updated it triggers,
		// both stamped with one `changed`. First writer wins; the second is
		// recorded as processed so the provider stops redelivering it.
		outcome, err := repo.ApplySubscriptionEvent(ctx, "evt-charge", "subscription.charge.completed",
			eventState(userID, subID, &account, "past_due", "pro", applied))

		require.NoError(t, err)
		assert.Equal(t, model.WebhookSuperseded, outcome)

		sub, err := repo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "active", sub.Status, "the guard is strictly greater-than")
		assert.True(t, sub.LastEventAt.Equal(applied))

		assert.Contains(t, recordedEventIDs(t, pool), "evt-charge",
			"a superseded event is still processed — it must not be redelivered forever")
	})

	t.Run("an older event is superseded", func(t *testing.T) {
		// A manual resend of a stale event: fresh ID, fresh envelope, but the
		// provider's own `changed` still points at the older state change.
		outcome, err := repo.ApplySubscriptionEvent(ctx, "evt-stale-resend", "subscription.updated",
			eventState(userID, subID, &account, "cancelled", "free", earlier))

		require.NoError(t, err)
		assert.Equal(t, model.WebhookSuperseded, outcome)

		sub, err := repo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "active", sub.Status, "a resend must not resurrect older state")
		assert.Equal(t, "pro", sub.Plan)
		assert.True(t, sub.LastEventAt.Equal(applied))
	})

	t.Run("a newer event is applied", func(t *testing.T) {
		outcome, err := repo.ApplySubscriptionEvent(ctx, "evt-updated", "subscription.updated",
			eventState(userID, subID, &account, "active", "enterprise", later))

		require.NoError(t, err)
		assert.Equal(t, model.WebhookApplied, outcome)

		sub, err := repo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "enterprise", sub.Plan)
		assert.True(t, sub.LastEventAt.Equal(later))
	})

	t.Run("an event without an account keeps the linked one", func(t *testing.T) {
		// Not every lifecycle payload carries the account. COALESCE is what stops
		// such an event from erasing the only server-side link between the
		// purchase and this user.
		outcome, err := repo.ApplySubscriptionEvent(ctx, "evt-deactivated", "subscription.deactivated",
			eventState(userID, subID, nil, "cancelled", "free", latest))

		require.NoError(t, err)
		assert.Equal(t, model.WebhookApplied, outcome)

		sub, err := repo.GetByUserID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "cancelled", sub.Status)
		assert.Equal(t, "free", sub.Plan)
		require.NotNil(t, sub.ExternalAccountID, "the account link must survive an event that omits it")
		assert.Equal(t, accountID, *sub.ExternalAccountID)
	})

	t.Run("every delivery is recorded exactly once", func(t *testing.T) {
		assert.ElementsMatch(t,
			[]string{"evt-activated", "evt-charge", "evt-stale-resend", "evt-updated", "evt-deactivated"},
			recordedEventIDs(t, pool))
	})
}

func TestIntegrationApplySubscriptionEventConcurrentDeliveries(t *testing.T) {
	repo, pool := newIntegrationRepo(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const (
		userID    = "22222222-2222-4222-8222-222222222222"
		subID     = "SuBsCr1PT10nJobberPro02"
		accountID = "acctJobberIntegration02"
	)
	seedUser(t, pool, userID)
	account := accountID

	var (
		seeded = time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		older  = time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC)
		newer  = time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
	)

	// Seed the row first, so both racing deliveries take the
	// ON CONFLICT ... DO UPDATE branch — that WHERE clause is what has to hold.
	outcome, err := repo.ApplySubscriptionEvent(ctx, "evt-seed", "subscription.activated",
		eventState(userID, subID, &account, "active", "pro", seeded))
	require.NoError(t, err)
	require.Equal(t, model.WebhookApplied, outcome)

	deliveries := []struct {
		eventID   string
		eventType string
		status    string
		plan      string
		changedAt time.Time
	}{
		{eventID: "evt-old", eventType: "subscription.updated", status: "past_due", plan: "pro", changedAt: older},
		{eventID: "evt-new", eventType: "subscription.deactivated", status: "cancelled", plan: "free", changedAt: newer},
	}

	outcomes := make([]model.WebhookApplyOutcome, len(deliveries))
	errs := make([]error, len(deliveries))

	// Released together so both statements contend for the same row. Whichever
	// order PostgreSQL picks, the guard is re-evaluated against the version the
	// other writer committed — the two can never interleave into the old state.
	var ready, done sync.WaitGroup
	start := make(chan struct{})
	for i, delivery := range deliveries {
		ready.Add(1)
		done.Add(1)
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			outcomes[i], errs[i] = repo.ApplySubscriptionEvent(ctx, delivery.eventID, delivery.eventType,
				eventState(userID, subID, &account, delivery.status, delivery.plan, delivery.changedAt))
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()

	for i, delivery := range deliveries {
		require.NoError(t, errs[i], "delivery %s failed", delivery.eventID)
		assert.NotEqual(t, model.WebhookDuplicate, outcomes[i],
			"%s has a distinct event ID and must be claimed, not dismissed as a repeat", delivery.eventID)
	}
	assert.Equal(t, model.WebhookApplied, outcomes[1],
		"the newer event is newer than anything already on the row, so it must always apply")

	sub, err := repo.GetByUserID(ctx, userID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", sub.Status, "the newer state must be the one left standing")
	assert.Equal(t, "free", sub.Plan)
	require.NotNil(t, sub.LastEventAt)
	assert.True(t, sub.LastEventAt.Equal(newer),
		"expected the row stamped with the newer change, got %s", sub.LastEventAt)

	// Both are claimed even though only one may have written: an unclaimed event
	// would be redelivered by the provider forever.
	assert.ElementsMatch(t, []string{"evt-seed", "evt-old", "evt-new"}, recordedEventIDs(t, pool))
}

// seedSubscriptionRow writes a user's row directly, so a test can start from any
// state the guard has to judge — including ones only the provider can produce.
func seedSubscriptionRow(t *testing.T, pool *pgxpool.Pool, userID string, linkedID *string, status, plan string) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO subscriptions (user_id, external_subscription_id, status, plan)
		 VALUES ($1::uuid, $2::text, $3::text, $4::text)`,
		userID, linkedID, status, plan)
	require.NoError(t, err)
}

func linkedSubscriptionID(t *testing.T, repo *SubscriptionRepository, userID string) string {
	t.Helper()
	sub, err := repo.GetByUserID(context.Background(), userID)
	require.NoError(t, err)
	if sub.ExternalSubscriptionID == nil {
		return ""
	}
	return *sub.ExternalSubscriptionID
}

// TestIntegrationApplySubscriptionEventLinkGuard runs the invariant that keeps a
// subscriber from being billed twice with only one of the two cancellable:
// the row holds a single external_subscription_id, so it may only be replaced
// when there is none, when the event describes that very subscription, or when
// the provider has already ended it.
//
// A mock can only replay what the test author believed the SQL does; this runs
// the real WHERE clause and the real row lock.
func TestIntegrationApplySubscriptionEventLinkGuard(t *testing.T) {
	changedAt := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	// What the row points at before the event lands. The identifiers themselves
	// are generated per case: external_subscription_id is globally unique, so no
	// two rows in one schema may name the same subscription.
	type link int
	const (
		linkNone  link = iota // no provider subscription yet
		linkSame              // the very subscription this event describes
		linkOther             // a different subscription
	)

	tests := []struct {
		name        string
		link        link
		status      string
		wantOutcome model.WebhookApplyOutcome
	}{
		{
			name:        "the first activation links a row that points at nothing",
			link:        linkNone,
			status:      "free",
			wantOutcome: model.WebhookApplied,
		},
		{
			name:        "the same subscription keeps moving through its lifecycle",
			link:        linkSame,
			status:      "active",
			wantOutcome: model.WebhookApplied,
		},
		{
			name:        "a second subscription cannot replace an active one",
			link:        linkOther,
			status:      "active",
			wantOutcome: model.WebhookLinkConflict,
		},
		{
			// Dunning: FastSpring is still trying to charge the old subscription.
			name:        "a second subscription cannot replace a past-due one",
			link:        linkOther,
			status:      "past_due",
			wantOutcome: model.WebhookLinkConflict,
		},
		{
			// A pause resumes into billing, so the identifier must survive it.
			name:        "a second subscription cannot replace a paused one",
			link:        linkOther,
			status:      "paused",
			wantOutcome: model.WebhookLinkConflict,
		},
		{
			// A scheduled cancellation is stored as active with cancel_at: access
			// and billing both run to the deactivation date.
			name:        "a second subscription cannot replace one winding down",
			link:        linkOther,
			status:      "active",
			wantOutcome: model.WebhookLinkConflict,
		},
		{
			// The provider ended it, so there is nothing left to strand and
			// buying again is the only way back to a paid plan.
			name:        "a cancelled subscription may be replaced",
			link:        linkOther,
			status:      "cancelled",
			wantOutcome: model.WebhookApplied,
		},
	}

	repo, pool := newIntegrationRepo(t)
	ctx := context.Background()

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			userID := fmt.Sprintf("33333333-3333-4333-8333-3333333333%02d", i)
			incomingID := fmt.Sprintf("SuBsCr1PT10nIncoming%02d", i)
			eventID := fmt.Sprintf("evt-link-guard-%02d", i)

			var linked *string
			switch tc.link {
			case linkSame:
				linked = &incomingID
			case linkOther:
				linked = ptr(fmt.Sprintf("SuBsCr1PT10nAlreadyLinked%02d", i))
			}

			seedUser(t, pool, userID)
			seedSubscriptionRow(t, pool, userID, linked, tc.status, "pro")

			outcome, err := repo.ApplySubscriptionEvent(ctx, eventID, "subscription.activated",
				eventState(userID, incomingID, nil, "active", "enterprise", changedAt))

			require.NoError(t, err)
			assert.Equal(t, tc.wantOutcome, outcome)

			if tc.wantOutcome == model.WebhookLinkConflict {
				require.NotNil(t, linked)
				assert.Equal(t, *linked, linkedSubscriptionID(t, repo, userID),
					"the live subscription must still be the one the row points at")
				assert.NotContains(t, recordedEventIDs(t, pool), eventID,
					"a refused event claims nothing, so a resend can still land once the old link is ended")
				return
			}
			assert.Equal(t, incomingID, linkedSubscriptionID(t, repo, userID))
			assert.Contains(t, recordedEventIDs(t, pool), eventID)
		})
	}
}

// TestIntegrationApplySubscriptionEventConcurrentActivations is the race the
// guard exists for: a user starts two checkouts while free and pays for both, so
// two activations for two different provider subscriptions land at once.
//
// Exactly one may link. If both were allowed the second would overwrite the
// first identifier and leave that subscription billing at FastSpring with
// nothing in Jobber able to cancel it.
func TestIntegrationApplySubscriptionEventConcurrentActivations(t *testing.T) {
	repo, pool := newIntegrationRepo(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	const (
		userID = "44444444-4444-4444-8444-444444444444"
		subA   = "SuBsCr1PT10nJobberRaceA"
		subB   = "SuBsCr1PT10nJobberRaceB"
	)
	seedUser(t, pool, userID)
	seedSubscriptionRow(t, pool, userID, nil, "free", "free")

	activations := []struct {
		eventID string
		subID   string
	}{
		{eventID: "evt-activated-a", subID: subA},
		{eventID: "evt-activated-b", subID: subB},
	}
	// Distinct change times, so nothing but the link guard can decide the loser:
	// the later event would sail past the ordering guard.
	changedAt := []time.Time{
		time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 1, 11, 0, 0, 0, time.UTC),
	}

	outcomes := make([]model.WebhookApplyOutcome, len(activations))
	errs := make([]error, len(activations))

	var ready, done sync.WaitGroup
	start := make(chan struct{})
	for i, activation := range activations {
		ready.Add(1)
		done.Add(1)
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			outcomes[i], errs[i] = repo.ApplySubscriptionEvent(ctx, activation.eventID, "subscription.activated",
				eventState(userID, activation.subID, nil, "active", "pro", changedAt[i]))
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()

	for i, activation := range activations {
		require.NoError(t, errs[i], "delivery %s failed", activation.eventID)
	}

	applied, conflicted := 0, 0
	for _, outcome := range outcomes {
		switch outcome {
		case model.WebhookApplied:
			applied++
		case model.WebhookLinkConflict:
			conflicted++
		}
	}
	assert.Equal(t, 1, applied, "exactly one activation may link, got outcomes %v", outcomes)
	assert.Equal(t, 1, conflicted,
		"the loser must be reported as a link conflict, not as a routine supersede: got %v", outcomes)

	linked := linkedSubscriptionID(t, repo, userID)
	assert.Contains(t, []string{subA, subB}, linked)

	// The whole point: the winner is still the one the row names. Neither
	// activation may have overwritten the other.
	winner := activations[0]
	if linked == subB {
		winner = activations[1]
	}
	assert.ElementsMatch(t, []string{winner.eventID}, recordedEventIDs(t, pool),
		"only the activation that linked may be claimed — the refused one wrote nothing at all")
}
