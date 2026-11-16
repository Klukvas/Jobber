//go:build integration

package repository

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIntegrationRejectedResultRefundsUserKeepsProviderCost(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := context.Background()
	key := uuid.NewString()
	id, err := repo.ReserveCreditAttempt(ctx, creditAttempt(owner, key, "generate", 7))
	require.NoError(t, err)
	require.NoError(t, repo.DispatchCreditAttempt(ctx, id))
	require.NoError(t, repo.SettleCreditAttempt(ctx, id, 10_000_000))
	require.NoError(t, repo.RejectCreditOperation(ctx, creditAttempt(owner, key, "generate", 7).Operation))
	require.NoError(t, repo.RejectCreditOperation(ctx, creditAttempt(owner, key, "generate", 7).Operation))
	balance, err := repo.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 30, balance.Remaining)
	var cost int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&cost))
	require.EqualValues(t, 10_000_000, cost)
}

func TestIntegrationResultRetryChargesOnceAndKeepsAttempts(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := t.Context()
	key := uuid.NewString()
	attempt := creditAttempt(owner, key, "1:1", 7)
	id, err := repo.ReserveCreditAttempt(ctx, attempt)
	require.NoError(t, err)
	require.NoError(t, repo.DispatchCreditAttempt(ctx, id))
	require.NoError(t, repo.SettleCreditAttempt(ctx, id, 10_000_000))
	require.NoError(t, repo.RejectCreditOperation(ctx, attempt.Operation))
	_, err = repo.ReserveCreditAttempt(ctx, attempt)
	require.Error(t, err, "same stage cannot reset the provider call limit")
	attempt.Stage = "2:1"
	id, err = repo.ReserveCreditAttempt(ctx, attempt)
	require.NoError(t, err)
	require.NoError(t, repo.DispatchCreditAttempt(ctx, id))
	require.NoError(t, repo.SettleCreditAttempt(ctx, id, 10_000_000))
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); require.NoError(t, repo.CompleteCreditOperation(ctx, attempt.Operation)) }()
	}
	wg.Wait()
	require.NoError(t, repo.RejectCreditOperation(ctx, attempt.Operation))
	balance, err := repo.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 23, balance.Remaining)
	var calls int
	require.NoError(t, pool.QueryRow(ctx, `SELECT count(*) FROM ai_credit_attempts`).Scan(&calls))
	require.Equal(t, 2, calls)
}

func TestIntegrationResultSaveRollbackAndCrashRefund(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := t.Context()
	attempt := creditAttempt(owner, uuid.NewString(), "1:1", 7)
	id, err := repo.ReserveCreditAttempt(ctx, attempt)
	require.NoError(t, err)
	require.NoError(t, repo.DispatchCreditAttempt(ctx, id))
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, FinalizeCreditResultInTransaction(ctx, tx, owner, attempt.Operation.Key, true, ""))
	require.NoError(t, tx.Rollback(ctx))
	_, err = pool.Exec(ctx, `UPDATE ai_credit_operations SET created_at=now()-interval '6 minutes' WHERE user_id=$1`, owner)
	require.NoError(t, err)
	balance, err := repo.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 30, balance.Remaining)
	require.Error(t, repo.CompleteCreditOperation(ctx, attempt.Operation), "late save cannot charge a refunded crashed operation")
	var cost int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&cost))
	require.EqualValues(t, 2_000_000_000, cost)
}

func TestIntegrationChargeMigrationExistingFailedAndUnknownOperations(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := t.Context()
	failed := creditAttempt(owner, uuid.NewString(), "1:1", 7)
	unknown := creditAttempt(owner, uuid.NewString(), "1:1", 7)
	for _, a := range []model.AIAttempt{failed, unknown} {
		id, err := repo.ReserveCreditAttempt(ctx, a)
		require.NoError(t, err)
		require.NoError(t, repo.DispatchCreditAttempt(ctx, id))
		require.NoError(t, repo.SettleCreditAttempt(ctx, id, 10_000_000))
	}
	_, err := pool.Exec(ctx, `WITH legacy AS (UPDATE ai_credit_operations SET charge_state='legacy') INSERT INTO interview_operations(id,user_id,status,charge_released) VALUES($1,$2,'failed',false)`, failed.Operation.Key, owner)
	require.NoError(t, err)
	down, err := os.ReadFile("../../../migrations/000059_ai_result_charges.down.sql")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(down))
	require.NoError(t, err)
	up, err := os.ReadFile("../../../migrations/000059_ai_result_charges.up.sql")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, string(up))
	require.NoError(t, err)
	b, err := repo.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 23, b.Remaining, "proved failure refunded, unknown settled retained")
	var costs int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&costs))
	require.EqualValues(t, 20_000_000, costs)
}

func TestIntegrationParsedResultMigrationAndOriginalGrantExpiry(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := t.Context()
	grant := paidFixture(t, repo, pool, owner)
	applyPaidFixture(t, repo, grant, "paid")
	a := creditAttempt(owner, uuid.NewString(), "1:1", 7)
	id, err := repo.ReserveCreditAttempt(ctx, a)
	require.NoError(t, err)
	require.NoError(t, repo.DispatchCreditAttempt(ctx, id))
	require.NoError(t, repo.SettleCreditAttempt(ctx, id, 1000))
	_, err = pool.Exec(ctx, `WITH legacy AS (UPDATE ai_credit_operations SET charge_state='legacy') INSERT INTO resume_tailoring_operations(user_id,ai_receipt_id,status) VALUES($1,$2,'parsed')`, owner, a.Operation.Key)
	require.NoError(t, err)
	for _, direction := range []string{"down", "up"} {
		ddl, err := os.ReadFile("../../../migrations/000059_ai_result_charges." + direction + ".sql")
		require.NoError(t, err)
		_, err = pool.Exec(ctx, string(ddl))
		require.NoError(t, err)
	}
	var state string
	require.NoError(t, pool.QueryRow(ctx, `SELECT charge_state FROM ai_credit_operations WHERE operation_key=$1`, a.Operation.Key).Scan(&state))
	require.Equal(t, "reserved", state)
	require.NoError(t, repo.RejectCreditOperation(ctx, a.Operation))
	_, err = pool.Exec(ctx, `UPDATE credit_grants SET period_start=now()-interval '30 days',period_end=now()-interval '1 day' WHERE kind='pro' AND user_id=$1`, owner)
	require.NoError(t, err)
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, RestoreSavedResultCharge(ctx, tx, owner, a.Operation.Key))
	require.NoError(t, FinalizeCreditResultInTransaction(ctx, tx, owner, a.Operation.Key, true, ""))
	require.NoError(t, tx.Commit(ctx))
	_, err = repo.ReserveCreditAttempt(ctx, creditAttempt(owner, a.Operation.Key, "2:1", 7))
	require.Error(t, err, "saving a parsed result cannot admit a new provider call in a later period")
	b, err := repo.CreditBalance(ctx, owner)
	require.NoError(t, err)
	require.Equal(t, 30, b.Remaining)
	var charged, cost int64
	require.NoError(t, pool.QueryRow(ctx, `SELECT sum(credits) FROM ai_credit_operations WHERE user_id=$1 AND charge_state='charged'`, owner).Scan(&charged))
	require.EqualValues(t, 7, charged)
	require.NoError(t, pool.QueryRow(ctx, `SELECT allocated_nanodollars FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&cost))
	require.EqualValues(t, 1000, cost)
}
