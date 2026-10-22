package repository

import (
	"context"
	"errors"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Legacy AI requests have a 90-second enforced deadline; pending reservations
// live 120 seconds so an active provider call cannot outlive its counted slot.
const AIReservationLifetime = 120 * time.Second
const aiCountWithPending = `SELECT
 (SELECT count(*) FROM ai_usage WHERE user_id=$1 AND usage_type IN ('match_score','resume_autofill_parse') AND created_at>=date_trunc('month',now())) +
 (SELECT count(*) FROM ai_quota_reservations WHERE user_id=$1 AND expires_at>now())`

func lockAIQuota(ctx context.Context, tx pgx.Tx, userID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "ai-quota:"+userID)
	return err
}
func checkAIQuota(ctx context.Context, tx pgx.Tx, userID string, limit int) error {
	var count int
	if err := tx.QueryRow(ctx, aiCountWithPending, userID).Scan(&count); err != nil {
		return err
	}
	if limit >= 0 && count >= limit {
		return model.ErrLimitReached
	}
	return nil
}

// ReserveAIInTransaction coordinates durable operations with legacy pending
// reservations. Existing ledger rows and plan prices remain unchanged.
func ReserveAIInTransaction(ctx context.Context, tx pgx.Tx, userID string, limit int, receiptIDs ...string) error {
	if err := lockAIQuota(ctx, tx, userID); err != nil {
		return err
	}
	if err := checkAIQuota(ctx, tx, userID, limit); err != nil {
		return err
	}
	id := uuid.NewString()
	if len(receiptIDs) > 0 {
		id = receiptIDs[0]
	}
	_, err := tx.Exec(ctx, `INSERT INTO ai_usage(id,user_id,usage_type) VALUES($1,$2,'match_score')`, id, userID)
	return err
}
func (r *SubscriptionRepository) ReserveAI(ctx context.Context, userID string, limit int) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if err := lockAIQuota(ctx, tx, userID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM ai_quota_reservations WHERE user_id=$1 AND expires_at<=now()`, userID); err != nil {
		return "", err
	}
	if err := checkAIQuota(ctx, tx, userID, limit); err != nil {
		return "", err
	}
	id := uuid.NewString()
	_, err = tx.Exec(ctx, `INSERT INTO ai_quota_reservations(id,user_id,expires_at) VALUES($1,$2,now()+$3 * interval '1 second')`, id, userID, int64(AIReservationLifetime/time.Second))
	if err != nil {
		return "", err
	}
	return id, tx.Commit(ctx)
}
func (r *SubscriptionRepository) CommitAIReservation(ctx context.Context, userID, id, usageType string) error {
	if usageType != "match_score" && usageType != "resume_autofill_parse" {
		return errors.New("invalid ai usage type")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockAIQuota(ctx, tx, userID); err != nil {
		return err
	}
	var active bool
	err = tx.QueryRow(ctx, `SELECT expires_at>now() FROM ai_quota_reservations WHERE id=$1 AND user_id=$2 FOR UPDATE`, id, userID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		var consumed bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ai_usage WHERE id=$1 AND user_id=$2)`, id, userID).Scan(&consumed); err != nil {
			return err
		}
		if consumed {
			return tx.Commit(ctx)
		}
		return model.ErrLimitReached
	}
	if err != nil {
		return err
	}
	if !active {
		return model.ErrLimitReached
	}
	_, err = tx.Exec(ctx, `INSERT INTO ai_usage(id,user_id,usage_type) VALUES($1,$2,$3) ON CONFLICT(id) DO NOTHING`, id, userID, usageType)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM ai_quota_reservations WHERE id=$1 AND user_id=$2`, id, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *SubscriptionRepository) ReleaseAI(ctx context.Context, userID, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM ai_quota_reservations WHERE user_id=$1 AND id=$2`, userID, id)
	return err
}
