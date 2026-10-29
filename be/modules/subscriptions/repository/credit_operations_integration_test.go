//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestIntegrationCreditCleanupCannotReleaseConcurrentProviderReservation(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	key := uuid.NewString()
	request := creditAttempt(owner, key, "generate", 7)
	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	require.NoError(t, ReserveCreditOperationInTransaction(ctx, tx, request.Operation))
	require.NoError(t, tx.Commit(ctx))
	// Hold the real reservation after its operation lock but before attempt visibility.
	// The trigger exists only in this test's isolated schema and never changes production SQL.
	const barrier int64 = 72496103
	blocker, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer blocker.Release()
	_, err = blocker.Exec(ctx, "SELECT pg_advisory_lock($1)", barrier)
	require.NoError(t, err)
	defer func() {
		_, unlockErr := blocker.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", barrier)
		require.NoError(t, unlockErr)
	}()
	_, err = pool.Exec(ctx, `CREATE FUNCTION hold_credit_attempt() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_advisory_xact_lock(72496103); RETURN NEW; END $$; CREATE TRIGGER hold_credit_attempt BEFORE INSERT ON ai_credit_attempts FOR EACH ROW EXECUTE FUNCTION hold_credit_attempt()`)
	require.NoError(t, err)
	makeRepo := func(label string) (*SubscriptionRepository, *pgxpool.Pool) {
		cfg := pool.Config()
		cfg.ConnConfig.RuntimeParams["application_name"] = label
		named, openErr := pgxpool.NewWithConfig(ctx, cfg)
		require.NoError(t, openErr)
		t.Cleanup(named.Close)
		return NewSubscriptionRepository(named), named
	}
	reserveName := "release-reserve-" + uuid.NewString()
	reserve, _ := makeRepo(reserveName)
	cleanupName := "release-cleanup-" + uuid.NewString()
	cleanup, _ := makeRepo(cleanupName)
	type reserved struct {
		id  string
		err error
	}
	reservedCh := make(chan reserved, 1)
	go func() {
		id, reserveErr := reserve.ReserveCreditAttempt(ctx, request)
		reservedCh <- reserved{id, reserveErr}
	}()
	waitBlocked := func(name string) {
		require.Eventually(t, func() bool {
			var n int
			err := pool.QueryRow(ctx, "SELECT count(*) FROM pg_stat_activity WHERE application_name=$1 AND wait_event_type='Lock'", name).Scan(&n)
			return err == nil && n == 1
		}, 5*time.Second, 10*time.Millisecond, "transaction did not reach controlled lock barrier: %s", name)
	}
	waitBlocked(reserveName)
	cleanupCh := make(chan error, 1)
	go func() { cleanupCh <- cleanup.FinishCreditOperation(ctx, owner, key) }()
	waitBlocked(cleanupName)
	_, err = blocker.Exec(ctx, "SELECT pg_advisory_unlock($1)", barrier)
	require.NoError(t, err)
	result := <-reservedCh
	require.NoError(t, result.err)
	require.NoError(t, <-cleanupCh)
	require.NoError(t, repo.DispatchCreditAttempt(ctx, result.id))
	balance, err := repo.CreditBalance(ctx, owner)
	require.NoError(t, err)
	var opState, attemptState string
	require.NoError(t, pool.QueryRow(ctx, `SELECT o.state,a.state FROM ai_credit_operations o JOIN ai_credit_attempts a ON a.operation_id=o.id WHERE a.id=$1`, result.id).Scan(&opState, &attemptState))
	t.Logf("concurrent cleanup: operation=%s attempt=%s remaining=%d", opState, attemptState, balance.Remaining)
	require.Equal(t, 23, balance.Remaining, "provider dispatch must retain the 7-credit charge")
	require.Equal(t, "dispatched", opState)
}

func TestIntegrationCreditDispatchRejectsReleasedOperation(t *testing.T) {
	repo, pool, owner := newCreditFixture(t)
	ctx := t.Context()
	id, err := repo.ReserveCreditAttempt(ctx, creditAttempt(owner, uuid.NewString(), "generate", 7))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `UPDATE ai_credit_operations SET state='released' WHERE id=(SELECT operation_id FROM ai_credit_attempts WHERE id=$1)`, id)
	require.NoError(t, err)
	err = repo.DispatchCreditAttempt(ctx, id)
	require.ErrorIs(t, err, model.ErrAIOperationConflict)
	var state string
	require.NoError(t, pool.QueryRow(ctx, "SELECT state FROM ai_credit_attempts WHERE id=$1", id).Scan(&state))
	require.Equal(t, "reserved", state, "failed dispatch must atomically roll back attempt CAS")
}

func TestIntegrationCreditCleanupBindsOperationIdentity(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(*model.CreditOperation)
	}{
		{"different feature", func(op *model.CreditOperation) { op.Kind = "cover_letter" }},
		{"different body fingerprint", func(op *model.CreditOperation) { op.Fingerprint = "another-body" }},
		{"different credit cost", func(op *model.CreditOperation) { op.Credits = 5 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			repo, pool, owner := newCreditFixture(t)
			ctx := t.Context()
			op := creditAttempt(owner, uuid.NewString(), "generate", 7).Operation
			tx, err := pool.Begin(ctx)
			require.NoError(t, err)
			require.NoError(t, ReserveCreditOperationInTransaction(ctx, tx, op))
			require.NoError(t, tx.Commit(ctx))
			conflicting := op
			tt.change(&conflicting)
			require.NoError(t, repo.FinishCreditOperation(ctx, owner, op.Key, conflicting))
			var state string
			require.NoError(t, pool.QueryRow(ctx, "SELECT state FROM ai_credit_operations WHERE user_id=$1 AND operation_key=$2", owner, op.Key).Scan(&state))
			require.Equal(t, "reserved", state, "conflicting HTTP cleanup must not release another operation")
			balance, err := repo.CreditBalance(ctx, owner)
			require.NoError(t, err)
			require.Equal(t, 23, balance.Remaining)
			require.NoError(t, repo.FinishCreditOperation(ctx, owner, op.Key, op))
			balance, err = repo.CreditBalance(ctx, owner)
			require.NoError(t, err)
			require.Equal(t, 30, balance.Remaining, "matching undispatched cleanup still refunds its own reservation")
		})
	}
}
