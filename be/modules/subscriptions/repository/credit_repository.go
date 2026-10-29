package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ConfigureAIBudget never erases existing allocation, including unknown outcomes.
func (r *SubscriptionRepository) ConfigureAIBudget(ctx context.Context, limit int64) error {
	if limit < 0 {
		return model.ErrAIBudgetReached
	}
	_, err := r.pool.Exec(ctx, `INSERT INTO ai_provider_budget(provider,limit_nanodollars) VALUES('anthropic',$1) ON CONFLICT(provider) DO UPDATE SET limit_nanodollars=EXCLUDED.limit_nanodollars`, limit)
	if err != nil {
		return fmt.Errorf("configure ai budget: %w", err)
	}
	return nil
}

func ensureFreeCredits(ctx context.Context, tx pgx.Tx, user string) error {
	_, err := tx.Exec(ctx, `INSERT INTO credit_grants(user_id,kind,credits) VALUES($1,'free_once',30) ON CONFLICT(user_id) WHERE kind='free_once' DO NOTHING`, user)
	return err
}

// activeGrant selects a paid period only after money was confirmed; lifecycle
// rows by themselves cannot manufacture a paid allowance. Free never refreshes.
const activeGrant = `SELECT g.id::text,g.kind,g.credits,g.period_end,
 COALESCE((SELECT sum(o.credits) FROM ai_credit_operations o WHERE o.grant_id=g.id AND o.state<>'released'),0)::integer
 FROM credit_grants g WHERE g.user_id=$1 AND g.revoked_at IS NULL AND
 (g.kind='free_once' OR (g.kind='manual_pro' AND g.period_start<=now() AND g.period_end>now() AND EXISTS(SELECT 1 FROM manual_access_grants m WHERE m.grant_id=g.id)) OR (g.kind='pro' AND g.period_start<=now() AND g.period_end>now()
 AND EXISTS(SELECT 1 FROM subscriptions s WHERE s.user_id=g.user_id AND s.external_subscription_id=g.subscription_id AND s.status='active')))
 ORDER BY (g.kind='pro') DESC,(g.kind='manual_pro') DESC,g.period_start DESC NULLS LAST LIMIT 1`

func (r *SubscriptionRepository) CreditBalance(ctx context.Context, user string) (model.CreditBalance, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.CreditBalance{}, err
	}
	defer tx.Rollback(ctx)
	if _, _, err := lockCreditBudget(ctx, tx); err != nil {
		return model.CreditBalance{}, err
	}
	if err := lockAIQuota(ctx, tx, user); err != nil {
		return model.CreditBalance{}, err
	}
	if _, err := recoverUndispatchedCredits(ctx, tx, user); err != nil {
		return model.CreditBalance{}, err
	}
	if err := ensureFreeCredits(ctx, tx, user); err != nil {
		return model.CreditBalance{}, err
	}
	var id string
	var balance model.CreditBalance
	err = tx.QueryRow(ctx, activeGrant, user).Scan(&id, &balance.Kind, &balance.Granted, &balance.PeriodEnd, &balance.Used)
	if err != nil {
		return balance, fmt.Errorf("read credit balance: %w", err)
	}
	if err := tx.QueryRow(ctx, `SELECT g.spend_limit_nanodollars,COALESCE((SELECT sum(COALESCE(a.actual_nanodollars,a.reserved_nanodollars)) FROM ai_credit_attempts a JOIN ai_credit_operations o ON o.id=a.operation_id WHERE o.grant_id=g.id AND a.state<>'released'),0)::bigint FROM credit_grants g WHERE g.id=$1`, id).Scan(&balance.SpendLimitNanodollars, &balance.SpendAllocatedNanodollars); err != nil {
		return balance, err
	}
	balance.Remaining = balance.Granted - balance.Used
	if balance.Remaining < 0 {
		balance.Remaining = 0
	}
	return balance, tx.Commit(ctx)
}

func lockCreditBudget(ctx context.Context, tx pgx.Tx) (int64, int64, error) {
	var limit, allocated int64
	var blocked bool
	err := tx.QueryRow(ctx, `SELECT limit_nanodollars,allocated_nanodollars,blocked FROM ai_provider_budget WHERE provider='anthropic' FOR UPDATE`).Scan(&limit, &allocated, &blocked)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, model.ErrAIBudgetReached
	}
	if blocked {
		limit = 0
	}
	return limit, allocated, err
}

// ReserveCreditAttempt atomically acquires both the weighted user allowance and
// full provider liability. A duplicate dispatch is refused across processes and
// restarts; business workflow receipts own successful result replay.
func (r *SubscriptionRepository) ReserveCreditAttempt(ctx context.Context, request model.AIAttempt) (string, error) {
	op := request.Operation
	if op.UserID == "" || op.Key == "" || op.Fingerprint == "" || op.Credits <= 0 || request.Stage == "" || request.MaxNanodollars <= 0 {
		return "", model.ErrCreditUnavailable
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	limit, allocated, err := lockCreditBudget(ctx, tx)
	if err != nil {
		return "", err
	}
	if err := lockAIQuota(ctx, tx, op.UserID); err != nil {
		return "", err
	}
	released, err := recoverUndispatchedCredits(ctx, tx, op.UserID)
	if err != nil {
		return "", err
	}
	allocated -= released
	if request.MaxNanodollars > limit-allocated {
		return "", model.ErrAIBudgetReached
	}
	if err := ensureFreeCredits(ctx, tx, op.UserID); err != nil {
		return "", err
	}
	var grant, kind string
	var granted, used int
	var balance model.CreditBalance
	if err := tx.QueryRow(ctx, activeGrant, op.UserID).Scan(&grant, &kind, &granted, &balance.PeriodEnd, &used); err != nil {
		return "", err
	}
	var operationID, fingerprint, existingKind, state, operationGrant string
	var credits int
	err = tx.QueryRow(ctx, `SELECT id::text,fingerprint,kind,credits,state,grant_id::text FROM ai_credit_operations WHERE user_id=$1 AND operation_key=$2 FOR UPDATE`, op.UserID, op.Key).Scan(&operationID, &fingerprint, &existingKind, &credits, &state, &operationGrant)
	if errors.Is(err, pgx.ErrNoRows) {
		if granted-used < op.Credits {
			return "", model.ErrCreditUnavailable
		}
		operationID = uuid.NewString()
		_, err = tx.Exec(ctx, `INSERT INTO ai_credit_operations(id,user_id,operation_key,kind,fingerprint,grant_id,credits) VALUES($1,$2,$3,$4,$5,$6,$7)`, operationID, op.UserID, op.Key, op.Kind, op.Fingerprint, grant, op.Credits)
	} else if err == nil {
		if fingerprint != op.Fingerprint || existingKind != op.Kind || credits != op.Credits {
			return "", model.ErrAIOperationConflict
		}
		if state == "released" {
			if granted-used < op.Credits {
				return "", model.ErrCreditUnavailable
			}
			_, err = tx.Exec(ctx, `UPDATE ai_credit_operations SET state='reserved',grant_id=$2,created_at=now() WHERE id=$1`, operationID, grant)
		} else if operationGrant != grant {
			return "", model.ErrAIOperationConflict
		}
	}
	if err != nil {
		return "", err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ai_credit_attempts WHERE operation_id=$1 AND stage=$2)`, operationID, request.Stage).Scan(&exists); err != nil {
		return "", err
	}
	if exists {
		return "", model.ErrAIOperationConflict
	}
	var spendLimit, spendAllocated int64
	err = tx.QueryRow(ctx, `SELECT g.spend_limit_nanodollars,COALESCE((SELECT sum(COALESCE(a.actual_nanodollars,a.reserved_nanodollars)) FROM ai_credit_attempts a JOIN ai_credit_operations o ON o.id=a.operation_id WHERE o.grant_id=g.id AND a.state<>'released'),0)::bigint FROM credit_grants g WHERE g.id=$1`, grant).Scan(&spendLimit, &spendAllocated)
	if err != nil {
		return "", err
	}
	if request.MaxNanodollars > spendLimit-spendAllocated {
		return "", model.ErrAIBudgetReached
	}
	id := uuid.NewString()
	tag, err := tx.Exec(ctx, `INSERT INTO ai_credit_attempts(id,operation_id,stage,model,provider,reserved_nanodollars,state) VALUES($1,$2,$3,$4,'anthropic',$5,'reserved') ON CONFLICT(operation_id,stage) DO NOTHING`, id, operationID, request.Stage, request.Model, request.MaxNanodollars)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() != 1 {
		return "", model.ErrAIOperationConflict
	}
	if _, err := tx.Exec(ctx, `UPDATE ai_provider_budget SET allocated_nanodollars=allocated_nanodollars+$1 WHERE provider='anthropic'`, request.MaxNanodollars); err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}

// DispatchCreditAttempt persists intent BEFORE the first byte can leave the app.
func (r *SubscriptionRepository) DispatchCreditAttempt(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	limit, allocated, err := lockCreditBudget(ctx, tx)
	if err != nil {
		return err
	}
	if limit < allocated {
		return model.ErrAIBudgetReached
	}
	var blocked bool
	if err := tx.QueryRow(ctx, `SELECT blocked FROM ai_provider_budget WHERE provider='anthropic'`).Scan(&blocked); err != nil {
		return err
	}
	if blocked {
		return model.ErrAIBudgetReached
	}
	var owner string
	if err := tx.QueryRow(ctx, `SELECT o.user_id::text FROM ai_credit_operations o JOIN ai_credit_attempts a ON a.operation_id=o.id WHERE a.id=$1`, id).Scan(&owner); err != nil {
		return err
	}
	if err := lockAIQuota(ctx, tx, owner); err != nil {
		return err
	}
	var operation string
	err = tx.QueryRow(ctx, `UPDATE ai_credit_attempts SET state='dispatched',dispatched_at=now() WHERE id=$1 AND state='reserved' AND EXISTS(SELECT 1 FROM ai_credit_operations o JOIN credit_grants g ON g.id=o.grant_id WHERE o.id=ai_credit_attempts.operation_id AND g.revoked_at IS NULL AND (g.kind='free_once' OR (g.kind='manual_pro' AND g.period_start<=now() AND g.period_end>now() AND EXISTS(SELECT 1 FROM manual_access_grants m WHERE m.grant_id=g.id)) OR (g.kind='pro' AND g.period_start<=now() AND g.period_end>now() AND EXISTS(SELECT 1 FROM subscriptions s WHERE s.user_id=g.user_id AND s.external_subscription_id=g.subscription_id AND s.status='active')))) RETURNING operation_id::text`, id).Scan(&operation)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrAIOperationConflict
	}
	if err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE ai_credit_operations SET state='dispatched' WHERE id=$1 AND state<>'released'`, operation)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return model.ErrAIOperationConflict
	}
	return tx.Commit(ctx)
}

// SettleCreditAttempt returns only the proven unused monetary reservation.
// Invalid/missing usage must leave the full liability outstanding.
func (r *SubscriptionRepository) SettleCreditAttempt(ctx context.Context, id string, actual int64) error {
	if actual < 0 {
		return model.ErrAIBudgetReached
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, _, err := lockCreditBudget(ctx, tx); err != nil {
		return err
	}
	var reserved int64
	var operation, state string
	var previous *int64
	err = tx.QueryRow(ctx, `SELECT reserved_nanodollars,operation_id::text,state,actual_nanodollars FROM ai_credit_attempts WHERE id=$1 FOR UPDATE`, id).Scan(&reserved, &operation, &state, &previous)
	if err != nil {
		return err
	}
	if state == "settled" {
		if previous != nil && *previous == actual {
			return tx.Commit(ctx)
		}
		return model.ErrAIOperationConflict
	}
	if state != "dispatched" || actual > reserved {
		return model.ErrAIBudgetReached
	}
	if _, err := tx.Exec(ctx, `UPDATE ai_provider_budget SET allocated_nanodollars=allocated_nanodollars-$1 WHERE provider='anthropic'`, reserved-actual); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE ai_credit_attempts SET state='settled',actual_nanodollars=$2,settled_at=now() WHERE id=$1`, id, actual); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE ai_credit_operations SET state='settled' WHERE id=$1`, operation); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ReleaseCreditAttempt is legal only before dispatch. There is intentionally no
// time-based cleanup: timeouts and crashes do not prove that a provider was free.
func (r *SubscriptionRepository) ReleaseCreditAttempt(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, _, err := lockCreditBudget(ctx, tx); err != nil {
		return err
	}
	var reserved int64
	var owner string
	if err := tx.QueryRow(ctx, `SELECT o.user_id::text FROM ai_credit_operations o JOIN ai_credit_attempts a ON a.operation_id=o.id WHERE a.id=$1`, id).Scan(&owner); err != nil {
		return err
	}
	if err := lockAIQuota(ctx, tx, owner); err != nil {
		return err
	}
	var operation string
	err = tx.QueryRow(ctx, `UPDATE ai_credit_attempts SET state='released' WHERE id=$1 AND state='reserved' RETURNING reserved_nanodollars,operation_id::text`, id).Scan(&reserved, &operation)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrAIOperationConflict
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE ai_provider_budget SET allocated_nanodollars=allocated_nanodollars-$1 WHERE provider='anthropic'`, reserved); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE ai_credit_operations SET state='released' WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM ai_credit_attempts WHERE operation_id=$1 AND state IN ('reserved','dispatched','settled'))`, operation); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// BlockAIProvider is sticky across process restarts and budget reconfiguration.
// It stores a low-cardinality reason only, never raw provider content.
func (r *SubscriptionRepository) BlockAIProvider(ctx context.Context, reason string) error {
	if reason != "response_contract" {
		return model.ErrAIBudgetReached
	}
	_, err := r.pool.Exec(ctx, `UPDATE ai_provider_budget SET blocked=true,block_reason=$1 WHERE provider='anthropic'`, reason)
	return err
}
