package repository

import (
	"context"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/jackc/pgx/v5"
)

// FinalizeCreditResultInTransaction couples a usable saved result with its charge.
// It never releases provider attempts or their monetary liability. Callers hold
// their workflow fence and use the same transaction that stores the result.
func FinalizeCreditResultInTransaction(ctx context.Context, tx pgx.Tx, user, key string, success bool, reason string) error {
	if err := lockAIQuota(ctx, tx, user); err != nil {
		return err
	}
	if !success {
		_, err := tx.Exec(ctx, `UPDATE ai_credit_operations SET charge_state='released',charge_reason=$3,charge_finished_at=now() WHERE user_id=$1 AND operation_key=$2 AND charge_state='reserved'`, user, key, reason)
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE ai_credit_operations SET charge_state='charged',charge_reason='usable_result',charge_finished_at=COALESCE(charge_finished_at,now()) WHERE user_id=$1 AND operation_key=$2 AND charge_state IN ('reserved','charged')`, user, key)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return model.ErrAIOperationConflict
	}
	return nil
}

func (r *SubscriptionRepository) CompleteCreditOperation(ctx context.Context, op model.CreditOperation) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var matches bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ai_credit_operations WHERE user_id=$1 AND operation_key=$2 AND kind=$3 AND fingerprint=$4 AND credits=$5)`, op.UserID, op.Key, op.Kind, op.Fingerprint, op.Credits).Scan(&matches); err != nil {
		return err
	}
	// A cache-only success made no provider call and has no charge to commit.
	if !matches {
		return nil
	}
	if err := FinalizeCreditResultInTransaction(ctx, tx, op.UserID, op.Key, true, ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RejectCreditOperation is explicit application failure cleanup. Unlike unsent
// attempt cleanup it refunds the user even when the provider already incurred cost.
func (r *SubscriptionRepository) RejectCreditOperation(ctx context.Context, op model.CreditOperation) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockAIQuota(ctx, tx, op.UserID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE ai_credit_operations SET charge_state='released',charge_reason='rejected_result',charge_finished_at=now() WHERE user_id=$1 AND operation_key=$2 AND kind=$3 AND fingerprint=$4 AND credits=$5 AND charge_state='reserved'`, op.UserID, op.Key, op.Kind, op.Fingerprint, op.Credits)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RestoreSavedResultCharge is only for a locked durable parsed-result receipt.
// No provider call is admitted: the original grant funds the already-produced
// result, including across period expiry. Provider attempts never change grants.
func RestoreSavedResultCharge(ctx context.Context, tx pgx.Tx, user, key string) error {
	if err := lockAIQuota(ctx, tx, user); err != nil {
		return err
	}
	var state, grant string
	var credits int
	if err := tx.QueryRow(ctx, `SELECT charge_state,grant_id::text,credits FROM ai_credit_operations WHERE user_id=$1 AND operation_key=$2 FOR UPDATE`, user, key).Scan(&state, &grant, &credits); err != nil {
		return err
	}
	if state == "reserved" || state == "charged" {
		return nil
	}
	if state != "released" {
		return model.ErrAIOperationConflict
	}
	var remaining int
	if err := tx.QueryRow(ctx, `SELECT credits-COALESCE((SELECT sum(credits) FROM ai_credit_operations WHERE grant_id=$1 AND charge_state<>'released'),0)::integer FROM credit_grants WHERE id=$1 AND user_id=$2`, grant, user).Scan(&remaining); err != nil {
		return err
	}
	if remaining < credits {
		return model.ErrCreditUnavailable
	}
	_, err := tx.Exec(ctx, `UPDATE ai_credit_operations SET charge_state='reserved',charge_reason='',charge_finished_at=NULL WHERE user_id=$1 AND operation_key=$2`, user, key)
	return err
}
