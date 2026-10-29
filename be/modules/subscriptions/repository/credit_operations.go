package repository

import (
	"context"
	"errors"

	"github.com/andreypavlenko/jobber/internal/platform/ai"
	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ReserveCreditOperationInTransaction couples an existing durable workflow receipt
// with its weighted allowance before enqueueing or doing expensive preparation.
func ReserveCreditOperationInTransaction(ctx context.Context, tx pgx.Tx, op model.CreditOperation) error {
	if err := lockAIQuota(ctx, tx, op.UserID); err != nil {
		return err
	}
	if err := ensureFreeCredits(ctx, tx, op.UserID); err != nil {
		return err
	}
	var grant, kind string
	var granted, used int
	var b model.CreditBalance
	if err := tx.QueryRow(ctx, activeGrant, op.UserID).Scan(&grant, &kind, &granted, &b.PeriodEnd, &used); err != nil {
		return err
	}
	var fp, oldKind, state string
	var credits int
	err := tx.QueryRow(ctx, `SELECT fingerprint,kind,credits,state FROM ai_credit_operations WHERE user_id=$1 AND operation_key=$2 FOR UPDATE`, op.UserID, op.Key).Scan(&fp, &oldKind, &credits, &state)
	if err == nil {
		if fp != op.Fingerprint || oldKind != op.Kind || credits != op.Credits {
			return model.ErrAIOperationConflict
		}
		if state == "released" {
			if granted-used < op.Credits {
				return model.ErrCreditUnavailable
			}
			_, err = tx.Exec(ctx, `UPDATE ai_credit_operations SET state='reserved',grant_id=$3,created_at=now() WHERE user_id=$1 AND operation_key=$2`, op.UserID, op.Key, grant)
			return err
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if granted-used < op.Credits {
		return model.ErrCreditUnavailable
	}
	_, err = tx.Exec(ctx, `INSERT INTO ai_credit_operations(id,user_id,operation_key,kind,fingerprint,grant_id,credits) VALUES($1,$2,$3,$4,$5,$6,$7)`, uuid.NewString(), op.UserID, op.Key, op.Kind, op.Fingerprint, grant, op.Credits)
	return err
}

// FinishCreditOperation shares reservation's provider/user lock order. The cleanup
// statement must take its snapshot after an in-flight attempt has committed.
// HTTP cleanup also binds the complete operation, so a reused key cannot release
// another feature's durable reservation. Trusted workflow cleanup may omit it.
func (r *SubscriptionRepository) FinishCreditOperation(ctx context.Context, user, key string, expected ...model.CreditOperation) error {
	if len(expected) > 1 {
		return model.ErrAIOperationConflict
	}
	var operation model.CreditOperation
	bound := len(expected) == 1
	if bound {
		operation = expected[0]
		if operation.UserID != user || operation.Key != key || operation.Kind == "" || operation.Fingerprint == "" || operation.Credits <= 0 {
			return model.ErrAIOperationConflict
		}
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, _, err := lockCreditBudget(ctx, tx); err != nil {
		return err
	}
	if err := lockAIQuota(ctx, tx, user); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE ai_credit_operations SET state='released' WHERE user_id=$1 AND operation_key=$2 AND state='reserved' AND (NOT $3::boolean OR (fingerprint=$4 AND kind=$5 AND credits=$6)) AND NOT EXISTS(SELECT 1 FROM ai_credit_attempts a WHERE a.operation_id=ai_credit_operations.id AND a.state<>'released')`, user, key, bound, operation.Fingerprint, operation.Kind, operation.Credits)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func ContextCreditReservation(ctx context.Context, tx pgx.Tx, user, receipt string) (bool, error) {
	op, ok := ai.CreditOperationFromContext(ctx, user, receipt)
	if !ok {
		return false, nil
	}
	return true, ReserveCreditOperationInTransaction(ctx, tx, op)
}

// recoverUndispatchedCredits runs with the provider row and this user's quota
// lock held. Expiry is safe only for reserved attempts: Dispatch's CAS fences any
// late sender. Unknown/dispatched liability is deliberately excluded forever.
func recoverUndispatchedCredits(ctx context.Context, tx pgx.Tx, user string) (int64, error) {
	var released int64
	err := tx.QueryRow(ctx, `WITH expired AS(UPDATE ai_credit_attempts a SET state='released' FROM ai_credit_operations o WHERE a.operation_id=o.id AND o.user_id=$1 AND a.state='reserved' AND a.created_at<now()-interval '5 minutes' RETURNING a.reserved_nanodollars) SELECT COALESCE(sum(reserved_nanodollars),0)::bigint FROM expired`, user).Scan(&released)
	if err != nil {
		return 0, err
	}
	if released > 0 {
		if _, err := tx.Exec(ctx, `UPDATE ai_provider_budget SET allocated_nanodollars=allocated_nanodollars-$1 WHERE provider='anthropic'`, released); err != nil {
			return 0, err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE ai_credit_operations o SET state='released' WHERE user_id=$1 AND state='reserved' AND created_at<now()-interval '5 minutes' AND NOT EXISTS(SELECT 1 FROM ai_credit_attempts a WHERE a.operation_id=o.id AND a.state<>'released')`, user)
	return released, err
}
func (r *SubscriptionRepository) CreditOperationDispatched(ctx context.Context, user, key string) (bool, error) {
	var dispatched bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ai_credit_attempts a JOIN ai_credit_operations o ON o.id=a.operation_id WHERE o.user_id=$1 AND o.operation_key=$2 AND a.state IN ('dispatched','settled'))`, user, key).Scan(&dispatched)
	return dispatched, err
}

// AbortUnsentCreditAttempt compensates a dispatch intent only when local control
// flow proves that the provider transport was never invoked (source preflight).
// It is deliberately distinct from transport-error cleanup and has no HTTP API.
func (r *SubscriptionRepository) AbortUnsentCreditAttempt(ctx context.Context, id string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, _, err := lockCreditBudget(ctx, tx); err != nil {
		return err
	}
	var reserved int64
	var operation string
	err = tx.QueryRow(ctx, `UPDATE ai_credit_attempts SET state='released' WHERE id=$1 AND state='dispatched' AND actual_nanodollars IS NULL RETURNING reserved_nanodollars,operation_id::text`, id).Scan(&reserved, &operation)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrAIOperationConflict
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE ai_provider_budget SET allocated_nanodollars=allocated_nanodollars-$1 WHERE provider='anthropic'`, reserved); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE ai_credit_operations SET state='released' WHERE id=$1 AND NOT EXISTS(SELECT 1 FROM ai_credit_attempts WHERE operation_id=$1 AND state<>'released')`, operation); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
