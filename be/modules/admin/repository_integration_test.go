package admin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	subModel "github.com/andreypavlenko/jobber/modules/subscriptions/model"
	subRepo "github.com/andreypavlenko/jobber/modules/subscriptions/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const fixtureSchema = `CREATE TABLE users(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),email text NOT NULL,name text NOT NULL,locale text NOT NULL DEFAULT 'en',email_verified bool NOT NULL DEFAULT false,password_hash text NOT NULL DEFAULT '',created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE subscriptions(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),user_id uuid UNIQUE REFERENCES users(id),external_subscription_id text UNIQUE,external_account_id text,status text NOT NULL,plan text NOT NULL,current_period_start timestamptz,current_period_end timestamptz,cancel_at timestamptz,last_event_at timestamptz,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE webhook_events(event_id text PRIMARY KEY,event_type text NOT NULL,processed_at timestamptz NOT NULL DEFAULT now());`

func isolatedPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("ADMIN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("ADMIN_TEST_DATABASE_URL unset: isolated database integration not requested")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", cfg.ConnConfig.Host)
	require.EqualValues(t, 55439, cfg.ConnConfig.Port)
	require.Equal(t, "admin_fixture", cfg.ConnConfig.Database, "tests only accept the disposable admin fixture database")
	root, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	schema := "admin_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = root.Exec(t.Context(), "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize())
	require.NoError(t, err)
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Close()
		_, err := root.Exec(context.Background(), "DROP SCHEMA "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
		require.NoError(t, err)
		root.Close()
	})
	_, err = pool.Exec(t.Context(), fixtureSchema)
	require.NoError(t, err)
	applyMigration(t, pool, "000052_subscription_credits.up.sql")
	applyMigration(t, pool, "000053_manual_admin_access.up.sql")
	return pool
}
func migration(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "migrations", name))
	require.NoError(t, err)
	return string(data)
}
func applyMigration(t *testing.T, pool *pgxpool.Pool, name string) {
	t.Helper()
	_, err := pool.Exec(t.Context(), migration(t, name))
	require.NoError(t, err)
}
func fixtureUsers(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(t.Context(), `INSERT INTO users(id,email,name,email_verified) VALUES($1,'admin@example.com','Admin',true),($2,'user@example.com','User',true)`, testUser, testGrant)
	require.NoError(t, err)
}
func TestIntegrationManualAccessLedgerAndProviderIndependence(t *testing.T) {
	pool := isolatedPool(t)
	fixtureUsers(t, pool)
	ctx := t.Context()
	admin := NewRepository(pool)
	ledger := subRepo.NewSubscriptionRepository(pool)
	require.NoError(t, ledger.ConfigureAIBudget(ctx, 100_000_000_000))
	p := GrantRequest{ID: uuid.NewString(), Days: 30, Reason: "Support trial"}
	require.NoError(t, admin.Grant(ctx, testGrant, testUser, p))
	before, err := ledger.ManualAccess(ctx, testGrant)
	require.NoError(t, err)
	require.NotNil(t, before)
	b, err := ledger.CreditBalance(ctx, testGrant)
	require.NoError(t, err)
	require.Equal(t, "manual_pro", b.Kind)
	require.Equal(t, 500, b.Remaining)
	require.EqualValues(t, 3_000_000_000, b.SpendLimitNanodollars)
	op := subModel.CreditOperation{UserID: testGrant, Key: uuid.NewString(), Fingerprint: "manual-test", Kind: "cover_letter", Credits: 7}
	attempt, err := ledger.ReserveCreditAttempt(ctx, subModel.AIAttempt{Operation: op, Stage: "generate", Model: "fixture", MaxNanodollars: 1_000_000})
	require.NoError(t, err)
	require.NoError(t, ledger.DispatchCreditAttempt(ctx, attempt))
	require.NoError(t, admin.Grant(ctx, testGrant, testUser, p))
	after, err := ledger.ManualAccess(ctx, testGrant)
	require.NoError(t, err)
	require.Equal(t, before, after)
	b, err = ledger.CreditBalance(ctx, testGrant)
	require.NoError(t, err)
	require.Equal(t, 493, b.Remaining)
	modified := p
	modified.Days = 31
	require.ErrorIs(t, admin.Grant(ctx, testGrant, testUser, modified), ErrConflict)
	// A real provider lifecycle update does not overwrite the separate manual grant.
	now := time.Now().UTC()
	subID, account := "fixture-sub", "fixture-customer"
	end := now.Add(30 * 24 * time.Hour)
	sub := &subModel.Subscription{UserID: testGrant, ExternalSubscriptionID: &subID, ExternalAccountID: &account, Status: "active", Plan: "pro", CurrentPeriodStart: &now, CurrentPeriodEnd: &end, LastEventAt: &now}
	outcome, err := ledger.ApplySubscriptionEvent(ctx, "manual-lifecycle", "subscription.active", sub)
	require.NoError(t, err)
	require.Equal(t, subModel.WebhookApplied, outcome)
	manual, err := ledger.ManualAccess(ctx, testGrant)
	require.NoError(t, err)
	require.NotNil(t, manual)
	require.NoError(t, admin.Revoke(ctx, testGrant, p.ID, testUser, "Trial ended"))
	require.NoError(t, admin.Revoke(ctx, testGrant, p.ID, testUser, "Repeated revoke"))
	manual, err = ledger.ManualAccess(ctx, testGrant)
	require.NoError(t, err)
	require.Nil(t, manual)
	provider, err := ledger.GetByUserID(ctx, testGrant)
	require.NoError(t, err)
	require.Equal(t, "active", provider.Status)
	require.Equal(t, "pro", provider.Plan)
	var allocated int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&allocated))
	require.EqualValues(t, 1_000_000, allocated, "revocation must retain dispatched liability")
	require.NoError(t, admin.Grant(ctx, testGrant, testUser, p))
	manual, err = ledger.ManualAccess(ctx, testGrant)
	require.NoError(t, err)
	require.Nil(t, manual, "retry cannot revive revoked receipt")
	// Provider payment allowance survives manual revocation, and is preferred.
	_, err = pool.Exec(ctx, `INSERT INTO credit_grants(user_id,kind,credits,provider,mode,transaction_id,subscription_id,customer_id,product_id,period_start,period_end) VALUES($1,'pro',500,'creem','test','fixture-tx',$2,$3,'fixture-product',$4,$5)`, testGrant, subID, account, now, end)
	require.NoError(t, err)
	b, err = ledger.CreditBalance(ctx, testGrant)
	require.NoError(t, err)
	require.Equal(t, "pro", b.Kind)
	require.Equal(t, 500, b.Remaining)
	d, err := admin.Detail(ctx, testGrant)
	require.NoError(t, err)
	require.Equal(t, "pro", d.EffectivePlan)
	require.Len(t, d.ManualGrants, 1)
	require.Equal(t, "Trial ended", *d.ManualGrants[0].RevokeReason)
}
func TestIntegrationManualRevokeFencesDispatchAndExpiry(t *testing.T) {
	pool := isolatedPool(t)
	fixtureUsers(t, pool)
	ctx := t.Context()
	admin := NewRepository(pool)
	ledger := subRepo.NewSubscriptionRepository(pool)
	require.NoError(t, ledger.ConfigureAIBudget(ctx, 100_000_000_000))
	p := GrantRequest{ID: uuid.NewString(), Days: 1, Reason: "Fixture trial"}
	require.NoError(t, admin.Grant(ctx, testGrant, testUser, p))
	op := subModel.CreditOperation{UserID: testGrant, Key: uuid.NewString(), Fingerprint: "test", Kind: "cover_letter", Credits: 7}
	attempt, err := ledger.ReserveCreditAttempt(ctx, subModel.AIAttempt{Operation: op, Stage: "generate", Model: "fixture", MaxNanodollars: 1_000_000})
	require.NoError(t, err)
	require.NoError(t, admin.Revoke(ctx, testGrant, p.ID, testUser, "Stop access"))
	require.ErrorIs(t, ledger.DispatchCreditAttempt(ctx, attempt), subModel.ErrAIOperationConflict)
	b, err := ledger.CreditBalance(ctx, testGrant)
	require.NoError(t, err)
	require.Equal(t, "free_once", b.Kind)
	next := GrantRequest{ID: uuid.NewString(), Days: 1, Reason: "Second trial"}
	require.NoError(t, admin.Grant(ctx, testGrant, testUser, next))
	_, err = pool.Exec(ctx, `UPDATE credit_grants SET period_start=now()-interval '2 days',period_end=now()-interval '1 day' WHERE id=$1`, next.ID)
	require.NoError(t, err)
	manual, err := ledger.ManualAccess(ctx, testGrant)
	require.NoError(t, err)
	require.Nil(t, manual)
	b, err = ledger.CreditBalance(ctx, testGrant)
	require.NoError(t, err)
	require.Equal(t, "free_once", b.Kind)
}
func TestIntegrationManualGrantRaceAndReadOnlyOverview(t *testing.T) {
	pool := isolatedPool(t)
	fixtureUsers(t, pool)
	admin := NewRepository(pool)
	ctx := t.Context()
	_, err := admin.List(ctx, "%", 1, 20)
	require.NoError(t, err)
	d, err := admin.Detail(ctx, testGrant)
	require.NoError(t, err)
	require.Empty(t, d.ManualGrants)
	var grants, budgets int
	require.NoError(t, pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM credit_grants),(SELECT count(*) FROM ai_provider_budget)`).Scan(&grants, &budgets))
	require.Zero(t, grants)
	require.Zero(t, budgets)
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results <- admin.Grant(ctx, testGrant, testUser, GrantRequest{ID: uuid.NewString(), Days: 7, Reason: "Concurrent trial"})
		}()
	}
	wait.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if err == ErrConflict {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)
}
func TestIntegrationManualMigrationRollbackGuard(t *testing.T) {
	pool := isolatedPool(t)
	applyMigration(t, pool, "000053_manual_admin_access.down.sql")
	applyMigration(t, pool, "000053_manual_admin_access.up.sql")
	fixtureUsers(t, pool)
	require.NoError(t, NewRepository(pool).Grant(t.Context(), testGrant, testUser, GrantRequest{ID: uuid.NewString(), Days: 1, Reason: "Preserve history"}))
	conn, err := pool.Acquire(t.Context())
	require.NoError(t, err)
	defer conn.Release()
	_, err = conn.Exec(t.Context(), migration(t, "000053_manual_admin_access.down.sql"))
	require.ErrorContains(t, err, "Manual access history exists")
	_, err = conn.Exec(t.Context(), "ROLLBACK")
	require.NoError(t, err)
	var count int
	require.NoError(t, pool.QueryRow(t.Context(), `SELECT count(*) FROM manual_access_grants`).Scan(&count))
	require.Equal(t, 1, count)
}
