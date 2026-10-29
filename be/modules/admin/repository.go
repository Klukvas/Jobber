package admin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}
type Repository struct{ db DB }

func NewRepository(db DB) *Repository { return &Repository{db: db} }

const userColumns = `u.id::text,u.email,u.name,u.locale,u.email_verified,u.created_at,COALESCE(s.plan,'free'),COALESCE(s.status,'free'),(SELECT max(g.period_end) FROM credit_grants g JOIN manual_access_grants m ON m.grant_id=g.id WHERE g.user_id=u.id AND g.kind='manual_pro' AND g.revoked_at IS NULL AND g.period_start<=now() AND g.period_end>now())`
const userJoin = ` FROM users u LEFT JOIN subscriptions s ON s.user_id=u.id `

func scanUser(row pgx.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Locale, &u.EmailVerified, &u.CreatedAt, &u.ProviderPlan, &u.ProviderStatus, &u.ManualUntil)
	return u, err
}
func (r *Repository) List(ctx context.Context, search string, page, limit int) (UserList, error) {
	result := UserList{Users: []User{}, Page: page, Limit: limit}
	// strpos makes wildcard characters literal and keeps the query parameterized.
	filter := ` WHERE strpos(lower(u.email),lower($1))>0 OR strpos(lower(u.name),lower($1))>0 `
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM users u`+filter, search).Scan(&result.Total); err != nil {
		return result, fmt.Errorf("count admin users: %w", err)
	}
	rows, err := r.db.Query(ctx, `SELECT `+userColumns+userJoin+filter+` ORDER BY u.created_at DESC,u.id LIMIT $2 OFFSET $3`, search, limit, (page-1)*limit)
	if err != nil {
		return result, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return result, err
		}
		result.Users = append(result.Users, u)
	}
	return result, rows.Err()
}
func (r *Repository) Detail(ctx context.Context, id string) (Detail, error) {
	d := Detail{ManualGrants: []Grant{}}
	u, err := scanUser(r.db.QueryRow(ctx, `SELECT `+userColumns+userJoin+` WHERE u.id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return d, ErrNotFound
	}
	if err != nil {
		return d, err
	}
	d.User = u
	err = r.db.QueryRow(ctx, `SELECT external_subscription_id,external_account_id,current_period_end,cancel_at FROM subscriptions WHERE user_id=$1`, id).Scan(&d.ProviderSubscriptionID, &d.ProviderAccountID, &d.ProviderPeriodEnd, &d.ProviderCancelAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return d, err
	}
	d.EffectivePlan = u.ProviderPlan
	if (u.ProviderStatus != "active" && u.ProviderStatus != "past_due") || (u.ProviderStatus == "active" && d.ProviderCancelAt != nil && time.Now().After(d.ProviderCancelAt.Add(48*time.Hour))) {
		d.EffectivePlan = "free"
	}
	if d.EffectivePlan == "free" && u.ManualUntil != nil {
		d.EffectivePlan = "pro"
	}
	rows, err := r.db.Query(ctx, `SELECT g.id::text,g.period_start,g.period_end,g.revoked_at,m.actor_id::text,m.reason,m.revoked_by::text,m.revoke_reason,g.credits,(g.revoked_at IS NULL AND g.period_start<=now() AND g.period_end>now()),COALESCE((SELECT sum(o.credits) FROM ai_credit_operations o WHERE o.grant_id=g.id AND o.state<>'released'),0)::int FROM credit_grants g JOIN manual_access_grants m ON m.grant_id=g.id WHERE g.user_id=$1 ORDER BY g.created_at DESC LIMIT 50`, id)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var g Grant
		if err := rows.Scan(&g.ID, &g.Start, &g.End, &g.RevokedAt, &g.ActorID, &g.Reason, &g.RevokedBy, &g.RevokeReason, &g.Credits, &g.Active, &g.Used); err != nil {
			return d, err
		}
		d.ManualGrants = append(d.ManualGrants, g)
	}
	return d, rows.Err()
}
func (r *Repository) Update(ctx context.Context, id string, p ProfileUpdate) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET name=$2,locale=$3,updated_at=now() WHERE id=$1`, id, p.Name, p.Locale)
	if err != nil {
		return fmt.Errorf("update admin profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func lockUser(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "ai-quota:"+id)
	return err
}
func (r *Repository) Grant(ctx context.Context, user, actor string, p GrantRequest) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockUser(ctx, tx, user); err != nil {
		return err
	}
	// The immutable UUID is the idempotency key. Replays never extend a period,
	// replenish credits, or reactivate a revoked grant.
	var owner, oldActor, reason string
	var days int
	err = tx.QueryRow(ctx, `SELECT g.user_id::text,m.actor_id::text,m.reason,extract(epoch FROM(g.period_end-g.period_start))::int/86400 FROM credit_grants g JOIN manual_access_grants m ON m.grant_id=g.id WHERE g.id=$1`, p.ID).Scan(&owner, &oldActor, &reason, &days)
	if err == nil {
		if owner != user || oldActor != actor || reason != p.Reason || days != p.Days {
			return ErrConflict
		}
		return tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var exists, active bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1),EXISTS(SELECT 1 FROM credit_grants WHERE user_id=$1 AND kind='manual_pro' AND revoked_at IS NULL AND period_end>now())`, user).Scan(&exists, &active)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	if active {
		return ErrConflict
	}
	_, err = tx.Exec(ctx, `INSERT INTO credit_grants(id,user_id,kind,credits,period_start,period_end) VALUES($1,$2,'manual_pro',500,now(),now()+make_interval(days=>$3))`, p.ID, user, p.Days)
	if err != nil {
		return fmt.Errorf("create manual credits: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO manual_access_grants(grant_id,actor_id,reason) VALUES($1,$2,$3)`, p.ID, actor, p.Reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *Repository) Revoke(ctx context.Context, user, grant, actor, reason string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockUser(ctx, tx, user); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `UPDATE credit_grants SET revoked_at=COALESCE(revoked_at,now()) WHERE id=$1 AND user_id=$2 AND kind='manual_pro' AND EXISTS(SELECT 1 FROM manual_access_grants WHERE grant_id=$1)`, grant, user)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_, err = tx.Exec(ctx, `UPDATE manual_access_grants SET revoked_by=$2,revoke_reason=$3 WHERE grant_id=$1 AND revoked_by IS NULL`, grant, actor, reason)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
