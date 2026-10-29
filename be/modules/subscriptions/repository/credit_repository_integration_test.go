//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func newCreditFixture(t *testing.T) (*SubscriptionRepository, *pgxpool.Pool, string) {
	t.Helper()
	repo, pool := newIntegrationRepo(t)
	ddl, err := os.ReadFile("../../../migrations/000052_subscription_credits.up.sql")
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), string(ddl))
	require.NoError(t, err)
	owner := uuid.NewString()
	seedUser(t, pool, owner)
	require.NoError(t, repo.EnsureFree(context.Background(), owner))
	require.NoError(t, repo.ConfigureAIBudget(context.Background(), 100_000_000_000))
	return repo, pool, owner
}
func creditAttempt(owner, key, stage string, credits int) model.AIAttempt {
	return model.AIAttempt{Operation: model.CreditOperation{UserID: owner, Key: key, Kind: "tailor", Fingerprint: "fixture", Credits: credits}, Stage: stage, Model: "fixture", MaxNanodollars: 2_000_000_000}
}

func TestIntegrationCreditFreeOnceConcurrentAndPersistent(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := context.Background()
	var success atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			attempt := creditAttempt(owner, uuid.NewString(), "generate", 7)
			attempt.MaxNanodollars = 10_000_000
			_, err := repo.ReserveCreditAttempt(ctx, attempt)
			if err == nil {
				success.Add(1)
			} else if !errors.Is(err, model.ErrCreditUnavailable) {
				t.Errorf("reserve: %v", err)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 4, success.Load())
	restarted := NewSubscriptionRepository(pool)
	balance, err := restarted.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 2, balance.Remaining)
	var grants int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM credit_grants WHERE user_id=$1`, owner).Scan(&grants))
	require.Equal(t, 1, grants)
	_, err = pool.Exec(ctx, `UPDATE credit_grants SET created_at=now()-interval '2 months' WHERE user_id=$1`, owner)
	require.NoError(t, err)
	balance, err = restarted.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 2, balance.Remaining)
}
func TestIntegrationCreditUnknownCannotExpireRefundOrReplay(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := context.Background()
	key := uuid.NewString()
	id, err := repo.ReserveCreditAttempt(ctx, creditAttempt(owner, key, "generate", 7))
	require.NoError(t, err)
	require.NoError(t, repo.DispatchCreditAttempt(ctx, id))
	_, err = pool.Exec(ctx, `UPDATE ai_credit_attempts SET created_at=now()-interval '2 days' WHERE id=$1`, id)
	require.NoError(t, err)
	require.ErrorIs(t, repo.ReleaseCreditAttempt(ctx, id), model.ErrAIOperationConflict)
	_, err = repo.ReserveCreditAttempt(ctx, creditAttempt(owner, key, "generate", 7))
	require.ErrorIs(t, err, model.ErrAIOperationConflict)
	balance, err := repo.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 23, balance.Remaining)
	var burn int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&burn))
	require.EqualValues(t, 2_000_000_000, burn)
}
func TestIntegrationCreditStagesChargeOnceSettleAndGlobalCap(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := context.Background()
	key := uuid.NewString()
	require.NoError(t, repo.ConfigureAIBudget(ctx, 3_000_000_000))
	id, err := repo.ReserveCreditAttempt(ctx, creditAttempt(owner, key, "generate", 7))
	require.NoError(t, err)
	require.NoError(t, repo.DispatchCreditAttempt(ctx, id))
	_, err = repo.ReserveCreditAttempt(ctx, creditAttempt(owner, key, "audit", 7))
	require.ErrorIs(t, err, model.ErrAIBudgetReached)
	require.NoError(t, repo.SettleCreditAttempt(ctx, id, 10_000_000))
	require.NoError(t, repo.SettleCreditAttempt(ctx, id, 10_000_000))
	audit, err := repo.ReserveCreditAttempt(ctx, creditAttempt(owner, key, "audit", 7))
	require.NoError(t, err)
	require.NoError(t, repo.DispatchCreditAttempt(ctx, audit))
	balance, err := repo.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 7, balance.Used)
	require.ErrorIs(t, repo.SettleCreditAttempt(ctx, audit, 3_000_000_000), model.ErrAIBudgetReached)
	var burn int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&burn))
	require.EqualValues(t, 2_010_000_000, burn)
}
func paidFixture(t *testing.T, repo *SubscriptionRepository, pool *pgxpool.Pool, owner string) model.PaidCreditGrant {
	t.Helper()
	now := time.Now().UTC()
	subID := "sub_" + uuid.NewString()
	customer := "cust_" + uuid.NewString()
	_, err := pool.Exec(context.Background(), `UPDATE subscriptions SET external_subscription_id=$2,external_account_id=$3,status='active',plan='pro' WHERE user_id=$1`, owner, subID, customer)
	require.NoError(t, err)
	return model.PaidCreditGrant{UserID: owner, Provider: "creem", Mode: "test", TransactionID: "txn_" + uuid.NewString(), SubscriptionID: subID, CustomerID: customer, ProductID: "prod_24_month", PeriodStart: now.Add(-time.Hour).Truncate(time.Microsecond), PeriodEnd: now.Add(30*24*time.Hour - time.Hour).Truncate(time.Microsecond)}
}
func applyPaidFixture(t *testing.T, repo *SubscriptionRepository, g model.PaidCreditGrant, event string) {
	t.Helper()
	now := time.Now().UTC()
	sub := &model.Subscription{UserID: g.UserID, ExternalSubscriptionID: &g.SubscriptionID, ExternalAccountID: &g.CustomerID, Status: "active", Plan: "pro", CurrentPeriodStart: &g.PeriodStart, CurrentPeriodEnd: &g.PeriodEnd, LastEventAt: &now}
	_, err := repo.ApplyPaidCreditEvent(context.Background(), event, "subscription.paid", sub, g)
	require.NoError(t, err)
}
func TestIntegrationCreditPaidDuplicateAndRefundBeforePaid(t *testing.T) {
	for _, holdFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "refund-first"}[holdFirst], func(t *testing.T) {
			repo, pool, owner := newCreditFixture(t)
			ctx := context.Background()
			g := paidFixture(t, repo, pool, owner)
			if holdFirst {
				require.NoError(t, repo.HoldCreditPayment(ctx, "refund1", "refund.created", "test", g.TransactionID))
			}
			applyPaidFixture(t, repo, g, "paid1")
			applyPaidFixture(t, repo, g, "paid1")
			applyPaidFixture(t, repo, g, "paid2")
			var count int
			require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM credit_grants WHERE kind='pro'`).Scan(&count))
			require.Equal(t, 1, count)
			b, err := repo.CreditBalance(ctx, owner)
			require.NoError(t, err)
			if holdFirst {
				require.Equal(t, 30, b.Remaining)
			} else {
				require.Equal(t, 500, b.Remaining)
				require.NoError(t, repo.HoldCreditPayment(ctx, "refund1", "refund.created", "test", g.TransactionID))
				b, err = repo.CreditBalance(ctx, owner)
				require.NoError(t, err)
				require.Equal(t, 30, b.Remaining)
			}
		})
	}
}
func TestIntegrationCreditRevocationBetweenStagesAndAccountDelete(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := context.Background()
	g := paidFixture(t, repo, pool, owner)
	applyPaidFixture(t, repo, g, "paid")
	key := uuid.NewString()
	id, err := repo.ReserveCreditAttempt(ctx, creditAttempt(owner, key, "generate", 7))
	require.NoError(t, err)
	require.NoError(t, repo.DispatchCreditAttempt(ctx, id))
	require.NoError(t, repo.HoldCreditPayment(ctx, "refund", "refund.created", "test", g.TransactionID))
	_, err = repo.ReserveCreditAttempt(ctx, creditAttempt(owner, key, "audit", 7))
	require.ErrorIs(t, err, model.ErrAIOperationConflict)
	_, err = pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, owner)
	require.NoError(t, err)
	var burn int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&burn))
	require.EqualValues(t, 2_000_000_000, burn)
	var holds int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM credit_payment_holds`).Scan(&holds))
	require.Equal(t, 1, holds)
}
func TestIntegrationCreditMigrationRollbackSafety(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := context.Background()
	down, err := os.ReadFile("../../../migrations/000052_subscription_credits.down.sql")
	require.NoError(t, err)
	id, err := repo.ReserveCreditAttempt(ctx, creditAttempt(owner, "rollback", "generate", 7))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(down))
	require.ErrorContains(t, err, "used AI credit ledger")
	require.NoError(t, repo.ReleaseCreditAttempt(ctx, id))
	_, err = pool.Exec(ctx, string(down))
	require.ErrorContains(t, err, "used AI credit ledger")
}

func TestIntegrationCreditPerGrantSpendingAndUndispatchedRecovery(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := context.Background()
	id, err := repo.ReserveCreditAttempt(ctx, creditAttempt(owner, "abandoned", "generate", 7))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE ai_credit_attempts SET created_at=now()-interval '6 minutes' WHERE id=$1`, id)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE ai_credit_operations SET created_at=now()-interval '6 minutes' WHERE user_id=$1`, owner)
	require.NoError(t, err)
	balance, err := repo.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 30, balance.Remaining)
	require.ErrorIs(t, repo.DispatchCreditAttempt(ctx, id), model.ErrAIOperationConflict)
	sent, err := repo.ReserveCreditAttempt(ctx, creditAttempt(owner, "actually-sent", "generate", 7))
	require.NoError(t, err)
	require.NoError(t, repo.DispatchCreditAttempt(ctx, sent))
	_, err = repo.ReserveCreditAttempt(ctx, creditAttempt(owner, "another", "generate", 7))
	require.ErrorIs(t, err, model.ErrAIBudgetReached)
	var burn int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&burn))
	require.EqualValues(t, 2_000_000_000, burn)
	require.NoError(t, repo.ConfigureAIBudget(ctx, 0))
	_, err = repo.ReserveCreditAttempt(ctx, creditAttempt(owner, "blocked", "generate", 7))
	require.ErrorIs(t, err, model.ErrAIBudgetReached)
	require.NoError(t, pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&burn))
	require.EqualValues(t, 2_000_000_000, burn)
}
func TestIntegrationCreditPaidPeriodRenewsDespiteEqualLifecycleTimestamp(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := context.Background()
	g := paidFixture(t, repo, pool, owner)
	at := time.Now().UTC()
	old := g
	old.TransactionID = "prior-transaction"
	old.PeriodEnd = g.PeriodStart
	old.PeriodStart = g.PeriodStart.Add(-30 * 24 * time.Hour)
	sub := &model.Subscription{UserID: owner, ExternalSubscriptionID: &g.SubscriptionID, ExternalAccountID: &g.CustomerID, Status: "active", Plan: "pro", CurrentPeriodStart: &old.PeriodStart, CurrentPeriodEnd: &old.PeriodEnd, LastEventAt: &at}
	_, err := repo.ApplyPaidCreditEvent(ctx, "prior-paid", "subscription.paid", sub, old)
	require.NoError(t, err)
	sub.CurrentPeriodStart = &g.PeriodStart
	sub.CurrentPeriodEnd = &g.PeriodEnd
	outcome, err := repo.ApplyPaidCreditEvent(ctx, "renewal-paid", "subscription.paid", sub, g)
	require.NoError(t, err)
	require.Equal(t, model.WebhookSuperseded, outcome)
	b, err := repo.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 500, b.Remaining)
	_, err = pool.Exec(ctx, `UPDATE credit_grants SET period_end=now()-interval '1 minute',period_start=now()-interval '30 days' WHERE kind='pro' AND transaction_id=$1`, g.TransactionID)
	require.NoError(t, err)
	b, err = repo.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 30, b.Remaining)
}

func TestIntegrationCreditVirginMigrationRoundTripAndStickyProviderBlock(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := context.Background()
	down, err := os.ReadFile("../../../migrations/000052_subscription_credits.down.sql")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(down))
	require.NoError(t, err)
	up, err := os.ReadFile("../../../migrations/000052_subscription_credits.up.sql")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(up))
	require.NoError(t, err)
	require.NoError(t, repo.ConfigureAIBudget(ctx, 10_000_000_000))
	id, err := repo.ReserveCreditAttempt(ctx, creditAttempt(owner, "reserved-before-block", "generate", 7))
	require.NoError(t, err)
	require.NoError(t, repo.BlockAIProvider(ctx, "response_contract"))
	restarted := NewSubscriptionRepository(pool)
	require.NoError(t, restarted.ConfigureAIBudget(ctx, 10_000_000_000))
	_, err = restarted.ReserveCreditAttempt(ctx, creditAttempt(owner, "blocked-after-restart", "generate", 7))
	require.ErrorIs(t, err, model.ErrAIBudgetReached)
	require.ErrorIs(t, restarted.DispatchCreditAttempt(ctx, id), model.ErrAIBudgetReached)
	_, err = pool.Exec(ctx, string(down))
	require.ErrorContains(t, err, "used AI credit ledger")
}

func TestIntegrationCreditRollbackWaitsForPendingFirstWriter(t *testing.T) {
	for _, reservation := range []bool{false, true} {
		t.Run(map[bool]string{false: "first grant", true: "first reservation"}[reservation], func(t *testing.T) {
			_, pool, owner := newCreditFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			writer, err := pool.Begin(ctx)
			require.NoError(t, err)
			defer writer.Rollback(context.Background())
			var grant string
			require.NoError(t, writer.QueryRow(ctx, `INSERT INTO credit_grants(user_id,kind,credits) VALUES($1,'free_once',30) RETURNING id`, owner).Scan(&grant))
			if reservation {
				var operation string
				require.NoError(t, writer.QueryRow(ctx, `INSERT INTO ai_credit_operations(user_id,operation_key,kind,fingerprint,grant_id,credits) VALUES($1,'pending-first','tailor','fixture',$2,7) RETURNING id`, owner, grant).Scan(&operation))
				_, err = writer.Exec(ctx, `UPDATE ai_provider_budget SET allocated_nanodollars=2000000000 WHERE provider='anthropic'`)
				require.NoError(t, err)
				_, err = writer.Exec(ctx, `INSERT INTO ai_credit_attempts(operation_id,stage,model,provider,reserved_nanodollars,state) VALUES($1,'first','fixture','anthropic',2000000000,'reserved')`, operation)
				require.NoError(t, err)
			}
			down, err := os.ReadFile("../../../migrations/000052_subscription_credits.down.sql")
			require.NoError(t, err)
			rollback, err := pool.Acquire(ctx)
			require.NoError(t, err)
			defer rollback.Release()
			pid := rollback.Conn().PgConn().PID()
			result := make(chan error, 1)
			go func() {
				// Simulate a runner sending the lock prefix separately from the
				// remaining statements: explicit BEGIN must retain its locks.
				split := strings.Index(string(down), "DO $$")
				_, runErr := rollback.Exec(ctx, string(down[:split]))
				if runErr == nil {
					_, runErr = rollback.Exec(ctx, string(down[split:]))
				}
				_, _ = rollback.Exec(context.Background(), "ROLLBACK")
				result <- runErr
			}()
			require.Eventually(t, func() bool {
				var waiting bool
				queryErr := pool.QueryRow(ctx, `SELECT cardinality(pg_blocking_pids($1))>0`, pid).Scan(&waiting)
				return queryErr == nil && waiting
			}, 3*time.Second, 10*time.Millisecond, "rollback must wait for the pending financial writer")
			require.NoError(t, writer.Commit(ctx))
			require.ErrorContains(t, <-result, "used AI credit ledger")
			var count int
			require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM credit_grants WHERE id=$1`, grant).Scan(&count))
			require.Equal(t, 1, count)
			if reservation {
				var burn int64
				require.NoError(t, pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&burn))
				require.EqualValues(t, 2_000_000_000, burn)
				require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM ai_credit_attempts`).Scan(&count))
				require.Equal(t, 1, count)
			}
		})
	}
}
