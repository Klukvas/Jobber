package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/andreypavlenko/jobber/modules/subscriptions/model"
	"github.com/jackc/pgx/v5"
)

func validCreditGrant(g model.PaidCreditGrant) bool {
	duration := g.PeriodEnd.Sub(g.PeriodStart)
	return g.UserID != "" && g.Provider == "creem" && (g.Mode == "live" || g.Mode == "test") && g.TransactionID != "" && g.SubscriptionID != "" && g.CustomerID != "" && g.ProductID != "" && !g.PeriodStart.IsZero() && duration >= 27*24*time.Hour && duration <= 32*24*time.Hour
}

// ApplyPaidCreditEvent keeps the payment grant and lifecycle claim atomic. A
// later lifecycle update may supersede the state while this genuine paid period
// still needs its grant; grant deduplication therefore has its own payment keys.
func (r *SubscriptionRepository) ApplyPaidCreditEvent(ctx context.Context, eventID, eventType string, sub *model.Subscription, g model.PaidCreditGrant) (model.WebhookApplyOutcome, error) {
	if !validCreditGrant(g) || sub.UserID != g.UserID || sub.ExternalSubscriptionID == nil || *sub.ExternalSubscriptionID != g.SubscriptionID || sub.ExternalAccountID == nil || *sub.ExternalAccountID != g.CustomerID {
		return "", model.ErrCreditUnavailable
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "credit-payment:creem:"+g.Mode+":"+g.TransactionID); err != nil {
		return "", err
	}
	if err := lockAIQuota(ctx, tx, g.UserID); err != nil {
		return "", err
	}
	conflict, err := linkConflicts(ctx, tx, sub)
	if err != nil {
		return "", err
	}
	if conflict {
		return model.WebhookLinkConflict, nil
	}
	var claimed, applied bool
	err = tx.QueryRow(ctx, applySubscriptionEventSQL, eventID, eventType, sub.UserID, sub.ExternalSubscriptionID, sub.ExternalAccountID, sub.Status, sub.Plan, sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.CancelAt, sub.LastEventAt).Scan(&claimed, &applied)
	if isAccountUniqueViolation(err) {
		return model.WebhookAccountConflict, nil
	}
	if err != nil {
		return "", fmt.Errorf("apply paid lifecycle: %w", err)
	}
	if !claimed {
		return model.WebhookDuplicate, tx.Commit(ctx)
	}
	// Refunded/disputed money is never usable, including a refund delivered first.
	_, err = tx.Exec(ctx, `INSERT INTO credit_grants(user_id,kind,credits,provider,mode,transaction_id,subscription_id,customer_id,product_id,period_start,period_end,revoked_at)
 VALUES($1,'pro',500,$2,$3,$4,$5,$6,$7,$8,$9,CASE WHEN EXISTS(SELECT 1 FROM credit_payment_holds WHERE provider=$2 AND mode=$3 AND transaction_id=$4) THEN now() ELSE NULL END)
 ON CONFLICT DO NOTHING`, g.UserID, g.Provider, g.Mode, g.TransactionID, g.SubscriptionID, g.CustomerID, g.ProductID, g.PeriodStart, g.PeriodEnd)
	if err != nil {
		return "", fmt.Errorf("grant paid credits: %w", err)
	}
	// A colliding transaction or period must describe the SAME owner and period.
	var same bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM credit_grants WHERE provider=$1 AND mode=$2 AND transaction_id=$3 AND user_id=$4 AND subscription_id=$5 AND period_start=$6 AND period_end=$7) OR EXISTS(SELECT 1 FROM credit_grants WHERE provider=$1 AND mode=$2 AND subscription_id=$5 AND period_start=$6 AND period_end=$7 AND user_id=$4)`, g.Provider, g.Mode, g.TransactionID, g.UserID, g.SubscriptionID, g.PeriodStart, g.PeriodEnd).Scan(&same)
	if err != nil {
		return "", err
	}
	if !same {
		return "", model.ErrBillingAccountTaken
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	if !applied {
		return model.WebhookSuperseded, nil
	}
	return model.WebhookApplied, nil
}

// HoldCreditPayment is deliberately conservative for partial refunds too. The
// immutable spend history and provider liability remain untouched.
func (r *SubscriptionRepository) HoldCreditPayment(ctx context.Context, eventID, eventType, mode, transaction string) error {
	if eventID == "" || transaction == "" || (mode != "live" && mode != "test") {
		return model.ErrCreditUnavailable
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// The payment lock serializes an early hold with any paid grant for this transaction.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "credit-payment:creem:"+mode+":"+transaction); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO credit_payment_holds(provider,mode,transaction_id,reason) VALUES('creem',$1,$2,$3) ON CONFLICT DO NOTHING`, mode, transaction, eventType)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE credit_grants SET revoked_at=COALESCE(revoked_at,now()) WHERE provider='creem' AND mode=$1 AND transaction_id=$2`, mode, transaction); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO webhook_events(event_id,event_type) VALUES($1,$2) ON CONFLICT DO NOTHING`, eventID, eventType)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// HasConfirmedPaidPeriod distinguishes purchased legacy state from an actual
// validated payment grant. SQL failures must not turn into paid access.
func (r *SubscriptionRepository) HasConfirmedPaidPeriod(ctx context.Context, user string) (bool, error) {
	var paid bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM credit_grants g JOIN subscriptions s ON s.user_id=g.user_id AND s.external_subscription_id=g.subscription_id WHERE g.user_id=$1 AND g.kind='pro' AND g.revoked_at IS NULL AND g.period_start<=now() AND g.period_end>now() AND s.status='active') OR EXISTS(SELECT 1 FROM credit_grants g JOIN manual_access_grants m ON m.grant_id=g.id WHERE g.user_id=$1 AND g.kind='manual_pro' AND g.revoked_at IS NULL AND g.period_start<=now() AND g.period_end>now())`, user).Scan(&paid)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return paid, err
}
